package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

const defaultTheme = "light"

var themeMu sync.Mutex

// handleThemeSave はログイン中の利用者のテーマを保存します。
func (s *Server) handleThemeSave(w http.ResponseWriter, r *http.Request) {
	theme := formText(r, "theme")
	if !validTheme(theme) {
		http.Error(w, "テーマが不正です。", http.StatusBadRequest)
		return
	}
	if err := s.saveTheme(personFrom(r).ID, theme); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// themeOf は personID の保存テーマを返します。無ければライトです。
func (s *Server) themeOf(personID string) string {
	if personID == "" {
		return defaultTheme
	}
	themeMu.Lock()
	defer themeMu.Unlock()
	themes, err := s.readThemes()
	if err != nil {
		return defaultTheme
	}
	theme := themes[personID]
	if !validTheme(theme) {
		return defaultTheme
	}
	return theme
}

// saveTheme は personID のテーマを data ディレクトリへ書き込みます。
func (s *Server) saveTheme(personID, theme string) error {
	themeMu.Lock()
	defer themeMu.Unlock()
	themes, err := s.readThemes()
	if err != nil {
		return err
	}
	if themes == nil {
		themes = map[string]string{}
	}
	themes[personID] = theme
	raw, err := json.MarshalIndent(themes, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	path := s.themeFile()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// readThemes はテーマ設定ファイルを読みます。無ければ空の map です。
func (s *Server) readThemes() (map[string]string, error) {
	raw, err := os.ReadFile(s.themeFile())
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var themes map[string]string
	if err := json.Unmarshal(raw, &themes); err != nil {
		return nil, err
	}
	if themes == nil {
		themes = map[string]string{}
	}
	return themes, nil
}

func (s *Server) themeFile() string {
	return filepath.Join(s.cfg.DataDir, "themes.json")
}

func validTheme(theme string) bool {
	return theme == "light" || theme == "dark"
}
