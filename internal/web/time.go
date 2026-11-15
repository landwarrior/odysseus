package web

import (
	"errors"
	"net/http"
	"sort"
	"time"

	"odysseus/internal/calc"
	"odysseus/internal/store"
)

// timeListPage は、自分が実績を入れられる案件の一覧です。
type timeListPage struct {
	base
	Projects []store.Project
}

// timePage は1案件の月次の実績入力です。Cells はフォームの日付に既にある時間です。
type timePage struct {
	base
	Project store.Project
	Month   string
	Prev    string
	Next    string
	Date    string
	Columns []store.Assignment
	Cells   map[string]string
	Saved   []savedDay
}

// savedDay は、その月に保存済みの1日です。Hours は Columns と同じ順の時間表示です。
type savedDay struct {
	Date  string
	Hours []string
}

// handleTimeList はログイン中の利用者に割り当てられ、完了前の案件を出します。
func (s *Server) handleTimeList(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListAssignedProjects(r.Context(), personFrom(r).ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, http.StatusOK, "time_list", timeListPage{base: s.page(w, r, "実績"), Projects: projects})
}

// handleTimeEdit はパスの案件について、月を移動できる実績入力画面を出します。
func (s *Server) handleTimeEdit(w http.ResponseWriter, r *http.Request) {
	page, ok := s.prepareTime(w, r, "")
	if !ok {
		return
	}
	s.render(w, http.StatusOK, "time_form", page)
}

// handleTimeSave はフォームの日付と工程ごとの時間を、本人のアサインに保存します。
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
	s.redirect(w, r, "/time/"+page.Project.ID+"?month="+date[:7]+"&date="+date, "登録しました。")
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

// timeContext はパスの案件と、ログイン中の利用者のアサイン、指定月の実績を読みます。
// 完了した案件やアサインが無い案件では false です。
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
	columns, err := s.store.ListPersonAssignments(r.Context(), project.ID, personFrom(r).ID)
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
		Date: date, Columns: columns, Cells: cells, Saved: saved,
	}
	return page, columns, true
}
