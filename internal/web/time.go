package web

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"time"

	"odysseus/internal/calc"
	"odysseus/internal/store"
)

// timePerson は実績入力で選ぶ人材1人です。
type timePerson struct {
	ID   string
	Code string
	Name string
}

// timeColumn は実績入力の1欄です。タスクがある工程はタスクごと、無い工程はアサイン1欄です。
type timeColumn struct {
	Key          string
	AssignmentID string
	TaskID       string
	Label        string
	StartDate    string
	EndDate      string
	InRange      bool
}

// timeProcessGroup は1工程の入力欄の塊です。
type timeProcessGroup struct {
	ProcessName string
	Period      string
	Columns     []timeColumn
}

// timeProjectGroup は1案件の工程グループの塊です。
type timeProjectGroup struct {
	ProjectID   string
	ProjectCode string
	ProjectName string
	Processes   []timeProcessGroup
}

// timePage は横断の日次実績入力です。Cells はフォームの日付に既にある時間です（キーは Columns.Key）。
type timePage struct {
	base
	Date          string
	PrevDate      string
	NextDate      string
	PersonID      string
	People        []timePerson
	ShowPerson    bool
	CanPickPerson bool
	Groups        []timeProjectGroup
	Columns       []timeColumn
	Cells          map[string]string
	DayTotal       string
	HiddenDayTotal string // フォームに出ない同日実績。JS の合計に足す。
	SavedDates     []string
}

// handleTime は横断の実績入力画面を出します。
func (s *Server) handleTime(w http.ResponseWriter, r *http.Request) {
	page, ok := s.prepareTime(w, r, "")
	if !ok {
		return
	}
	s.render(w, http.StatusOK, "time_form", page)
}

// handleTimeProjectRedirect は旧・案件単位 URL を横断画面へ移します。
func (s *Server) handleTimeProjectRedirect(w http.ResponseWriter, r *http.Request) {
	q := url.Values{}
	if date := r.URL.Query().Get("date"); date != "" {
		q.Set("date", date)
	} else if month, ok := normalizeMonth(r.URL.Query().Get("month")); ok {
		today := time.Now().Format("2006-01-02")
		if today[:7] == month {
			q.Set("date", today)
		} else {
			q.Set("date", month+"-01")
		}
	}
	if person := r.URL.Query().Get("person"); person != "" {
		q.Set("person", person)
	}
	path := "/time"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	http.Redirect(w, r, path, http.StatusFound)
}

// handleTimeSave はフォームの日付と、全案件のタスク（または工程）ごとの時間を保存します。
// 空欄と 0 は削除、1 日の合計が 24 時間を超える入力と期間外の入力は拒否します。
func (s *Server) handleTimeSave(w http.ResponseWriter, r *http.Request) {
	page, columns, ok := s.timeContext(w, r)
	if !ok {
		return
	}
	date, msg := normalizeDate(r.FormValue("work_date"), true)
	if msg != "" {
		page.Warn = msg
		s.render(w, http.StatusBadRequest, "time_form", page)
		return
	}
	var cells []store.DayCell
	var dayTotal int64
	visible := map[string]bool{}
	for _, column := range columns {
		visible[column.AssignmentID] = true
		raw := r.FormValue("minutes_" + column.Key)
		var minutes int64
		if raw != "" {
			parsed, errText := parseOptionalHours(raw)
			if errText != "" {
				page.Date = date
				page.Warn = column.Label + "の" + errText
				s.render(w, http.StatusBadRequest, "time_form", page)
				return
			}
			minutes = parsed
		}
		inRange := date >= column.StartDate && date <= column.EndDate
		if minutes > 0 && !inRange {
			page.Date = date
			page.Warn = column.Label + "は " + column.StartDate + " から " + column.EndDate + " の範囲で入力してください。"
			s.render(w, http.StatusBadRequest, "time_form", page)
			return
		}
		dayTotal += minutes
		if minutes > 0 || inRange {
			cells = append(cells, store.DayCell{
				AssignmentID: column.AssignmentID, TaskID: column.TaskID, WorkDate: date, Minutes: minutes,
			})
		}
	}
	// 代行時は担当案件以外の欄がフォームに出ない。同じ日の他案件実績も 24 時間上限に含める。
	other, err := s.hiddenDayMinutes(r.Context(), page.PersonID, date, visible)
	if err != nil {
		s.fail(w, err)
		return
	}
	dayTotal += other
	if dayTotal > 24*60 {
		page.Date = date
		page.Warn = "1 日の入力は合計 24 時間までです。"
		s.render(w, http.StatusBadRequest, "time_form", page)
		return
	}
	if err := s.store.SaveDay(r.Context(), cells); err != nil {
		s.fail(w, err)
		return
	}
	s.redirect(w, r, timeURL(date, page.PersonID), "登録しました。")
}

