package web

import (
	"errors"
	"net/http"
	"time"

	"odysseus/internal/calc"
	"odysseus/internal/store"
)

// personListPage は人材の一覧です。
type personListPage struct {
	base
	People []personView
}

// personView は一覧の1行です。RoleName は表示日時点で有効な、いちばん新しい単価の役割です。
// 同じ適用開始日に複数あるときは、単価が最も高い役割です。
type personView struct {
	store.Person
	RoleName string
}

// personPage は人材の登録と編集です。IsNew が真のときは単価欄を出しません。
type personPage struct {
	base
	Person store.Person
	Rates  []store.Rate
	Roles  []store.Role
	IsNew  bool
}

// handlePeople は人材の一覧を出します。役割は、表示日時点で有効な最新の単価から付けます。
// 同じ適用開始日なら、単価が最も高い役割です。
func (s *Server) handlePeople(w http.ResponseWriter, r *http.Request) {
	people, err := s.store.ListPeople(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	roles, err := s.store.LatestRoleNames(r.Context(), time.Now().Format("2006-01-02"))
	if err != nil {
		s.fail(w, err)
		return
	}
	views := make([]personView, len(people))
	for i, person := range people {
		views[i] = personView{Person: person, RoleName: roles[person.ID]}
	}
	s.render(w, http.StatusOK, "people", personListPage{base: s.page(w, r, "人材"), People: views})
}

// handlePersonNew は空の人材登録画面を出します。初期状態は有効です。
func (s *Server) handlePersonNew(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusOK, "person_form", personPage{
		base: s.page(w, r, "人材の追加"), IsNew: true, Person: store.Person{Active: true},
	})
}

// handlePersonCreate はフォームの人材を登録し、単価を続けて入れる編集画面へ移します。
func (s *Server) handlePersonCreate(w http.ResponseWriter, r *http.Request) {
	in, msg := readPerson(r, true)
	if msg != "" {
		s.render(w, http.StatusBadRequest, "person_form", personPage{
			base:   warnBase(s.page(w, r, "人材の追加"), msg),
			IsNew:  true,
			Person: postedPerson(in),
		})
		return
	}
	person, err := s.store.InsertPerson(r.Context(), in)
	if errors.Is(err, store.ErrConflict) {
		s.render(w, http.StatusBadRequest, "person_form", personPage{
			base: warnBase(s.page(w, r, "人材の追加"), "このユーザーコードは既に使われています。"), IsNew: true, Person: postedPerson(in),
		})
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/people/"+person.ID, "登録しました。続けて単価を登録してください。")
}

// handlePersonEdit はパスの人材 ID の編集画面を出します。
func (s *Server) handlePersonEdit(w http.ResponseWriter, r *http.Request) {
	page, ok := s.loadPerson(w, r)
	if !ok {
		return
	}
	s.render(w, http.StatusOK, "person_form", page)
}

// handlePersonUpdate はパスの人材をフォームの内容で更新します。自分自身は無効にできません。
func (s *Server) handlePersonUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	in, msg := readPerson(r, false)
	if msg != "" {
		s.renderPersonError(w, r, id, in, msg)
		return
	}
	if id == personFrom(r).ID && !in.Active {
		s.renderPersonError(w, r, id, in, "自分自身は無効にできません。")
		return
	}
	if err := s.store.UpdatePerson(r.Context(), id, in); err != nil {
		switch {
		case errors.Is(err, store.ErrConflict):
			s.renderPersonError(w, r, id, in, "このユーザーコードは既に使われています。")
		case errors.Is(err, store.ErrLastAdmin):
			s.renderPersonError(w, r, id, in, "最後の有効な管理者は外せません。")
		case errors.Is(err, store.ErrNotFound):
			http.Error(w, "見つかりません。", http.StatusNotFound)
		default:
			s.fail(w, err)
		}
		return
	}
	s.redirect(w, r, "/people/"+id, "登録しました。")
}

// handlePersonDelete は、確認用に入力したユーザーコードが一致したときだけ削除します。
func (s *Server) handlePersonDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == personFrom(r).ID {
		s.redirectWarn(w, r, "/people/"+id, "自分自身は削除できません。")
		return
	}
	person, err := s.store.GetPerson(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "見つかりません。", http.StatusNotFound)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	if formText(r, "confirm_code") != person.Code {
		s.redirectWarn(w, r, "/people/"+id, "削除するにはユーザーコードを正しく入力してください。")
		return
	}
	if err := s.store.DeletePerson(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrInUse) {
			s.redirectWarn(w, r, "/people/"+id, "アサインがあるため削除できません。無効にしてください。")
			return
		}
		if errors.Is(err, store.ErrLastAdmin) {
			s.redirectWarn(w, r, "/people/"+id, "最後の有効な管理者は削除できません。")
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/people", "削除しました。")
}

