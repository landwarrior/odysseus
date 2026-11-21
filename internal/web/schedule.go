package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"odysseus/internal/calc"
	"odysseus/internal/schedule"
	"odysseus/internal/store"
)

// scheduleCell は旬1マスです。
type scheduleCell struct {
	YM      string
	Decade  int
	Minutes int64
	Days    string
	Current bool
}

// scheduleTaskRow はスケジュール表の1行（タスク）です。
type scheduleTaskRow struct {
	TaskID    string
	Name      string
	Process   string
	Color     string
	Estimate  string
	Allocated string
	Diff      string
	DiffClass string
	Cells     []scheduleCell
}

// scheduleCapCell はプロジェクト単位のキャパまたは割当合計の1マスです。
type scheduleCapCell struct {
	Days     string
	Current  bool
	CapClass string
}

// scheduleProjectBlock は案件ごとの行の塊です。
type scheduleProjectBlock struct {
	Project   store.Project
	Tasks     []scheduleTaskRow
	RowCount  int
	Allocated []scheduleCapCell
}

// schedulePeriodCol はフッターの旬列です。
type schedulePeriodCol struct {
	YM           string
	Decade       int
	Current      bool
	BusinessDays int
	Allocated    string
}

// schedulePage はプロジェクト一覧（スケジュール）です。
type schedulePage struct {
	base
	Status      string
	From        string
	To          string
	Months      []string
	CurrentYM   string
	CurrentDec  int
	Decades     []int
	DecadeLabel map[int]string
	Blocks      []scheduleProjectBlock
	Periods     []schedulePeriodCol
	ColSpan     int
}

// handleProjects は案件をスケジュール表で出します。
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && !validStatus(status) {
		status = ""
	}
	from, to, months, err := s.scheduleRange(r)
	if err != nil {
		s.redirectWarn(w, r, "/projects", err.Error())
		return
	}
	projects, err := s.store.ListProjects(r.Context(), status)
	if err != nil {
		s.fail(w, err)
		return
	}
	ids := make([]string, len(projects))
	for i, p := range projects {
		ids[i] = p.ID
	}
	tasks, err := s.store.ListTasksForProjects(r.Context(), ids)
	if err != nil {
		s.fail(w, err)
		return
	}
	taskIDs := make([]string, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID
	}
	allocs, err := s.store.ListAllocationsByTasks(r.Context(), taskIDs)
	if err != nil {
		s.fail(w, err)
		return
	}
	totals, err := s.store.AllocationTotalsByTask(r.Context(), taskIDs)
	if err != nil {
		s.fail(w, err)
		return
	}
	holidays, err := s.store.ListHolidayDates(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	currentYM, currentDec := schedule.CurrentPeriod(time.Time{})
	std := s.cfg.StandardMinutes

	allocMap := map[string]int64{}
	for _, a := range allocs {
		allocMap[a.TaskID+"|"+a.YearMonth+"|"+strconv.Itoa(a.Decade)] = a.PlannedMinutes
	}
	bizDays := map[string]int{}
	for _, ym := range months {
		for decade := 1; decade <= 3; decade++ {
			days, _ := schedule.BusinessDays(ym, decade, holidays)
			bizDays[ym+"|"+strconv.Itoa(decade)] = days
		}
	}

	periodAllocated := map[string]int64{}
	tasksByProject := map[string][]store.Task{}
	for _, t := range tasks {
		tasksByProject[t.ProjectID] = append(tasksByProject[t.ProjectID], t)
	}

	blocks := make([]scheduleProjectBlock, 0, len(projects))
	for _, project := range projects {
		projectTasks := tasksByProject[project.ID]
		projectPeriod := map[string]int64{}
		block := scheduleProjectBlock{
			Project:  project,
			RowCount: len(projectTasks) + 1, // タスク行 + 割当合計
		}
		if len(projectTasks) == 0 {
			block.RowCount = 2 // 未登録案内 + 割当合計
		}
		for _, task := range projectTasks {
			est := task.PlannedMinutes
			allocated := totals[task.ID]
			diff := est - allocated
			diffClass := ""
			if diff > 0 {
				diffClass = "warn"
			} else if diff < 0 {
				diffClass = "diff-over"
			}
			row := scheduleTaskRow{
				TaskID:    task.ID,
				Name:      task.Name,
				Process:   task.ProcessName,
				Color:     schedule.NormalizeColor(task.ProcessColor),
				Estimate:  calc.FormatPersonDays(est, std),
				Allocated: calc.FormatPersonDays(allocated, std),
				Diff:      calc.FormatPersonDays(diff, std),
				DiffClass: diffClass,
			}
			for _, ym := range months {
				for decade := 1; decade <= 3; decade++ {
					key := task.ID + "|" + ym + "|" + strconv.Itoa(decade)
					minutes := allocMap[key]
					periodKey := ym + "|" + strconv.Itoa(decade)
					periodAllocated[periodKey] += minutes
					projectPeriod[periodKey] += minutes
					cell := scheduleCell{
						YM: ym, Decade: decade, Minutes: minutes,
						Current: ym == currentYM && decade == currentDec,
					}
					if minutes > 0 {
						cell.Days = calc.FormatPersonDays(minutes, std)
					}
					row.Cells = append(row.Cells, cell)
				}
			}
			block.Tasks = append(block.Tasks, row)
		}
		for _, ym := range months {
			for decade := 1; decade <= 3; decade++ {
				periodKey := ym + "|" + strconv.Itoa(decade)
				current := ym == currentYM && decade == currentDec
				block.Allocated = append(block.Allocated, scheduleCapCell{
					Days: calc.FormatPersonDays(projectPeriod[periodKey], std), Current: current,
				})
			}
		}
		blocks = append(blocks, block)
	}

	periods := make([]schedulePeriodCol, 0, len(months)*3)
	for _, ym := range months {
		for decade := 1; decade <= 3; decade++ {
			periodKey := ym + "|" + strconv.Itoa(decade)
			periods = append(periods, schedulePeriodCol{
				YM: ym, Decade: decade,
				Current:      ym == currentYM && decade == currentDec,
				BusinessDays: bizDays[periodKey],
				Allocated:    calc.FormatPersonDays(periodAllocated[periodKey], std),
			})
		}
	}

	page := schedulePage{
		base:        s.page(w, r, "プロジェクト"),
		Status:      status,
		From:        from,
		To:          to,
		Months:      months,
		CurrentYM:   currentYM,
		CurrentDec:  currentDec,
		Decades:     []int{1, 2, 3},
		DecadeLabel: schedule.DecadeLabels,
		Blocks:      blocks,
		Periods:     periods,
		ColSpan:     5 + len(months)*3,
	}
	s.render(w, http.StatusOK, "projects", page)
}

