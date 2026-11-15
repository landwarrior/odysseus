// Package web はブラウザへ返す画面と、入力の受け取りです。
//
// ハンドラの w はブラウザへの応答、r は届いたリクエストです。
// ログイン中の利用者は r から取ります。パスの {id} は r.PathValue("id") です。
// 各ハンドラの説明では、w と r を繰り返しません。
package web

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"odysseus/internal/calc"
	"odysseus/internal/config"
	"odysseus/internal/store"
)

//go:embed templates static
var assets embed.FS

// Server は画面の描画と、ログイン状態の確認を持っています。
type Server struct {
	store *store.Store
	cfg   config.Config
	tmpl  *template.Template
}

// base はどの画面にも共通の見出し、CSRF トークン、通知、ログイン利用者です。
// Flash は成功、Warn は入力や業務ルールの誤り、Error は想定外の失敗です。
type base struct {
	Title string
	CSRF  string
	Flash string
	Warn  string
	Error string
	Me    *store.Person
}

// session はログインクッキーの中身です。CSRF はフォーム改ざんの照合に使います。
type session struct {
	PersonID string `json:"pid"`
	CSRF     string `json:"csrf"`
	Exp      int64  `json:"exp"`
}

// ctxKey は、リクエストの中に利用者とセッションを入れるときのキーの型です。
type ctxKey int

const (
	ctxPerson ctxKey = iota
	ctxSession
)

// NewServer は st（保存処理）と cfg（設定）から画面サーバを作ります。
// cfg.SessionKey が 32 バイト未満のときは error です。
func NewServer(st *store.Store, cfg config.Config) (*Server, error) {
	if len(cfg.SessionKey) < 32 {
		return nil, errors.New("session key is missing")
	}
	if cfg.StandardMinutes <= 0 {
		cfg.StandardMinutes = 480
	}
	funcs := template.FuncMap{
		"yen":    calc.FormatYen,
		"yenp":   formatYenPtr,
		"nump":   formatNumberPtr,
		"hours":  calc.FormatHours,
		"pdays":  func(n int64) string { return calc.FormatPersonDays(n, cfg.StandardMinutes) },
		"status": statusLabel,
		"sub":    func(a, b int64) int64 { return a - b },
		"day": func(s string) string {
			if s == "" {
				return "—"
			}
			return s
		},
	}
	tmpl, err := template.New("").Funcs(funcs).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{store: st, cfg: cfg, tmpl: tmpl}, nil
}

// Handler は URL と処理の対応表を返します。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", s.handleLoginGet)
	mux.HandleFunc("POST /login", s.handleLoginPost)
	mux.HandleFunc("POST /logout", s.requireLogin(s.handleLogout))
	mux.HandleFunc("GET /password", s.requireLogin(s.handlePasswordGet))
	mux.HandleFunc("POST /password", s.requireLogin(s.handlePasswordPost))
	mux.HandleFunc("GET /", s.requireLogin(s.handleHome))

	mux.HandleFunc("GET /projects", s.requireAdmin(s.handleProjects))
	mux.HandleFunc("GET /projects/new", s.requireAdmin(s.handleProjectNew))
	mux.HandleFunc("POST /projects", s.requireAdmin(s.handleProjectCreate))
	mux.HandleFunc("GET /projects/{id}", s.requireAdmin(s.handleProjectEdit))
	mux.HandleFunc("POST /projects/{id}", s.requireAdmin(s.handleProjectUpdate))
	mux.HandleFunc("POST /projects/{id}/delete", s.requireAdmin(s.handleProjectDelete))
	mux.HandleFunc("GET /projects/{id}/assign", s.requireAdmin(s.handleAssign))
	mux.HandleFunc("POST /projects/{id}/assignments", s.requireAdmin(s.handleAssignCreate))
	mux.HandleFunc("POST /projects/{id}/assignments/{aid}", s.requireAdmin(s.handleAssignUpdate))
	mux.HandleFunc("POST /projects/{id}/assignments/{aid}/delete", s.requireAdmin(s.handleAssignDelete))
	mux.HandleFunc("POST /projects/{id}/recalc", s.requireAdmin(s.handleRecalc))
	mux.HandleFunc("GET /projects/{id}/forecast", s.requireAdmin(s.handleForecast))

	mux.HandleFunc("GET /people", s.requireAdmin(s.handlePeople))
	mux.HandleFunc("GET /people/new", s.requireAdmin(s.handlePersonNew))
	mux.HandleFunc("POST /people", s.requireAdmin(s.handlePersonCreate))
	mux.HandleFunc("GET /people/{id}", s.requireAdmin(s.handlePersonEdit))
	mux.HandleFunc("POST /people/{id}", s.requireAdmin(s.handlePersonUpdate))
	mux.HandleFunc("POST /people/{id}/delete", s.requireAdmin(s.handlePersonDelete))
	mux.HandleFunc("POST /people/{id}/rates", s.requireAdmin(s.handleRateCreate))
	mux.HandleFunc("POST /people/{id}/rates/{rateID}/delete", s.requireAdmin(s.handleRateDelete))

	mux.HandleFunc("GET /roles", s.requireAdmin(s.handleRoles))
	mux.HandleFunc("POST /roles", s.requireAdmin(s.handleRoleCreate))
	mux.HandleFunc("POST /roles/{id}/move", s.requireAdmin(s.handleRoleMove))
	mux.HandleFunc("POST /roles/{id}", s.requireAdmin(s.handleRoleUpdate))
	mux.HandleFunc("POST /roles/{id}/delete", s.requireAdmin(s.handleRoleDelete))
	mux.HandleFunc("GET /processes", s.requireAdmin(s.handleProcesses))
	mux.HandleFunc("POST /processes", s.requireAdmin(s.handleProcessCreate))
	mux.HandleFunc("POST /processes/{id}/move", s.requireAdmin(s.handleProcessMove))
	mux.HandleFunc("POST /processes/{id}", s.requireAdmin(s.handleProcessUpdate))
	mux.HandleFunc("POST /processes/{id}/delete", s.requireAdmin(s.handleProcessDelete))

	mux.HandleFunc("GET /time", s.requireLogin(s.handleTimeList))
	mux.HandleFunc("GET /time/{projectID}", s.requireLogin(s.handleTimeEdit))
	mux.HandleFunc("POST /time/{projectID}", s.requireLogin(s.handleTimeSave))

	sub, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))
	return secure(mux)
}

