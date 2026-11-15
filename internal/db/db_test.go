package db

import "testing"

func TestRebind(t *testing.T) {
	q := "SELECT * FROM t WHERE a = ? AND b = ?"
	if got := Rebind("sqlite", q); got != q {
		t.Fatalf("sqlite: %s", got)
	}
	want := "SELECT * FROM t WHERE a = $1 AND b = $2"
	if got := Rebind("postgres", q); got != want {
		t.Fatalf("postgres: %s", got)
	}
}
