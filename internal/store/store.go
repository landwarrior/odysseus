// Package store はテーブルの読み書きです。画面と計算はデータベース製品を知りません。
//
// メソッドの第1引数 ctx は処理の中断を伝えるものです。各メソッドの説明では繰り返しません。
// 金額の単位は円、工数の単位は分、日付は YYYY-MM-DD です。
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"odysseus/internal/db"
	"odysseus/internal/id"
)

var (
	ErrNotFound  = errors.New("not found")  // 指定した行が無い
	ErrConflict  = errors.New("conflict")   // 番号や名前が既に使われている
	ErrInUse     = errors.New("in use")     // 他から参照されていて削除できない
	ErrLastAdmin = errors.New("last admin") // 最後の有効な管理者は外せない
)

// InUseError は削除や工程の取り外しが参照で止まったときに、対象の名前を持たせます。
type InUseError struct{ Name string }

// Error は e.Name（外せなかった対象の名前）を返します。
func (e *InUseError) Error() string { return e.Name }

// Is は target が ErrInUse のとき真を返します。errors.Is がこの判定を使います。
func (e *InUseError) Is(target error) bool { return target == ErrInUse }

// Store は開いたデータベースへの読み書きです。
type Store struct{ db *db.DB }

// New は database（開いた接続）を使う Store を返します。
func New(database *db.DB) *Store { return &Store{db: database} }

// Person はログインも兼ねる人材です。パスワードそのものは持ちません。
// Theme は light または dark です。
type Person struct {
	ID                 string
	Code               string
	Name               string
	NameKana           string
	IsAdmin            bool
	Active             bool
	Remarks            string
	MustChangePassword bool
	Theme              string
}

// PersonAuth は Person に、照合用のパスワードハッシュを足したものです。画面へは渡しません。
type PersonAuth struct {
	Person
	PasswordHash string
}

// NewPerson は人材の登録と更新に使う入力です。Password が空の更新では、パスワードを変えません。
type NewPerson struct {
	Code     string
	Name     string
	NameKana string
	Password string
	IsAdmin  bool
	Active   bool
	Remarks  string
}

// Role は PG や SE などの役割です。SortOrder が小さいほど先に並びます。
type Role struct {
	ID        string
	Name      string
	SortOrder int64
	Active    bool
}

// Process は要件定義や製造などの工程マスタです。案件ごとの期間や工数は別の型です。
type Process struct {
	ID        string
	Name      string
	SortOrder int64
	Active    bool
}

// Rate は人材と役割の時間単価です。FromDate 以降、次の単価が現れるまで有効です。
type Rate struct {
	ID        string
	PersonID  string
	RoleID    string
	RoleName  string
	FromDate  string
	HourlyYen int64
}

// Project は案件です。Status は planning、active、closed のいずれかです。
// OrderAmount が nil のときは受注金額が未設定です。
type Project struct {
	ID          string
	Code        string
	Name        string
	OrderAmount *int64
	StartDate   string
	EndDate     string
	Status      string
}

// ProcessLine は案件にぶら下がる工程です。BudgetYen が nil のときは予算未入力です。
type ProcessLine struct {
	ID             string
	ProjectID      string
	ProcessID      string
	ProcessName    string
	StartDate      string
	EndDate        string
	PlannedMinutes int64
	BudgetYen      *int64
	SortOrder      int64
}

// LineWrite は工程明細の保存内容です。ProcessID は工程マスタの ID です。
type LineWrite struct {
	ProcessID      string
	StartDate      string
	EndDate        string
	PlannedMinutes int64
	BudgetYen      *int64
}

// Task は工程明細を細分化した作業です。見積工数とスケジュール割当の単位です。
type Task struct {
	ID               string
	ProjectProcessID string
	ProjectID        string
	ProcessID        string
	ProcessName      string
	ProcessSort      int64
	Name             string
	SortOrder        int64
	PlannedMinutes   int64
}

// TaskWrite はタスクの保存内容です。
type TaskWrite struct {
	ID             string
	Name           string
	PlannedMinutes int64
}

// TaskAllocation はタスクの旬への工数割当です。YearMonth は YYYY-MM、Decade は 1〜3 です。
type TaskAllocation struct {
	TaskID         string
	YearMonth      string
	Decade         int
	PlannedMinutes int64
}