// secure は next の前に、内容の種類をブラウザが誤認しないためのヘッダを付けます。
func secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// requireLogin は next の前に、ログイン済みかを確認します。未ログインはログイン画面へ戻します。
// パスワード変更が済んでいなければ、変更画面以外へは進ませません。
func (s *Server) requireLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "入力が大きすぎるか、形式が不正です。", http.StatusBadRequest)
				return
			}
		}
		me, sess, ok := s.current(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodPost && !csrfOK(r.FormValue("csrf"), sess.CSRF) {
			http.Error(w, "ページの有効期限が切れました。戻って再度操作してください。", http.StatusBadRequest)
			return
		}
		if me.MustChangePassword && r.URL.Path != "/password" && r.URL.Path != "/logout" {
			http.Redirect(w, r, "/password", http.StatusSeeOther)
			return
		}
		if r.URL.Path != "/logout" {
			s.writeSession(w, me.ID, sess.CSRF)
		}
		ctx := context.WithValue(r.Context(), ctxPerson, me)
		ctx = context.WithValue(ctx, ctxSession, sess)
		next(w, r.WithContext(ctx))
	}
}

// requireAdmin は next の前に、ログインと管理者であることを確認します。
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireLogin(func(w http.ResponseWriter, r *http.Request) {
		if !personFrom(r).IsAdmin {
			http.Error(w, "管理者権限がありません。", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

// personFrom は r に載っているログイン中の利用者を返します。未ログインなら nil です。
func personFrom(r *http.Request) *store.Person {
	p, _ := r.Context().Value(ctxPerson).(*store.Person)
	return p
}

// sessionFrom は r に載っているログイン情報を返します。
func sessionFrom(r *http.Request) session {
	sess, _ := r.Context().Value(ctxSession).(session)
	return sess
}

// current は r のクッキーから利用者とセッションを復元します。
// クッキーが無効、期限切れ、利用者が無効なら false です。
func (s *Server) current(r *http.Request) (*store.Person, session, bool) {
	c, err := r.Cookie("odysseus_session")
	if err != nil {
		return nil, session{}, false
	}
	sess, err := s.openSession(c.Value)
	if err != nil || sess.Exp < time.Now().Unix() {
		return nil, session{}, false
	}
	me, err := s.store.GetPerson(r.Context(), sess.PersonID)
	if err != nil || !me.Active {
		return nil, session{}, false
	}
	return &me, sess, true
}

// writeSession は personID のログインを w のクッキーに書きます。
// csrf が空なら新しい照合トークンを作ります。有効期限は 12 時間です。
func (s *Server) writeSession(w http.ResponseWriter, personID, csrf string) {
	if csrf == "" {
		csrf = newToken()
	}
	raw, err := json.Marshal(session{PersonID: personID, CSRF: csrf, Exp: time.Now().Add(12 * time.Hour).Unix()})
	if err != nil {
		return
	}
	mac := hmac.New(sha256.New, s.cfg.SessionKey)
	mac.Write(raw)
	value := base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	http.SetCookie(w, &http.Cookie{
		Name: "odysseus_session", Value: value, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: 12 * 3600,
	})
}

// clearSession は w に、ログインクッキーを消す指示を書きます。
func (s *Server) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: "odysseus_session", Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}

// openSession は value（クッキーの値）を検証して中身を返します。署名が違えば error です。
func (s *Server) openSession(value string) (session, error) {
	rawPart, sigPart, ok := strings.Cut(value, ".")
	if !ok {
		return session{}, errors.New("format")
	}
	raw, err := base64.RawURLEncoding.DecodeString(rawPart)
	if err != nil {
		return session{}, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigPart)
	if err != nil {
		return session{}, err
	}
	mac := hmac.New(sha256.New, s.cfg.SessionKey)
	mac.Write(raw)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return session{}, errors.New("signature")
	}
	var sess session
	if err := json.Unmarshal(raw, &sess); err != nil {
		return session{}, err
	}
	return sess, nil
}

const (
	noticeOK    = "ok"
	noticeWarn  = "warn"
	noticeError = "error"
)

// setNotice は kind（ok / warn / error）と msg を、次の画面で一度だけ出す通知として w に書きます。
func (s *Server) setNotice(w http.ResponseWriter, kind, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name: "odysseus_flash", Value: url.QueryEscape(kind + "|" + msg), Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: 120,
	})
}

