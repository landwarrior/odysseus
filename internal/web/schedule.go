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

// scheduleCell は旬1マスです。InRange は工程の開始〜終了に掛かるとき true です。
// Headcount と CapClass は工程の枠行用です。CapClass は cap-warn（枠の 80% 以上）か cap-over（枠以上）です。
type scheduleCell struct {
	YM        string
	Decade    int
	Minutes   int64
	Days      string
	InRange   bool
	Current   bool
	Headcount int
	CapClass  string
}

// scheduleRow はスケジュール表の1行です。一覧は工程、詳細はタスクです。
type scheduleRow struct {
	RowID     string
	ProcessID string
	Process   string
	Name      string
	Remarks   string
	Color     string
	Estimate  string
	Allocated string
	Diff      string
	DiffClass string
	Editable  bool
	Cells     []scheduleCell
}

// scheduleProcessGroup は案件スケジュールの工程とそのタスク行です。
// Capacity はアサイン人数×営業日の枠（目安）です。
type scheduleProcessGroup struct {
	ProcessID string
	Process   string
	Color     string
	RowSpan   int
	Tasks     []scheduleRow
	Capacity  scheduleRow
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
	Rows      []scheduleRow
	Groups    []scheduleProcessGroup
	RowCount  int
	Allocated []scheduleCapCell
}

// schedulePeriodCol はフッターの旬列です。
// Capacity は、その旬にアサイン期間が重なるユニーク人数 × 営業日（人日）です。
// CapClass は割当合計に対する黄（80% 以上）／赤（キャパ以上）です。
type schedulePeriodCol struct {
	YM           string
	Decade       int
	Current      bool
	BusinessDays int
	Headcount    int
	Capacity     string
	Allocated    string
	CapClass     string
}

// schedulePage はプロジェクト一覧または案件のスケジュールです。
type schedulePage struct {
	base
	Detail      bool
	Project     store.Project
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
	RangeAction string
}

// handleProjects は案件を工程行のスケジュール表で出します。
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
	page, err := s.buildSchedulePage(w, r, scheduleBuildInput{
		Title:       "プロジェクト",
		Projects:    projects,
		From:        from,
		To:          to,
		Months:      months,
		Status:      status,
		Detail:      false,
		RangeAction: "/projects",
	})
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, http.StatusOK, "projects", page)
}

// handleProjectSchedule は1案件のタスク行スケジュールを出します。
func (s *Server) handleProjectSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, err := s.store.GetProject(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	from, to, months, err := s.scheduleRange(r)
	if err != nil {
		s.redirectWarn(w, r, "/projects/"+id+"/schedule", err.Error())
		return
	}
	page, err := s.buildSchedulePage(w, r, scheduleBuildInput{
		Title:       "スケジュール",
		Projects:    []store.Project{project},
		Project:     project,
		From:        from,
		To:          to,
		Months:      months,
		Detail:      true,
		RangeAction: "/projects/" + id + "/schedule",
	})
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, http.StatusOK, "projects", page)
}

type scheduleBuildInput struct {
	Title       string
	Projects    []store.Project
	Project     store.Project
	From        string
	To          string
	Months      []string
	Status      string
	Detail      bool
	RangeAction string
}