// handleRateCreate はパスの人材に、役割、適用開始日、時間単価・月間単価を追加します。
func (s *Server) handleRateCreate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.store.GetPerson(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	date, msg := normalizeDate(r.FormValue("from_date"), true)
	hourly, herr := calc.ParseYen(r.FormValue("hourly_yen"))
	if herr != nil {
		msg = firstMsg(msg, "時間単価（円 / 時）は 0 以上の整数で入力してください。")
	}
	monthly, merr := calc.ParseYen(r.FormValue("monthly_yen"))
	if merr != nil {
		msg = firstMsg(msg, "月間単価（円 / 月）は 0 以上の整数で入力してください。")
	}
	if hourly == nil && monthly == nil {
		msg = firstMsg(msg, "時間単価か月間単価のどちらかを入力してください。0 円でも登録できます。")
	}
	roleID := formText(r, "role_id")
	if roleID == "" {
		msg = firstMsg(msg, "役割を選択してください。")
	}
	if msg != "" {
		s.redirectWarn(w, r, "/people/"+id, msg)
		return
	}
	if _, err := s.store.InsertRate(r.Context(), store.Rate{
		PersonID: id, RoleID: roleID, FromDate: date, HourlyYen: hourly, MonthlyYen: monthly,
	}); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.redirectWarn(w, r, "/people/"+id, "同じ役割と適用開始日の単価が既にあります。")
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/people/"+id, "単価を登録しました。")
}

// handleRateDelete はパスの人材に属する単価を削除します。
func (s *Server) handleRateDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteRate(r.Context(), id, r.PathValue("rateID")); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/people/"+id, "単価を削除しました。")
}

// loadPerson はパスの人材と、その単価、役割一覧を読みます。無ければ 404 で false です。
func (s *Server) loadPerson(w http.ResponseWriter, r *http.Request) (personPage, bool) {
	person, err := s.store.GetPerson(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "見つかりません。", http.StatusNotFound)
		return personPage{}, false
	}
	if err != nil {
		s.fail(w, err)
		return personPage{}, false
	}
	rates, err := s.store.ListRates(r.Context(), person.ID)
	if err != nil {
		s.fail(w, err)
		return personPage{}, false
	}
	roles, err := s.store.ListRoles(r.Context())
	if err != nil {
		s.fail(w, err)
		return personPage{}, false
	}
	return personPage{base: s.page(w, r, "人材の編集"), Person: person, Rates: rates, Roles: roles}, true
}

// renderPersonError は id の編集画面を、in の入力内容と msg の警告で描き直します。
func (s *Server) renderPersonError(w http.ResponseWriter, r *http.Request, id string, in store.NewPerson, msg string) {
	page, ok := s.loadPerson(w, r)
	if !ok {
		return
	}
	page.Warn = msg
	page.Person.Code = in.Code
	page.Person.Name = in.Name
	page.Person.NameKana = in.NameKana
	page.Person.Remarks = in.Remarks
	page.Person.IsAdmin = in.IsAdmin
	page.Person.IsManager = in.IsManager
	page.Person.Active = in.Active
	page.Person.ID = id
	s.render(w, http.StatusBadRequest, "person_form", page)
}

// readPerson は r のフォームを人材の入力にします。
// passwordRequired が真のときはパスワード必須です。エラーがあればメッセージを返します。
func readPerson(r *http.Request, passwordRequired bool) (store.NewPerson, string) {
	in := store.NewPerson{
		Code: formText(r, "code"), Name: formText(r, "name"), NameKana: formText(r, "name_kana"),
		Password: r.FormValue("password"), Remarks: formText(r, "remarks"),
		IsAdmin: checkbox(r, "is_admin"), IsManager: checkbox(r, "is_manager"), Active: checkbox(r, "active"),
	}
	var msg string
	if in.Code == "" {
		msg = "ユーザーコードを入力してください。"
	}
	msg = firstMsg(msg, limitRunes(in.Code, "ユーザーコード", 32))
	if in.Name == "" {
		msg = firstMsg(msg, "氏名を入力してください。")
	}
	msg = firstMsg(msg, limitRunes(in.Name, "氏名", 128))
	msg = firstMsg(msg, limitRunes(in.NameKana, "カナ", 128))
	msg = firstMsg(msg, limitRunes(in.Remarks, "備考", 1000))
	msg = firstMsg(msg, passwordIssue(in.Password, passwordRequired))
	return in, msg
}

// postedPerson は in を、登録に失敗した画面へ戻す表示用の人材にします。パスワードは含めません。
func postedPerson(in store.NewPerson) store.Person {
	return store.Person{
		Code: in.Code, Name: in.Name, NameKana: in.NameKana, Remarks: in.Remarks,
		IsAdmin: in.IsAdmin, IsManager: in.IsManager, Active: in.Active,
	}
}

// warnBase は b に msg を警告として載せたコピーを返します。
func warnBase(b base, msg string) base {
	b.Warn = msg
	return b
}