// handleAllocationSave は旬セルの割当を保存し、更新後の見積・割当・差分を JSON で返します。
func (s *Server) handleAllocationSave(w http.ResponseWriter, r *http.Request) {
	taskID := formText(r, "task_id")
	ym, err := schedule.NormalizeYM(r.FormValue("year_month"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	decade, err := strconv.Atoi(formText(r, "decade"))
	if err != nil || decade < 1 || decade > 3 {
		http.Error(w, "旬が不正です。", http.StatusBadRequest)
		return
	}
	raw := formText(r, "effort")
	var minutes int64
	if raw != "" {
		minutes, err = calc.ParsePersonDays(raw, s.cfg.StandardMinutes)
		if err != nil {
			http.Error(w, "工数は 0.1 人日単位で入力してください。", http.StatusBadRequest)
			return
		}
	}
	task, err := s.store.GetTask(r.Context(), taskID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "タスクが見つかりません。", http.StatusNotFound)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.store.UpsertTaskAllocation(r.Context(), task.ID, ym, decade, minutes); err != nil {
		s.fail(w, err)
		return
	}
	totals, err := s.store.AllocationTotalsByTask(r.Context(), []string{task.ID})
	if err != nil {
		s.fail(w, err)
		return
	}
	allocated := totals[task.ID]
	diff := task.PlannedMinutes - allocated
	std := s.cfg.StandardMinutes
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"task_id":   task.ID,
		"effort":    calc.FormatPersonDays(minutes, std),
		"estimate":  calc.FormatPersonDays(task.PlannedMinutes, std),
		"allocated": calc.FormatPersonDays(allocated, std),
		"diff":      calc.FormatPersonDays(diff, std),
		"diff_sign": diff,
	})
}

// scheduleRange は表示期間の from/to と月一覧を返します。
func (s *Server) scheduleRange(r *http.Request) (from, to string, months []string, err error) {
	currentYM, _ := schedule.CurrentPeriod(time.Time{})
	from = r.URL.Query().Get("from")
	to = r.URL.Query().Get("to")
	if from == "" && to == "" {
		from, err = schedule.AddMonths(currentYM, -1)
		if err != nil {
			return "", "", nil, err
		}
		to, err = schedule.AddMonths(currentYM, 5)
		if err != nil {
			return "", "", nil, err
		}
	} else {
		from, err = schedule.NormalizeYM(from)
		if err != nil {
			return "", "", nil, err
		}
		to, err = schedule.NormalizeYM(to)
		if err != nil {
			return "", "", nil, err
		}
	}
	months, err = schedule.IterMonths(from, to)
	return from, to, months, err
}