// prepareTime は実績画面の表示データを作り、errMsg を警告として載せます。表示できないときは false です。
func (s *Server) prepareTime(w http.ResponseWriter, r *http.Request, errMsg string) (timePage, bool) {
	page, _, ok := s.timeContext(w, r)
	if !ok {
		return timePage{}, false
	}
	page.Warn = errMsg
	return page, true
}

// timeContext は入力対象人材の全アサイン・タスクと、指定日の実績を読みます。
func (s *Server) timeContext(w http.ResponseWriter, r *http.Request) (timePage, []timeColumn, bool) {
	me := personFrom(r)
	people, err := s.timePeople(w, r, me)
	if err != nil {
		return timePage{}, nil, false
	}
	canProxy := me.IsAdmin || s.hasManagedProjects(r, me.ID)
	if len(people) == 0 {
		page := timePage{
			base: s.page(w, r, "実績登録"), Date: time.Now().Format("2006-01-02"),
			ShowPerson: canProxy,
		}
		page.Warn = "入力できるアサインがありません。"
		s.render(w, http.StatusOK, "time_form", page)
		return timePage{}, nil, false
	}
	personID, ok := resolveTimePerson(me, people, r)
	if !ok {
		s.redirectWarn(w, r, "/time", "入力できるアサインがありません。")
		return timePage{}, nil, false
	}
	allAssignments, err := s.store.ListPersonOpenAssignments(r.Context(), personID)
	if err != nil {
		s.fail(w, err)
		return timePage{}, nil, false
	}
	assignments, err := s.filterTimeAssignments(r, me, personID, allAssignments)
	if err != nil {
		s.fail(w, err)
		return timePage{}, nil, false
	}
	if len(assignments) == 0 {
		page := timePage{
			base: s.page(w, r, "実績登録"), Date: time.Now().Format("2006-01-02"),
			PersonID: personID, People: people, ShowPerson: canProxy, CanPickPerson: canProxy && len(people) > 1,
		}
		page.Warn = "入力できるアサインがありません。"
		s.render(w, http.StatusOK, "time_form", page)
		return timePage{}, nil, false
	}
	projectIDs := uniqueProjectIDs(assignments)
	tasks, err := s.store.ListTasksForProjects(r.Context(), projectIDs)
	if err != nil {
		s.fail(w, err)
		return timePage{}, nil, false
	}
	date := r.URL.Query().Get("date")
	if r.Method == http.MethodPost {
		if d := r.FormValue("work_date"); d != "" {
			date = d
		}
	}
	if _, msg := normalizeDate(date, true); msg != "" {
		date = time.Now().Format("2006-01-02")
	}
	day, _ := time.Parse("2006-01-02", date)
	prevDate := day.AddDate(0, 0, -1).Format("2006-01-02")
	nextDate := day.AddDate(0, 0, 1).Format("2006-01-02")
	monthStart := date[:7] + "-01"
	monthEnd, _ := time.Parse("2006-01-02", monthStart)
	monthUntil := monthEnd.AddDate(0, 1, 0).Format("2006-01-02")

	// 画面の入力欄は絞り込み後だが、日次合計と日付リンクは対象者の全完了前アサイン分を使う。
	allIDs := make([]string, len(allAssignments))
	for i, a := range allAssignments {
		allIDs[i] = a.ID
	}
	entries, err := s.store.ListEntries(r.Context(), allIDs, monthStart, monthUntil)
	if err != nil {
		s.fail(w, err)
		return timePage{}, nil, false
	}
	legacy := map[string]bool{}
	for _, entry := range entries {
		if entry.TaskID == "" {
			legacy[entry.AssignmentID] = true
		}
	}
	groups, columns := buildTimeGroups(assignments, tasks, legacy, date)
	columnKeys := map[string]bool{}
	for _, column := range columns {
		columnKeys[column.Key] = true
	}
	cells := map[string]string{}
	savedSet := map[string]bool{}
	var dayMinutes, hiddenMinutes int64
	for _, entry := range entries {
		key := entryKey(entry.AssignmentID, entry.TaskID, entry.TaskID == "" && hasTasksForAssignment(assignments, tasks, entry.AssignmentID))
		if entry.WorkDate == date {
			dayMinutes += entry.Minutes
			if columnKeys[key] {
				cells[key] = calc.FormatHours(entry.Minutes)
			} else {
				hiddenMinutes += entry.Minutes
			}
		}
		savedSet[entry.WorkDate] = true
	}
	var savedDates []string
	for d := range savedSet {
		savedDates = append(savedDates, d)
	}
	sort.Strings(savedDates)

	page := timePage{
		base: s.page(w, r, "実績登録"), Date: date, PrevDate: prevDate, NextDate: nextDate,
		PersonID: personID, People: people,
		ShowPerson: canProxy || len(people) > 1, CanPickPerson: canProxy && len(people) > 1,
		Groups: groups, Columns: columns, Cells: cells,
		DayTotal: calc.FormatHours(dayMinutes), HiddenDayTotal: calc.FormatHours(hiddenMinutes),
		SavedDates: savedDates,
	}
	return page, columns, true
}

