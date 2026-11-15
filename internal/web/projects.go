package web

import (
	"errors"
	"net/http"
	"strconv"

	"odysseus/internal/calc"
	"odysseus/internal/report"
	"odysseus/internal/store"
)

// lineForm は工程明細の入力欄です。時間と予算は、まだ数値に変換する前の文字列です。
type lineForm struct {
	ProcessID string
	Start     string
	End       string
	Hours     string
	Budget    string
}

// projectPage は案件の登録と編集です。IsNew が真のときは削除欄を出しません。
type projectPage struct {
	base
	Project   store.Project
	Lines     []lineForm
	Processes []store.Process
	IsNew     bool
	Status    string
}

// projectListPage は案件一覧です。Status が空ならすべての状態を出します。
type projectListPage struct {
	base
	Projects []store.Project
	Status   string
}

// handleProjects は案件一覧を出します。クエリ status があれば、その状態だけに絞ります。
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && !validStatus(status) {
		status = ""
	}
	projects, err := s.store.ListProjects(r.Context(), status)
	if err != nil {
		s.fail(w, err)
		return
	}
	page := projectListPage{base: s.page(w, r, "プロジェクト"), Projects: projects, Status: status}
	s.render(w, http.StatusOK, "projects", page)
}

// handleProjectNew は空の案件作成画面を出します。初期状態は進行中です。
func (s *Server) handleProjectNew(w http.ResponseWriter, r *http.Request) {
	s.renderProject(w, r, projectPage{
		base: s.page(w, r, "プロジェクト作成"), IsNew: true,
		Project: store.Project{Status: "active"}, Lines: []lineForm{{}},
	}, http.StatusOK)
}

// handleProjectCreate はフォームの案件と工程を登録し、編集画面へ移します。
func (s *Server) handleProjectCreate(w http.ResponseWriter, r *http.Request) {
	project, lines, forms, msg := s.readProject(r, "")
	if msg != "" {
		s.renderProject(w, r, projectPage{base: s.page(w, r, "プロジェクト作成"), Project: project, Lines: forms, IsNew: true}, http.StatusBadRequest, msg)
		return
	}
	id, err := s.store.SaveProject(r.Context(), project, lines)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.renderProject(w, r, projectPage{base: s.page(w, r, "プロジェクト作成"), Project: project, Lines: forms, IsNew: true}, http.StatusBadRequest, "このプロジェクト番号は既に使われています。")
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/projects/"+id, "登録しました。")
}

// handleProjectEdit はパスの案件の編集画面を出します。
func (s *Server) handleProjectEdit(w http.ResponseWriter, r *http.Request) {
	project, lines, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	s.renderProject(w, r, projectPage{
		base: s.page(w, r, "プロジェクト編集"), Project: project, Lines: lineForms(lines),
	}, http.StatusOK)
}

// handleProjectUpdate はパスの案件をフォームの内容で更新します。
func (s *Server) handleProjectUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, lines, forms, msg := s.readProject(r, id)
	if msg != "" {
		s.renderProject(w, r, projectPage{base: s.page(w, r, "プロジェクト編集"), Project: project, Lines: forms}, http.StatusBadRequest, msg)
		return
	}
	if _, err := s.store.SaveProject(r.Context(), project, lines); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.renderProject(w, r, projectPage{base: s.page(w, r, "プロジェクト編集"), Project: project, Lines: forms}, http.StatusBadRequest, "このプロジェクト番号は既に使われています。")
			return
		}
		var used *store.InUseError
		if errors.As(err, &used) {
			s.renderProject(w, r, projectPage{base: s.page(w, r, "プロジェクト編集"), Project: project, Lines: forms}, http.StatusBadRequest, used.Name+" はアサインがあるため外せません。先にアサインを削除してください。")
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/projects/"+id, "登録しました。")
}

