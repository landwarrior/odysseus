package web

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"odysseus/internal/calc"
	"odysseus/internal/report"
	"odysseus/internal/store"
)

// taskForm は工程下のタスク入力欄です。Days は人日表示です。
type taskForm struct {
	ID      string
	Name    string
	Remarks string
	Days    string
}

// lineForm は工程明細の入力欄です。Days はタスク合計の人日、Budget は円の文字列です。
type lineForm struct {
	ProcessID string
	Start     string
	End       string
	Days      string
	Budget    string
	Tasks     []taskForm
}

// managerOpt は案件責任者の候補1人です。Selected が真なら既に責任者です。
type managerOpt struct {
	ID       string
	Label    string
	Selected bool
}

// projectPage は案件の登録と編集です。IsNew が真のときは削除欄を出しません。
// CanManage は共通導線用、CanEditManagers が真のときだけ責任者の割当欄を出します。
type projectPage struct {
	base
	Project         store.Project
	Lines           []lineForm
	Processes       []store.Process
	Managers        []managerOpt
	CanManage       bool
	CanEditManagers bool
	IsNew           bool
	Status          string
}

// handleProjectNew は空の案件作成画面を出します。初期状態は進行中です。
func (s *Server) handleProjectNew(w http.ResponseWriter, r *http.Request) {
	s.renderProject(w, r, projectPage{
		base: s.page(w, r, "プロジェクト作成"), IsNew: true,
		Project: store.Project{Status: "active"}, Lines: []lineForm{{Tasks: []taskForm{{}}}},
	}, http.StatusOK)
}

// handleProjectCreate はフォームの案件と工程を登録し、編集画面へ移します。
// 作成者が責任者権限を持つときは、自分を案件責任者に必ず含めます。
func (s *Server) handleProjectCreate(w http.ResponseWriter, r *http.Request) {
	project, lines, forms, msg := s.readProject(r, "")
	me := personFrom(r)
	managerIDs := withDefaultManager(r.Form["manager_id"], me)
	failPage := projectPage{base: s.page(w, r, "プロジェクト作成"), Project: project, Lines: forms, IsNew: true}
	if msg != "" {
		s.renderProject(w, r, failPage, http.StatusBadRequest, msg)
		return
	}
	if msg := s.validateManagerIDs(r, managerIDs); msg != "" {
		s.renderProject(w, r, failPage, http.StatusBadRequest, msg)
		return
	}
	id, err := s.store.SaveProject(r.Context(), project, lines)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.renderProject(w, r, failPage, http.StatusBadRequest, "このプロジェクト番号は既に使われています。")
			return
		}
		s.fail(w, err)
		return
	}
	if err := s.saveProjectTasks(r, id, forms); err != nil {
		s.fail(w, err)
		return
	}
	if err := s.store.ReplaceProjectManagers(r.Context(), id, managerIDs); err != nil {
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/projects/"+id, "登録しました。")
}

// withDefaultManager は me が責任者権限を持つとき、ids に me.ID を足して返します。
func withDefaultManager(ids []string, me *store.Person) []string {
	if me == nil || !me.IsManager {
		return ids
	}
	for _, id := range ids {
		if id == me.ID {
			return ids
		}
	}
	return append(append([]string{}, ids...), me.ID)
}

// validateManagerIDs は責任者候補が、有効かつ責任者権限を持つか確認します。
// 問題があればメッセージ、なければ空です。
func (s *Server) validateManagerIDs(r *http.Request, personIDs []string) string {
	if len(personIDs) == 0 {
		return ""
	}
	people, err := s.store.ListPeople(r.Context())
	if err != nil {
		return "責任者の確認に失敗しました。"
	}
	ok := map[string]bool{}
	for _, p := range people {
		if p.Active && p.IsManager {
			ok[p.ID] = true
		}
	}
	for _, id := range personIDs {
		if id == "" {
			continue
		}
		if !ok[id] {
			return "責任者に選べない人材が含まれています。"
		}
	}
	return ""
}

