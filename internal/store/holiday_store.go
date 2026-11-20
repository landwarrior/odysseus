package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// Holiday は祝日または休業日です。Date は YYYY-MM-DD です。
type Holiday struct {
	Date string
	Name string
}

// ListHolidays は祝日・休業日を日付順で返します。
func (s *Store) ListHolidays(ctx context.Context) ([]Holiday, error) {
	rows, err := s.db.Query(ctx, `SELECT date, name FROM holiday ORDER BY date`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Holiday
	for rows.Next() {
		var h Holiday
		if err := rows.Scan(&h.Date, &h.Name); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ListHolidayDates は祝日・休業日の日付だけを返します。
func (s *Store) ListHolidayDates(ctx context.Context) ([]string, error) {
	holidays, err := s.ListHolidays(ctx)
	if err != nil {
		return nil, err
	}
	dates := make([]string, len(holidays))
	for i, h := range holidays {
		dates[i] = h.Date
	}
	return dates, nil
}

// UpsertHoliday は date の祝日を登録または名称更新します。
func (s *Store) UpsertHoliday(ctx context.Context, date, name string) error {
	name = strings.TrimSpace(name)
	stamp := now()
	var existing string
	err := s.db.QueryRow(ctx, `SELECT date FROM holiday WHERE date = ?`, date).Scan(&existing)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.db.Exec(ctx, `
			INSERT INTO holiday (date, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			date, name, stamp, stamp)
		return err
	}
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `UPDATE holiday SET name = ?, updated_at = ? WHERE date = ?`, name, stamp, date)
	return err
}

// UpdateHolidayName は date の名称だけを更新します。無ければ ErrNotFound です。
func (s *Store) UpdateHolidayName(ctx context.Context, date, name string) error {
	res, err := s.db.Exec(ctx, `
		UPDATE holiday SET name = ?, updated_at = ? WHERE date = ?`,
		strings.TrimSpace(name), now(), date)
	if err != nil {
		return err
	}
	return missing(res)
}

// DeleteHoliday は date の祝日を削除します。無ければ ErrNotFound です。
func (s *Store) DeleteHoliday(ctx context.Context, date string) error {
	res, err := s.db.Exec(ctx, `DELETE FROM holiday WHERE date = ?`, date)
	if err != nil {
		return err
	}
	return missing(res)
}

// CountAssigneesByProjects は各案件にアサインされている人材の人数（重複なし）を返します。
func (s *Store) CountAssigneesByProjects(ctx context.Context, projectIDs []string) (map[string]int, error) {
	out := map[string]int{}
	if len(projectIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT pp.project_id, COUNT(DISTINCT a.person_id)
		FROM assignment a
		JOIN project_process pp ON pp.id = a.project_process_id
		WHERE pp.project_id IN (`+placeholders(len(projectIDs))+`)
		GROUP BY pp.project_id`, argsOf(projectIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var projectID string
		var n int
		if err := rows.Scan(&projectID, &n); err != nil {
			return nil, err
		}
		out[projectID] = n
	}
	return out, rows.Err()
}
