package web

import (
	"context"
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
// CapClass は一覧では工程キャパ超過（黄／赤）、案件スケジュールの残工数行ではマイナスのとき cap-over です。
// CapDays は一覧の工程キャパ（人日）。ActualDays は time_entry を旬に集めた実績（人日表示）です。
// CapHeads は工程キャパの人数（title 用）です。
type scheduleCell struct {
	YM         string
	Decade     int
	Minutes    int64
	Days       string
	ActualDays string
	HasActual  bool
	InRange    bool
	Current    bool
	CapClass   string
	CapDays    string
	CapHeads   int
}

// scheduleRow はスケジュール表の1行です。一覧は工程、詳細はタスクです。
// Actual は行の実績合計（人日）。ActualOver は実績が見積を超えたとき true です。
type scheduleRow struct {
	RowID       string
	ProcessID   string
	Process     string
	Name        string
	Remarks     string
	Color       string
	Estimate    string
	Allocated   string
	Actual      string
	Diff        string
	DiffClass   string
	ActualDiff  string
	ActualClass string
	ActualOver  bool
	Editable    bool
	Cells       []scheduleCell
}

// scheduleProcessGroup は案件スケジュールの工程とそのタスク行です。
// Remain は工程の旬割当からタスク合計を引いた残工数です。
type scheduleProcessGroup struct {
	ProcessID string
	Process   string
	Color     string
	RowSpan   int
	Tasks     []scheduleRow
	Remain    scheduleRow
}

// scheduleProjectBlock は案件ごとの行の塊です。
type scheduleProjectBlock struct {
	Project  store.Project
	Rows     []scheduleRow
	Groups   []scheduleProcessGroup
	RowCount int
}

// schedulePeriodCol はフッターの旬列です。
// Capacity は、その旬にアサイン期間が重なるユニーク人数 × 営業日（人日）です。
// 0 円単価のアサインは人数に含めません。CapClass は割当合計に対する黄（80% 以上）／赤（キャパ以上）です。
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
// CanEdit が偽のときは閲覧のみで、工数セルは入力欄にしません。
type schedulePage struct {
	base
	Detail      bool
	CanEdit     bool
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
	taskAllocs, err := s.store.ListAllocationsByTasks(r.Context(), taskIDs)
	if err != nil {
		return schedulePage{}, err
	}
	taskTotals, err := s.store.AllocationTotalsByTask(r.Context(), taskIDs)
	if err != nil {
		return schedulePage{}, err
	}
	lineIDs := make([]string, 0, len(lines))
	for _, line := range lines {
		lineIDs = append(lineIDs, line.ID)
	}
	processAllocs, err := s.store.ListAllocationsByProcesses(r.Context(), lineIDs)
	if err != nil {
		return schedulePage{}, err
	}
	processTotals, err := s.store.AllocationTotalsByProcess(r.Context(), lineIDs)
	if err != nil {
		return schedulePage{}, err
	}
	holidays, err := s.store.ListHolidayDates(r.Context())
	if err != nil {
		return schedulePage{}, err
	}
	currentYM, currentDec := schedule.CurrentPeriod(time.Time{})
	std := s.cfg.StandardMinutes

	taskAllocMap := map[string]int64{}
	for _, a := range taskAllocs {
		taskAllocMap[a.TaskID+"|"+a.YearMonth+"|"+strconv.Itoa(a.Decade)] = a.PlannedMinutes
	}
	processAllocMap := map[string]int64{}
	periodAllocated := map[string]int64{}
	for _, a := range processAllocs {
		processAllocMap[a.ProjectProcessID+"|"+a.YearMonth+"|"+strconv.Itoa(a.Decade)] = a.PlannedMinutes
		periodAllocated[a.YearMonth+"|"+strconv.Itoa(a.Decade)] += a.PlannedMinutes
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
	assigns, err := s.store.AssignmentsForLines(r.Context(), lineIDs)
	if err != nil {
		return schedulePage{}, err
	}
	billable, err := s.assignmentBillable(r.Context(), assigns)
	if err != nil {
		return schedulePage{}, err
	}
	processActualMap, taskActualMap, processActualTotals, taskActualTotals, err := s.scheduleActualMaps(r.Context(), assigns, in.From, in.To)
	if err != nil {
		return schedulePage{}, err
	}

	canEdit := personFrom(r).IsAdmin
	blocks := make([]scheduleProjectBlock, 0, len(in.Projects))
	for _, project := range in.Projects {
		projectTasks := tasksByProject[project.ID]
		projectLines := linesByProject[project.ID]
		block := scheduleProjectBlock{Project: project}

		if in.Detail {
			block.Groups, block.RowCount = s.scheduleTaskGroups(projectLines, projectTasks, in.Months, currentYM, currentDec, taskAllocMap, taskTotals, processAllocMap, taskActualMap, taskActualTotals, processActualMap, processActualTotals, std, canEdit)
		} else {
			block.Rows, block.RowCount = s.scheduleProcessRows(projectLines, projectTasks, in.Months, currentYM, currentDec, processAllocMap, processTotals, processActualMap, processActualTotals, assigns, billable, bizDays, std, canEdit)
		}
		blocks = append(blocks, block)
	}

	periods := make([]schedulePeriodCol, 0, len(in.Months)*3)
	for _, ym := range in.Months {
		for decade := 1; decade <= 3; decade++ {
			periodKey := ym + "|" + strconv.Itoa(decade)
			heads := uniqueAssigneesInDecade(assigns, billable, ym, decade, "")
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
		CanEdit:     canEdit,
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

// scheduleActualMaps は表示期間の実績を、工程／タスク × 旬と行合計にまとめます。
func (s *Server) scheduleActualMaps(ctx context.Context, assigns []store.Assignment, fromYM, toYM string) (
	processCells, taskCells, processTotals, taskTotals map[string]int64, err error,
) {
	processCells = map[string]int64{}
	taskCells = map[string]int64{}
	processTotals = map[string]int64{}
	taskTotals = map[string]int64{}
	if len(assigns) == 0 {
		return processCells, taskCells, processTotals, taskTotals, nil
	}
	ids := make([]string, len(assigns))
	lineOf := map[string]string{}
	for i, a := range assigns {
		ids[i] = a.ID
		lineOf[a.ID] = a.ProjectProcessID
	}
	fromDay := fromYM + "-01"
	end, err := time.Parse("2006-01", toYM)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	until := end.AddDate(0, 1, 0).Format("2006-01-02")
	entries, err := s.store.ListEntries(ctx, ids, fromDay, until)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for _, e := range entries {
		day, err := time.Parse("2006-01-02", e.WorkDate)
		if err != nil {
			continue
		}
		ym := schedule.FormatYM(day.Year(), int(day.Month()))
		decade := schedule.DecadeOf(day)
		periodKey := ym + "|" + strconv.Itoa(decade)
		lineID := lineOf[e.AssignmentID]
		if lineID != "" {
			processCells[lineID+"|"+periodKey] += e.Minutes
			processTotals[lineID] += e.Minutes
		}
		if e.TaskID != "" {
			taskCells[e.TaskID+"|"+periodKey] += e.Minutes
			taskTotals[e.TaskID] += e.Minutes
		}
	}
	return processCells, taskCells, processTotals, taskTotals, nil
}

// attachActualMeta は行の実績合計と、見積との差分表示を付けます。
func attachActualMeta(row *scheduleRow, estimateMinutes, actualMinutes, std int64) {
	estStr, actualStr, diffStr, diffClass, diffSign := personDayRowMeta(estimateMinutes, actualMinutes, std)
	row.Actual = actualStr
	row.ActualDiff = diffStr
	row.ActualClass = diffClass
	row.ActualOver = diffSign < 0
	_ = estStr
}

// fillCellActual はセルに旬の実績表示を付けます。
func fillCellActual(cell *scheduleCell, actualMinutes, std int64) {
	if actualMinutes <= 0 {
		return
	}
	cell.HasActual = true
	cell.ActualDays = calc.FormatPersonDays(actualMinutes, std)
}

// scheduleProcessRows は工程ごとの行を作ります（一覧用）。
// canEdit が真のときだけ工数セルを編集可にします。
// 各旬セルに、その工程のアサイン人数ベースのキャパ（CapDays）と負荷色（CapClass）を付けます。
func (s *Server) scheduleProcessRows(
	lines []store.ProcessLine,
	tasks []store.Task,
	months []string,
	currentYM string,
	currentDec int,
	processAllocMap map[string]int64,
	processTotals map[string]int64,
	processActualMap map[string]int64,
	processActualTotals map[string]int64,
	assigns []store.Assignment,
	billable map[string]bool,
	bizDays map[string]int,
	std int64,
	canEdit bool,
) ([]scheduleRow, int) {
	if len(lines) == 0 {
		return nil, 1 // 未登録案内
	}
	tasksByLine := map[string][]store.Task{}
	for _, t := range tasks {
		tasksByLine[t.ProjectProcessID] = append(tasksByLine[t.ProjectProcessID], t)
	}
	rows := make([]scheduleRow, 0, len(lines))
	for _, line := range lines {
		lineTasks := tasksByLine[line.ID]
		var est int64
		color := schedule.NormalizeColor(line.ProcessColor)
		if len(lineTasks) == 0 {
			est = line.PlannedMinutes
		} else {
			for _, t := range lineTasks {
				est += t.PlannedMinutes
				color = schedule.NormalizeColor(t.ProcessColor)
			}
		}
		allocated := processTotals[line.ID]
		estStr, allocStr, diffStr, diffClass, _ := personDayRowMeta(est, allocated, std)
		row := scheduleRow{
			RowID:     line.ID,
			ProcessID: line.ID,
			Process:   line.ProcessName,
			Color:     color,
			Estimate:  estStr,
			Allocated: allocStr,
			Diff:      diffStr,
			DiffClass: diffClass,
			Editable:  canEdit,
		}
		attachActualMeta(&row, est, processActualTotals[line.ID], std)
		for _, ym := range months {
			for decade := 1; decade <= 3; decade++ {
				periodKey := ym + "|" + strconv.Itoa(decade)
				minutes := processAllocMap[line.ID+"|"+periodKey]
				heads := uniqueAssigneesInDecade(assigns, billable, ym, decade, line.ID)
				days := bizDays[periodKey]
				capMinutes := int64(heads) * int64(days) * std
				cell := scheduleCell{
					YM: ym, Decade: decade, Minutes: minutes,
					InRange:  schedule.OverlapsDecade(ym, decade, line.StartDate, line.EndDate),
					Current:  ym == currentYM && decade == currentDec,
					CapClass: capacityLoadClass(minutes, capMinutes),
					CapHeads: heads,
				}
				if minutes > 0 {
					cell.Days = calc.FormatPersonDays(minutes, std)
				}
				if heads > 0 {
					cell.CapDays = calc.FormatPersonDays(capMinutes, std)
				}
				fillCellActual(&cell, processActualMap[line.ID+"|"+periodKey], std)
				row.Cells = append(row.Cells, cell)
			}
		}
		rows = append(rows, row)
	}
	return rows, len(rows)
}

// scheduleTaskGroups は工程ごとのタスク行と残工数行を作ります（案件スケジュール用）。
// canEdit が真でタスクがあるときだけ工数セルを編集可にします。タスクのない工程は常に編集不可です。
// 残工数は一覧の工程割当 − タスク合計です。
func (s *Server) scheduleTaskGroups(
	lines []store.ProcessLine,
	tasks []store.Task,
	months []string,
	currentYM string,
	currentDec int,
	taskAllocMap map[string]int64,
	taskTotals map[string]int64,
	processAllocMap map[string]int64,
	taskActualMap map[string]int64,
	taskActualTotals map[string]int64,
	processActualMap map[string]int64,
	processActualTotals map[string]int64,
	std int64,
	canEdit bool,
) ([]scheduleProcessGroup, int) {
	if len(lines) == 0 {
		return nil, 1
	}
	tasksByLine := map[string][]store.Task{}
	for _, task := range tasks {
		tasksByLine[task.ProjectProcessID] = append(tasksByLine[task.ProjectProcessID], task)
	}
	groups := make([]scheduleProcessGroup, 0, len(lines))
	rowCount := 0
	for _, line := range lines {
		color := schedule.NormalizeColor(line.ProcessColor)
		lineTasks := tasksByLine[line.ID]
		taskRows := make([]scheduleRow, 0, len(lineTasks))
		periodMinutes := map[string]int64{}
		for _, task := range lineTasks {
			est := task.PlannedMinutes
			allocated := taskTotals[task.ID]
			estStr, allocStr, diffStr, diffClass, _ := personDayRowMeta(est, allocated, std)
			row := scheduleRow{
				RowID:     task.ID,
				ProcessID: line.ID,
				Process:   line.ProcessName,
				Name:      task.Name,
				Remarks:   task.Remarks,
				Color:     color,
				Estimate:  estStr,
				Allocated: allocStr,
				Diff:      diffStr,
				DiffClass: diffClass,
				Editable:  canEdit,
			}
			attachActualMeta(&row, est, taskActualTotals[task.ID], std)
			for _, ym := range months {
				for decade := 1; decade <= 3; decade++ {
					periodKey := ym + "|" + strconv.Itoa(decade)
					minutes := taskAllocMap[task.ID+"|"+periodKey]
					periodMinutes[periodKey] += minutes
					cell := scheduleCell{
						YM: ym, Decade: decade, Minutes: minutes,
						InRange: schedule.OverlapsDecade(ym, decade, line.StartDate, line.EndDate),
						Current: ym == currentYM && decade == currentDec,
					}
					if minutes > 0 {
						cell.Days = calc.FormatPersonDays(minutes, std)
					}
					fillCellActual(&cell, taskActualMap[task.ID+"|"+periodKey], std)
					row.Cells = append(row.Cells, cell)
				}
			}
			taskRows = append(taskRows, row)
		}
		if len(taskRows) == 0 {
			est := line.PlannedMinutes
			estStr, allocStr, diffStr, diffClass, _ := personDayRowMeta(est, 0, std)
			row := scheduleRow{
				RowID:     line.ID,
				ProcessID: line.ID,
				Process:   line.ProcessName,
				Name:      "—",
				Color:     color,
				Estimate:  estStr,
				Allocated: allocStr,
				Diff:      diffStr,
				DiffClass: diffClass,
				Editable:  false,
			}
			attachActualMeta(&row, est, processActualTotals[line.ID], std)
			for _, ym := range months {
				for decade := 1; decade <= 3; decade++ {
					periodKey := ym + "|" + strconv.Itoa(decade)
					cell := scheduleCell{
						YM: ym, Decade: decade,
						InRange: schedule.OverlapsDecade(ym, decade, line.StartDate, line.EndDate),
						Current: ym == currentYM && decade == currentDec,
					}
					fillCellActual(&cell, processActualMap[line.ID+"|"+periodKey], std)
					row.Cells = append(row.Cells, cell)
				}
			}
			taskRows = append(taskRows, row)
		}
		remain := scheduleRow{
			RowID:     line.ID,
			ProcessID: line.ID,
			Process:   line.ProcessName,
			Name:      "残工数",
			Color:     color,
		}
		for _, ym := range months {
			for decade := 1; decade <= 3; decade++ {
				periodKey := ym + "|" + strconv.Itoa(decade)
				procMinutes := processAllocMap[line.ID+"|"+periodKey]
				taskMinutes := periodMinutes[periodKey]
				remainT := calc.PersonDayTenths(procMinutes, std) - calc.PersonDayTenths(taskMinutes, std)
				cell := scheduleCell{
					YM: ym, Decade: decade, Minutes: procMinutes,
					Current: ym == currentYM && decade == currentDec,
				}
				if procMinutes > 0 || taskMinutes > 0 {
					cell.Days = calc.FormatPersonDayTenths(remainT)
				}
				if remainT < 0 {
					cell.CapClass = "cap-over"
				}
				remain.Cells = append(remain.Cells, cell)
			}
		}
		span := len(taskRows) + 1
		groups = append(groups, scheduleProcessGroup{
			ProcessID: line.ID,
			Process:   line.ProcessName,
			Color:     color,
			RowSpan:   span,
			Tasks:     taskRows,
			Remain:    remain,
		})
		rowCount += span
	}
	return groups, rowCount
}

// assignmentBillable はアサイン ID ごとに、開始日時点の単価が 0 円超かどうかを返します。
// 0 円単価（OJT）はキャパに含めないための判定です。
func (s *Server) assignmentBillable(ctx context.Context, assigns []store.Assignment) (map[string]bool, error) {
	out := make(map[string]bool, len(assigns))
	cache := map[string]bool{}
	for _, a := range assigns {
		if a.ID == "" {
			continue
		}
		key := a.PersonID + "|" + a.RoleID + "|" + a.StartDate + "|" + a.RateKind
		if v, ok := cache[key]; ok {
			out[a.ID] = v
			continue
		}
		rate, ok, err := s.store.FindRate(ctx, a.PersonID, a.RoleID, a.StartDate)
		if err != nil {
			return nil, err
		}
		v := store.RatePositive(a.RateKind, rate, ok)
		cache[key] = v
		out[a.ID] = v
	}
	return out, nil
}

// uniqueAssigneesInDecade は、ym の decade 旬にアサイン期間が重なる人材のユニーク人数です。
// processID が空なら全工程、指定があればその工程のアサインだけを数えます。
// 同じ人が複数工程・複数案件にいても 1 人と数えます。0 円単価のアサインは含めません。
func uniqueAssigneesInDecade(assigns []store.Assignment, billable map[string]bool, ym string, decade int, processID string) int {
	seen := map[string]bool{}
	for _, a := range assigns {
		if a.PersonID == "" || !billable[a.ID] {
			continue
		}
		if processID != "" && a.ProjectProcessID != processID {
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
// task_id があればタスク割当、project_process_id があれば工程割当です。
func (s *Server) handleAllocationSave(w http.ResponseWriter, r *http.Request) {
	taskID := formText(r, "task_id")
	processID := formText(r, "project_process_id")
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
	std := s.cfg.StandardMinutes
	if taskID != "" {
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
		estStr, allocStr, diffStr, _, diffSign := personDayRowMeta(task.PlannedMinutes, allocated, std)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"kind":      "task",
			"task_id":   task.ID,
			"effort":    calc.FormatPersonDays(minutes, std),
			"estimate":  estStr,
			"allocated": allocStr,
			"diff":      diffStr,
			"diff_sign": diffSign,
		})
		return
	}
	if processID == "" {
		http.Error(w, "割当対象が不正です。", http.StatusBadRequest)
		return
	}
	line, err := s.store.GetProcessLine(r.Context(), processID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "工程が見つかりません。", http.StatusNotFound)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.store.UpsertProcessAllocation(r.Context(), line.ID, ym, decade, minutes); err != nil {
		s.fail(w, err)
		return
	}
	est, err := s.processEstimateMinutes(r.Context(), line)
	if err != nil {
		s.fail(w, err)
		return
	}
	totals, err := s.store.AllocationTotalsByProcess(r.Context(), []string{line.ID})
	if err != nil {
		s.fail(w, err)
		return
	}
	allocated := totals[line.ID]
	estStr, allocStr, diffStr, _, diffSign := personDayRowMeta(est, allocated, std)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"kind":               "process",
		"project_process_id": line.ID,
		"effort":             calc.FormatPersonDays(minutes, std),
		"estimate":           estStr,
		"allocated":          allocStr,
		"diff":               diffStr,
		"diff_sign":          diffSign,
	})
}

// personDayRowMeta は見積・割当・差分を、画面の人日（0.1 刻み）同士で揃えて返します。
// diffSign は 0.1 人日の個数の差（正: 割当不足、負: 超過）で、JS の色分けに使います。
func personDayRowMeta(estimateMinutes, allocatedMinutes, standardMinutes int64) (estimate, allocated, diff, diffClass string, diffSign int64) {
	estT := calc.PersonDayTenths(estimateMinutes, standardMinutes)
	allocT := calc.PersonDayTenths(allocatedMinutes, standardMinutes)
	diffT := estT - allocT
	if diffT > 0 {
		diffClass = "warn"
	} else if diffT < 0 {
		diffClass = "diff-over"
	}
	return calc.FormatPersonDayTenths(estT), calc.FormatPersonDayTenths(allocT), calc.FormatPersonDayTenths(diffT), diffClass, diffT
}

// processEstimateMinutes は工程明細の見積分です。タスクがあればその合計、無ければ工程の予定分です。
func (s *Server) processEstimateMinutes(ctx context.Context, line store.ProcessLine) (int64, error) {
	tasks, err := s.store.ListTasksByProject(ctx, line.ProjectID)
	if err != nil {
		return 0, err
	}
	var sum int64
	var n int
	for _, t := range tasks {
		if t.ProjectProcessID == line.ID {
			sum += t.PlannedMinutes
			n++
		}
	}
	if n == 0 {
		return line.PlannedMinutes, nil
	}
	return sum, nil
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
