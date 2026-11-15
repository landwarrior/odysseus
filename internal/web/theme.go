package web

import (
	"net/http"

	"odysseus/internal/store"
)

const defaultTheme = "light"

// handleThemeSave はログイン中の利用者のテーマをデータベースへ保存します。
func (s *Server) handleThemeSave(w http.ResponseWriter, r *http.Request) {
	theme := formText(r, "theme")
	if !validTheme(theme) {
		http.Error(w, "テーマが不正です。", http.StatusBadRequest)
		return
	}
	if err := s.store.UpdateTheme(r.Context(), personFrom(r).ID, theme); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// personTheme は p のテーマを返します。不正ならライトです。
func personTheme(p *store.Person) string {
	if p != nil && validTheme(p.Theme) {
		return p.Theme
	}
	return defaultTheme
}

func validTheme(theme string) bool {
	return theme == "light" || theme == "dark"
}
