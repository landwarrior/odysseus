// Package report は保存済みの明細から予実の数字を組み立てます。
package report

import (
	"time"

	"odysseus/internal/calc"
	"odysseus/internal/store"
	"sort"
)

// Report は1案件の予実です。金額の単位は円、工数の単位は分です。
// OrderAmount と GrossMargin が nil のときは、画面では未設定と出します。
// ActualKnown が偽のときは、UnsetMinutes の分だけ単価が無く、金額に含めていません。
type Report struct {
	OrderAmount           *int64
	BudgetYen             int64
	PlannedCostYen        int64
	ActualCostYen         int64
	RemainingCostYen      int64
	GrossMargin           *int64
	ActualKnown           bool
	UnsetMinutes          int64
	PlannedMinutes        int64
	ProcessPlannedMinutes int64
	ActualMinutes         int64
	DerivedBudget         bool
	Processes             []ProcessRow
	Tasks                 []TaskRow
	People                []PersonRow
}

// ProcessRow は工程1行の予実です。
// BudgetFromAssignments が真のときは、手入力の予算が空なので予定原価を予算にしています。
// OverPlan が真のときは、実績工数が工程の予定工数を超えています。
type ProcessRow struct {
	Name                  string
	StartDate             string
	EndDate               string
	PlannedMinutes        int64
	ActualMinutes         int64
	BudgetYen             int64
	BudgetFromAssignments bool
	PlannedCostYen        int64
	ActualCostYen         int64
	UnsetMinutes          int64
	OverBudget            bool
	OverPlan              bool
}

// TaskRow はタスク1行の見積と実績です。OverPlan が真のときは実績が見積を超えています。
type TaskRow struct {
	ProcessName    string
	Name           string
	PlannedMinutes int64
	ActualMinutes  int64
	OverPlan       bool
}

// PersonRow はアサイン1件の予実です。同じ人が複数工程にいると、行も複数になります。
type PersonRow struct {
	ProcessName      string
	PersonName       string
	PersonCode       string
	RoleName         string
	StartDate        string
	EndDate          string
	ActualMinutes    int64
	PlannedCostYen   int64
	ActualCostYen    int64
	RemainingCostYen int64
	UnsetMinutes     int64
}

