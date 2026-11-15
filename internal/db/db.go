// Package db はデータベース製品の差を閉じ込める層です。
//
// 案件や人材の SQL はここには書きません。それらは internal/store にあります。
// このファイルが担当するのは次の4つです。
//
//   - 接続を開く
//   - SQL の ? を、製品が受け付ける書き方に直す
//   - 複数の更新をまとめるトランザクション
//   - 起動時に、未適用のテーブル定義を順に当てる
//
// メソッドの ctx は処理の中断を伝えるものです。画面の処理が終わると一緒に終わります。
// query は SQL 文、args はその中の ? に順番に入る値です。以下の説明では、この3つを繰り返しません。
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	// 名前を付けない import は、接続ドライバをデータベース/sql へ登録するためだけに置いています。
	// このファイルの中で型としては使いません。
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL
	_ "modernc.org/sqlite"             // SQLite
)

// migrationFS は sql ディレクトリの .sql ファイルを、実行ファイルの中に埋め込んだものです。
// 起動する PC に、テーブル定義のファイルを別に置かなくて済みます。
//
//go:embed sql/*.sql
var migrationFS embed.FS

// DB は開いた接続です。問い合わせは必ずここのメソッドを通します。
// SQL フィールドを直接使うと、プレースホルダの変換が飛ばされます。
type DB struct {
	SQL    *sql.DB // 標準ライブラリの接続プール
	Driver string  // "sqlite" または "postgres"。アプリから見た製品名
}

// Tx は Begin で始めたトランザクションです。
// Commit するまで確定せず、失敗したら Rollback で全部戻します。
type Tx struct {
	tx     *sql.Tx
	driver string
}

// Querier は、普通の接続とトランザクションのどちらでも同じ関数を呼べるようにする窓口です。
// Exec は更新、Query は複数行の取得、QueryRow は1行の取得です。
type Querier interface {
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
}

// Open は driver（sqlite または postgres）と dsn（接続文字列）でデータベースを開きます。
// SQLite ならファイルの親フォルダを作り、外部キーを有効にします。
// 開けても、5秒以内に応答がなければ接続失敗として閉じます。
func Open(driver, dsn string) (*DB, error) {
	sqlDriver := driver
	switch driver {
	case "sqlite", "":
		driver = "sqlite"
		sqlDriver = "sqlite"
		if err := ensureSQLiteDir(dsn); err != nil {
			return nil, err
		}
		dsn = sqliteDSN(dsn)
	case "postgres":
		sqlDriver = "pgx"
	default:
		return nil, fmt.Errorf("未対応のDBドライバです: %s", driver)
	}
	sqlDB, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		return nil, err
	}
	if driver == "sqlite" {
		// SQLite は同時に書くとロックしやすいので、接続は1本にします。
		sqlDB.SetMaxOpenConns(1)
		if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
			sqlDB.Close()
			return nil, err
		}
		if _, err := sqlDB.Exec("PRAGMA busy_timeout = 5000"); err != nil {
			sqlDB.Close()
			return nil, err
		}
		if _, err := sqlDB.Exec("PRAGMA journal_mode = WAL"); err != nil {
			sqlDB.Close()
			return nil, err
		}
	} else {
		sqlDB.SetMaxOpenConns(10)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("データベースに接続できません: %w", err)
	}
	return &DB{SQL: sqlDB, Driver: driver}, nil
}

// Exec は d の接続で、行を返さない SQL を実行します。INSERT、UPDATE、DELETE に使います。
func (d *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.SQL.ExecContext(ctx, Rebind(d.Driver, query), args...)
}

// Query は d の接続で、複数行の SELECT を実行します。読み終えたら呼び出し側が Close します。
func (d *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.SQL.QueryContext(ctx, Rebind(d.Driver, query), args...)
}

// QueryRow は d の接続で、1行の SELECT を実行します。行が無いと、あとで Scan がエラーになります。
func (d *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.SQL.QueryRowContext(ctx, Rebind(d.Driver, query), args...)
}

// Begin は d の接続でトランザクションを開始します。Commit するまで変更は確定しません。
func (d *DB) Begin(ctx context.Context) (*Tx, error) {
	tx, err := d.SQL.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, driver: d.Driver}, nil
}

