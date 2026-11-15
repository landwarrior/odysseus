package web

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"odysseus/internal/config"
	"odysseus/internal/db"
	"odysseus/internal/store"
)

func TestForecastFlow(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	database, err := db.Open("sqlite", filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.SQL.Close() })
	if err := db.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	st := store.New(database)
	adminPassword, _, err := st.Seed(ctx)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(st, config.Config{
		DataDir: dir, Driver: "sqlite", StandardMinutes: 480,
		SessionKey: bytes.Repeat([]byte("k"), 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	admin := newClient(t, ts.URL)

	body := admin.submit(t, "/login", url.Values{"code": {"admin"}, "password": {adminPassword}})
	body = admin.submit(t, "/password", url.Values{
		"csrf": {csrfOf(t, body)}, "current_password": {adminPassword},
		"new_password": {"adminpass1"}, "confirm_password": {"adminpass1"},
	})
	if !strings.Contains(body, "進行中のプロジェクトはありません") {
		t.Fatalf("home after password change:\n%s", body)
	}

	body = admin.submit(t, "/people", url.Values{
		"csrf": {csrfOf(t, body)}, "code": {"m01"}, "name": {"山田"},
		"password": {"memberpass"}, "active": {"1"},
	})
	if !strings.Contains(body, "時間単価") {
		t.Fatalf("person page:\n%s", body)
	}
	roles, err := st.ListRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var pg string
	for _, role := range roles {
		if role.Name == "PG" {
			pg = role.ID
		}
	}
	body = admin.submit(t, "/people/"+personID(t, st, "m01")+"/rates", url.Values{
		"csrf": {csrfOf(t, body)}, "role_id": {pg}, "from_date": {"2026-04-01"}, "hourly_yen": {"5000"},
	})
	processes, err := st.ListProcesses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var build string
	for _, process := range processes {
		if process.Name == "製造" {
			build = process.ID
		}
	}
	body = admin.submit(t, "/projects", url.Values{
		"csrf": {csrfOf(t, body)}, "code": {"P1"}, "name": {"予実案件"}, "order_amount": {"1000000"},
		"status": {"active"}, "start_date": {"2026-04-01"}, "end_date": {"2026-04-30"},
		"process_id": {build}, "line_start": {"2026-04-01"}, "line_end": {"2026-04-30"}, "line_hours": {"80"},
	})
	project := projectByCode(t, st, "P1")
	lines, err := st.ListProcessLines(ctx, project.ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("lines: %+v %v", lines, err)
	}
	body = admin.submit(t, "/projects/"+project.ID+"/assignments", url.Values{
		"csrf": {csrfOf(t, body)}, "project_process_id": {lines[0].ID},
		"person_id": {personID(t, st, "m01")}, "role_id": {pg},
		"start_date": {"2026-04-01"}, "end_date": {"2026-04-30"}, "hours": {"80"},
	})
	if !strings.Contains(body, "¥400,000") {
		t.Fatalf("assignment cost missing:\n%s", body)
	}

	member := newClient(t, ts.URL)
	body = member.submit(t, "/login", url.Values{"code": {"m01"}, "password": {"memberpass"}})
	body = member.submit(t, "/password", url.Values{
		"csrf": {csrfOf(t, body)}, "current_password": {"memberpass"},
		"new_password": {"memberpass2"}, "confirm_password": {"memberpass2"},
	})
	if strings.Contains(body, "粗利見込み") {
		t.Fatal("member home shows money")
	}
	status, forbidden := member.get(t, "/projects")
	if status != http.StatusForbidden || !strings.Contains(forbidden, "管理者権限がありません") {
		t.Fatalf("member projects: %d %s", status, forbidden)
	}
	assignments, err := st.ListAssignmentsByProject(ctx, project.ID)
	if err != nil || len(assignments) != 1 {
		t.Fatal(err)
	}
	timePage := member.getBody(t, "/time/"+project.ID+"?month=2026-04")
	member.submit(t, "/time/"+project.ID+"?month=2026-04", url.Values{
		"csrf": {csrfOf(t, timePage)}, "work_date": {"2026-04-02"},
		"minutes_" + assignments[0].ID: {"8"},
	})

	forecast := admin.getBody(t, "/projects/"+project.ID+"/forecast")
	for _, want := range []string{"¥1,000,000", "¥400,000", "¥40,000", "¥360,000", "¥600,000"} {
		if !strings.Contains(forecast, want) {
			t.Fatalf("forecast missing %s\n%s", want, forecast)
		}
	}
	home := admin.getBody(t, "/")
	if !strings.Contains(home, "¥600,000") {
		t.Fatalf("home missing gross:\n%s", home)
	}
}

type client struct {
	t    *testing.T
	base string
	http *http.Client
}

func newClient(t *testing.T, base string) *client {
	t.Helper()
	return &client{t: t, base: base, http: &http.Client{Jar: &memJar{cookies: map[string]*http.Cookie{}}}}
}

func (c *client) submit(t *testing.T, path string, form url.Values) string {
	t.Helper()
	res, err := c.http.PostForm(c.base+path, form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode >= 400 {
		t.Fatalf("POST %s -> %d\n%s", path, res.StatusCode, body)
	}
	return string(body)
}

func (c *client) get(t *testing.T, path string) (int, string) {
	t.Helper()
	res, err := c.http.Get(c.base + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(body)
}

func (c *client) getBody(t *testing.T, path string) string {
	t.Helper()
	status, body := c.get(t, path)
	if status != http.StatusOK {
		t.Fatalf("GET %s -> %d\n%s", path, status, body)
	}
	return body
}

func csrfOf(t *testing.T, body string) string {
	t.Helper()
	const key = `name="csrf" value="`
	i := strings.Index(body, key)
	if i < 0 {
		t.Fatalf("csrf not found:\n%s", body)
	}
	rest := body[i+len(key):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		t.Fatal("csrf truncated")
	}
	return rest[:j]
}

func personID(t *testing.T, st *store.Store, code string) string {
	t.Helper()
	people, err := st.ListPeople(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, person := range people {
		if person.Code == code {
			return person.ID
		}
	}
	t.Fatalf("person %s not found", code)
	return ""
}

func projectByCode(t *testing.T, st *store.Store, code string) store.Project {
	t.Helper()
	projects, err := st.ListProjects(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, project := range projects {
		if project.Code == code {
			return project
		}
	}
	t.Fatalf("project %s not found", code)
	return store.Project{}
}

func TestRolePageSeparatesAddAndReorder(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	database, err := db.Open("sqlite", filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.SQL.Close() })
	if err := db.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	st := store.New(database)
	adminPassword, _, err := st.Seed(ctx)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(st, config.Config{
		DataDir: dir, Driver: "sqlite", StandardMinutes: 480,
		SessionKey: bytes.Repeat([]byte("k"), 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	admin := newClient(t, ts.URL)
	body := admin.submit(t, "/login", url.Values{"code": {"admin"}, "password": {adminPassword}})
	admin.submit(t, "/password", url.Values{
		"csrf": {csrfOf(t, body)}, "current_password": {adminPassword},
		"new_password": {"adminpass1"}, "confirm_password": {"adminpass1"},
	})
	page := admin.getBody(t, "/roles")
	for _, want := range []string{"役割を追加", "登録済みの役割", "上へ", "下へ", "disabled>上へ", `class="role-page"`} {
		if !strings.Contains(page, want) {
			t.Fatalf("roles page missing %s", want)
		}
	}
	if strings.Contains(page, "並び順") && strings.Contains(page, `name="sort_order"`) {
		t.Fatal("roles page should not ask for a sort number")
	}
	if strings.Index(page, `value="PG"`) > strings.Index(page, `value="SE"`) {
		t.Fatal("PG should be listed before SE")
	}
	roles, err := st.ListRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var seID string
	for _, role := range roles {
		if role.Name == "SE" {
			seID = role.ID
		}
	}
	moved := admin.submit(t, "/roles/"+seID+"/move", url.Values{
		"csrf": {csrfOf(t, page)}, "direction": {"up"},
	})
	if strings.Index(moved, `value="SE"`) > strings.Index(moved, `value="PG"`) {
		t.Fatal("SE should move above PG")
	}
	added := admin.submit(t, "/roles", url.Values{
		"csrf": {csrfOf(t, moved)}, "name": {"顧問"}, "active": {"1"},
	})
	if strings.Index(added, `value="PM"`) > strings.Index(added, `value="顧問"`) {
		t.Fatal("new role should be listed last")
	}
}

func TestProcessPageSeparatesAddAndReorder(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	database, err := db.Open("sqlite", filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.SQL.Close() })
	if err := db.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	st := store.New(database)
	adminPassword, _, err := st.Seed(ctx)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(st, config.Config{
		DataDir: dir, Driver: "sqlite", StandardMinutes: 480,
		SessionKey: bytes.Repeat([]byte("k"), 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	admin := newClient(t, ts.URL)
	body := admin.submit(t, "/login", url.Values{"code": {"admin"}, "password": {adminPassword}})
	admin.submit(t, "/password", url.Values{
		"csrf": {csrfOf(t, body)}, "current_password": {adminPassword},
		"new_password": {"adminpass1"}, "confirm_password": {"adminpass1"},
	})
	page := admin.getBody(t, "/processes")
	for _, want := range []string{"工程を追加", "登録済みの工程", "上へ", "下へ", "disabled>上へ"} {
		if !strings.Contains(page, want) {
			t.Fatalf("processes page missing %s", want)
		}
	}
	if strings.Index(page, `value="提案"`) > strings.Index(page, `value="要件定義"`) {
		t.Fatal("提案 should be listed before 要件定義")
	}
	processes, err := st.ListProcesses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var requirementID string
	for _, process := range processes {
		if process.Name == "要件定義" {
			requirementID = process.ID
		}
	}
	moved := admin.submit(t, "/processes/"+requirementID+"/move", url.Values{
		"csrf": {csrfOf(t, page)}, "direction": {"up"},
	})
	if strings.Index(moved, `value="要件定義"`) > strings.Index(moved, `value="提案"`) {
		t.Fatal("要件定義 should move above 提案")
	}
}

type memJar struct{ cookies map[string]*http.Cookie }

func (j *memJar) SetCookies(u *url.URL, cs []*http.Cookie) {
	for _, c := range cs {
		if c.MaxAge < 0 {
			delete(j.cookies, c.Name)
			continue
		}
		j.cookies[c.Name] = c
	}
}

func (j *memJar) Cookies(u *url.URL) []*http.Cookie {
	out := make([]*http.Cookie, 0, len(j.cookies))
	for _, c := range j.cookies {
		out = append(out, c)
	}
	return out
}
