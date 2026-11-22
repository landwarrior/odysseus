// Package report は保存済みの明細から予実の数字を組み立てます。
package report

import (
	"strings"
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
// 準委任の原価は工程には載せません（案件合計と人材行のみ）。
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

// PersonRow はアサインの予実です。請負は工程ごとに1行、準委任は案件内で人材1行にまとめます。
// RateKind は hourly（請負）または monthly（準委任）です。
type PersonRow struct {
	ProcessName      string
	PersonID         string
	PersonName       string
	PersonCode       string
	RoleName         string
	RateKind         string
	StartDate        string
	EndDate          string
	ActualMinutes    int64
	PlannedCostYen   int64
	ActualCostYen    int64
	RemainingCostYen int64
	UnsetMinutes     int64
}

// Build は b だけから予実を計算します。準委任の按分は b 内の案件・アサインだけで行います。
// 案件横断の按分が必要なら BuildMany か BuildWithContext を使います。
// asOf は準委任の実績原価をどこまで計上するかの基準日です。ゼロ値なら実行時点の日付です。
func Build(b store.Bundle, asOf time.Time) Report {
	return BuildWithContext(b, []store.Bundle{b}, asOf)
}

// BuildMany は bundles 全部を案件横断の準委任按分コンテキストとして、同じ並びに Report を返します。
func BuildMany(bundles []store.Bundle, asOf time.Time) []Report {
	out := make([]Report, len(bundles))
	for i, b := range bundles {
		out[i] = BuildWithContext(b, bundles, asOf)
	}
	return out
}

// BuildWithContext は b の予実を計算します。peers は準委任按分用の他案件（b 自身を含めてよい）です。
// asOf がゼロ値なら実行時点の日付です。
func BuildWithContext(b store.Bundle, peers []store.Bundle, asOf time.Time) Report {
	if asOf.IsZero() {
		asOf = time.Now()
	}
	asOfDate := asOf.UTC().Format("2006-01-02")
	idx := indexRates(b.Rates)
	for _, peer := range peers {
		for _, r := range peer.Rates {
			idx = mergeRate(idx, r)
		}
	}
	entries := map[string][]store.TimeEntry{}
	for _, e := range b.Entries {
		entries[e.AssignmentID] = append(entries[e.AssignmentID], e)
	}
	byLine := map[string][]store.Assignment{}
	for _, a := range b.Assignments {
		byLine[a.ProjectProcessID] = append(byLine[a.ProjectProcessID], a)
	}

	orders := map[string]*int64{}
	var stakes []calc.MonthlyStake
	for _, peer := range peers {
		orders[peer.Project.ID] = peer.Project.OrderAmount
		for _, a := range peer.Assignments {
			if a.RateKind != store.RateKindMonthly {
				continue
			}
			projectID := a.ProjectID
			if projectID == "" {
				projectID = peer.Project.ID
			}
			rate, hasRate := rateAt(idx, a.PersonID, a.RoleID, a.StartDate)
			stake := calc.MonthlyStake{
				PersonID:  a.PersonID,
				ProjectID: projectID,
				StartDate: a.StartDate,
				EndDate:   a.EndDate,
				HasRate:   hasRate && rate.MonthlyYen != nil,
			}
			if stake.HasRate {
				stake.MonthlyYen = *rate.MonthlyYen
			}
			stakes = append(stakes, stake)
		}
	}
	monthlyCosts, monthlyMissing := calc.MonthlyPersonCosts(stakes, orders, asOfDate)

	rep := Report{
		OrderAmount: b.Project.OrderAmount,
		ActualKnown: !monthlyMissing,
	}
	type monthlyAgg struct {
		row      PersonRow
		roles    []string
		roleSeen map[string]bool
		procs    []string
		procSeen map[string]bool
	}
	monthlyPeople := map[string]*monthlyAgg{}

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
			for _, e := range entries[a.ID] {
				row.ActualMinutes += e.Minutes
				rep.ActualMinutes += e.Minutes
			}
			if a.RateKind == store.RateKindMonthly {
				agg := monthlyPeople[a.PersonID]
				if agg == nil {
					agg = &monthlyAgg{
						row: PersonRow{
							PersonID:   a.PersonID,
							PersonName: a.PersonName,
							PersonCode: a.PersonCode,
							RateKind:   store.RateKindMonthly,
							StartDate:  a.StartDate,
							EndDate:    a.EndDate,
						},
						roleSeen: map[string]bool{},
						procSeen: map[string]bool{},
					}
					cost := monthlyCosts[calc.PersonProjectKey(a.PersonID, b.Project.ID)]
					agg.row.PlannedCostYen = cost.PlannedYen
					agg.row.ActualCostYen = cost.ActualYen
					agg.row.RemainingCostYen = calc.RemainingCost(cost.PlannedYen, cost.ActualYen)
					monthlyPeople[a.PersonID] = agg
				}
				if a.StartDate != "" && (agg.row.StartDate == "" || a.StartDate < agg.row.StartDate) {
					agg.row.StartDate = a.StartDate
				}
				if a.EndDate != "" && (agg.row.EndDate == "" || a.EndDate > agg.row.EndDate) {
					agg.row.EndDate = a.EndDate
				}
				if a.RoleName != "" && !agg.roleSeen[a.RoleName] {
					agg.roleSeen[a.RoleName] = true
					agg.roles = append(agg.roles, a.RoleName)
				}
				if line.ProcessName != "" && !agg.procSeen[line.ProcessName] {
					agg.procSeen[line.ProcessName] = true
					agg.procs = append(agg.procs, line.ProcessName)
				}
				for _, e := range entries[a.ID] {
					agg.row.ActualMinutes += e.Minutes
				}
				rate, hasRate := rateAt(idx, a.PersonID, a.RoleID, a.StartDate)
				if !hasRate || rate.MonthlyYen == nil {
					rep.ActualKnown = false
				}
				continue
			}

			person := PersonRow{
				ProcessName:    line.ProcessName,
				PersonID:       a.PersonID,
				PersonName:     a.PersonName,
				PersonCode:     a.PersonCode,
				RoleName:       a.RoleName,
				RateKind:       a.RateKind,
				StartDate:      a.StartDate,
				EndDate:        a.EndDate,
				PlannedCostYen: a.PlannedCostYen,
			}
			row.PlannedCostYen += a.PlannedCostYen
			rep.PlannedCostYen += a.PlannedCostYen
			for _, e := range entries[a.ID] {
				person.ActualMinutes += e.Minutes
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

	var monthlyIDs []string
	for id := range monthlyPeople {
		monthlyIDs = append(monthlyIDs, id)
	}
	sort.Strings(monthlyIDs)
	for _, id := range monthlyIDs {
		agg := monthlyPeople[id]
		if len(agg.procs) == 0 {
			agg.row.ProcessName = "—"
		} else {
			agg.row.ProcessName = strings.Join(agg.procs, "、")
		}
		agg.row.RoleName = strings.Join(agg.roles, "、")
		rep.PlannedCostYen += agg.row.PlannedCostYen
		rep.ActualCostYen += agg.row.ActualCostYen
		rep.RemainingCostYen += agg.row.RemainingCostYen
		rep.People = append(rep.People, agg.row)
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

// mergeRate は r を idx に足します（同一キーは FromDate 新しい順）。
func mergeRate(idx map[string][]store.Rate, r store.Rate) map[string][]store.Rate {
	if idx == nil {
		idx = map[string][]store.Rate{}
	}
	k := r.PersonID + "\x00" + r.RoleID
	for _, existing := range idx[k] {
		if existing.ID != "" && existing.ID == r.ID {
			return idx
		}
		if existing.ID == "" && existing.FromDate == r.FromDate && existing.PersonID == r.PersonID && existing.RoleID == r.RoleID {
			return idx
		}
	}
	idx[k] = append(idx[k], r)
	sort.Slice(idx[k], func(i, j int) bool {
		return idx[k][i].FromDate > idx[k][j].FromDate
	})
	return idx
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
// 該当が無いとき、または時間単価が未設定のときは false です。0 円は有効です。
func hourlyAt(idx map[string][]store.Rate, personID, roleID, date string) (int64, bool) {
	r, ok := rateAt(idx, personID, roleID, date)
	if !ok || r.HourlyYen == nil {
		return 0, false
	}
	return *r.HourlyYen, true
}