// Exec はトランザクションの中で、結果の行を読まない SQL を実行します。
func (t *Tx) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, Rebind(t.driver, query), args...)
}

// Query はトランザクションの中で、複数行の SELECT を実行します。
func (t *Tx) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, Rebind(t.driver, query), args...)
}

// QueryRow はトランザクションの中で、1行の SELECT を実行します。
func (t *Tx) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, Rebind(t.driver, query), args...)
}

// Commit はトランザクション中の変更を確定します。
func (t *Tx) Commit() error { return t.tx.Commit() }

// Rollback はトランザクション中の変更を取り消します。Commit のあとでは何もしません。
func (t *Tx) Rollback() error { return t.tx.Rollback() }

// Rebind は query の中の ? を、driver が postgres のときだけ $1, $2 に置き換えます。
// SQLite はそのまま返します。query の文字定数の中に ? は書けません。
func Rebind(driver, query string) string {
	if driver != "postgres" {
		return query
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(fmt.Sprint(n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

// Migrate は database に、まだ当てていない sql/*.sql をファイル名の順に実行します。
// schema_migration に記録済みのファイルは飛ばします。
func Migrate(ctx context.Context, database *DB) error {
	entries, err := fs.ReadDir(migrationFS, "sql")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := strings.TrimSuffix(entry.Name(), ".sql")
		applied, err := migrationApplied(ctx, database, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := migrationFS.ReadFile("sql/" + entry.Name())
		if err != nil {
			return err
		}
		if err := applyMigration(ctx, database, version, string(body)); err != nil {
			return fmt.Errorf("マイグレーション %s: %w", version, err)
		}
	}
	return nil
}

// migrationApplied は database に version（ファイル名から拡張子を除いたもの）が適用済みかを返します。
// 初回で schema_migration テーブルが無いときは、未適用として false を返します。
func migrationApplied(ctx context.Context, database *DB, version string) (bool, error) {
	var n int
	err := database.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migration WHERE version = ?`, version).Scan(&n)
	if err == nil {
		return n > 0, nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "no such table") || strings.Contains(msg, "does not exist") {
		return false, nil
	}
	return false, err
}

// applyMigration は body（SQL ファイルの中身）を database で実行し、version を適用済みにします。
// 途中で失敗したら、そのファイルの変更は残しません。
func applyMigration(ctx context.Context, database *DB, version, body string) error {
	tx, err := database.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range splitSQL(body) {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("%w\nSQL: %s", err, stmt)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migration (version, applied_at) VALUES (?, ?)`, version, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return tx.Commit()
}

// splitSQL は body を、セミコロンで終わる文の列に分けます。
// 行コメント（--）は捨てます。文の途中のセミコロンには対応していません。
func splitSQL(body string) []string {
	var stmts []string
	var b strings.Builder
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
		if strings.HasSuffix(trim, ";") {
			stmt := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(b.String()), ";"))
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			b.Reset()
		}
	}
	if rest := strings.TrimSpace(b.String()); rest != "" {
		stmts = append(stmts, strings.TrimSuffix(rest, ";"))
	}
	return stmts
}

// sqliteDSN は dsn（ファイルパス、または file: で始まる接続文字列）を、ドライバの形式にします。
// すでに file: で始まっていれば、そのまま返します。
func sqliteDSN(dsn string) string {
	if strings.HasPrefix(dsn, "file:") {
		return dsn
	}
	abs, err := filepath.Abs(dsn)
	if err != nil {
		abs = dsn
	}
	return "file:" + filepath.ToSlash(abs) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}

// ensureSQLiteDir は dsn が指す SQLite ファイルの親フォルダを、無いときだけ作ります。
func ensureSQLiteDir(dsn string) error {
	path := dsn
	if strings.HasPrefix(dsn, "file:") {
		path = strings.TrimPrefix(dsn, "file:")
		if i := strings.Index(path, "?"); i >= 0 {
			path = path[:i]
		}
		path = filepath.FromSlash(path)
	}
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o700)
}