// takeNotice は r の通知を読み、w でそのクッキーを消します。種類ごとに分けて返します。
func (s *Server) takeNotice(w http.ResponseWriter, r *http.Request) (flash, warn, errMsg string) {
	c, err := r.Cookie("odysseus_flash")
	if err != nil {
		return "", "", ""
	}
	http.SetCookie(w, &http.Cookie{
		Name: "odysseus_flash", Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
	raw, err := url.QueryUnescape(c.Value)
	if err != nil {
		return "", "", ""
	}
	kind, msg, ok := strings.Cut(raw, "|")
	if !ok {
		return raw, "", ""
	}
	switch kind {
	case noticeOK:
		return msg, "", ""
	case noticeWarn:
		return "", msg, ""
	case noticeError:
		return "", "", msg
	default:
		return raw, "", ""
	}
}

// redirect は path へ移動させます。msg が空でなければ、移動先で成功の通知を一度だけ出します。
func (s *Server) redirect(w http.ResponseWriter, r *http.Request, path, msg string) {
	if msg != "" {
		s.setNotice(w, noticeOK, msg)
	}
	http.Redirect(w, r, path, http.StatusSeeOther)
}

// redirectWarn は path へ移動させます。msg が空でなければ、移動先で警告を一度だけ出します。
func (s *Server) redirectWarn(w http.ResponseWriter, r *http.Request, path, msg string) {
	if msg != "" {
		s.setNotice(w, noticeWarn, msg)
	}
	http.Redirect(w, r, path, http.StatusSeeOther)
}

// page は title を見出しにした、画面共通の情報を作ります。r の通知はここで消費します。
func (s *Server) page(w http.ResponseWriter, r *http.Request, title string) base {
	flash, warn, errMsg := s.takeNotice(w, r)
	return base{Title: title, CSRF: sessionFrom(r).CSRF, Flash: flash, Warn: warn, Error: errMsg, Me: personFrom(r)}
}

// render は name のテンプレートを data で描画し、status の HTTP 状態で w に書きます。
func (s *Server) render(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}

// fail は err をログに残し、w には想定外の失敗として赤い通知だけ返します。SQL の中身は画面に出しません。
func (s *Server) fail(w http.ResponseWriter, err error) {
	log.Printf("error: %v", err)
	s.render(w, http.StatusInternalServerError, "failure", base{Title: "エラー", Error: "処理に失敗しました。"})
}

// initialHint は初期パスワードのファイルが残っているとき、ログイン画面用の案内を返します。
func (s *Server) initialHint() string {
	if _, err := os.Stat(filepath.Join(s.cfg.DataDir, "initial-admin.txt")); err == nil {
		return "初期パスワードは data/initial-admin.txt と、起動した画面に出しています。"
	}
	return ""
}

// formatYenPtr は n が nil なら「未設定」、そうでなければ円の表記を返します。
func formatYenPtr(n *int64) string {
	if n == nil {
		return "未設定"
	}
	return calc.FormatYen(*n)
}

// formatNumberPtr は n が nil なら空文字、そうでなければ桁区切り付きの数字を返します。金額入力欄の再表示に使います。
func formatNumberPtr(n *int64) string {
	if n == nil {
		return ""
	}
	return calc.FormatComma(*n)
}

// statusLabel は status（planning、active、closed）を画面用の日本語にします。
func statusLabel(status string) string {
	switch status {
	case "planning":
		return "計画中"
	case "active":
		return "進行中"
	case "closed":
		return "完了"
	default:
		return status
	}
}

// csrfOK は got（フォームの値）と want（セッションの値）が一致するかを、比較時間を揃えて判定します。
func csrfOK(got, want string) bool {
	if got == "" || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// newToken はログインとフォーム照合に使う、推測しにくい文字列を返します。
func newToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