// handleProjectEdit はパスの案件の編集画面を出します。
func (s *Server) handleProjectEdit(w http.ResponseWriter, r *http.Request) {
	project, lines, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	forms, err := s.lineFormsWithTasks(r, project.ID, lines)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.renderProject(w, r, projectPage{
		base: s.page(w, r, "プロジェクト編集"), Project: project, Lines: forms, CanManage: true,
	}, http.StatusOK)
}

// handleProjectUpdate はパスの案件をフォームの内容で更新します。
// 責任者の割当は、管理者またはその案件を編集できる責任者が変えられます。
func (s *Server) handleProjectUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, lines, forms, msg := s.readProject(r, id)
	page := projectPage{base: s.page(w, r, "プロジェクト編集"), Project: project, Lines: forms, CanManage: true}
	if msg != "" {
		s.renderProject(w, r, page, http.StatusBadRequest, msg)
		return
	}
	if _, err := s.store.SaveProject(r.Context(), project, lines); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.renderProject(w, r, page, http.StatusBadRequest, "このプロジェクト番号は既に使われています。")
			return
		}
		var used *store.InUseError
		if errors.As(err, &used) {
			s.renderProject(w, r, page, http.StatusBadRequest, used.Name+" はアサインがあるため外せません。先にアサインを削除してください。")
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	if err := s.saveProjectTasks(r, id, forms); err != nil {
		s.fail(w, err)
		return
	}
	managerIDs := r.Form["manager_id"]
	if msg := s.validateManagerIDs(r, managerIDs); msg != "" {
		s.renderProject(w, r, page, http.StatusBadRequest, msg)
		return
	}
	if err := s.store.ReplaceProjectManagers(r.Context(), id, managerIDs); err != nil {
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
		s.redirectWarn(w, r, "/projects/"+project.ID, "削除するにはプロジェクト番号を正しく入力してください。")
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
	for i := range forms {
		form := &forms[i]
		emptyLine := form.ProcessID == "" && form.Start == "" && form.End == "" && form.Budget == "" && !hasNamedTask(form.Tasks)
		if emptyLine {
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
		budget, berr := calc.ParseYen(form.Budget)
		if berr != nil {
			msg = firstMsg(msg, "予算は 0 以上の整数で入力してください。")
		}
		var taskSum int64
		for _, task := range form.Tasks {
			if stringsTrim(task.Name) == "" {
				continue
			}
			if m := limitRunes(stringsTrim(task.Name), "タスク名", 128); m != "" {
				msg = firstMsg(msg, m)
			}
			if m := limitRunes(stringsTrim(task.Remarks), "タスク備考", 500); m != "" {
				msg = firstMsg(msg, m)
			}
			taskMinutes, tm := parseOptionalPersonDays(task.Days, s.cfg.StandardMinutes)
			msg = firstMsg(msg, tm)
			if taskMinutes > 100000*60 {
				msg = firstMsg(msg, "タスクの見積工数が大きすぎます。")
			}
			taskSum += taskMinutes
		}
		if taskSum > 100000*60 {
			msg = firstMsg(msg, "合計工数が大きすぎます。")
		}
		form.Days = calc.FormatPersonDays(taskSum, s.cfg.StandardMinutes)
		lines = append(lines, store.LineWrite{
			ProcessID: form.ProcessID, StartDate: ls, EndDate: le, PlannedMinutes: taskSum, BudgetYen: budget,
		})
	}
	if len(forms) == 0 {
		forms = []lineForm{{Tasks: []taskForm{{}}}}
	}
	return project, lines, forms, msg
}

// readLineForms は r の繰り返し項目を、画面の行の順にまとめます。
func readLineForms(r *http.Request) []lineForm {
	ids := r.Form["process_id"]
	starts := r.Form["line_start"]
	ends := r.Form["line_end"]
	days := r.Form["line_days"]
	budgets := r.Form["line_budget"]
	n := len(ids)
	if len(starts) > n {
		n = len(starts)
	}
	out := make([]lineForm, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, lineForm{
			ProcessID: at(ids, i), Start: at(starts, i), End: at(ends, i), Days: at(days, i), Budget: at(budgets, i),
			Tasks: []taskForm{{}},
		})
	}
	attachTasks(out, r)
	return out
}

// attachTasks は task_process_id の並びに合わせて、各工程行へタスクを載せます。
func attachTasks(lines []lineForm, r *http.Request) {
	processIDs := r.Form["task_process_id"]
	ids := r.Form["task_id"]
	names := r.Form["task_name"]
	remarks := r.Form["task_remarks"]
	days := r.Form["task_days"]
	if len(processIDs) == 0 {
		return
	}
	byProcess := map[string][]taskForm{}
	n := len(processIDs)
	for i := 0; i < n; i++ {
		pid := at(processIDs, i)
		if pid == "" {
			continue
		}
		byProcess[pid] = append(byProcess[pid], taskForm{
			ID: at(ids, i), Name: at(names, i), Remarks: at(remarks, i), Days: at(days, i),
		})
	}
	for i := range lines {
		if lines[i].ProcessID == "" {
			continue
		}
		if tasks, ok := byProcess[lines[i].ProcessID]; ok {
			lines[i].Tasks = tasks
		}
	}
}

// at は values の i 番目を返します。範囲外なら空文字です。
func at(values []string, i int) string {
	if i < 0 || i >= len(values) {
		return ""
	}
	return values[i]
}

// lineForms は lines を入力欄の文字列に戻します。空なら空行を1つ足します。
func (s *Server) lineForms(lines []store.ProcessLine) []lineForm {
	if len(lines) == 0 {
		return []lineForm{{Tasks: []taskForm{{}}}}
	}
	std := s.cfg.StandardMinutes
	out := make([]lineForm, 0, len(lines))
	for _, line := range lines {
		budget := ""
		if line.BudgetYen != nil {
			budget = calc.FormatComma(*line.BudgetYen)
		}
		out = append(out, lineForm{
			ProcessID: line.ProcessID, Start: line.StartDate, End: line.EndDate,
			Days: calc.FormatPersonDays(line.PlannedMinutes, std), Budget: budget,
			Tasks: []taskForm{{}},
		})
	}
	return out
}

// lineFormsWithTasks は工程明細に、保存済みタスクを載せて返します。
func (s *Server) lineFormsWithTasks(r *http.Request, projectID string, lines []store.ProcessLine) ([]lineForm, error) {
	forms := s.lineForms(lines)
	tasks, err := s.store.ListTasksByProject(r.Context(), projectID)
	if err != nil {
		return nil, err
	}
	std := s.cfg.StandardMinutes
	byProcess := map[string][]taskForm{}
	for _, task := range tasks {
		byProcess[task.ProcessID] = append(byProcess[task.ProcessID], taskForm{
			ID: task.ID, Name: task.Name, Remarks: task.Remarks, Days: calc.FormatPersonDays(task.PlannedMinutes, std),
		})
	}
	for i := range forms {
		if tasks, ok := byProcess[forms[i].ProcessID]; ok && len(tasks) > 0 {
			forms[i].Tasks = tasks
		}
	}
	return forms, nil
}

// saveProjectTasks は forms のタスクを projectID へ保存します。
func (s *Server) saveProjectTasks(r *http.Request, projectID string, forms []lineForm) error {
	byProcess := map[string][]store.TaskWrite{}
	std := s.cfg.StandardMinutes
	for _, form := range forms {
		if form.ProcessID == "" {
			continue
		}
		var writes []store.TaskWrite
		for _, task := range form.Tasks {
			name := stringsTrim(task.Name)
			if name == "" {
				continue
			}
			minutes, _ := parseOptionalPersonDays(task.Days, std)
			writes = append(writes, store.TaskWrite{
				ID: task.ID, Name: name, Remarks: stringsTrim(task.Remarks), PlannedMinutes: minutes,
			})
		}
		// 工程行にタスク欄があったものだけ置き換える（空ならその工程のタスクを消す）。
		if len(form.Tasks) > 0 {
			byProcess[form.ProcessID] = writes
		}
	}
	return s.store.ReplaceTasksByProcess(r.Context(), projectID, byProcess)
}

func hasNamedTask(tasks []taskForm) bool {
	for _, task := range tasks {
		if stringsTrim(task.Name) != "" {
			return true
		}
	}
	return false
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}

// renderProject は page を status で描画します。errs の先頭があれば、それを警告として出します。
func (s *Server) renderProject(w http.ResponseWriter, r *http.Request, page projectPage, status int, errs ...string) {
	if len(errs) > 0 {
		page.Warn = errs[0]
	}
	processes, err := s.store.ListProcesses(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	page.Processes = selectableProcesses(processes, page.Lines)
	me := personFrom(r)
	if !page.IsNew {
		page.CanManage = true
	}
	page.CanEditManagers = (page.IsNew && canCreateProject(me)) || page.CanManage
	var formSelected []string
	if r.Method == http.MethodPost {
		formSelected = r.Form["manager_id"]
	}
	managers, err := s.projectManagerOpts(r, page.Project.ID, page.IsNew, formSelected, me)
	if err != nil {
		s.fail(w, err)
		return
	}
	page.Managers = managers
	s.render(w, status, "project_form", page)
}

// projectManagerOpts は責任者権限を持つ人材を候補にします。
// selectedIDs があればそれを優先し、新規作成で未指定なら作成者（責任者権限あり）を選びます。
// 既に案件責任者で権限を外された人も、選択済みとして残します。
func (s *Server) projectManagerOpts(r *http.Request, projectID string, isNew bool, selectedIDs []string, me *store.Person) ([]managerOpt, error) {
	people, err := s.store.ListPeople(r.Context())
	if err != nil {
		return nil, err
	}
	selected := map[string]bool{}
	if selectedIDs != nil {
		for _, id := range selectedIDs {
			selected[id] = true
		}
		if isNew {
			for _, id := range withDefaultManager(nil, me) {
				selected[id] = true
			}
		}
	} else if projectID != "" {
		ids, err := s.store.ListProjectManagerIDs(r.Context(), projectID)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			selected[id] = true
		}
	} else if isNew {
		for _, id := range withDefaultManager(nil, me) {
			selected[id] = true
		}
	}
	var out []managerOpt
	for _, p := range people {
		if !(p.IsManager && p.Active) && !selected[p.ID] {
			continue
		}
		label := p.Code + " " + p.FullName()
		if !p.IsManager {
			label += "（権限なし）"
		} else if !p.Active {
			label += "（無効）"
		}
		out = append(out, managerOpt{
			ID: p.ID, Label: label, Selected: selected[p.ID],
		})
	}
	return out, nil
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

// assignPage は案件の人材アサイン画面です。人材ごとに既存アサインをまとめ、追加は人→工程の順です。
type assignPage struct {
	base
	Project      store.Project
	CanManage    bool
	Groups       []assignPersonGroup
	Lines        []assignLineOpt
	People       []opt
	Roles        []opt
	DefaultStart string
	DefaultEnd   string
}

// assignPersonGroup は1人分の既存アサインです。
type assignPersonGroup struct {
	PersonID   string
	PersonCode string
	PersonName string
	Rows       []assignRow
}

// assignRow は既存アサイン1件と、役割の選択肢です。人材は見出し側なので行には出しません。
type assignRow struct {
	Assignment store.Assignment
	Roles      []opt
}

// assignLineOpt は追加フォームの工程1行です。Days は見積の人日表示です。
type assignLineOpt struct {
	ID    string
	Name  string
	Start string
	End   string
	Days  string
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
	byPerson := map[string][]store.Assignment{}
	var personOrder []string
	for _, a := range assignments {
		if _, ok := byPerson[a.PersonID]; !ok {
			personOrder = append(personOrder, a.PersonID)
		}
		byPerson[a.PersonID] = append(byPerson[a.PersonID], a)
	}
	sort.SliceStable(personOrder, func(i, j int) bool {
		a, b := byPerson[personOrder[i]][0], byPerson[personOrder[j]][0]
		if a.PersonCode != b.PersonCode {
			return a.PersonCode < b.PersonCode
		}
		return a.PersonName < b.PersonName
	})
	page := assignPage{
		base: s.page(w, r, "人材アサイン"), Project: project, CanManage: true,
		People: activePersonOpts(people), Roles: activeRoleOpts(roles),
		DefaultStart: project.StartDate, DefaultEnd: project.EndDate,
	}
	for _, personID := range personOrder {
		list := byPerson[personID]
		group := assignPersonGroup{
			PersonID: personID, PersonCode: list[0].PersonCode, PersonName: list[0].PersonName,
		}
		for _, a := range list {
			group.Rows = append(group.Rows, assignRow{
				Assignment: a, Roles: roleOpts(roles, a),
			})
		}
		page.Groups = append(page.Groups, group)
	}
	for _, line := range lines {
		page.Lines = append(page.Lines, assignLineOpt{
			ID: line.ID, Name: line.ProcessName, Start: line.StartDate, End: line.EndDate,
			Days: calc.FormatPersonDays(line.PlannedMinutes, s.cfg.StandardMinutes),
		})
	}
	return page, true
}

// handleAssignCreate はフォームの人材を、選んだ工程へまとめて追加します。
func (s *Server) handleAssignCreate(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	processIDs := r.Form["project_process_id"]
	if len(processIDs) == 0 {
		s.redirectWarn(w, r, "/projects/"+projectID+"/assign", "担当する工程を1つ以上選んでください。")
		return
	}
	personID := formText(r, "person_id")
	rateKind := formText(r, "rate_kind")
	startRaw := r.FormValue("start_date")
	endRaw := r.FormValue("end_date")
	seen := map[string]bool{}
	var writes []store.AssignmentWrite
	for _, processID := range processIDs {
		if seen[processID] {
			continue
		}
		seen[processID] = true
		roleID := formText(r, "role_"+processID)
		if roleID == "" {
			roleID = formText(r, "role_id")
		}
		write, msg := s.readAssignmentValues(r, projectID, "", personID, processID, roleID, rateKind, startRaw, endRaw)
		if msg != "" {
			s.redirectWarn(w, r, "/projects/"+projectID+"/assign", msg)
			return
		}
		writes = append(writes, write)
	}
	existing, err := s.store.ListAssignmentsByProject(r.Context(), projectID)
	if err != nil {
		s.fail(w, err)
		return
	}
	taken := map[string]bool{}
	for _, a := range existing {
		if a.PersonID == personID {
			taken[a.ProjectProcessID] = true
		}
	}
	for _, write := range writes {
		if taken[write.ProjectProcessID] {
			s.redirectWarn(w, r, "/projects/"+projectID+"/assign", "同じ人材が既に入っている工程があります。")
			return
		}
	}
	created := 0
	for _, write := range writes {
		if _, err := s.store.CreateAssignment(r.Context(), write); err != nil {
			if errors.Is(err, store.ErrConflict) {
				s.redirectWarn(w, r, "/projects/"+projectID+"/assign", "同じ人材が既に入っている工程があります。")
				return
			}
			s.fail(w, err)
			return
		}
		created++
	}
	if created == 1 {
		s.redirect(w, r, "/projects/"+projectID+"/assign", "登録しました。")
		return
	}
	s.redirect(w, r, "/projects/"+projectID+"/assign", "登録しました（"+strconv.Itoa(created)+" 件）。")
}

// handleAssignUpdate はパスのアサインをフォームの内容で更新し、予定原価を計算し直します。
func (s *Server) handleAssignUpdate(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	assignment, ok := s.ownAssignment(w, r)
	if !ok {
		return
	}
	write, msg := s.readAssignmentValues(r, projectID, assignment.ID,
		formText(r, "person_id"), assignment.ProjectProcessID, formText(r, "role_id"),
		formText(r, "rate_kind"), r.FormValue("start_date"), r.FormValue("end_date"))
	if msg != "" {
		s.redirectWarn(w, r, "/projects/"+projectID+"/assign", msg)
		return
	}
	write.ProjectProcessID = assignment.ProjectProcessID
	if err := s.store.UpdateAssignment(r.Context(), assignment.ID, write); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.redirectWarn(w, r, "/projects/"+projectID+"/assign", "この工程には、同じ人材が既にいます。")
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

// readAssignmentValues はアサイン1件分の入力を検証して返します。
// projectID は案件、assignmentID は更新対象です。新規なら assignmentID は空です。
// 開始日時点の単価が無いときはメッセージを返し、予定原価は計算しません。
func (s *Server) readAssignmentValues(r *http.Request, projectID, assignmentID, personID, processID, roleID, rateKind, startRaw, endRaw string) (store.AssignmentWrite, string) {
	write := store.AssignmentWrite{
		ProjectProcessID: processID,
		PersonID:         personID,
		RoleID:           roleID,
	}
	if assignmentID == "" {
		line, err := s.store.GetProcessLine(r.Context(), write.ProjectProcessID)
		if err != nil || line.ProjectID != projectID {
			return write, "工程が不正です。"
		}
	}
	start, msg := normalizeDate(startRaw, true)
	if msg != "" {
		return write, msg
	}
	end, msg := normalizeDate(endRaw, true)
	if msg != "" {
		return write, msg
	}
	if m := dateOrder(start, end, "開始日", "終了日"); m != "" {
		return write, m
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
	rate, ok, err := s.store.FindRate(r.Context(), person.ID, role.ID, start)
	if err != nil {
		return write, "単価の取得に失敗しました。"
	}
	if !ok {
		return write, "この開始日に、その役割の単価（時間または月間）がありません。"
	}
	switch rateKind {
	case store.RateKindHourly, store.RateKindMonthly:
	default:
		return write, "単価の種類（請負または準委任）を選んでください。"
	}
	if !store.RateKindOK(rateKind, rate, true) {
		if rateKind == store.RateKindMonthly {
			return write, "この開始日に月間単価がありません。準委任以外を選ぶか、単価を登録してください。"
		}
		return write, "この開始日に時間単価がありません。請負以外を選ぶか、単価を登録してください。"
	}
	write.StartDate = start
	write.EndDate = end
	write.RateKind = rateKind
	// 準委任の予定原価は予実で受注金額比按分するため、アサインには保存しない。
	write.PlannedCostYen = 0
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
	if len(missing) > 0 {
		s.redirectWarn(w, r, "/projects/"+id+"/forecast", "一部のアサインは開始日時点の単価がないため更新していません。")
		return
	}
	if updated == 0 {
		s.redirectWarn(w, r, "/projects/"+id+"/forecast", "再計算するアサインはありません。")
		return
	}
	s.redirect(w, r, "/projects/"+id+"/forecast", "予定原価を再計算しました。")
}

// forecastPage は1案件の予実画面です。
type forecastPage struct {
	base
	Project   store.Project
	CanManage bool
	Report    report.Report
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
	peers, err := s.store.ExpandMonthlyPeers(r.Context(), []store.Bundle{bundle})
	if err != nil {
		s.fail(w, err)
		return
	}
	page := forecastPage{
		base:      s.page(w, r, "予実"),
		Project:   bundle.Project,
		CanManage: true,
		Report:    report.BuildWithContext(bundle, peers, time.Time{}),
	}
	s.render(w, http.StatusOK, "forecast", page)
}

// activePersonOpts は people のうち有効な人だけを、プルダウンの選択肢にします。
func activePersonOpts(people []store.Person) []opt {
	var out []opt
	for _, p := range people {
		if p.Active {
			out = append(out, opt{ID: p.ID, Label: p.Code + " " + p.FullName()})
		}
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
