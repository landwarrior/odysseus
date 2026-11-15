package db

import (
	"context"
	"os"
	"testing"
)

func TestPostgresMigrate(t *testing.T) {
	dsn := os.Getenv("ODYSSEUS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("ODYSSEUS_TEST_POSTGRES_DSN が無いため PostgreSQL テストを飛ばします")
	}
	database, err := Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.SQL.Close() })
	if err := Migrate(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(context.Background(), database); err != nil {
		t.Fatal(err)
	}
}
