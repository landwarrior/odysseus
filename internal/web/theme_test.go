package web

import (
	"context"
	"path/filepath"
	"testing"

	"odysseus/internal/config"
	"odysseus/internal/db"
	"odysseus/internal/store"
)

func TestThemePersistsInDatabase(t *testing.T) {
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
	if _, _, err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	admin, err := st.GetPersonByCode(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if admin.Theme != "light" {
		t.Fatalf("default theme = %q", admin.Theme)
	}
	if err := st.UpdateTheme(ctx, admin.ID, "dark"); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetPerson(ctx, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Theme != "dark" {
		t.Fatalf("theme = %q", got.Theme)
	}
	srv, err := NewServer(st, config.Config{
		DataDir: dir, Driver: "sqlite", StandardMinutes: 480,
		SessionKey: make([]byte, 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	if personTheme(&got) != "dark" {
		t.Fatal("personTheme")
	}
	_ = srv
}