// Assignment は工程に人を割り当てた1件です。PlannedCostYen は保存時点の予定原価で、単価変更では自動更新しません。
type Assignment struct {
	ID               string
	ProjectProcessID string
	ProjectID        string
	ProcessName      string
	PersonID         string
	PersonCode       string
	PersonName       string
	RoleID           string
	RoleName         string
	StartDate        string
	EndDate          string
	PlannedMinutes   int64
	PlannedCostYen   int64
}

// AssignmentWrite はアサインの登録と更新に使う入力です。
type AssignmentWrite struct {
	ProjectProcessID string
	PersonID         string
	RoleID           string
	StartDate        string
	EndDate          string
	PlannedMinutes   int64
	PlannedCostYen   int64
}

// TimeEntry はアサインに対する1日の実績です。Minutes は 0 より大きい分です。
type TimeEntry struct {
	ID           string
	AssignmentID string
	WorkDate     string
	Minutes      int64
}

// DayCell は1日の保存指示です。Minutes が 0 以下のときは、その日の実績を削除します。
type DayCell struct {
	AssignmentID string
	WorkDate     string
	Minutes      int64
}

// Bundle は予実計算に必要な、1案件分の明細をまとめたものです。
type Bundle struct {
	Project     Project
	Lines       []ProcessLine
	Assignments []Assignment
	Entries     []TimeEntry
	Rates       []Rate
}

// Seed は役割、工程、人が空のときだけ初期データを作ります。
// 人を新しく作ったときだけ created が真で、adminPassword に初期パスワードが入ります。
func (s *Store) Seed(ctx context.Context) (adminPassword string, created bool, err error) {
	if err := s.seedNames(ctx, `SELECT COUNT(*) FROM role`, []Role{
		{Name: "PG", SortOrder: 10, Active: true},
		{Name: "SE", SortOrder: 20, Active: true},
		{Name: "PL", SortOrder: 30, Active: true},
		{Name: "PM", SortOrder: 40, Active: true},
	}, nil); err != nil {
		return "", false, err
	}
	if err := s.seedNames(ctx, `SELECT COUNT(*) FROM process`, nil, []Process{
		{Name: "提案", SortOrder: 10, Active: true},
		{Name: "要件定義", SortOrder: 20, Active: true},
		{Name: "基本設計", SortOrder: 30, Active: true},
		{Name: "詳細設計", SortOrder: 40, Active: true},
		{Name: "製造", SortOrder: 50, Active: true},
		{Name: "単体テスト", SortOrder: 60, Active: true},
		{Name: "結合テスト", SortOrder: 70, Active: true},
		{Name: "シナリオテスト", SortOrder: 80, Active: true},
	}); err != nil {
		return "", false, err
	}
	n, err := s.count(ctx, `SELECT COUNT(*) FROM person`)
	if err != nil {
		return "", false, err
	}
	if n > 0 {
		return "", false, nil
	}
	password, err := randomPassword(12)
	if err != nil {
		return "", false, err
	}
	_, err = s.InsertPerson(ctx, NewPerson{
		Code: "admin", Name: "管理者", NameKana: "カンリシャ", Password: password,
		IsAdmin: true, Active: true, Remarks: "初期管理者",
	})
	if err != nil {
		return "", false, err
	}
	if _, err := s.db.Exec(ctx, `UPDATE person SET must_change_password = 1 WHERE code = ?`, "admin"); err != nil {
		return "", false, err
	}
	return password, true, nil
}

// seedNames は countSQL の件数が 0 のときだけ、roles と processes を登録します。
// 役割と工程のどちらを入れるかは、空でない方のスライスで決まります。
func (s *Store) seedNames(ctx context.Context, countSQL string, roles []Role, processes []Process) error {
	n, err := s.count(ctx, countSQL)
	if err != nil || n > 0 {
		return err
	}
	for _, role := range roles {
		if _, err := s.InsertRole(ctx, role); err != nil {
			return err
		}
	}
	for _, process := range processes {
		if _, err := s.InsertProcess(ctx, process); err != nil {
			return err
		}
	}
	return nil
}

// GetPerson は personID の人材を返します。無ければ ErrNotFound です。
func (s *Store) GetPerson(ctx context.Context, personID string) (Person, error) {
	row := s.db.QueryRow(ctx, personSelect+` WHERE id = ?`, personID)
	return scanPerson(row)
}

