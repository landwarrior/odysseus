package store

import (
	"context"
	"path/filepath"
	"testing"

	"odysseus/internal/db"
)

func TestSeedRateAndCascade(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	password, created, err := st.Seed(ctx)
	if err != nil || !created || len(password) < 12 {
		t.Fatalf("seed: pw=%q created=%v err=%v", password, created, err)
	}
	if _, created, err := st.Seed(ctx); err != nil || created {
		t.Fatalf("second seed created=%v err=%v", created, err)
	}
	roles, err := st.ListRoles(ctx)
	if err != nil || len(roles) != 4 {
		t.Fatalf("roles: %+v %v", roles, err)
	}
	processes, err := st.ListProcesses(ctx)
	if err != nil || len(processes) != 8 {
		t.Fatalf("processes: %d %v", len(processes), err)
	}

	var pg Role
	var build Process
	for _, role := range roles {
		if role.Name == "PG" {
			pg = role
		}
	}
	for _, process := range processes {
		if process.Name == "製造" {
			build = process
		}
	}
	person, err := st.InsertPerson(ctx, NewPerson{Code: "m01", Name: "山田", Password: "memberpass", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertRate(ctx, Rate{PersonID: person.ID, RoleID: pg.ID, FromDate: "2026-01-01", HourlyYen: Yen(3000)}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertRate(ctx, Rate{PersonID: person.ID, RoleID: pg.ID, FromDate: "2026-02-01", HourlyYen: Yen(9000)}); err != nil {
		t.Fatal(err)
	}
	yen, ok, err := st.FindHourly(ctx, person.ID, pg.ID, "2026-03-01")
	if err != nil || !ok || yen != 9000 {
		t.Fatalf("hourly = %d ok=%v err=%v", yen, ok, err)
	}
	if _, ok, err := st.FindHourly(ctx, person.ID, pg.ID, "2025-12-31"); err != nil || ok {
		t.Fatalf("early rate ok=%v err=%v", ok, err)
	}

	order := int64(1000000)
	projectID, err := st.SaveProject(ctx, Project{
		Code: "P1", Name: "案件", OrderAmount: &order, StartDate: "2026-04-01", EndDate: "2026-04-30", Status: "active",
	}, []LineWrite{{ProcessID: build.ID, StartDate: "2026-04-01", EndDate: "2026-04-30", PlannedMinutes: 4800}})
	if err != nil {
		t.Fatal(err)
	}
	lines, err := st.ListProcessLines(ctx, projectID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("lines: %+v %v", lines, err)
	}
	if _, err := st.CreateAssignment(ctx, AssignmentWrite{
		ProjectProcessID: lines[0].ID, PersonID: person.ID, RoleID: pg.ID,
		StartDate: "2026-04-01", EndDate: "2026-04-30", RateKind: RateKindMonthly, PlannedCostYen: 400000,
	}); err != nil {
		t.Fatal(err)
	}
	assignments, err := st.ListAssignmentsByProject(ctx, projectID)
	if err != nil || len(assignments) != 1 {
		t.Fatal(err)
	}
	if err := st.ReplaceTasksByProcess(ctx, projectID, map[string][]TaskWrite{
		build.ID: {{Name: "画面", PlannedMinutes: 240}, {Name: "API", PlannedMinutes: 240}},
	}); err != nil {
		t.Fatal(err)
	}
	tasks, err := st.ListTasksByProject(ctx, projectID)
	if err != nil || len(tasks) != 2 {
		t.Fatalf("tasks: %+v %v", tasks, err)
	}
	if err := st.SaveDay(ctx, []DayCell{
		{AssignmentID: assignments[0].ID, TaskID: tasks[0].ID, WorkDate: "2026-04-02", Minutes: 120},
		{AssignmentID: assignments[0].ID, TaskID: tasks[1].ID, WorkDate: "2026-04-02", Minutes: 180},
	}); err != nil {
		t.Fatal(err)
	}
	entries, err := st.ListEntries(ctx, []string{assignments[0].ID}, "", "")
	if err != nil || len(entries) != 2 {
		t.Fatalf("entries: %+v %v", entries, err)
	}
	if err := st.DeletePerson(ctx, person.ID); err != ErrInUse && !isInUse(err) {
		t.Fatalf("delete person: %v", err)
	}
	if err := st.DeleteProject(ctx, projectID); err != nil {
		t.Fatal(err)
	}
	entries, err = st.ListEntries(ctx, []string{assignments[0].ID}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries remained: %+v", entries)
	}
	admin, err := st.GetPersonByCode(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdatePerson(ctx, admin.ID, NewPerson{Code: "admin", Name: "管理者", Active: true, IsAdmin: false}); err != ErrLastAdmin {
		t.Fatalf("last admin: %v", err)
	}
}

func TestLatestRoleNames(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	if _, _, err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	roles, err := st.ListRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	roleID := map[string]string{}
	for _, role := range roles {
		roleID[role.Name] = role.ID
	}
	person, err := st.InsertPerson(ctx, NewPerson{Code: "m01", Name: "山田", Password: "memberpass", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertRate(ctx, Rate{PersonID: person.ID, RoleID: roleID["PG"], FromDate: "2026-01-01", HourlyYen: Yen(3000)}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertRate(ctx, Rate{PersonID: person.ID, RoleID: roleID["SE"], FromDate: "2026-03-01", HourlyYen: Yen(4000)}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertRate(ctx, Rate{PersonID: person.ID, RoleID: roleID["PL"], FromDate: "2026-06-01", HourlyYen: Yen(5000)}); err != nil {
		t.Fatal(err)
	}
	names, err := st.LatestRoleNames(ctx, "2026-04-15")
	if err != nil {
		t.Fatal(err)
	}
	if names[person.ID] != "SE" {
		t.Fatalf("latest role = %q", names[person.ID])
	}
	names, err = st.LatestRoleNames(ctx, "2025-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := names[person.ID]; ok {
		t.Fatal("future-only rates should stay empty")
	}

	other, err := st.InsertPerson(ctx, NewPerson{Code: "m02", Name: "佐藤", Password: "memberpass", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertRate(ctx, Rate{PersonID: other.ID, RoleID: roleID["PG"], FromDate: "2026-01-01", HourlyYen: Yen(6000)}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertRate(ctx, Rate{PersonID: other.ID, RoleID: roleID["SE"], FromDate: "2026-01-01", HourlyYen: Yen(4500)}); err != nil {
		t.Fatal(err)
	}
	names, err = st.LatestRoleNames(ctx, "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if names[other.ID] != "PG" {
		t.Fatalf("same-date highest rate role = %q", names[other.ID])
	}
}

func TestMoveRole(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	if _, _, err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	roles, err := st.ListRoles(ctx)
	if err != nil || len(roles) < 2 {
		t.Fatal(err)
	}
	first, second := roles[0].Name, roles[1].Name
	moved, err := st.MoveRole(ctx, roles[0].ID, "up")
	if err != nil || moved {
		t.Fatalf("top item should stay: moved=%v err=%v", moved, err)
	}
	moved, err = st.MoveRole(ctx, roles[1].ID, "up")
	if err != nil || !moved {
		t.Fatal(err)
	}
	roles, err = st.ListRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if roles[0].Name != second || roles[1].Name != first {
		t.Fatalf("order = %s, %s", roles[0].Name, roles[1].Name)
	}
}

func TestMoveProcess(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	if _, _, err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	processes, err := st.ListProcesses(ctx)
	if err != nil || len(processes) < 2 {
		t.Fatal(err)
	}
	first, second := processes[0].Name, processes[1].Name
	moved, err := st.MoveProcess(ctx, processes[0].ID, "up")
	if err != nil || moved {
		t.Fatalf("top item should stay: moved=%v err=%v", moved, err)
	}
	moved, err = st.MoveProcess(ctx, processes[1].ID, "up")
	if err != nil || !moved {
		t.Fatal(err)
	}
	processes, err = st.ListProcesses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if processes[0].Name != second || processes[1].Name != first {
		t.Fatalf("order = %s, %s", processes[0].Name, processes[1].Name)
	}
}

func isInUse(err error) bool {
	return err != nil && (err == ErrInUse || (func() bool { _, ok := err.(*InUseError); return ok })())
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	database, err := db.Open("sqlite", filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.SQL.Close() })
	if err := db.Migrate(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	return New(database)
}
