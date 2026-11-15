package web

import (
	"os"
	"path/filepath"
	"testing"

	"odysseus/internal/config"
)

func TestThemePersistsAcrossServer(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{DataDir: dir}
	s1 := &Server{cfg: cfg}
	if err := s1.saveTheme("person-1", "dark"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "themes.json")); err != nil {
		t.Fatal(err)
	}
	s2 := &Server{cfg: cfg}
	if got := s2.themeOf("person-1"); got != "dark" {
		t.Fatalf("theme = %q", got)
	}
	if got := s2.themeOf("unknown"); got != "light" {
		t.Fatalf("default theme = %q", got)
	}
}
