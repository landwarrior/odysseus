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
	if !strings.Contains(body, "単価") {
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
		"csrf": {csrfOf(t, body)}, "role_id": {pg}, "from_date": {"2026-04-01"},
		"hourly_yen": {"5000"}, "monthly_yen": {"400000"},
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
		"process_id": {build}, "line_start": {"2026-04-01"}, "line_end": {"2026-04-30"}, "line_days": {"10.0"},
	})
	project := projectByCode(t, st, "P1")
	lines, err := st.ListProcessLines(ctx, project.ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("lines: %+v %v", lines, err)
	}
	body = admin.submit(t, "/projects/"+project.ID+"/assignments", url.Values{
		"csrf": {csrfOf(t, body)}, "project_process_id": {lines[0].ID},
		"person_id": {personID(t, st, "m01")}, "role_" + lines[0].ID: {pg},
		"rate_kind": {"monthly"}, "start_date": {"2026-04-01"}, "end_date": {"2026-04-30"},
	})
	if !strings.Contains(body, "予実で按分") {
		t.Fatalf("monthly assignment should defer planned cost to forecast:\n%s", body)
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
	status, projectsPage := member.get(t, "/projects")
	if status != http.StatusOK {
		t.Fatalf("member projects: %d %s", status, projectsPage)
	}
	if !strings.Contains(projectsPage, "予実案件") {
		t.Fatalf("member should see schedule:\n%s", projectsPage)
	}
	if strings.Contains(projectsPage, "effort-input") {
		t.Fatal("member must not edit schedule cells")
	}
	if strings.Contains(projectsPage, "新規作成") || strings.Contains(projectsPage, "/projects/new") {
		t.Fatal("member must not see project create")
	}
	status, schedulePage := member.get(t, "/projects/"+project.ID+"/schedule")
	if status != http.StatusOK {
		t.Fatalf("member project schedule: %d %s", status, schedulePage)
	}
	if strings.Contains(schedulePage, "effort-input") {
		t.Fatal("member must not edit project schedule cells")
	}
	status, forbidden := member.get(t, "/projects/"+project.ID)
	if status != http.StatusForbidden || !strings.Contains(forbidden, "この案件を編集する権限がありません") {
		t.Fatalf("member project edit: %d %s", status, forbidden)
	}
	allocRes, err := member.http.PostForm(ts.URL+"/projects/allocations", url.Values{
		"csrf": {csrfOf(t, projectsPage)}, "year_month": {"2026-04"}, "decade": {"1"}, "effort": {"1.0"},
		"project_process_id": {lines[0].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	allocBody, err := io.ReadAll(allocRes.Body)
	allocRes.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if allocRes.StatusCode != http.StatusForbidden || !strings.Contains(string(allocBody), "この案件を編集する権限がありません") {
		t.Fatalf("member allocation save: %d %s", allocRes.StatusCode, allocBody)
	}
	assignments, err := st.ListAssignmentsByProject(ctx, project.ID)
	if err != nil || len(assignments) != 1 {
		t.Fatal(err)
	}
	memberID := personID(t, st, "m01")
	if err := st.ReplaceTasksByProcess(ctx, project.ID, map[string][]store.TaskWrite{
		build: {{Name: "画面", PlannedMinutes: 40 * 60}, {Name: "API", PlannedMinutes: 40 * 60}},
	}); err != nil {
		t.Fatal(err)
	}
	tasks, err := st.ListTasksByProject(ctx, project.ID)
	if err != nil || len(tasks) != 2 {
		t.Fatalf("tasks: %+v %v", tasks, err)
	}

	adminTime := admin.getBody(t, "/time?date=2026-04-01&person="+memberID)
	if !strings.Contains(adminTime, "P1") {
		t.Fatalf("admin time form missing project:\n%s", adminTime)
	}
	if !strings.Contains(adminTime, "入力対象: m01 山田") && !strings.Contains(adminTime, "m01 山田") {
		t.Fatalf("admin time form missing person:\n%s", adminTime)
	}
	if !strings.Contains(adminTime, "画面") {
		t.Fatalf("admin time form missing task column:\n%s", adminTime)
	}
	admin.submit(t, "/time", url.Values{
		"csrf": {csrfOf(t, adminTime)}, "person_id": {memberID}, "work_date": {"2026-04-01"},
		"minutes_" + assignments[0].ID + "_" + tasks[0].ID: {"4"},
	})

	timePage := member.getBody(t, "/time?date=2026-04-02")
	member.submit(t, "/time", url.Values{
		"csrf": {csrfOf(t, timePage)}, "work_date": {"2026-04-02"},
		"minutes_" + assignments[0].ID + "_" + tasks[1].ID: {"8"},
	})
	forecastPage := admin.getBody(t, "/projects/"+project.ID+"/forecast")
	if !strings.Contains(forecastPage, "タスク") || !strings.Contains(forecastPage, "画面") {
		t.Fatalf("forecast missing tasks:\n%s", forecastPage)
	}

	forecast := admin.getBody(t, "/projects/"+project.ID+"/forecast")
	// 月間単価があるので実績原価は暦月分（4月1ヶ月 = ¥400,000）。時間実績は金額に使わない。
	for _, want := range []string{"¥1,000,000", "残予定原価", "粗利見込み", "¥600,000"} {
		if !strings.Contains(forecast, want) {
			t.Fatalf("forecast missing %s\n%s", want, forecast)
		}
	}
	home := admin.getBody(t, "/")
	for _, want := range []string{"進行中のプロジェクト", "P1", "¥600,000", "編集", "アサイン", "予実"} {
		if !strings.Contains(home, want) {
			t.Fatalf("home missing %s:\n%s", want, home)
		}
	}
}

func TestProjectManagerFlow(t *testing.T) {
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
	body = admin.submit(t, "/people", url.Values{
		"csrf": {csrfOf(t, body)}, "code": {"mgr1"}, "name": {"管理"}, "name_kana": {"次郎"},
		"password": {"managerpass"}, "active": {"1"}, "is_manager": {"1"},
	})
	body = admin.submit(t, "/people", url.Values{
		"csrf": {csrfOf(t, body)}, "code": {"m02"}, "name": {"作業"}, "name_kana": {"花子"},
		"password": {"workerpass"}, "active": {"1"},
	})
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
	mgrID := personID(t, st, "mgr1")
	workerID := personID(t, st, "m02")
	body = admin.submit(t, "/people/"+mgrID+"/rates", url.Values{
		"csrf": {csrfOf(t, body)}, "role_id": {pg}, "from_date": {"2026-04-01"}, "hourly_yen": {"5000"},
	})
	body = admin.submit(t, "/people/"+workerID+"/rates", url.Values{
		"csrf": {csrfOf(t, body)}, "role_id": {pg}, "from_date": {"2026-04-01"}, "hourly_yen": {"4000"},
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
		"csrf": {csrfOf(t, body)}, "code": {"PM1"}, "name": {"責任者案件"}, "order_amount": {"800000"},
		"status": {"active"}, "start_date": {"2026-04-01"}, "end_date": {"2026-04-30"},
		"process_id": {build}, "line_start": {"2026-04-01"}, "line_end": {"2026-04-30"}, "line_days": {"5.0"},
		"manager_id": {mgrID},
	})
	project := projectByCode(t, st, "PM1")
	ok, err := st.IsProjectManager(ctx, project.ID, mgrID)
	if err != nil || !ok {
		t.Fatalf("manager not saved: %v %v", ok, err)
	}
	body = admin.submit(t, "/projects", url.Values{
		"csrf": {csrfOf(t, body)}, "code": {"PM2"}, "name": {"別案件"}, "order_amount": {"100000"},
		"status": {"active"}, "start_date": {"2026-04-01"}, "end_date": {"2026-04-30"},
		"process_id": {build}, "line_start": {"2026-04-01"}, "line_end": {"2026-04-30"}, "line_days": {"1.0"},
	})
	other := projectByCode(t, st, "PM2")
	lines, err := st.ListProcessLines(ctx, project.ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("lines: %+v %v", lines, err)
	}
	body = admin.submit(t, "/projects/"+project.ID+"/assignments", url.Values{
		"csrf": {csrfOf(t, body)}, "project_process_id": {lines[0].ID},
		"person_id": {workerID}, "role_" + lines[0].ID: {pg},
		"rate_kind": {"hourly"}, "start_date": {"2026-04-01"}, "end_date": {"2026-04-30"},
	})

	manager := newClient(t, ts.URL)
	body = manager.submit(t, "/login", url.Values{"code": {"mgr1"}, "password": {"managerpass"}})
	body = manager.submit(t, "/password", url.Values{
		"csrf": {csrfOf(t, body)}, "current_password": {"managerpass"},
		"new_password": {"managerpass2"}, "confirm_password": {"managerpass2"},
	})
	if !strings.Contains(body, "担当の進行中プロジェクト") || !strings.Contains(body, "責任者案件") {
		t.Fatalf("manager home:\n%s", body)
	}
	if strings.Contains(body, "別案件") {
		t.Fatal("manager home shows unmanaged project")
	}
	status, peoplePage := manager.get(t, "/people")
	if status != http.StatusForbidden {
		t.Fatalf("manager people: %d %s", status, peoplePage)
	}
	status, editPage := manager.get(t, "/projects/"+project.ID)
	if status != http.StatusOK || !strings.Contains(editPage, "責任者案件") {
		t.Fatalf("manager project edit: %d %s", status, editPage)
	}
	if !strings.Contains(editPage, `name="manager_id"`) {
		t.Fatal("manager should edit manager list on owned project")
	}
	if strings.Contains(editPage, "m02 作業") {
		t.Fatal("non-manager person must not appear in manager candidates")
	}
	status, otherEdit := manager.get(t, "/projects/"+other.ID)
	if status != http.StatusForbidden {
		t.Fatalf("manager other project: %d %s", status, otherEdit)
	}
	projectsPage := manager.getBody(t, "/projects")
	if !strings.Contains(projectsPage, "effort-input") {
		t.Fatal("manager should edit managed schedule cells")
	}
	allocRes, err := manager.http.PostForm(ts.URL+"/projects/allocations", url.Values{
		"csrf": {csrfOf(t, projectsPage)}, "year_month": {"2026-04"}, "decade": {"1"}, "effort": {"1.0"},
		"project_process_id": {lines[0].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	allocBody, err := io.ReadAll(allocRes.Body)
	allocRes.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if allocRes.StatusCode != http.StatusOK {
		t.Fatalf("manager allocation save: %d %s", allocRes.StatusCode, allocBody)
	}
	otherLines, err := st.ListProcessLines(ctx, other.ID)
	if err != nil || len(otherLines) != 1 {
		t.Fatalf("other lines: %+v %v", otherLines, err)
	}
	allocRes, err = manager.http.PostForm(ts.URL+"/projects/allocations", url.Values{
		"csrf": {csrfOf(t, projectsPage)}, "year_month": {"2026-04"}, "decade": {"1"}, "effort": {"1.0"},
		"project_process_id": {otherLines[0].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	allocBody, err = io.ReadAll(allocRes.Body)
	allocRes.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if allocRes.StatusCode != http.StatusForbidden {
		t.Fatalf("manager other allocation: %d %s", allocRes.StatusCode, allocBody)
	}
	forecast := manager.getBody(t, "/projects/"+project.ID+"/forecast")
	if !strings.Contains(forecast, "予実") {
		t.Fatalf("manager forecast:\n%s", forecast)
	}
	timePage := manager.getBody(t, "/time?date=2026-04-01&person="+workerID)
	if !strings.Contains(timePage, "PM1") && !strings.Contains(timePage, "責任者案件") {
		t.Fatalf("manager time proxy:\n%s", timePage)
	}
	status, newProject := manager.get(t, "/projects/new")
	if status != http.StatusOK {
		t.Fatalf("manager project create: %d %s", status, newProject)
	}
	if !strings.Contains(newProject, `name="manager_id" value="`+mgrID+`"`) && !strings.Contains(newProject, `value="`+mgrID+`"`) {
		t.Fatalf("creator should be preselected as manager:\n%s", newProject)
	}
	body = manager.submit(t, "/projects", url.Values{
		"csrf": {csrfOf(t, newProject)}, "code": {"PM3"}, "name": {"自己作成"}, "order_amount": {"200000"},
		"status": {"active"}, "start_date": {"2026-05-01"}, "end_date": {"2026-05-31"},
		"process_id": {build}, "line_start": {"2026-05-01"}, "line_end": {"2026-05-31"}, "line_days": {"2.0"},
	})
	created := projectByCode(t, st, "PM3")
	ok, err = st.IsProjectManager(ctx, created.ID, mgrID)
	if err != nil || !ok {
		t.Fatalf("creator not auto-registered as manager: %v %v", ok, err)
	}
	home := manager.getBody(t, "/")
	if !strings.Contains(home, "自己作成") {
		t.Fatalf("manager home missing created project:\n%s", home)
	}
}

// TestManagerProxyDayTotalIncludesHiddenAssignments は、責任者が他人分を代行するとき、
// 担当外案件に既にある同日の実績も 24 時間上限に含めることを確認します。
func TestManagerProxyDayTotalIncludesHiddenAssignments(t *testing.T) {
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
	body = admin.submit(t, "/people", url.Values{
		"csrf": {csrfOf(t, body)}, "code": {"mgr2"}, "name": {"責任"}, "name_kana": {"者"},
		"password": {"managerpass"}, "active": {"1"}, "is_manager": {"1"},
	})
	body = admin.submit(t, "/people", url.Values{
		"csrf": {csrfOf(t, body)}, "code": {"m03"}, "name": {"作業"}, "name_kana": {"者"},
		"password": {"workerpass"}, "active": {"1"},
	})
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
	mgrID := personID(t, st, "mgr2")
	workerID := personID(t, st, "m03")
	body = admin.submit(t, "/people/"+mgrID+"/rates", url.Values{
		"csrf": {csrfOf(t, body)}, "role_id": {pg}, "from_date": {"2026-04-01"}, "hourly_yen": {"5000"},
	})
	body = admin.submit(t, "/people/"+workerID+"/rates", url.Values{
		"csrf": {csrfOf(t, body)}, "role_id": {pg}, "from_date": {"2026-04-01"}, "hourly_yen": {"4000"},
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
		"csrf": {csrfOf(t, body)}, "code": {"HT1"}, "name": {"担当案件"}, "order_amount": {"500000"},
		"status": {"active"}, "start_date": {"2026-04-01"}, "end_date": {"2026-04-30"},
		"process_id": {build}, "line_start": {"2026-04-01"}, "line_end": {"2026-04-30"}, "line_days": {"5.0"},
		"manager_id": {mgrID},
	})
	managed := projectByCode(t, st, "HT1")
	body = admin.submit(t, "/projects", url.Values{
		"csrf": {csrfOf(t, body)}, "code": {"HT2"}, "name": {"他案件"}, "order_amount": {"200000"},
		"status": {"active"}, "start_date": {"2026-04-01"}, "end_date": {"2026-04-30"},
		"process_id": {build}, "line_start": {"2026-04-01"}, "line_end": {"2026-04-30"}, "line_days": {"5.0"},
	})
	other := projectByCode(t, st, "HT2")
	managedLines, err := st.ListProcessLines(ctx, managed.ID)
	if err != nil || len(managedLines) != 1 {
		t.Fatalf("managed lines: %+v %v", managedLines, err)
	}
	otherLines, err := st.ListProcessLines(ctx, other.ID)
	if err != nil || len(otherLines) != 1 {
		t.Fatalf("other lines: %+v %v", otherLines, err)
	}
	body = admin.submit(t, "/projects/"+managed.ID+"/assignments", url.Values{
		"csrf": {csrfOf(t, body)}, "project_process_id": {managedLines[0].ID},
		"person_id": {workerID}, "role_" + managedLines[0].ID: {pg},
		"rate_kind": {"hourly"}, "start_date": {"2026-04-01"}, "end_date": {"2026-04-30"},
	})
	body = admin.submit(t, "/projects/"+other.ID+"/assignments", url.Values{
		"csrf": {csrfOf(t, body)}, "project_process_id": {otherLines[0].ID},
		"person_id": {workerID}, "role_" + otherLines[0].ID: {pg},
		"rate_kind": {"hourly"}, "start_date": {"2026-04-01"}, "end_date": {"2026-04-30"},
	})
	managedAssigns, err := st.ListAssignmentsByProject(ctx, managed.ID)
	if err != nil || len(managedAssigns) != 1 {
		t.Fatalf("managed assigns: %+v %v", managedAssigns, err)
	}
	otherAssigns, err := st.ListAssignmentsByProject(ctx, other.ID)
	if err != nil || len(otherAssigns) != 1 {
		t.Fatalf("other assigns: %+v %v", otherAssigns, err)
	}
	if err := st.SaveDay(ctx, []store.DayCell{{
		AssignmentID: otherAssigns[0].ID, WorkDate: "2026-04-10", Minutes: 16 * 60,
	}}); err != nil {
		t.Fatal(err)
	}

	manager := newClient(t, ts.URL)
	body = manager.submit(t, "/login", url.Values{"code": {"mgr2"}, "password": {"managerpass"}})
	body = manager.submit(t, "/password", url.Values{
		"csrf": {csrfOf(t, body)}, "current_password": {"managerpass"},
		"new_password": {"managerpass2"}, "confirm_password": {"managerpass2"},
	})
	timePage := manager.getBody(t, "/time?date=2026-04-10&person="+workerID)
	if !strings.Contains(timePage, "担当案件") {
		t.Fatalf("manager should see managed project:\n%s", timePage)
	}
	if strings.Contains(timePage, "他案件") {
		t.Fatal("manager must not see unmanaged project columns")
	}
	if !strings.Contains(timePage, "data-time-total>16</") {
		t.Fatalf("day total should include hidden 16h:\n%s", timePage)
	}
	res, err := manager.http.PostForm(ts.URL+"/time", url.Values{
		"csrf": {csrfOf(t, timePage)}, "person_id": {workerID}, "work_date": {"2026-04-10"},
		"minutes_" + managedAssigns[0].ID: {"9"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	rejectBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when day exceeds 24h, got %d\n%s", res.StatusCode, rejectBody)
	}
	if !strings.Contains(string(rejectBody), "1 日の入力は合計 24 時間までです") {
		t.Fatalf("missing 24h warning:\n%s", rejectBody)
	}
	okPage := manager.submit(t, "/time", url.Values{
		"csrf": {csrfOf(t, timePage)}, "person_id": {workerID}, "work_date": {"2026-04-10"},
		"minutes_" + managedAssigns[0].ID: {"8"},
	})
	if !strings.Contains(okPage, "登録しました") && !strings.Contains(manager.getBody(t, "/time?date=2026-04-10&person="+workerID), "8") {
		t.Fatalf("8h on managed + 16h hidden should save:\n%s", okPage)
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

func TestPeopleListShowsLatestRoleAndEditButton(t *testing.T) {
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
	body = admin.submit(t, "/people", url.Values{
		"csrf": {csrfOf(t, body)}, "code": {"m01"}, "name": {"山田"}, "name_kana": {"太郎"},
		"password": {"memberpass"}, "active": {"1"},
		"remarks": {"長い備考テキストを一覧でも見えるようにする"},
	})
	roles, err := st.ListRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var pg, se string
	for _, role := range roles {
		switch role.Name {
		case "PG":
			pg = role.ID
		case "SE":
			se = role.ID
		}
	}
	admin.submit(t, "/people/"+personID(t, st, "m01")+"/rates", url.Values{
		"csrf": {csrfOf(t, body)}, "role_id": {pg}, "from_date": {"2020-01-01"}, "hourly_yen": {"3000"},
	})
	body = admin.submit(t, "/people/"+personID(t, st, "m01")+"/rates", url.Values{
		"csrf": {csrfOf(t, body)}, "role_id": {se}, "from_date": {"2020-06-01"}, "hourly_yen": {"4000"},
	})
	page := admin.getBody(t, "/people")
	for _, want := range []string{
		"<th>役割</th>", "<th>備考</th>", "SE", "山田 太郎", "長い備考テキストを一覧でも見えるようにする",
		`class="button" href="/people/` + personID(t, st, "m01") + `">編集</a>`,
		`class="clip"`, `class="person-fullname"`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("people page missing %s\n%s", want, page)
		}
	}
	if strings.Contains(page, "<td>PG</td>") {
		t.Fatal("people list should show the latest role, not an older one")
	}
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
	for _, want := range []string{"役割を追加", "登録済みの役割", `aria-label="上へ"`, `aria-label="下へ"`, `aria-label="上へ" disabled`, `class="role-page"`} {
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
	for _, want := range []string{"工程を追加", "登録済みの工程", `aria-label="上へ"`, `aria-label="下へ"`, `aria-label="上へ" disabled`} {
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