// buildSchedulePage は一覧（工程）または詳細（タスク）の表データを組み立てます。
func (s *Server) buildSchedulePage(w http.ResponseWriter, r *http.Request, in scheduleBuildInput) (schedulePage, error) {
	ids := make([]string, len(in.Projects))
	for i, p := range in.Projects {
		ids[i] = p.ID
	}
	tasks, err := s.store.ListTasksForProjects(r.Context(), ids)
	if err != nil {
		return schedulePage{}, err
	}
	lines, err := s.store.ListProcessLinesForProjects(r.Context(), ids)
	if err != nil {
		return schedulePage{}, err
	}
	taskIDs := make([]string, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID
	}
	allocs, err := s.store.ListAllocationsByTasks(r.Context(), taskIDs)
	if err != nil {
		return schedulePage{}, err
	}
	totals, err := s.store.AllocationTotalsByTask(r.Context(), taskIDs)
	if err != nil {
		return schedulePage{}, err
	}
	holidays, err := s.store.ListHolidayDates(r.Context())
	if err != nil {
		return schedulePage{}, err
	}
	currentYM, currentDec := schedule.CurrentPeriod(time.Time{})
	std := s.cfg.StandardMinutes

	allocMap := map[string]int64{}
	for _, a := range allocs {
		allocMap[a.TaskID+"|"+a.YearMonth+"|"+strconv.Itoa(a.Decade)] = a.PlannedMinutes
	}
	bizDays := map[string]int{}
	for _, ym := range in.Months {
		for decade := 1; decade <= 3; decade++ {
			days, _ := schedule.BusinessDays(ym, decade, holidays)
			bizDays[ym+"|"+strconv.Itoa(decade)] = days
		}
	}

	tasksByProject := map[string][]store.Task{}
	for _, t := range tasks {
		tasksByProject[t.ProjectID] = append(tasksByProject[t.ProjectID], t)
	}
	linesByProject := map[string][]store.ProcessLine{}
	for _, line := range lines {
		linesByProject[line.ProjectID] = append(linesByProject[line.ProjectID], line)
	}
	lineIDs := make([]string, 0, len(lines))
	for _, line := range lines {
		lineIDs = append(lineIDs, line.ID)
	}
	assigns, err := s.store.AssignmentsForLines(r.Context(), lineIDs)
	if err != nil {
		return schedulePage{}, err
	}
	assignmentsByLine := map[string][]store.Assignment{}
	for _, a := range assigns {
		assignmentsByLine[a.ProjectProcessID] = append(assignmentsByLine[a.ProjectProcessID], a)
	}

	periodAllocated := map[string]int64{}
	blocks := make([]scheduleProjectBlock, 0, len(in.Projects))
	for _, project := range in.Projects {
		projectTasks := tasksByProject[project.ID]
		projectLines := linesByProject[project.ID]
		projectPeriod := map[string]int64{}
		block := scheduleProjectBlock{Project: project}

		if in.Detail {
			block.Groups, block.RowCount = s.scheduleTaskGroups(projectLines, projectTasks, assignmentsByLine, in.Months, currentYM, currentDec, allocMap, totals, std, bizDays, projectPeriod, periodAllocated)
		} else {
			block.Rows, block.RowCount = s.scheduleProcessRows(projectLines, projectTasks, in.Months, currentYM, currentDec, allocMap, totals, std, projectPeriod, periodAllocated)
		}

		for _, ym := range in.Months {
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

	periods := make([]schedulePeriodCol, 0, len(in.Months)*3)
	for _, ym := range in.Months {
		for decade := 1; decade <= 3; decade++ {
			periodKey := ym + "|" + strconv.Itoa(decade)
			heads := uniqueAssigneesInDecade(assigns, ym, decade)
			days := bizDays[periodKey]
			capMinutes := int64(heads) * int64(days) * std
			allocated := periodAllocated[periodKey]
			periods = append(periods, schedulePeriodCol{
				YM: ym, Decade: decade,
				Current:      ym == currentYM && decade == currentDec,
				BusinessDays: days,
				Headcount:    heads,
				Capacity:     calc.FormatPersonDays(capMinutes, std),
				Allocated:    calc.FormatPersonDays(allocated, std),
				CapClass:     capacityLoadClass(allocated, capMinutes),
			})
		}
	}

	stickyCols := 5
	if in.Detail {
		stickyCols = 6
	}
	return schedulePage{
		base:        s.page(w, r, in.Title),
		Detail:      in.Detail,
		Project:     in.Project,
		Status:      in.Status,
		From:        in.From,
		To:          in.To,
		Months:      in.Months,
		CurrentYM:   currentYM,
		CurrentDec:  currentDec,
		Decades:     []int{1, 2, 3},
		DecadeLabel: schedule.DecadeLabels,
		Blocks:      blocks,
		Periods:     periods,
		ColSpan:     stickyCols + len(in.Months)*3,
		RangeAction: in.RangeAction,
	}, nil
}

// scheduleProcessRows は工程ごとの集約行を作ります（一覧用・編集不可）。
func (s *Server) scheduleProcessRows(
	lines []store.ProcessLine,
	tasks []store.Task,
	months []string,
	currentYM string,
	currentDec int,
	allocMap map[string]int64,
	totals map[string]int64,
	std int64,
	projectPeriod map[string]int64,
	periodAllocated map[string]int64,
) ([]scheduleRow, int) {
	if len(lines) == 0 {
		return nil, 2 // 未登録案内 + 割当合計
	}
	tasksByLine := map[string][]store.Task{}
	for _, t := range tasks {
		tasksByLine[t.ProjectProcessID] = append(tasksByLine[t.ProjectProcessID], t)
	}
	rows := make([]scheduleRow, 0, len(lines))
	for _, line := range lines {
		lineTasks := tasksByLine[line.ID]
		var est, allocated int64
		color := schedule.NormalizeColor(line.ProcessColor)
		if len(lineTasks) == 0 {
			est = line.PlannedMinutes
		} else {
			for _, t := range lineTasks {
				est += t.PlannedMinutes
				allocated += totals[t.ID]
				color = schedule.NormalizeColor(t.ProcessColor)
			}
		}
		diff := est - allocated
		diffClass := ""
		if diff > 0 {
			diffClass = "warn"
		} else if diff < 0 {
			diffClass = "diff-over"
		}
		row := scheduleRow{
			RowID:     line.ID,
			Process:   line.ProcessName,
			Color:     color,
			Estimate:  calc.FormatPersonDays(est, std),
			Allocated: calc.FormatPersonDays(allocated, std),
			Diff:      calc.FormatPersonDays(diff, std),
			DiffClass: diffClass,
			Editable:  false,
		}
		for _, ym := range months {
			for decade := 1; decade <= 3; decade++ {
				var minutes int64
				for _, t := range lineTasks {
					minutes += allocMap[t.ID+"|"+ym+"|"+strconv.Itoa(decade)]
				}
				periodKey := ym + "|" + strconv.Itoa(decade)
				periodAllocated[periodKey] += minutes
				projectPeriod[periodKey] += minutes
				cell := scheduleCell{
					YM: ym, Decade: decade, Minutes: minutes,
					InRange: schedule.OverlapsDecade(ym, decade, line.StartDate, line.EndDate),
					Current: ym == currentYM && decade == currentDec,
				}
				if minutes > 0 {
					cell.Days = calc.FormatPersonDays(minutes, std)
				}
				row.Cells = append(row.Cells, cell)
			}
		}
		rows = append(rows, row)
	}
	return rows, len(rows) + 1
}

// scheduleTaskGroups は工程ごとのタスク行と枠行を作ります（案件スケジュール用）。
// タスクのない工程（開始・終了だけなど）は編集不可の1行として出します。
// 枠は、その旬に期間が重なるアサイン人数 × 営業日（人日）です。
func (s *Server) scheduleTaskGroups(
	lines []store.ProcessLine,
	tasks []store.Task,
	assignmentsByLine map[string][]store.Assignment,
	months []string,
	currentYM string,
	currentDec int,
	allocMap map[string]int64,
	totals map[string]int64,
	std int64,
	bizDays map[string]int,
	projectPeriod map[string]int64,
	periodAllocated map[string]int64,
) ([]scheduleProcessGroup, int) {
	if len(lines) == 0 {
		return nil, 2
	}
	tasksByProcess := map[string][]store.Task{}
	for _, task := range tasks {
		tasksByProcess[task.ProcessID] = append(tasksByProcess[task.ProcessID], task)
	}
	groups := make([]scheduleProcessGroup, 0, len(lines))
	rowCount := 1 // プロジェクト割当合計
	for _, line := range lines {
		color := schedule.NormalizeColor(line.ProcessColor)
		lineTasks := tasksByProcess[line.ProcessID]
		lineAssigns := assignmentsByLine[line.ID]
		taskRows := make([]scheduleRow, 0, len(lineTasks))
		periodMinutes := map[string]int64{}
		for _, task := range lineTasks {
			est := task.PlannedMinutes
			allocated := totals[task.ID]
			diff := est - allocated
			diffClass := ""
			if diff > 0 {
				diffClass = "warn"
			} else if diff < 0 {
				diffClass = "diff-over"
			}
			row := scheduleRow{
				RowID:     task.ID,
				ProcessID: line.ProcessID,
				Process:   line.ProcessName,
				Name:      task.Name,
				Remarks:   task.Remarks,
				Color:     color,
				Estimate:  calc.FormatPersonDays(est, std),
				Allocated: calc.FormatPersonDays(allocated, std),
				Diff:      calc.FormatPersonDays(diff, std),
				DiffClass: diffClass,
				Editable:  true,
			}
			for _, ym := range months {
				for decade := 1; decade <= 3; decade++ {
					allocKey := task.ID + "|" + ym + "|" + strconv.Itoa(decade)
					minutes := allocMap[allocKey]
					periodKey := ym + "|" + strconv.Itoa(decade)
					periodAllocated[periodKey] += minutes
					projectPeriod[periodKey] += minutes
					periodMinutes[periodKey] += minutes
					cell := scheduleCell{
						YM: ym, Decade: decade, Minutes: minutes,
						InRange: schedule.OverlapsDecade(ym, decade, line.StartDate, line.EndDate),
						Current: ym == currentYM && decade == currentDec,
					}
					if minutes > 0 {
						cell.Days = calc.FormatPersonDays(minutes, std)
					}
					row.Cells = append(row.Cells, cell)
				}
			}
			taskRows = append(taskRows, row)
		}
		if len(taskRows) == 0 {
			est := line.PlannedMinutes
			row := scheduleRow{
				RowID:     line.ID,
				ProcessID: line.ProcessID,
				Process:   line.ProcessName,
				Name:      "—",
				Color:     color,
				Estimate:  calc.FormatPersonDays(est, std),
				Allocated: calc.FormatPersonDays(0, std),
				Diff:      calc.FormatPersonDays(est, std),
				Editable:  false,
			}
			if est > 0 {
				row.DiffClass = "warn"
			}
			for _, ym := range months {
				for decade := 1; decade <= 3; decade++ {
					row.Cells = append(row.Cells, scheduleCell{
						YM: ym, Decade: decade,
						InRange: schedule.OverlapsDecade(ym, decade, line.StartDate, line.EndDate),
						Current: ym == currentYM && decade == currentDec,
					})
				}
			}
			taskRows = append(taskRows, row)
		}
		capacity := scheduleRow{
			RowID:     line.ProcessID,
			ProcessID: line.ProcessID,
			Process:   line.ProcessName,
			Name:      "枠",
			Color:     color,
		}
		for _, ym := range months {
			for decade := 1; decade <= 3; decade++ {
				periodKey := ym + "|" + strconv.Itoa(decade)
				heads := 0
				for _, a := range lineAssigns {
					if schedule.OverlapsDecade(ym, decade, a.StartDate, a.EndDate) {
						heads++
					}
				}
				capMinutes := int64(heads) * int64(bizDays[periodKey]) * std
				cell := scheduleCell{
					YM: ym, Decade: decade, Minutes: capMinutes,
					Headcount: heads,
					Current:   ym == currentYM && decade == currentDec,
					CapClass:  capacityLoadClass(periodMinutes[periodKey], capMinutes),
				}
				if capMinutes > 0 {
					cell.Days = calc.FormatPersonDays(capMinutes, std)
				}
				capacity.Cells = append(capacity.Cells, cell)
			}
		}
		span := len(taskRows) + 1
		groups = append(groups, scheduleProcessGroup{
			ProcessID: line.ProcessID,
			Process:   line.ProcessName,
			Color:     color,
			RowSpan:   span,
			Tasks:     taskRows,
			Capacity:  capacity,
		})
		rowCount += span
	}
	return groups, rowCount
}

// uniqueAssigneesInDecade は、ym の decade 旬にアサイン期間が重なる人材のユニーク人数です。
// 同じ人が複数工程・複数案件にいても 1 人と数えます。
func uniqueAssigneesInDecade(assigns []store.Assignment, ym string, decade int) int {
	seen := map[string]bool{}
	for _, a := range assigns {
		if a.PersonID == "" {
			continue
		}
		if schedule.OverlapsDecade(ym, decade, a.StartDate, a.EndDate) {
			seen[a.PersonID] = true
		}
	}
	return len(seen)
}

// capacityLoadClass は割当分に対する枠の負荷クラスです。
// 枠の 80% 以上なら cap-warn、枠以上（枠 0 で割当ありを含む）なら cap-over です。
func capacityLoadClass(allocated, capacity int64) string {
	if capacity <= 0 {
		if allocated > 0 {
			return "cap-over"
		}
		return ""
	}
	if allocated >= capacity {
		return "cap-over"
	}
	if allocated*10 >= capacity*8 {
		return "cap-warn"
	}
	return ""
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