// GetPersonByCode は code（ユーザーコード）の人材とパスワードハッシュを返します。
func (s *Store) GetPersonByCode(ctx context.Context, code string) (PersonAuth, error) {
	var p PersonAuth
	var admin, active, must int
	err := s.db.QueryRow(ctx, `
		SELECT id, code, name, name_kana, is_admin, active, remarks, must_change_password, theme, password_hash
		FROM person WHERE code = ?`, code).Scan(
		&p.ID, &p.Code, &p.Name, &p.NameKana, &admin, &active, &p.Remarks, &must, &p.Theme, &p.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return PersonAuth{}, ErrNotFound
	}
	if err != nil {
		return PersonAuth{}, err
	}
	p.IsAdmin = admin != 0
	p.Active = active != 0
	p.MustChangePassword = must != 0
	if p.Theme != "dark" {
		p.Theme = "light"
	}
	return p, nil
}

// ListPeople は人材をユーザーコード順で返します。無効な人も含まれます。
func (s *Store) ListPeople(ctx context.Context) ([]Person, error) {
	rows, err := s.db.Query(ctx, personSelect+` ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Person
	for rows.Next() {
		p, err := scanPerson(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// InsertPerson は in を登録します。パスワードはハッシュ化し、次回ログイン時の変更を必須にします。
// ユーザーコードが重複すると ErrConflict です。
func (s *Store) InsertPerson(ctx context.Context, in NewPerson) (Person, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return Person{}, err
	}
	p := Person{
		ID: id.New(), Code: in.Code, Name: in.Name, NameKana: in.NameKana,
		IsAdmin: in.IsAdmin, Active: in.Active, Remarks: in.Remarks, MustChangePassword: true, Theme: "light",
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO person (
			id, code, name, name_kana, password_hash, is_admin, active, remarks, must_change_password, theme, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)`,
		p.ID, p.Code, p.Name, p.NameKana, string(hash), bint(p.IsAdmin), bint(p.Active), p.Remarks, p.Theme, now(), now())
	if err != nil {
		return Person{}, mapWriteErr(err)
	}
	return p, nil
}

// UpdatePerson は personID の人材を in の内容に更新します。
// in.Password が空ならパスワードは変えません。最後の管理者を外すと ErrLastAdmin です。
func (s *Store) UpdatePerson(ctx context.Context, personID string, in NewPerson) error {
	current, err := s.GetPerson(ctx, personID)
	if err != nil {
		return err
	}
	if current.IsAdmin && current.Active && (!in.IsAdmin || !in.Active) {
		n, err := s.count(ctx, `SELECT COUNT(*) FROM person WHERE is_admin = 1 AND active = 1`)
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastAdmin
		}
	}
	stamp := now()
	if in.Password == "" {
		_, err = s.db.Exec(ctx, `
			UPDATE person
			SET code = ?, name = ?, name_kana = ?, is_admin = ?, active = ?, remarks = ?, updated_at = ?
			WHERE id = ?`,
			in.Code, in.Name, in.NameKana, bint(in.IsAdmin), bint(in.Active), in.Remarks, stamp, personID)
	} else {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if hashErr != nil {
			return hashErr
		}
		_, err = s.db.Exec(ctx, `
			UPDATE person
			SET code = ?, name = ?, name_kana = ?, is_admin = ?, active = ?, remarks = ?,
				password_hash = ?, must_change_password = 1, updated_at = ?
			WHERE id = ?`,
			in.Code, in.Name, in.NameKana, bint(in.IsAdmin), bint(in.Active), in.Remarks, string(hash), stamp, personID)
	}
	return mapWriteErr(err)
}

// UpdateTheme は personID の画面テーマを theme（light または dark）に変えます。
func (s *Store) UpdateTheme(ctx context.Context, personID, theme string) error {
	if theme != "light" && theme != "dark" {
		return errors.New("invalid theme")
	}
	res, err := s.db.Exec(ctx, `UPDATE person SET theme = ?, updated_at = ? WHERE id = ?`, theme, now(), personID)
	if err != nil {
		return err
	}
	return missing(res)
}

