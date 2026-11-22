package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"odysseus/internal/db"
)

const taskSelect = `
	SELECT t.id, t.project_process_id, pp.project_id, pp.process_id, process.name, process.sort_order, process.color,
		t.name, t.remarks, t.sort_order, t.planned_minutes
	FROM project_task t
	JOIN project_process pp ON pp.id = t.project_process_id
	JOIN process ON process.id = pp.process_id`

// ListTasksByProject は projectID のタスクを、工程順・タスク順で返します。
func (s *Store) ListTasksByProject(ctx context.Context, projectID string) ([]Task, error) {
	rows, err := s.db.Query(ctx, taskSelect+`
		WHERE pp.project_id = ?
		ORDER BY process.sort_order, process.name, t.sort_order, t.name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

// ListTasksForProjects は projectIDs のタスクを、案件番号・工程・タスク順で返します。
func (s *Store) ListTasksForProjects(ctx context.Context, projectIDs []string) ([]Task, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, taskSelect+`
		JOIN project ON project.id = pp.project_id
		WHERE pp.project_id IN (`+placeholders(len(projectIDs))+`)
		ORDER BY CASE WHEN project.end_date IS NULL THEN '9999-12-31' ELSE project.end_date END,
			project.code, process.sort_order, process.name, t.sort_order, t.name`,
		argsOf(projectIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

// GetTask は taskID のタスクを返します。
func (s *Store) GetTask(ctx context.Context, taskID string) (Task, error) {
	return scanTask(s.db.QueryRow(ctx, taskSelect+` WHERE t.id = ?`, taskID))
}

// ReplaceTasksByProcess は projectID の工程ごとに tasksByProcess（工程マスタ ID → タスク）を置き換えます。
// フォームに載っていない工程のタスクは消しません。載せた工程で空なら、その工程のタスクを全部消します。
// 名前付きタスクがある工程は、工程明細の予定分をタスク合計に揃えます。
// タスクが無い工程の予定分は、案件保存時にタスク合計（0）として書かれた値のままです。
func (s *Store) ReplaceTasksByProcess(ctx context.Context, projectID string, tasksByProcess map[string][]TaskWrite) error {
	if len(tasksByProcess) == 0 {
		return nil
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	lines, err := listLines(ctx, tx, projectID)
	if err != nil {
		return err
	}
	lineByProcess := map[string]ProcessLine{}
	for _, line := range lines {
		lineByProcess[line.ProcessID] = line
	}
	stamp := now()
	for processID, tasks := range tasksByProcess {
		line, ok := lineByProcess[processID]
		if !ok {
			continue
		}
		existing, err := listTasksForLine(ctx, tx, line.ID)
		if err != nil {
			return err
		}
		byID := map[string]Task{}
		for _, task := range existing {
			byID[task.ID] = task
		}
		kept := map[string]bool{}
		var sum int64
		var order int64
		hadNamed := false
		for _, in := range tasks {
			name := strings.TrimSpace(in.Name)
			if name == "" {
				continue
			}
			hadNamed = true
			order++
			sum += in.PlannedMinutes
			remarks := strings.TrimSpace(in.Remarks)
			if ex, ok := byID[in.ID]; ok && in.ID != "" {
				_, err = tx.Exec(ctx, `
					UPDATE project_task
					SET name = ?, remarks = ?, sort_order = ?, planned_minutes = ?, updated_at = ?
					WHERE id = ? AND project_process_id = ?`,
					name, remarks, order, in.PlannedMinutes, stamp, ex.ID, line.ID)
				kept[ex.ID] = true
			} else {
				_, err = tx.Exec(ctx, `
					INSERT INTO project_task (
						id, project_process_id, name, remarks, sort_order, planned_minutes, created_at, updated_at
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
					newID(), line.ID, name, remarks, order, in.PlannedMinutes, stamp, stamp)
			}
			if err != nil {
				return mapWriteErr(err)
			}
		}
		for _, ex := range existing {
			if kept[ex.ID] {
				continue
			}
			if _, err := tx.Exec(ctx, `DELETE FROM project_task WHERE id = ?`, ex.ID); err != nil {
				return err
			}
		}
		// 名前付きタスクがあるときだけ、工程の予定分をタスク合計に揃える。
		// タスクが無い工程は、案件保存時にタスク合計（0）として既に書かれている。
		if hadNamed {
			if _, err := tx.Exec(ctx, `
				UPDATE project_process SET planned_minutes = ?, updated_at = ? WHERE id = ?`,
				sum, stamp, line.ID); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// ListAllocationsByProcesses は projectProcessIDs の工程旬割当をすべて返します。
func (s *Store) ListAllocationsByProcesses(ctx context.Context, projectProcessIDs []string) ([]ProcessAllocation, error) {
	if len(projectProcessIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT project_process_id, year_month, decade, planned_minutes
		FROM process_allocation
		WHERE project_process_id IN (`+placeholders(len(projectProcessIDs))+`)`, argsOf(projectProcessIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProcessAllocation
	for rows.Next() {
		var a ProcessAllocation
		if err := rows.Scan(&a.ProjectProcessID, &a.YearMonth, &a.Decade, &a.PlannedMinutes); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AllocationTotalsByProcess は各工程明細の割当合計（全期間）を返します。
func (s *Store) AllocationTotalsByProcess(ctx context.Context, projectProcessIDs []string) (map[string]int64, error) {
	out := map[string]int64{}
	if len(projectProcessIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT project_process_id, COALESCE(SUM(planned_minutes), 0)
		FROM process_allocation
		WHERE project_process_id IN (`+placeholders(len(projectProcessIDs))+`)
		GROUP BY project_process_id`, argsOf(projectProcessIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var minutes int64
		if err := rows.Scan(&id, &minutes); err != nil {
			return nil, err
		}
		out[id] = minutes
	}
	return out, rows.Err()
}

// UpsertProcessAllocation は projectProcessID の yearMonth/decade に minutes を保存します。
// minutes が 0 以下なら行を削除します。
func (s *Store) UpsertProcessAllocation(ctx context.Context, projectProcessID, yearMonth string, decade int, minutes int64) error {
	if decade < 1 || decade > 3 {
		return ErrConflict
	}
	stamp := now()
	if minutes <= 0 {
		_, err := s.db.Exec(ctx, `
			DELETE FROM process_allocation WHERE project_process_id = ? AND year_month = ? AND decade = ?`,
			projectProcessID, yearMonth, decade)
		return err
	}
	var existing string
	err := s.db.QueryRow(ctx, `
		SELECT project_process_id FROM process_allocation
		WHERE project_process_id = ? AND year_month = ? AND decade = ?`,
		projectProcessID, yearMonth, decade).Scan(&existing)
	if errorsIsNoRows(err) {
		_, err = s.db.Exec(ctx, `
			INSERT INTO process_allocation (project_process_id, year_month, decade, planned_minutes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			projectProcessID, yearMonth, decade, minutes, stamp, stamp)
		return mapWriteErr(err)
	}
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		UPDATE process_allocation SET planned_minutes = ?, updated_at = ?
		WHERE project_process_id = ? AND year_month = ? AND decade = ?`,
		minutes, stamp, projectProcessID, yearMonth, decade)
	return mapWriteErr(err)
}

// ListAllocationsByTasks は taskIDs の旬割当をすべて返します。
func (s *Store) ListAllocationsByTasks(ctx context.Context, taskIDs []string) ([]TaskAllocation, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT task_id, year_month, decade, planned_minutes
		FROM task_allocation
		WHERE task_id IN (`+placeholders(len(taskIDs))+`)`, argsOf(taskIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TaskAllocation
	for rows.Next() {
		var a TaskAllocation
		if err := rows.Scan(&a.TaskID, &a.YearMonth, &a.Decade, &a.PlannedMinutes); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AllocationTotalsByTask は各タスクの割当合計（全期間）を返します。
func (s *Store) AllocationTotalsByTask(ctx context.Context, taskIDs []string) (map[string]int64, error) {
	out := map[string]int64{}
	if len(taskIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT task_id, COALESCE(SUM(planned_minutes), 0)
		FROM task_allocation
		WHERE task_id IN (`+placeholders(len(taskIDs))+`)
		GROUP BY task_id`, argsOf(taskIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var taskID string
		var minutes int64
		if err := rows.Scan(&taskID, &minutes); err != nil {
			return nil, err
		}
		out[taskID] = minutes
	}
	return out, rows.Err()
}

// UpsertTaskAllocation は taskID の yearMonth/decade に minutes を保存します。
// minutes が 0 以下なら行を削除します。
func (s *Store) UpsertTaskAllocation(ctx context.Context, taskID, yearMonth string, decade int, minutes int64) error {
	if decade < 1 || decade > 3 {
		return ErrConflict
	}
	stamp := now()
	if minutes <= 0 {
		_, err := s.db.Exec(ctx, `
			DELETE FROM task_allocation WHERE task_id = ? AND year_month = ? AND decade = ?`,
			taskID, yearMonth, decade)
		return err
	}
	var existing string
	err := s.db.QueryRow(ctx, `
		SELECT task_id FROM task_allocation WHERE task_id = ? AND year_month = ? AND decade = ?`,
		taskID, yearMonth, decade).Scan(&existing)
	if errorsIsNoRows(err) {
		_, err = s.db.Exec(ctx, `
			INSERT INTO task_allocation (task_id, year_month, decade, planned_minutes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			taskID, yearMonth, decade, minutes, stamp, stamp)
		return mapWriteErr(err)
	}
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		UPDATE task_allocation SET planned_minutes = ?, updated_at = ?
		WHERE task_id = ? AND year_month = ? AND decade = ?`,
		minutes, stamp, taskID, yearMonth, decade)
	return mapWriteErr(err)
}

// CountActivePeople は有効な人材の人数を返します。
func (s *Store) CountActivePeople(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM person WHERE active = 1`).Scan(&n)
	return n, err
}

func listTasksForLine(ctx context.Context, q db.Querier, lineID string) ([]Task, error) {
	rows, err := q.Query(ctx, taskSelect+`
		WHERE t.project_process_id = ?
		ORDER BY t.sort_order, t.name`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func scanTasks(rows *sql.Rows) ([]Task, error) {
	var out []Task
	for rows.Next() {
		task, err := scanTaskRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

func scanTask(row *sql.Row) (Task, error) {
	var t Task
	err := row.Scan(&t.ID, &t.ProjectProcessID, &t.ProjectID, &t.ProcessID, &t.ProcessName, &t.ProcessSort, &t.ProcessColor,
		&t.Name, &t.Remarks, &t.SortOrder, &t.PlannedMinutes)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, err
	}
	return t, nil
}

func scanTaskRows(rows *sql.Rows) (Task, error) {
	var t Task
	err := rows.Scan(&t.ID, &t.ProjectProcessID, &t.ProjectID, &t.ProcessID, &t.ProcessName, &t.ProcessSort, &t.ProcessColor,
		&t.Name, &t.Remarks, &t.SortOrder, &t.PlannedMinutes)
	return t, err
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// SnapEffortToPersonDays は見積・割当の分数を、画面の 0.1 人日刻みに揃えます。
// 時間入力時代の端数が、表示 1.8 に対して割当 1.8 と食い違わないようにします。
// standardMinutes は 1 人日の分数です。何度呼んでも同じ結果になります。
func (s *Store) SnapEffortToPersonDays(ctx context.Context, standardMinutes int64) error {
	if standardMinutes <= 0 {
		standardMinutes = 480
	}
	half := standardMinutes / 2
	for _, table := range []string{"project_task", "project_process", "process_allocation", "task_allocation"} {
		_, err := s.db.Exec(ctx, `
			UPDATE `+table+` SET planned_minutes = ((planned_minutes * 10 + ?) / ?) * ? / 10
			WHERE planned_minutes > 0`, half, standardMinutes, standardMinutes)
		if err != nil {
			return err
		}
	}
	return nil
}