// handleProjectDelete は、確認用のプロジェクト番号が一致したときだけ案件を削除します。
func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request) {
	project, err := s.store.GetProject(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "見つかりません。", http.StatusNotFound)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	if formText(r, "confirm_code") != project.Code {
		s.redirect(w, r, "/projects/"+project.ID, "削除するにはプロジェクト番号を正しく入力してください。")
		return
	}
	if err := s.store.DeleteProject(r.Context(), project.ID); err != nil {
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/projects", "削除しました。")
}

// loadProject はパスの案件と工程明細を読みます。無ければ 404 で false です。
func (s *Server) loadProject(w http.ResponseWriter, r *http.Request) (store.Project, []store.ProcessLine, bool) {
	project, err := s.store.GetProject(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "見つかりません。", http.StatusNotFound)
		return store.Project{}, nil, false
	}
	if err != nil {
		s.fail(w, err)
		return store.Project{}, nil, false
	}
	lines, err := s.store.ListProcessLines(r.Context(), project.ID)
	if err != nil {
		s.fail(w, err)
		return store.Project{}, nil, false
	}
	return project, lines, true
}

// readProject は r のフォームを案件と工程にします。id は更新時の案件 ID で、新規なら空です。
// 4 つ目の戻り値は、入力エラーのメッセージです。正常なら空です。
func (s *Server) readProject(r *http.Request, id string) (store.Project, []store.LineWrite, []lineForm, string) {
	project := store.Project{
		ID: id, Code: formText(r, "code"), Name: formText(r, "name"), Status: formText(r, "status"),
	}
	var msg string
	if project.Code == "" {
		msg = "プロジェクト番号を入力してください。"
	} else if m := limitRunes(project.Code, "プロジェクト番号", 32); m != "" {
		msg = m
	}
	if project.Name == "" {
		msg = firstMsg(msg, "プロジェクト名を入力してください。")
	} else if m := limitRunes(project.Name, "プロジェクト名", 128); m != "" {
		msg = firstMsg(msg, m)
	}
	if !validStatus(project.Status) {
		msg = firstMsg(msg, "状態が不正です。")
	}
	start, m := normalizeDate(r.FormValue("start_date"), false)
	msg = firstMsg(msg, m)
	end, m := normalizeDate(r.FormValue("end_date"), false)
	msg = firstMsg(msg, m)
	msg = firstMsg(msg, dateOrder(start, end, "開始日", "終了日"))
	project.StartDate, project.EndDate = start, end
	order, err := calc.ParseYen(r.FormValue("order_amount"))
	if err != nil {
		msg = firstMsg(msg, "受注金額は 0 以上の整数で入力してください。")
	}
	project.OrderAmount = order

	forms := readLineForms(r)
	processes, perr := s.store.ListProcesses(r.Context())
	if perr != nil {
		return project, nil, forms, "工程の取得に失敗しました。"
	}
	allowed := map[string]store.Process{}
	for _, process := range processes {
		allowed[process.ID] = process
	}
	seen := map[string]bool{}
	var lines []store.LineWrite
	for _, form := range forms {
		if form.ProcessID == "" && form.Start == "" && form.End == "" && form.Hours == "" && form.Budget == "" {
			continue
		}
		if form.ProcessID == "" {
			msg = firstMsg(msg, "工程を選択してください。")
			continue
		}
		process, ok := allowed[form.ProcessID]
		if !ok {
			msg = firstMsg(msg, "工程が不正です。")
			continue
		}
		if seen[form.ProcessID] {
			msg = firstMsg(msg, "同じ工程は 1 件だけにしてください。")
			continue
		}
		seen[form.ProcessID] = true
		ls, m := normalizeDate(form.Start, false)
		msg = firstMsg(msg, m)
		le, m := normalizeDate(form.End, false)
		msg = firstMsg(msg, m)
		msg = firstMsg(msg, dateOrder(ls, le, process.Name+"の開始日", "終了日"))
		minutes, m := parseOptionalHours(form.Hours)
		msg = firstMsg(msg, m)
		if minutes > 100000*60 {
			msg = firstMsg(msg, "予定時間が大きすぎます。")
		}
		budget, berr := calc.ParseYen(form.Budget)
		if berr != nil {
			msg = firstMsg(msg, "予算は 0 以上の整数で入力してください。")
		}
		lines = append(lines, store.LineWrite{
			ProcessID: form.ProcessID, StartDate: ls, EndDate: le, PlannedMinutes: minutes, BudgetYen: budget,
		})
	}
	if len(forms) == 0 {
		forms = []lineForm{{}}
	}
	return project, lines, forms, msg
}