// timePeople は実績入力の人材候補です。
// 管理者は完了前案件のアサイン人材、責任者は担当案件のアサイン人材（と自分）、メンバーは本人のみです。
func (s *Server) timePeople(w http.ResponseWriter, r *http.Request, me *store.Person) ([]timePerson, error) {
	if me.IsAdmin {
		list, err := s.store.ListAssigneesOnOpenProjects(r.Context())
		if err != nil {
			s.fail(w, err)
			return nil, err
		}
		out := make([]timePerson, 0, len(list))
		for _, a := range list {
			out = append(out, timePerson{ID: a.ID, Code: a.Code, Name: a.Name})
		}
		return out, nil
	}

	byID := map[string]timePerson{}
	managed, err := s.store.ListAssigneesOnManagedProjects(r.Context(), me.ID)
	if err != nil {
		s.fail(w, err)
		return nil, err
	}
	for _, a := range managed {
		byID[a.ID] = timePerson{ID: a.ID, Code: a.Code, Name: a.Name}
	}
	assigns, err := s.store.ListPersonOpenAssignments(r.Context(), me.ID)
	if err != nil {
		s.fail(w, err)
		return nil, err
	}
	if len(assigns) > 0 {
		byID[me.ID] = timePerson{ID: me.ID, Code: me.Code, Name: me.Name}
	}
	if len(byID) == 0 {
		return nil, nil
	}
	out := make([]timePerson, 0, len(byID))
	for _, p := range byID {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// hasManagedProjects は personID がいずれかの案件の責任者なら真です。
func (s *Server) hasManagedProjects(r *http.Request, personID string) bool {
	ids, err := s.store.ListManagedProjectIDs(r.Context(), personID)
	return err == nil && len(ids) > 0
}

// filterTimeAssignments は、他人分を代行する責任者向けに、担当案件のアサインだけに絞ります。
// 管理者と本人分はそのまま返します。
func (s *Server) filterTimeAssignments(r *http.Request, me *store.Person, personID string, assignments []store.Assignment) ([]store.Assignment, error) {
	if me.IsAdmin || personID == me.ID {
		return assignments, nil
	}
	ids, err := s.store.ListManagedProjectIDs(r.Context(), me.ID)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, id := range ids {
		allowed[id] = true
	}
	var out []store.Assignment
	for _, a := range assignments {
		if allowed[a.ProjectID] {
			out = append(out, a)
		}
	}
	return out, nil
}

// hiddenDayMinutes は、visible に無い完了前アサインについて、date に既にある実績分の合計を返します。
// 責任者の代行保存で、フォームに出ていない他案件分を 24 時間上限へ加算するために使います。
func (s *Server) hiddenDayMinutes(ctx context.Context, personID, date string, visible map[string]bool) (int64, error) {
	assignments, err := s.store.ListPersonOpenAssignments(ctx, personID)
	if err != nil {
		return 0, err
	}
	var ids []string
	for _, a := range assignments {
		if visible[a.ID] {
			continue
		}
		ids = append(ids, a.ID)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, err
	}
	until := day.AddDate(0, 0, 1).Format("2006-01-02")
	entries, err := s.store.ListEntries(ctx, ids, date, until)
	if err != nil {
		return 0, err
	}
	var sum int64
	for _, entry := range entries {
		sum += entry.Minutes
	}
	return sum, nil
}

// buildTimeGroups はアサインとタスクから、案件→工程→入力欄の階層を作ります。
// 選んだ日がアサイン期間外の工程は出しません。
func buildTimeGroups(assignments []store.Assignment, tasks []store.Task, legacy map[string]bool, date string) ([]timeProjectGroup, []timeColumn) {
	byLine := map[string][]store.Task{}
	for _, task := range tasks {
		byLine[task.ProjectProcessID] = append(byLine[task.ProjectProcessID], task)
	}
	var groups []timeProjectGroup
	var flat []timeColumn
	var cur *timeProjectGroup
	for _, a := range assignments {
		if date < a.StartDate || date > a.EndDate {
			continue
		}
		if cur == nil || cur.ProjectID != a.ProjectID {
			groups = append(groups, timeProjectGroup{
				ProjectID: a.ProjectID, ProjectCode: a.ProjectCode, ProjectName: a.ProjectName,
			})
			cur = &groups[len(groups)-1]
		}
		lineTasks := byLine[a.ProjectProcessID]
		period := a.StartDate + " 〜 " + a.EndDate
		proc := timeProcessGroup{ProcessName: a.ProcessName, Period: period}
		if len(lineTasks) == 0 {
			col := timeColumn{
				Key: a.ID, AssignmentID: a.ID, Label: a.ProcessName,
				StartDate: a.StartDate, EndDate: a.EndDate, InRange: true,
			}
			proc.Columns = append(proc.Columns, col)
			flat = append(flat, col)
		} else {
			for _, task := range lineTasks {
				col := timeColumn{
					Key: a.ID + "_" + task.ID, AssignmentID: a.ID, TaskID: task.ID,
					Label: task.Name, StartDate: a.StartDate, EndDate: a.EndDate, InRange: true,
				}
				proc.Columns = append(proc.Columns, col)
				flat = append(flat, col)
			}
			if legacy[a.ID] {
				col := timeColumn{
					Key: a.ID + "_legacy", AssignmentID: a.ID, Label: "（未分類）",
					StartDate: a.StartDate, EndDate: a.EndDate, InRange: true,
				}
				proc.Columns = append(proc.Columns, col)
				flat = append(flat, col)
			}
		}
		cur.Processes = append(cur.Processes, proc)
	}
	return groups, flat
}

func uniqueProjectIDs(assignments []store.Assignment) []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range assignments {
		if seen[a.ProjectID] {
			continue
		}
		seen[a.ProjectID] = true
		out = append(out, a.ProjectID)
	}
	return out
}

// hasTasksForAssignment は assignmentID の工程にタスクがあるとき真です。
func hasTasksForAssignment(assignments []store.Assignment, tasks []store.Task, assignmentID string) bool {
	var lineID string
	for _, a := range assignments {
		if a.ID == assignmentID {
			lineID = a.ProjectProcessID
			break
		}
	}
	if lineID == "" {
		return false
	}
	for _, task := range tasks {
		if task.ProjectProcessID == lineID {
			return true
		}
	}
	return false
}

// entryKey は実績行をフォーム欄のキーにします。asLegacy が真なら未分類欄です。
func entryKey(assignmentID, taskID string, asLegacy bool) string {
	if taskID != "" {
		return assignmentID + "_" + taskID
	}
	if asLegacy {
		return assignmentID + "_legacy"
	}
	return assignmentID
}

// resolveTimePerson は実績入力の対象人材を決めます。
// 候補 people のうち、クエリ person またはフォーム person_id を使い、無ければ本人（候補にあれば）、それも無ければ先頭です。
// 候補に載らない人は選べません（管理者・責任者・メンバーの差は timePeople 側で付けます）。
func resolveTimePerson(me *store.Person, people []timePerson, r *http.Request) (string, bool) {
	if len(people) == 0 {
		return "", false
	}
	has := func(id string) bool {
		for _, p := range people {
			if p.ID == id {
				return true
			}
		}
		return false
	}
	requested := r.URL.Query().Get("person")
	if r.Method == http.MethodPost {
		if id := r.FormValue("person_id"); id != "" {
			requested = id
		}
	}
	if requested != "" && has(requested) {
		return requested, true
	}
	if has(me.ID) {
		return me.ID, true
	}
	return people[0].ID, true
}

// timeURL は横断実績画面のパスです。
func timeURL(date, personID string) string {
	path := "/time?date=" + url.QueryEscape(date)
	if personID != "" {
		path += "&person=" + url.QueryEscape(personID)
	}
	return path
}
