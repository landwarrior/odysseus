package web

import (
	"errors"
	"net/http"
	"sort"
	"time"

	"odysseus/internal/calc"
	"odysseus/internal/store"
)

// timeListPage は、実績を入れられる案件の一覧です。
type timeListPage struct {
	base
	Projects []store.Project
	AsAdmin  bool
}

// timePerson は実績入力で選ぶ人材1人です。
type timePerson struct {
	ID   string
	Code string
	Name string
}

// timePage は1案件の月次の実績入力です。Cells はフォームの日付に既にある時間です。
// PersonID は入力対象の人材、People は管理者が選べる人材です。
type timePage struct {
	base
	Project       store.Project
	Month         string
	Prev          string
	Next          string
	Date          string
	PersonID      string
	People        []timePerson
	ShowPerson    bool
	CanPickPerson bool
	Columns       []store.Assignment
	Cells         map[string]string
	Saved         []savedDay
}

// savedDay は、その月に保存済みの1日です。Hours は Columns と同じ順の時間表示です。
type savedDay struct {
	Date  string
	Hours []string
}

// handleTimeList は実績を入れられる完了前の案件を出します。
// 管理者はアサインがある全案件、メンバーは自分のアサイン案件だけです。
func (s *Server) handleTimeList(w http.ResponseWriter, r *http.Request) {
	me := personFrom(r)
	var projects []store.Project
	var err error
	if me.IsAdmin {
		projects, err = s.store.ListOpenProjectsWithAssignments(r.Context())
	} else {
		projects, err = s.store.ListAssignedProjects(r.Context(), me.ID)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, http.StatusOK, "time_list", timeListPage{
		base: s.page(w, r, "実績"), Projects: projects, AsAdmin: me.IsAdmin,
	})
}

// handleTimeEdit はパスの案件について、月を移動できる実績入力画面を出します。
func (s *Server) handleTimeEdit(w http.ResponseWriter, r *http.Request) {
	page, ok := s.prepareTime(w, r, "")
	if !ok {
		return
	}
	s.render(w, http.StatusOK, "time_form", page)
}

// handleTimeSave はフォームの日付と工程ごとの時間を、対象人材のアサインに保存します。
// 空欄と 0 は削除、24 時間を超える入力と期間外の入力は拒否します。
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
	for _, column := range columns {
		raw := r.FormValue("minutes_" + column.ID)
		var minutes int64
		if raw != "" {
			parsed, errText := parseOptionalHours(raw)
			if errText != "" {
				page.Date = date
				page.Warn = column.ProcessName + "の" + errText
				s.render(w, http.StatusBadRequest, "time_form", page)
				return
			}
			minutes = parsed
		}
		if minutes > 24*60 {
			page.Date = date
			page.Warn = "1 日の入力は 24 時間までです。"
			s.render(w, http.StatusBadRequest, "time_form", page)
			return
		}
		inRange := date >= column.StartDate && date <= column.EndDate
		if minutes > 0 && !inRange {
			page.Date = date
			page.Warn = column.ProcessName + "は " + column.StartDate + " から " + column.EndDate + " の範囲で入力してください。"
			s.render(w, http.StatusBadRequest, "time_form", page)
			return
		}
		if minutes > 0 || inRange {
			cells = append(cells, store.DayCell{AssignmentID: column.ID, WorkDate: date, Minutes: minutes})
		}
	}
	if err := s.store.SaveDay(r.Context(), cells); err != nil {
		s.fail(w, err)
		return
	}
	s.redirect(w, r, timeURL(page.Project.ID, date[:7], date, page.PersonID), "登録しました。")
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