// readLineForms は r の繰り返し項目を、画面の行の順にまとめます。
func readLineForms(r *http.Request) []lineForm {
	ids := r.Form["process_id"]
	starts := r.Form["line_start"]
	ends := r.Form["line_end"]
	hours := r.Form["line_hours"]
	budgets := r.Form["line_budget"]
	n := len(ids)
	if len(starts) > n {
		n = len(starts)
	}
	out := make([]lineForm, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, lineForm{
			ProcessID: at(ids, i), Start: at(starts, i), End: at(ends, i), Hours: at(hours, i), Budget: at(budgets, i),
		})
	}
	return out
}

// at は values の i 番目を返します。範囲外なら空文字です。
func at(values []string, i int) string {
	if i < 0 || i >= len(values) {
		return ""
	}
	return values[i]
}

// lineForms は lines を入力欄の文字列に戻します。空なら空行を1つ足します。
func lineForms(lines []store.ProcessLine) []lineForm {
	if len(lines) == 0 {
		return []lineForm{{}}
	}
	out := make([]lineForm, 0, len(lines))
	for _, line := range lines {
		budget := ""
		if line.BudgetYen != nil {
			budget = strconv.FormatInt(*line.BudgetYen, 10)
		}
		out = append(out, lineForm{
			ProcessID: line.ProcessID, Start: line.StartDate, End: line.EndDate,
			Hours: calc.FormatHours(line.PlannedMinutes), Budget: budget,
		})
	}
	return out
}