// Build は b（案件、工程、アサイン、実績、単価）から予実を計算します。データベースには触りません。
// asOf は準委任の実績原価をどこまで計上するかの基準日です。ゼロ値なら実行時点の日付です。
func Build(b store.Bundle, asOf time.Time) Report {
	if asOf.IsZero() {
		asOf = time.Now()
	}
	asOfDate := asOf.UTC().Format("2006-01-02")
	idx := indexRates(b.Rates)
	entries := map[string][]store.TimeEntry{}
	for _, e := range b.Entries {
		entries[e.AssignmentID] = append(entries[e.AssignmentID], e)
	}
	byLine := map[string][]store.Assignment{}
	for _, a := range b.Assignments {
		byLine[a.ProjectProcessID] = append(byLine[a.ProjectProcessID], a)
	}

	rep := Report{
		OrderAmount: b.Project.OrderAmount,
		ActualKnown: true,
	}
	for _, line := range b.Lines {
		row := ProcessRow{
			Name:           line.ProcessName,
			StartDate:      line.StartDate,
			EndDate:        line.EndDate,
			PlannedMinutes: line.PlannedMinutes,
		}
		rep.ProcessPlannedMinutes += line.PlannedMinutes
		rep.PlannedMinutes += line.PlannedMinutes
		for _, a := range byLine[line.ID] {
			person := PersonRow{
				ProcessName:    line.ProcessName,
				PersonName:     a.PersonName,
				PersonCode:     a.PersonCode,
				RoleName:       a.RoleName,
				StartDate:      a.StartDate,
				EndDate:        a.EndDate,
				PlannedCostYen: a.PlannedCostYen,
			}
			row.PlannedCostYen += a.PlannedCostYen
			rep.PlannedCostYen += a.PlannedCostYen
			for _, e := range entries[a.ID] {
				person.ActualMinutes += e.Minutes
				row.ActualMinutes += e.Minutes
				rep.ActualMinutes += e.Minutes
			}
			rate, hasRate := rateAt(idx, a.PersonID, a.RoleID, a.StartDate)
			if hasRate && rate.MonthlyYen > 0 {
				cost := calc.AccruedMonthlyCost(a.StartDate, a.EndDate, asOfDate, rate.MonthlyYen)
				person.ActualCostYen = cost
				row.ActualCostYen += cost
				rep.ActualCostYen += cost
			} else {
				for _, e := range entries[a.ID] {
					yen, ok := hourlyAt(idx, a.PersonID, a.RoleID, e.WorkDate)
					if !ok {
						person.UnsetMinutes += e.Minutes
						row.UnsetMinutes += e.Minutes
						rep.UnsetMinutes += e.Minutes
						rep.ActualKnown = false
						continue
					}
					cost := calc.CostYen(e.Minutes, yen)
					person.ActualCostYen += cost
					row.ActualCostYen += cost
					rep.ActualCostYen += cost
				}
			}
			person.RemainingCostYen = calc.RemainingCost(person.PlannedCostYen, person.ActualCostYen)
			rep.RemainingCostYen += person.RemainingCostYen
			rep.People = append(rep.People, person)
		}
		if line.BudgetYen != nil {
			row.BudgetYen = *line.BudgetYen
		} else {
			row.BudgetYen = row.PlannedCostYen
			row.BudgetFromAssignments = true
			rep.DerivedBudget = true
		}
		rep.BudgetYen += row.BudgetYen
		row.OverBudget = row.ActualMinutes > 0 && row.ActualCostYen > row.BudgetYen
		row.OverPlan = row.ActualMinutes > row.PlannedMinutes
		rep.Processes = append(rep.Processes, row)
	}
	actualByTask := map[string]int64{}
	for _, e := range b.Entries {
		if e.TaskID != "" {
			actualByTask[e.TaskID] += e.Minutes
		}
	}
	for _, task := range b.Tasks {
		actual := actualByTask[task.ID]
		rep.Tasks = append(rep.Tasks, TaskRow{
			ProcessName:    task.ProcessName,
			Name:           task.Name,
			PlannedMinutes: task.PlannedMinutes,
			ActualMinutes:  actual,
			OverPlan:       actual > task.PlannedMinutes,
		})
	}
	if rep.OrderAmount != nil {
		g := calc.GrossMargin(*rep.OrderAmount, rep.ActualCostYen, rep.RemainingCostYen)
		rep.GrossMargin = &g
	}
	return rep
}

// indexRates は rates を、人材と役割の組ごとに、適用開始日の新しい順へ並べます。
func indexRates(rates []store.Rate) map[string][]store.Rate {
	m := map[string][]store.Rate{}
	for _, r := range rates {
		k := r.PersonID + "\x00" + r.RoleID
		m[k] = append(m[k], r)
	}
	for k := range m {
		sort.Slice(m[k], func(i, j int) bool {
			return m[k][i].FromDate > m[k][j].FromDate
		})
	}
	return m
}

// rateAt は idx から、personID と roleID について date 以前で最も新しい単価を返します。
func rateAt(idx map[string][]store.Rate, personID, roleID, date string) (store.Rate, bool) {
	for _, r := range idx[personID+"\x00"+roleID] {
		if r.FromDate <= date {
			return r, true
		}
	}
	return store.Rate{}, false
}

// hourlyAt は idx から、personID と roleID について date 以前で最も新しい時間単価を返します。
// 該当が無いとき、または時間単価が 0 のときは false です。
func hourlyAt(idx map[string][]store.Rate, personID, roleID, date string) (int64, bool) {
	r, ok := rateAt(idx, personID, roleID, date)
	if !ok || r.HourlyYen <= 0 {
		return 0, false
	}
	return r.HourlyYen, true
}
