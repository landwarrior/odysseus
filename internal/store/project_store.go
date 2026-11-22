package store

import (
	"context"
	"database/sql"
	"errors"

	"odysseus/internal/calc"
	"odysseus/internal/db"
	"odysseus/internal/id"
)

// ListProjects は案件を終了日順で返します。status が空なら全状態、そうでなければその状態だけです。
func (s *Store) ListProjects(ctx context.Context, status string) ([]Project, error) {
	query := projectSelect
	var args []any
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY CASE WHEN end_date IS NULL THEN '9999-12-31' ELSE end_date END, code`
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjects(rows)
}

// GetProject は projectID の案件を返します。無ければ ErrNotFound です。
func (s *Store) GetProject(ctx context.Context, projectID string) (Project, error) {
	return scanProject(s.db.QueryRow(ctx, projectSelect+` WHERE id = ?`, projectID))
}

// SaveProject は project と lines をまとめて保存します。project.ID が空なら新規です。
// アサインが残っている工程を lines から外すと ErrInUse です。戻り値は案件 ID です。
func (s *Store) SaveProject(ctx context.Context, project Project, lines []LineWrite) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	stamp := now()
	if project.ID == "" {
		project.ID = newID()
		_, err = tx.Exec(ctx, `
			INSERT INTO project (id, code, name, order_amount, start_date, end_date, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			project.ID, project.Code, project.Name, ni(project.OrderAmount), ns(project.StartDate), ns(project.EndDate), project.Status, stamp, stamp)
	} else {
		res, execErr := tx.Exec(ctx, `
			UPDATE project
			SET code = ?, name = ?, order_amount = ?, start_date = ?, end_date = ?, status = ?, updated_at = ?
			WHERE id = ?`,
			project.Code, project.Name, ni(project.OrderAmount), ns(project.StartDate), ns(project.EndDate), project.Status, stamp, project.ID)
		if execErr != nil {
			err = execErr
		} else {
			err = missing(res)
		}
	}
	if err != nil {
		return "", mapWriteErr(err)
	}
	existing, err := listLines(ctx, tx, project.ID)
	if err != nil {
		return "", err
	}
	byProcess := map[string]ProcessLine{}
	for _, line := range existing {
		byProcess[line.ProcessID] = line
	}
	kept := map[string]bool{}
	for _, line := range lines {
		if ex, ok := byProcess[line.ProcessID]; ok {
			_, err = tx.Exec(ctx, `
				UPDATE project_process
				SET start_date = ?, end_date = ?, planned_minutes = ?, budget_yen = ?, updated_at = ?
				WHERE id = ?`,
				ns(line.StartDate), ns(line.EndDate), line.PlannedMinutes, ni(line.BudgetYen), stamp, ex.ID)
			kept[ex.ID] = true
		} else {
			_, err = tx.Exec(ctx, `
				INSERT INTO project_process (
					id, project_id, process_id, start_date, end_date, planned_minutes, budget_yen, created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				newID(), project.ID, line.ProcessID, ns(line.StartDate), ns(line.EndDate), line.PlannedMinutes, ni(line.BudgetYen), stamp, stamp)
		}
		if err != nil {
			return "", mapWriteErr(err)
		}
	}
	for _, ex := range existing {
		if kept[ex.ID] {
			continue
		}
		var n int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM assignment WHERE project_process_id = ?`, ex.ID).Scan(&n); err != nil {
			return "", err
		}
		if n > 0 {
			return "", &InUseError{Name: ex.ProcessName}
		}
		if _, err := tx.Exec(ctx, `DELETE FROM project_process WHERE id = ?`, ex.ID); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return project.ID, nil
}

// DeleteProject は projectID の案件を、工程、アサイン、実績ごと削除します。
func (s *Store) DeleteProject(ctx context.Context, projectID string) error {
	res, err := s.db.Exec(ctx, `DELETE FROM project WHERE id = ?`, projectID)
	if err != nil {
		return mapWriteErr(err)
	}
	return missing(res)
}

// ListProcessLines は projectID の工程明細を、工程の並び順で返します。
func (s *Store) ListProcessLines(ctx context.Context, projectID string) ([]ProcessLine, error) {
	return listLines(ctx, s.db, projectID)
}

// ListProcessLinesForProjects は projectIDs の工程明細を、工程の並び順で返します。
func (s *Store) ListProcessLinesForProjects(ctx context.Context, projectIDs []string) ([]ProcessLine, error) {
	return s.linesForProjects(ctx, projectIDs)
}

// GetProcessLine は lineID の工程明細を返します。
func (s *Store) GetProcessLine(ctx context.Context, lineID string) (ProcessLine, error) {
	row := s.db.QueryRow(ctx, lineSelect+` WHERE pp.id = ?`, lineID)
	return scanLine(row)
}

// ListAssignmentsByProject は projectID のアサインを、工程順、人材コード順で返します。
func (s *Store) ListAssignmentsByProject(ctx context.Context, projectID string) ([]Assignment, error) {
	rows, err := s.db.Query(ctx, assignmentSelect+`
		WHERE pp.project_id = ?
		ORDER BY process.sort_order, process.name, person.code`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssignments(rows)
}

// ListPersonAssignments は projectID のうち、personID 本人のアサインだけを返します。
func (s *Store) ListPersonAssignments(ctx context.Context, projectID, personID string) ([]Assignment, error) {
	rows, err := s.db.Query(ctx, assignmentSelect+`
		WHERE pp.project_id = ? AND a.person_id = ?
		ORDER BY process.sort_order, process.name`, projectID, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssignments(rows)
}

// GetAssignment は assignmentID のアサインを返します。
func (s *Store) GetAssignment(ctx context.Context, assignmentID string) (Assignment, error) {
	return scanAssignment(s.db.QueryRow(ctx, assignmentSelect+` WHERE a.id = ?`, assignmentID))
}

// CreateAssignment は in を登録し、新しいアサイン ID を返します。
// 同じ工程に同じ人がいると ErrConflict です。
func (s *Store) CreateAssignment(ctx context.Context, in AssignmentWrite) (string, error) {
	assignmentID := newID()
	_, err := s.db.Exec(ctx, `
		INSERT INTO assignment (
			id, project_process_id, person_id, role_id, start_date, end_date, rate_kind, planned_cost_yen, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		assignmentID, in.ProjectProcessID, in.PersonID, in.RoleID, in.StartDate, in.EndDate, in.RateKind, in.PlannedCostYen, now(), now())
	if err != nil {
		return "", mapWriteErr(err)
	}
	return assignmentID, nil
}

// UpdateAssignment は assignmentID のアサインを in の内容に更新します。工程の付け替えはしません。
func (s *Store) UpdateAssignment(ctx context.Context, assignmentID string, in AssignmentWrite) error {
	res, err := s.db.Exec(ctx, `
		UPDATE assignment
		SET person_id = ?, role_id = ?, start_date = ?, end_date = ?, rate_kind = ?, planned_cost_yen = ?, updated_at = ?
		WHERE id = ?`,
		in.PersonID, in.RoleID, in.StartDate, in.EndDate, in.RateKind, in.PlannedCostYen, now(), assignmentID)
	if err != nil {
		return mapWriteErr(err)
	}
	return missing(res)
}

// DeleteAssignment は assignmentID のアサインと、それにぶら下がる実績を削除します。
func (s *Store) DeleteAssignment(ctx context.Context, assignmentID string) error {
	res, err := s.db.Exec(ctx, `DELETE FROM assignment WHERE id = ?`, assignmentID)
	if err != nil {
		return mapWriteErr(err)
	}
	return missing(res)
}

// RecalcPlannedCosts は projectID の予定原価を、各アサインの開始日時点の単価と単価種別で計算し直します。
// updated は更新した件数、missingNames は単価が無くて見送った人の表示名です。
func (s *Store) RecalcPlannedCosts(ctx context.Context, projectID string) (updated int, missingNames []string, err error) {
	assignments, err := s.ListAssignmentsByProject(ctx, projectID)
	if err != nil {
		return 0, nil, err
	}
	for _, a := range assignments {
		rate, ok, err := s.FindRate(ctx, a.PersonID, a.RoleID, a.StartDate)
		if err != nil {
			return updated, missingNames, err
		}
		cost, okCost := plannedCostForKind(a.StartDate, a.EndDate, a.RateKind, rate, ok)
		if !okCost {
			missingNames = append(missingNames, a.PersonName+" / "+a.RoleName+"（"+RateKindLabel(a.RateKind)+"）")
			continue
		}
		if _, err := s.db.Exec(ctx, `UPDATE assignment SET planned_cost_yen = ?, updated_at = ? WHERE id = ?`, cost, now(), a.ID); err != nil {
			return updated, missingNames, err
		}
		updated++
	}
	return updated, missingNames, nil
}

// RateKindOK は kind に合う単価が rate にあるとき true です。0 円も有効です。
func RateKindOK(kind string, rate Rate, hasRate bool) bool {
	if !hasRate {
		return false
	}
	switch kind {
	case RateKindMonthly:
		return rate.MonthlyYen != nil
	case RateKindHourly:
		return rate.HourlyYen != nil
	default:
		return false
	}
}

// RatePositive は kind に合う単価が設定済みかつ 0 円超のとき true です。
// 0 円単価（OJT など）はアサイン可能でもキャパには含めません。
func RatePositive(kind string, rate Rate, hasRate bool) bool {
	if !RateKindOK(kind, rate, hasRate) {
		return false
	}
	if kind == RateKindMonthly {
		return *rate.MonthlyYen > 0
	}
	return *rate.HourlyYen > 0
}

// plannedCostForKind は start〜end の予定原価です。kind に合う単価が rate に無いときは false です。
func plannedCostForKind(start, end, kind string, rate Rate, hasRate bool) (int64, bool) {
	if !RateKindOK(kind, rate, hasRate) {
		return 0, false
	}
	if kind == RateKindMonthly {
		return calc.AssignmentPlannedCost(start, end, *rate.MonthlyYen), true
	}
	return 0, true
}

// ListEntries は assignmentIDs の実績を返します。
// from 以上 until 未満の作業日に限ります。どちらかが空なら、その端は制限しません。
func (s *Store) ListEntries(ctx context.Context, assignmentIDs []string, from, until string) ([]TimeEntry, error) {
	if len(assignmentIDs) == 0 {
		return nil, nil
	}
	query := `
		SELECT id, assignment_id, task_id, work_date, minutes
		FROM time_entry
		WHERE assignment_id IN (` + placeholders(len(assignmentIDs)) + `)`
	args := argsOf(assignmentIDs)
	if from != "" {
		query += ` AND work_date >= ?`
		args = append(args, from)
	}
	if until != "" {
		query += ` AND work_date < ?`
		args = append(args, until)
	}
	query += ` ORDER BY work_date, assignment_id, task_id`
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TimeEntry
	for rows.Next() {
		var e TimeEntry
		var taskID sql.NullString
		if err := rows.Scan(&e.ID, &e.AssignmentID, &taskID, &e.WorkDate, &e.Minutes); err != nil {
			return nil, err
		}
		if taskID.Valid {
			e.TaskID = taskID.String
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// SaveDay は cells を1つのトランザクションで保存します。分が 0 以下の要素は削除です。
// 同じアサイン・作業日でも、タスクが違えば別行です。TaskID が空なら task_id IS NULL を対象にします。
func (s *Store) SaveDay(ctx context.Context, cells []DayCell) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stamp := now()
	for _, cell := range cells {
		if cell.Minutes <= 0 {
			if err := deleteDayCell(ctx, tx, cell); err != nil {
				return err
			}
			continue
		}
		entryID, err := findDayCellID(ctx, tx, cell)
		if errors.Is(err, sql.ErrNoRows) {
			_, err = tx.Exec(ctx, `
				INSERT INTO time_entry (id, assignment_id, task_id, work_date, minutes, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				newID(), cell.AssignmentID, nullIfEmpty(cell.TaskID), cell.WorkDate, cell.Minutes, stamp, stamp)
		} else if err == nil {
			_, err = tx.Exec(ctx, `UPDATE time_entry SET minutes = ?, updated_at = ? WHERE id = ?`, cell.Minutes, stamp, entryID)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// deleteDayCell は cell に対応する実績行を消します。
func deleteDayCell(ctx context.Context, tx *db.Tx, cell DayCell) error {
	if cell.TaskID == "" {
		_, err := tx.Exec(ctx, `DELETE FROM time_entry WHERE assignment_id = ? AND work_date = ? AND task_id IS NULL`,
			cell.AssignmentID, cell.WorkDate)
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM time_entry WHERE assignment_id = ? AND work_date = ? AND task_id = ?`,
		cell.AssignmentID, cell.WorkDate, cell.TaskID)
	return err
}

// findDayCellID は cell に対応する実績行の ID を返します。無ければ sql.ErrNoRows です。
func findDayCellID(ctx context.Context, tx *db.Tx, cell DayCell) (string, error) {
	var entryID string
	var err error
	if cell.TaskID == "" {
		err = tx.QueryRow(ctx, `SELECT id FROM time_entry WHERE assignment_id = ? AND work_date = ? AND task_id IS NULL`,
			cell.AssignmentID, cell.WorkDate).Scan(&entryID)
	} else {
		err = tx.QueryRow(ctx, `SELECT id FROM time_entry WHERE assignment_id = ? AND work_date = ? AND task_id = ?`,
			cell.AssignmentID, cell.WorkDate, cell.TaskID).Scan(&entryID)
	}
	return entryID, err
}

// nullIfEmpty は value が空なら NULL、そうでなければ value を返します（SQL バインド用）。
func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// ListAssignedProjects は personID がアサインされ、まだ完了していない案件を返します。
func (s *Store) ListAssignedProjects(ctx context.Context, personID string) ([]Project, error) {
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT p.id, p.code, p.name, p.order_amount, p.start_date, p.end_date, p.status
		FROM project p
		JOIN project_process pp ON pp.project_id = p.id
		JOIN assignment a ON a.project_process_id = pp.id
		WHERE a.person_id = ? AND p.status <> 'closed'
		ORDER BY p.code`, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjects(rows)
}

// ListOpenProjectsWithAssignments はアサインがあり、まだ完了していない案件を返します。
// 管理者が他人の実績を入れるときの一覧に使います。
func (s *Store) ListOpenProjectsWithAssignments(ctx context.Context) ([]Project, error) {
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT p.id, p.code, p.name, p.order_amount, p.start_date, p.end_date, p.status
		FROM project p
		JOIN project_process pp ON pp.project_id = p.id
		JOIN assignment a ON a.project_process_id = pp.id
		WHERE p.status <> 'closed'
		ORDER BY p.code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjects(rows)
}

// LoadBundle は projectID の予実計算に必要な明細をまとめて返します。
func (s *Store) LoadBundle(ctx context.Context, projectID string) (Bundle, error) {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return Bundle{}, err
	}
	bundles, err := s.loadBundles(ctx, []Project{project})
	if err != nil {
		return Bundle{}, err
	}
	return bundles[0], nil
}

// LoadBundlesByStatus は status の案件全部について、予実計算用の明細を返します。
func (s *Store) LoadBundlesByStatus(ctx context.Context, status string) ([]Bundle, error) {
	projects, err := s.ListProjects(ctx, status)
	if err != nil {
		return nil, err
	}
	return s.loadBundles(ctx, projects)
}

// loadBundles は projects の並びに合わせて、工程、アサイン、実績、単価を読みます。
func (s *Store) loadBundles(ctx context.Context, projects []Project) ([]Bundle, error) {
	out := make([]Bundle, len(projects))
	if len(projects) == 0 {
		return out, nil
	}
	index := map[string]int{}
	ids := make([]string, len(projects))
	for i, project := range projects {
		out[i].Project = project
		index[project.ID] = i
		ids[i] = project.ID
	}
	lines, err := s.linesForProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	lineIDs := make([]string, 0, len(lines))
	lineProject := map[string]string{}
	for _, line := range lines {
		out[index[line.ProjectID]].Lines = append(out[index[line.ProjectID]].Lines, line)
		lineIDs = append(lineIDs, line.ID)
		lineProject[line.ID] = line.ProjectID
	}
	tasks, err := s.ListTasksForProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, task := range tasks {
		out[index[task.ProjectID]].Tasks = append(out[index[task.ProjectID]].Tasks, task)
	}
	assignments, err := s.assignmentsForLines(ctx, lineIDs)
	if err != nil {
		return nil, err
	}
	assignmentIDs := make([]string, 0, len(assignments))
	personIDs := map[string]struct{}{}
	assignmentProject := map[string]string{}
	for _, a := range assignments {
		projectID := lineProject[a.ProjectProcessID]
		out[index[projectID]].Assignments = append(out[index[projectID]].Assignments, a)
		assignmentIDs = append(assignmentIDs, a.ID)
		personIDs[a.PersonID] = struct{}{}
		assignmentProject[a.ID] = projectID
	}
	entries, err := s.ListEntries(ctx, assignmentIDs, "", "")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		out[index[assignmentProject[e.AssignmentID]]].Entries = append(out[index[assignmentProject[e.AssignmentID]]].Entries, e)
	}
	var people []string
	for personID := range personIDs {
		people = append(people, personID)
	}
	rates, err := s.ratesForPeople(ctx, people)
	if err != nil {
		return nil, err
	}
	for _, rate := range rates {
		for i := range out {
			for _, a := range out[i].Assignments {
				if a.PersonID == rate.PersonID {
					out[i].Rates = append(out[i].Rates, rate)
					break
				}
			}
		}
	}
	return out, nil
}

// linesForProjects は projectIDs に属する工程明細を返します。
func (s *Store) linesForProjects(ctx context.Context, projectIDs []string) ([]ProcessLine, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, lineSelect+` WHERE pp.project_id IN (`+placeholders(len(projectIDs))+`) ORDER BY process.sort_order, process.name`, argsOf(projectIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLines(rows)
}

// AssignmentsForLines は lineIDs（工程明細の ID）に属するアサインを返します。
func (s *Store) AssignmentsForLines(ctx context.Context, lineIDs []string) ([]Assignment, error) {
	return s.assignmentsForLines(ctx, lineIDs)
}

// assignmentsForLines は lineIDs（工程明細の ID）に属するアサインを返します。
func (s *Store) assignmentsForLines(ctx context.Context, lineIDs []string) ([]Assignment, error) {
	if len(lineIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, assignmentSelect+` WHERE a.project_process_id IN (`+placeholders(len(lineIDs))+`) ORDER BY person.code`, argsOf(lineIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssignments(rows)
}

// ratesForPeople は personIDs の時間単価を全部返します。
func (s *Store) ratesForPeople(ctx context.Context, personIDs []string) ([]Rate, error) {
	if len(personIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, person_id, role_id, '', from_date, hourly_yen, monthly_yen
		FROM person_rate
		WHERE person_id IN (`+placeholders(len(personIDs))+`)`, argsOf(personIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRates(rows)
}

const projectSelect = `SELECT id, code, name, order_amount, start_date, end_date, status FROM project`

const lineSelect = `
	SELECT pp.id, pp.project_id, pp.process_id, process.name, process.color, pp.start_date, pp.end_date, pp.planned_minutes, pp.budget_yen, process.sort_order
	FROM project_process pp
	JOIN process ON process.id = pp.process_id`

const assignmentSelect = `
	SELECT a.id, a.project_process_id, pp.project_id, process.name, a.person_id, person.code,
		CASE WHEN person.name_kana = '' THEN person.name ELSE person.name || ' ' || person.name_kana END,
		a.role_id, role.name, a.start_date, a.end_date, a.rate_kind, a.planned_cost_yen
	FROM assignment a
	JOIN project_process pp ON pp.id = a.project_process_id
	JOIN process ON process.id = pp.process_id
	JOIN person ON person.id = a.person_id
	JOIN role ON role.id = a.role_id`

// listLines は q から、projectID の工程明細を読みます。保存処理の途中でも使います。
func listLines(ctx context.Context, q db.Querier, projectID string) ([]ProcessLine, error) {
	rows, err := q.Query(ctx, lineSelect+` WHERE pp.project_id = ? ORDER BY process.sort_order, process.name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLines(rows)
}

// scanProjects は rows を最後まで Project の列に読みます。
func scanProjects(rows *sql.Rows) ([]Project, error) {
	var out []Project
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, project)
	}
	return out, rows.Err()
}

// scanProject は row の1件を Project に読みます。無ければ ErrNotFound です。
func scanProject(row scanner) (Project, error) {
	var project Project
	var order sql.NullInt64
	var start, end sql.NullString
	err := row.Scan(&project.ID, &project.Code, &project.Name, &order, &start, &end, &project.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	if err != nil {
		return Project{}, err
	}
	if order.Valid {
		v := order.Int64
		project.OrderAmount = &v
	}
	if start.Valid {
		project.StartDate = start.String
	}
	if end.Valid {
		project.EndDate = end.String
	}
	return project, nil
}

// scanLines は rows を最後まで ProcessLine の列に読みます。
func scanLines(rows *sql.Rows) ([]ProcessLine, error) {
	var out []ProcessLine
	for rows.Next() {
		line, err := scanLine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, line)
	}
	return out, rows.Err()
}

// scanLine は row の1件を ProcessLine に読みます。
func scanLine(row scanner) (ProcessLine, error) {
	var line ProcessLine
	var start, end sql.NullString
	var budget sql.NullInt64
	err := row.Scan(&line.ID, &line.ProjectID, &line.ProcessID, &line.ProcessName, &line.ProcessColor, &start, &end, &line.PlannedMinutes, &budget, &line.SortOrder)
	if errors.Is(err, sql.ErrNoRows) {
		return ProcessLine{}, ErrNotFound
	}
	if err != nil {
		return ProcessLine{}, err
	}
	if start.Valid {
		line.StartDate = start.String
	}
	if end.Valid {
		line.EndDate = end.String
	}
	if budget.Valid {
		v := budget.Int64
		line.BudgetYen = &v
	}
	return line, nil
}

// scanAssignments は rows を最後まで Assignment の列に読みます。
func scanAssignments(rows *sql.Rows) ([]Assignment, error) {
	var out []Assignment
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// scanAssignment は row の1件を Assignment に読みます。
func scanAssignment(row scanner) (Assignment, error) {
	var a Assignment
	err := row.Scan(
		&a.ID, &a.ProjectProcessID, &a.ProjectID, &a.ProcessName, &a.PersonID, &a.PersonCode, &a.PersonName,
		&a.RoleID, &a.RoleName, &a.StartDate, &a.EndDate, &a.RateKind, &a.PlannedCostYen)
	if errors.Is(err, sql.ErrNoRows) {
		return Assignment{}, ErrNotFound
	}
	return a, err
}

// newID は新しい主キーを返します。
func newID() string { return id.New() }