// renderProject は page を status で描画します。errs の先頭があれば、それをエラーとして出します。
func (s *Server) renderProject(w http.ResponseWriter, r *http.Request, page projectPage, status int, errs ...string) {
	if len(errs) > 0 {
		page.Error = errs[0]
	}
	processes, err := s.store.ListProcesses(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	page.Processes = selectableProcesses(processes, page.Lines)
	s.render(w, status, "project_form", page)
}

// selectableProcesses は all のうち、有効な工程と、lines で既に選ばれている無効な工程を返します。
func selectableProcesses(all []store.Process, lines []lineForm) []store.Process {
	used := map[string]bool{}
	for _, line := range lines {
		if line.ProcessID != "" {
			used[line.ProcessID] = true
		}
	}
	var out []store.Process
	for _, process := range all {
		if process.Active || used[process.ID] {
			out = append(out, process)
		}
	}
	return out
}

// firstMsg は current が空でなければそれを、空なら next を返します。最初のエラーを残すためです。
func firstMsg(current, next string) string {
	if current != "" {
		return current
	}
	return next
}

// assignPage は案件の人材アサイン画面です。
type assignPage struct {
	base
	Project store.Project
	Groups  []assignGroup
	People  []opt
	Roles   []opt
}

// assignGroup は工程1つと、そこへの既存アサインです。
type assignGroup struct {
	Line  store.ProcessLine
	Rows  []assignRow
	Hours string
}

// assignRow は既存アサイン1件と、選択肢、予定時間の表示文字列です。
type assignRow struct {
	Assignment store.Assignment
	People     []opt
	Roles      []opt
	Hours      string
}

// opt はプルダウンの1件です。ID が送信値、Label が画面の文言です。
type opt struct {
	ID    string
	Label string
}

// handleAssign はパスの案件のアサイン画面を出します。
func (s *Server) handleAssign(w http.ResponseWriter, r *http.Request) {
	page, ok := s.prepareAssign(w, r)
	if !ok {
		return
	}
	s.render(w, http.StatusOK, "assign", page)
}

// prepareAssign はパスの案件のアサイン画面に必要なデータを集めます。
func (s *Server) prepareAssign(w http.ResponseWriter, r *http.Request) (assignPage, bool) {
	project, lines, ok := s.loadProject(w, r)
	if !ok {
		return assignPage{}, false
	}
	assignments, err := s.store.ListAssignmentsByProject(r.Context(), project.ID)
	if err != nil {
		s.fail(w, err)
		return assignPage{}, false
	}
	people, err := s.store.ListPeople(r.Context())
	if err != nil {
		s.fail(w, err)
		return assignPage{}, false
	}
	roles, err := s.store.ListRoles(r.Context())
	if err != nil {
		s.fail(w, err)
		return assignPage{}, false
	}
	byLine := map[string][]store.Assignment{}
	for _, a := range assignments {
		byLine[a.ProjectProcessID] = append(byLine[a.ProjectProcessID], a)
	}
	page := assignPage{
		base: s.page(w, r, "人材アサイン"), Project: project,
		People: activePersonOpts(people), Roles: activeRoleOpts(roles),
	}
	for _, line := range lines {
		group := assignGroup{Line: line, Hours: calc.FormatHours(line.PlannedMinutes)}
		for _, a := range byLine[line.ID] {
			group.Rows = append(group.Rows, assignRow{
				Assignment: a, Hours: calc.FormatHours(a.PlannedMinutes),
				People: personOpts(people, a), Roles: roleOpts(roles, a),
			})
		}
		page.Groups = append(page.Groups, group)
	}
	return page, true
}

// handleAssignCreate はパスの案件の工程へ、フォームの人材を追加します。
func (s *Server) handleAssignCreate(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	write, msg := s.readAssignment(r, projectID, "")
	if msg != "" {
		s.redirect(w, r, "/projects/"+projectID+"/assign", msg)
		return
	}
	if _, err := s.store.CreateAssignment(r.Context(), write); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.redirect(w, r, "/projects/"+projectID+"/assign", "この工程には、同じ人材が既にいます。")
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/projects/"+projectID+"/assign", "登録しました。")
}

// handleAssignUpdate はパスのアサインをフォームの内容で更新し、予定原価を計算し直します。
func (s *Server) handleAssignUpdate(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	assignment, ok := s.ownAssignment(w, r)
	if !ok {
		return
	}
	write, msg := s.readAssignment(r, projectID, assignment.ID)
	if msg != "" {
		s.redirect(w, r, "/projects/"+projectID+"/assign", msg)
		return
	}
	write.ProjectProcessID = assignment.ProjectProcessID
	if err := s.store.UpdateAssignment(r.Context(), assignment.ID, write); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.redirect(w, r, "/projects/"+projectID+"/assign", "この工程には、同じ人材が既にいます。")
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/projects/"+projectID+"/assign", "登録しました。")
}

// handleAssignDelete はパスのアサインと、その実績を削除します。
func (s *Server) handleAssignDelete(w http.ResponseWriter, r *http.Request) {
	assignment, ok := s.ownAssignment(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteAssignment(r.Context(), assignment.ID); err != nil {
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/projects/"+assignment.ProjectID+"/assign", "削除しました。")
}

// ownAssignment はパスのアサインが、同じパスの案件に属することを確認して返します。
func (s *Server) ownAssignment(w http.ResponseWriter, r *http.Request) (store.Assignment, bool) {
	assignment, err := s.store.GetAssignment(r.Context(), r.PathValue("aid"))
	if errors.Is(err, store.ErrNotFound) || (err == nil && assignment.ProjectID != r.PathValue("id")) {
		http.Error(w, "見つかりません。", http.StatusNotFound)
		return store.Assignment{}, false
	}
	if err != nil {
		s.fail(w, err)
		return store.Assignment{}, false
	}
	return assignment, true
}

// readAssignment は r のフォームをアサイン入力にします。
// projectID は案件、assignmentID は更新対象です。新規なら assignmentID は空です。
// 開始日時点の単価が無いときはメッセージを返し、予定原価は計算しません。
func (s *Server) readAssignment(r *http.Request, projectID, assignmentID string) (store.AssignmentWrite, string) {
	write := store.AssignmentWrite{
		ProjectProcessID: formText(r, "project_process_id"),
		PersonID:         formText(r, "person_id"),
		RoleID:           formText(r, "role_id"),
	}
	if assignmentID == "" {
		line, err := s.store.GetProcessLine(r.Context(), write.ProjectProcessID)
		if err != nil || line.ProjectID != projectID {
			return write, "工程が不正です。"
		}
	}
	start, msg := normalizeDate(r.FormValue("start_date"), true)
	if msg != "" {
		return write, msg
	}
	end, msg := normalizeDate(r.FormValue("end_date"), true)
	if msg != "" {
		return write, msg
	}
	if m := dateOrder(start, end, "開始日", "終了日"); m != "" {
		return write, m
	}
	minutes, msg := parseRequiredHours(r.FormValue("hours"))
	if msg != "" {
		return write, msg
	}
	person, err := s.store.GetPerson(r.Context(), write.PersonID)
	if err != nil {
		return write, "人材が不正です。"
	}
	if !person.Active {
		allowed := false
		if assignmentID != "" {
			current, err := s.store.GetAssignment(r.Context(), assignmentID)
			allowed = err == nil && current.PersonID == person.ID
		}
		if !allowed {
			return write, "無効な人材は新しく割り当てられません。"
		}
	}
	roles, err := s.store.ListRoles(r.Context())
	if err != nil {
		return write, "役割の取得に失敗しました。"
	}
	var role store.Role
	found := false
	for _, item := range roles {
		if item.ID == write.RoleID {
			role = item
			found = true
		}
	}
	if !found {
		return write, "役割が不正です。"
	}
	if !role.Active && assignmentID != "" {
		current, err := s.store.GetAssignment(r.Context(), assignmentID)
		if err != nil || current.RoleID != role.ID {
			return write, "無効な役割は新しく割り当てられません。"
		}
	}
	if !role.Active && assignmentID == "" {
		return write, "無効な役割は新しく割り当てられません。"
	}
	yen, ok, err := s.store.FindHourly(r.Context(), person.ID, role.ID, start)
	if err != nil {
		return write, "単価の取得に失敗しました。"
	}
	if !ok {
		return write, "この開始日に、その役割の時間単価がありません。"
	}
	write.StartDate = start
	write.EndDate = end
	write.PlannedMinutes = minutes
	write.PlannedCostYen = calc.CostYen(minutes, yen)
	return write, ""
}

// handleRecalc はパスの案件の予定原価を、現在の単価で計算し直して予実画面へ戻します。
func (s *Server) handleRecalc(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.store.GetProject(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	updated, missing, err := s.store.RecalcPlannedCosts(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	msg := "予定原価を再計算しました。"
	if len(missing) > 0 {
		msg = "一部のアサインは開始日時点の単価がないため更新していません。"
	} else if updated == 0 {
		msg = "再計算するアサインはありません。"
	}
	s.redirect(w, r, "/projects/"+id+"/forecast", msg)
}

// forecastPage は1案件の予実画面です。
type forecastPage struct {
	base
	Project store.Project
	Report  report.Report
}

// handleForecast はパスの案件の予実を計算して出します。
func (s *Server) handleForecast(w http.ResponseWriter, r *http.Request) {
	bundle, err := s.store.LoadBundle(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "見つかりません。", http.StatusNotFound)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	page := forecastPage{base: s.page(w, r, "予実"), Project: bundle.Project, Report: report.Build(bundle)}
	s.render(w, http.StatusOK, "forecast", page)
}

// activePersonOpts は people のうち有効な人だけを、プルダウンの選択肢にします。
func activePersonOpts(people []store.Person) []opt {
	var out []opt
	for _, p := range people {
		if p.Active {
			out = append(out, opt{ID: p.ID, Label: p.Code + " " + p.Name})
		}
	}
	return out
}

// personOpts は有効な人に加え、current の人が無効でも選択肢に残します。
func personOpts(people []store.Person, current store.Assignment) []opt {
	out := activePersonOpts(people)
	for _, item := range out {
		if item.ID == current.PersonID {
			return out
		}
	}
	if current.PersonID != "" {
		out = append(out, opt{ID: current.PersonID, Label: current.PersonCode + " " + current.PersonName + "（無効）"})
	}
	return out
}

// activeRoleOpts は roles のうち有効な役割だけを選択肢にします。
func activeRoleOpts(roles []store.Role) []opt {
	var out []opt
	for _, role := range roles {
		if role.Active {
			out = append(out, opt{ID: role.ID, Label: role.Name})
		}
	}
	return out
}

// roleOpts は有効な役割に加え、current の役割が無効でも選択肢に残します。
func roleOpts(roles []store.Role, current store.Assignment) []opt {
	out := activeRoleOpts(roles)
	for _, item := range out {
		if item.ID == current.RoleID {
			return out
		}
	}
	if current.RoleID != "" {
		out = append(out, opt{ID: current.RoleID, Label: current.RoleName + "（無効）"})
	}
	return out
}