// UpdatePassword は personID のパスワードを password に変え、変更必須を解除します。
func (s *Store) UpdatePassword(ctx context.Context, personID, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(ctx, `
		UPDATE person SET password_hash = ?, must_change_password = 0, updated_at = ? WHERE id = ?`,
		string(hash), now(), personID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeletePerson は personID の人材を削除します。アサインがあると ErrInUse、最後の管理者だと ErrLastAdmin です。
func (s *Store) DeletePerson(ctx context.Context, personID string) error {
	current, err := s.GetPerson(ctx, personID)
	if err != nil {
		return err
	}
	if current.IsAdmin && current.Active {
		n, err := s.count(ctx, `SELECT COUNT(*) FROM person WHERE is_admin = 1 AND active = 1`)
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastAdmin
		}
	}
	n, err := s.count(ctx, `SELECT COUNT(*) FROM assignment WHERE person_id = ?`, personID)
	if err != nil {
		return err
	}
	if n > 0 {
		return &InUseError{Name: current.Name}
	}
	res, err := s.db.Exec(ctx, `DELETE FROM person WHERE id = ?`, personID)
	if err != nil {
		return mapWriteErr(err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListRoles は役割を並び順、名前の順で返します。
func (s *Store) ListRoles(ctx context.Context) ([]Role, error) {
	return s.listRoles(ctx, s.db)
}

// listRoles は q（通常の接続かトランザクション）から役割を読みます。
func (s *Store) listRoles(ctx context.Context, q db.Querier) ([]Role, error) {
	rows, err := q.Query(ctx, `SELECT id, name, sort_order, active FROM role ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Role
	for rows.Next() {
		var role Role
		var active int
		if err := rows.Scan(&role.ID, &role.Name, &role.SortOrder, &active); err != nil {
			return nil, err
		}
		role.Active = active != 0
		out = append(out, role)
	}
	return out, rows.Err()
}

// InsertRole は role を登録して、採番した ID を入れたものを返します。名前の重複は ErrConflict です。
func (s *Store) InsertRole(ctx context.Context, role Role) (Role, error) {
	role.ID = id.New()
	_, err := s.db.Exec(ctx, `
		INSERT INTO role (id, name, sort_order, active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		role.ID, role.Name, role.SortOrder, bint(role.Active), now(), now())
	if err != nil {
		return Role{}, mapWriteErr(err)
	}
	return role, nil
}

// MoveRole は id の役割を、direction が up なら一つ上、down なら一つ下へ動かします。
// 端にあるときは moved が false で、並びは変えません。
func (s *Store) MoveRole(ctx context.Context, id, direction string) (bool, error) {
	list, err := s.ListRoles(ctx)
	if err != nil {
		return false, err
	}
	index := -1
	for i, role := range list {
		if role.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return false, ErrNotFound
	}
	swap := index
	switch direction {
	case "up":
		swap = index - 1
	case "down":
		swap = index + 1
	default:
		return false, nil
	}
	if swap < 0 || swap >= len(list) {
		return false, nil
	}
	list[index], list[swap] = list[swap], list[index]
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	stamp := now()
	for i, role := range list {
		if _, err := tx.Exec(ctx, `UPDATE role SET sort_order = ?, updated_at = ? WHERE id = ?`, int64(i+1)*10, stamp, role.ID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// UpdateRole は role.ID の役割を、role の名前、並び順、有効フラグで更新します。
func (s *Store) UpdateRole(ctx context.Context, role Role) error {
	res, err := s.db.Exec(ctx, `
		UPDATE role SET name = ?, sort_order = ?, active = ?, updated_at = ? WHERE id = ?`,
		role.Name, role.SortOrder, bint(role.Active), now(), role.ID)
	if err != nil {
		return mapWriteErr(err)
	}
	return missing(res)
}

// DeleteRole は roleID の役割を削除します。単価かアサインが参照していると ErrInUse です。
func (s *Store) DeleteRole(ctx context.Context, roleID string) error {
	n, err := s.count(ctx, `SELECT
		(SELECT COUNT(*) FROM person_rate WHERE role_id = ?) +
		(SELECT COUNT(*) FROM assignment WHERE role_id = ?)`, roleID, roleID)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrInUse
	}
	res, err := s.db.Exec(ctx, `DELETE FROM role WHERE id = ?`, roleID)
	if err != nil {
		return mapWriteErr(err)
	}
	return missing(res)
}

// ListProcesses は工程マスタを並び順、名前の順で返します。
func (s *Store) ListProcesses(ctx context.Context) ([]Process, error) {
	rows, err := s.db.Query(ctx, `SELECT id, name, sort_order, active FROM process ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Process
	for rows.Next() {
		var process Process
		var active int
		if err := rows.Scan(&process.ID, &process.Name, &process.SortOrder, &active); err != nil {
			return nil, err
		}
		process.Active = active != 0
		out = append(out, process)
	}
	return out, rows.Err()
}

// InsertProcess は process を登録して、採番した ID を入れたものを返します。
func (s *Store) InsertProcess(ctx context.Context, process Process) (Process, error) {
	process.ID = id.New()
	_, err := s.db.Exec(ctx, `
		INSERT INTO process (id, name, sort_order, active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		process.ID, process.Name, process.SortOrder, bint(process.Active), now(), now())
	if err != nil {
		return Process{}, mapWriteErr(err)
	}
	return process, nil
}

// MoveProcess は id の工程を、direction が up なら一つ上、down なら一つ下へ動かします。
// 端にあるときは moved が false で、並びは変えません。
func (s *Store) MoveProcess(ctx context.Context, id, direction string) (bool, error) {
	list, err := s.ListProcesses(ctx)
	if err != nil {
		return false, err
	}
	index := -1
	for i, process := range list {
		if process.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return false, ErrNotFound
	}
	swap := index
	switch direction {
	case "up":
		swap = index - 1
	case "down":
		swap = index + 1
	default:
		return false, nil
	}
	if swap < 0 || swap >= len(list) {
		return false, nil
	}
	list[index], list[swap] = list[swap], list[index]
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	stamp := now()
	for i, process := range list {
		if _, err := tx.Exec(ctx, `UPDATE process SET sort_order = ?, updated_at = ? WHERE id = ?`, int64(i+1)*10, stamp, process.ID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// UpdateProcess は process.ID の工程を、process の名前、並び順、有効フラグで更新します。
func (s *Store) UpdateProcess(ctx context.Context, process Process) error {
	res, err := s.db.Exec(ctx, `
		UPDATE process SET name = ?, sort_order = ?, active = ?, updated_at = ? WHERE id = ?`,
		process.Name, process.SortOrder, bint(process.Active), now(), process.ID)
	if err != nil {
		return mapWriteErr(err)
	}
	return missing(res)
}

// DeleteProcess は processID の工程を削除します。案件で使われていると ErrInUse です。
func (s *Store) DeleteProcess(ctx context.Context, processID string) error {
	n, err := s.count(ctx, `SELECT COUNT(*) FROM project_process WHERE process_id = ?`, processID)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrInUse
	}
	res, err := s.db.Exec(ctx, `DELETE FROM process WHERE id = ?`, processID)
	if err != nil {
		return mapWriteErr(err)
	}
	return missing(res)
}

// LatestRoleNames は、date 以前に適用が始まっている単価のうち、人材ごとに
// 適用開始日が最も新しいものの役割名を返します。同じ適用開始日なら単価が最も高い役割です。
// キーは人材 ID です。該当する単価が無い人材は map に入りません。
func (s *Store) LatestRoleNames(ctx context.Context, date string) (map[string]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT r.person_id, role.name
		FROM person_rate r
		JOIN role ON role.id = r.role_id
		WHERE r.from_date <= ?
		ORDER BY r.person_id, r.from_date DESC, r.hourly_yen DESC, role.sort_order, role.name`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var personID, roleName string
		if err := rows.Scan(&personID, &roleName); err != nil {
			return nil, err
		}
		if _, ok := out[personID]; ok {
			continue
		}
		out[personID] = roleName
	}
	return out, rows.Err()
}

// ListRates は personID の時間単価を、適用開始日の順で返します。
func (s *Store) ListRates(ctx context.Context, personID string) ([]Rate, error) {
	rows, err := s.db.Query(ctx, `
		SELECT r.id, r.person_id, r.role_id, role.name, r.from_date, r.hourly_yen
		FROM person_rate r
		JOIN role ON role.id = r.role_id
		WHERE r.person_id = ?
		ORDER BY r.from_date, role.sort_order, role.name`, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRates(rows)
}

// InsertRate は rate を登録します。同じ人材、役割、適用開始日があると ErrConflict です。
func (s *Store) InsertRate(ctx context.Context, rate Rate) (Rate, error) {
	rate.ID = id.New()
	_, err := s.db.Exec(ctx, `
		INSERT INTO person_rate (id, person_id, role_id, from_date, hourly_yen, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rate.ID, rate.PersonID, rate.RoleID, rate.FromDate, rate.HourlyYen, now(), now())
	if err != nil {
		return Rate{}, mapWriteErr(err)
	}
	return rate, nil
}

// DeleteRate は personID に属する rateID の単価を削除します。他人の単価 ID では削除しません。
func (s *Store) DeleteRate(ctx context.Context, personID, rateID string) error {
	res, err := s.db.Exec(ctx, `DELETE FROM person_rate WHERE id = ? AND person_id = ?`, rateID, personID)
	if err != nil {
		return err
	}
	return missing(res)
}

// FindHourly はその日以前で最も新しい適用開始日の時間単価です。
// FindHourly は personID と roleID について、date 以前で最も新しい時間単価を返します。
// 該当が無いときは false です。
func (s *Store) FindHourly(ctx context.Context, personID, roleID, date string) (int64, bool, error) {
	var yen int64
	err := s.db.QueryRow(ctx, `
		SELECT hourly_yen FROM person_rate
		WHERE person_id = ? AND role_id = ? AND from_date <= ?
		ORDER BY from_date DESC
		LIMIT 1`, personID, roleID, date).Scan(&yen)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return yen, true, nil
}

const personSelect = `SELECT id, code, name, name_kana, is_admin, active, remarks, must_change_password, theme FROM person`

// scanner は、1行を列の値へ読み込むものです。*sql.Row と *sql.Rows の両方が満たします。
type scanner interface{ Scan(dest ...any) error }

// scanPerson は row（1行、または結果セットの現在行）を Person に読みます。
func scanPerson(row scanner) (Person, error) {
	var p Person
	var admin, active, must int
	err := row.Scan(&p.ID, &p.Code, &p.Name, &p.NameKana, &admin, &active, &p.Remarks, &must, &p.Theme)
	if errors.Is(err, sql.ErrNoRows) {
		return Person{}, ErrNotFound
	}
	if err != nil {
		return Person{}, err
	}
	p.IsAdmin = admin != 0
	p.Active = active != 0
	p.MustChangePassword = must != 0
	if p.Theme != "dark" {
		p.Theme = "light"
	}
	return p, nil
}

// scanRates は rows を最後まで Rate の列に読みます。
func scanRates(rows *sql.Rows) ([]Rate, error) {
	var out []Rate
	for rows.Next() {
		var rate Rate
		if err := rows.Scan(&rate.ID, &rate.PersonID, &rate.RoleID, &rate.RoleName, &rate.FromDate, &rate.HourlyYen); err != nil {
			return nil, err
		}
		out = append(out, rate)
	}
	return out, rows.Err()
}

// count は query（COUNT を返す SQL）を args で実行し、件数を返します。
func (s *Store) count(ctx context.Context, query string, args ...any) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}

// missing は res が1行も更新していなければ ErrNotFound を返します。
func missing(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// mapWriteErr は err が重複なら ErrConflict、外部キー違反なら ErrInUse に読み替えます。それ以外はそのまま返します。
func mapWriteErr(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique") || strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key") {
		return ErrConflict
	}
	if strings.Contains(msg, "foreign key") || strings.Contains(msg, "23503") {
		return ErrInUse
	}
	return err
}

// bint は v を、データベースに保存する 1 または 0 にします。
func bint(v bool) int {
	if v {
		return 1
	}
	return 0
}

// now は現在時刻を UTC の文字列で返します。作成日時と更新日時に使います。
func now() string { return time.Now().UTC().Format(time.RFC3339) }

// randomPassword は n 文字の初期パスワードを作ります。紛らわしい文字は含めません。
func randomPassword(n int) (string, error) {
	const alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf), nil
}

// ns は s が空なら SQL の NULL、そうでなければ s そのものを返します。
func ns(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ni は p が nil なら SQL の NULL、そうでなければ指している数を返します。
func ni(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// placeholders は IN 句に使う、n 個の ? をカンマでつないだ文字列を返します。
func placeholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

// argsOf は ids を、SQL の引数として渡せる形にします。
func argsOf(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}