// timeContext はパスの案件と、入力対象人材のアサイン、指定月の実績を読みます。
// 管理者は person / person_id で人材を選べます。完了した案件やアサインが無い案件では false です。
func (s *Server) timeContext(w http.ResponseWriter, r *http.Request) (timePage, []store.Assignment, bool) {
	project, err := s.store.GetProject(r.Context(), r.PathValue("projectID"))
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "見つかりません。", http.StatusNotFound)
		return timePage{}, nil, false
	}
	if err != nil {
		s.fail(w, err)
		return timePage{}, nil, false
	}
	if project.Status == "closed" {
		s.redirectWarn(w, r, "/time", "完了したプロジェクトには実績を入力できません。")
		return timePage{}, nil, false
	}
	all, err := s.store.ListAssignmentsByProject(r.Context(), project.ID)
	if err != nil {
		s.fail(w, err)
		return timePage{}, nil, false
	}
	people := assigneesOf(all)
	if len(people) == 0 {
		s.redirectWarn(w, r, "/time", "このプロジェクトへのアサインがありません。")
		return timePage{}, nil, false
	}
	me := personFrom(r)
	personID, ok := resolveTimePerson(me, people, r)
	if !ok {
		s.redirectWarn(w, r, "/time", "このプロジェクトへのアサインがありません。")
		return timePage{}, nil, false
	}
	columns, err := s.store.ListPersonAssignments(r.Context(), project.ID, personID)
	if err != nil {
		s.fail(w, err)
		return timePage{}, nil, false
	}
	if len(columns) == 0 {
		s.redirectWarn(w, r, "/time", "このプロジェクトへのアサインがありません。")
		return timePage{}, nil, false
	}
	month := time.Now().Format("2006-01")
	if q, ok := normalizeMonth(r.URL.Query().Get("month")); ok {
		month = q
	}
	start, _ := time.Parse("2006-01", month)
	next := start.AddDate(0, 1, 0)
	ids := make([]string, len(columns))
	for i, column := range columns {
		ids[i] = column.ID
	}
	entries, err := s.store.ListEntries(r.Context(), ids, start.Format("2006-01-02"), next.Format("2006-01-02"))
	if err != nil {
		s.fail(w, err)
		return timePage{}, nil, false
	}
	date := r.URL.Query().Get("date")
	if _, msg := normalizeDate(date, true); msg != "" {
		today := time.Now().Format("2006-01-02")
		if today[:7] == month {
			date = today
		} else {
			date = month + "-01"
		}
	}
	cells := map[string]string{}
	byDate := map[string]map[string]int64{}
	for _, entry := range entries {
		if entry.WorkDate == date {
			cells[entry.AssignmentID] = calc.FormatHours(entry.Minutes)
		}
		if byDate[entry.WorkDate] == nil {
			byDate[entry.WorkDate] = map[string]int64{}
		}
		byDate[entry.WorkDate][entry.AssignmentID] = entry.Minutes
	}
	var dates []string
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	var saved []savedDay
	for _, d := range dates {
		row := savedDay{Date: d, Hours: make([]string, len(columns))}
		for i, column := range columns {
			if minutes, ok := byDate[d][column.ID]; ok {
				row.Hours[i] = calc.FormatHours(minutes)
			}
		}
		saved = append(saved, row)
	}
	page := timePage{
		base: s.page(w, r, "実績登録"), Project: project, Month: month,
		Prev: start.AddDate(0, -1, 0).Format("2006-01"), Next: next.Format("2006-01"),
		Date: date, PersonID: personID, People: people,
		ShowPerson: me.IsAdmin, CanPickPerson: me.IsAdmin && len(people) > 1,
		Columns: columns, Cells: cells, Saved: saved,
	}
	return page, columns, true
}

// assigneesOf は assignments に出てくる人材を、コード順の重複なしで返します。
func assigneesOf(assignments []store.Assignment) []timePerson {
	seen := map[string]bool{}
	var out []timePerson
	for _, a := range assignments {
		if seen[a.PersonID] {
			continue
		}
		seen[a.PersonID] = true
		out = append(out, timePerson{ID: a.PersonID, Code: a.PersonCode, Name: a.PersonName})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// resolveTimePerson は実績入力の対象人材を決めます。
// メンバーは本人だけ。管理者はクエリ person またはフォーム person_id を使い、無ければ本人（アサインがあれば）、それも無ければ先頭の人材です。
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
	if !me.IsAdmin {
		if !has(me.ID) {
			return "", false
		}
		return me.ID, true
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

// timeURL は実績入力画面のパスです。personID が空でなければ人材クエリを付けます。
func timeURL(projectID, month, date, personID string) string {
	path := "/time/" + projectID + "?month=" + month
	if date != "" {
		path += "&date=" + date
	}
	if personID != "" {
		path += "&person=" + personID
	}
	return path
}
