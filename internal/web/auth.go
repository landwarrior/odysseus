package web

import (
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

// loginPage はログイン画面です。Code は入力し直すユーザーコード、Hint は初期パスワードの案内です。
type loginPage struct {
	base
	Code string
	Hint string
}

// handleLoginGet はログイン画面を出します。ログイン済みならホームかパスワード変更へ進みます。
func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	if me, _, ok := s.current(r); ok {
		if me.MustChangePassword {
			http.Redirect(w, r, "/password", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	flash, warn, errMsg := s.takeNotice(w, r)
	page := loginPage{base: base{Title: "ログイン", Flash: flash, Warn: warn, Error: errMsg}, Hint: s.initialHint()}
	s.render(w, http.StatusOK, "login", page)
}

// handleLoginPost は code と password を照合し、成功したらセッションを発行します。
func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "入力が不正です。", http.StatusBadRequest)
		return
	}
	code := formText(r, "code")
	password := r.FormValue("password")
	page := loginPage{base: base{Title: "ログイン"}, Code: code, Hint: s.initialHint()}
	auth, err := s.store.GetPersonByCode(r.Context(), code)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(password)) != nil || len(password) > 72 {
		page.Warn = "ユーザーコードまたはパスワードが違います。"
		s.render(w, http.StatusBadRequest, "login", page)
		return
	}
	if !auth.Active {
		page.Warn = "このユーザーは無効です。"
		s.render(w, http.StatusBadRequest, "login", page)
		return
	}
	s.writeSession(w, auth.ID, "")
	if auth.MustChangePassword {
		http.Redirect(w, r, "/password", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleLogout はセッションを消してログイン画面へ戻します。
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSession(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handlePasswordGet は自分のパスワード変更画面を出します。
func (s *Server) handlePasswordGet(w http.ResponseWriter, r *http.Request) {
	page := s.page(w, r, "パスワード変更")
	s.render(w, http.StatusOK, "password", page)
}

// handlePasswordPost は現在のパスワードを確認し、新しいパスワードへ更新します。
func (s *Server) handlePasswordPost(w http.ResponseWriter, r *http.Request) {
	me := personFrom(r)
	page := s.page(w, r, "パスワード変更")
	current := r.FormValue("current_password")
	next := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")
	auth, err := s.store.GetPersonByCode(r.Context(), me.Code)
	if err != nil {
		s.fail(w, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(current)) != nil {
		page.Warn = "現在のパスワードが違います。"
		s.render(w, http.StatusBadRequest, "password", page)
		return
	}
	if msg := passwordIssue(next, true); msg != "" {
		page.Warn = msg
		s.render(w, http.StatusBadRequest, "password", page)
		return
	}
	if next != confirm {
		page.Warn = "新しいパスワードが一致しません。"
		s.render(w, http.StatusBadRequest, "password", page)
		return
	}
	if err := s.store.UpdatePassword(r.Context(), me.ID, next); err != nil {
		s.fail(w, err)
		return
	}
	if me.MustChangePassword {
		_ = os.Remove(filepath.Join(s.cfg.DataDir, "initial-admin.txt"))
	}
	s.redirect(w, r, "/", "パスワードを変更しました。")
}
