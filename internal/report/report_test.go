package report

import (
	"testing"
	"time"

	"odysseus/internal/store"
)

func TestBuildUsesLatestRateAndGrossMargin(t *testing.T) {
	order := int64(1000000)
	b := store.Bundle{
		Project: store.Project{ID: "p", Code: "P1", Name: "案件", OrderAmount: &order, Status: "active"},
		Lines: []store.ProcessLine{{
			ID: "l", ProjectID: "p", ProcessID: "proc", ProcessName: "製造", PlannedMinutes: 4800,
		}},
		Assignments: []store.Assignment{{
			ID: "a", ProjectProcessID: "l", PersonID: "h", PersonCode: "m01", PersonName: "山田",
			RoleID: "pg", RoleName: "PG", StartDate: "2026-04-01", EndDate: "2026-04-30",
			RateKind: store.RateKindHourly, PlannedCostYen: 400000,
		}},
		Entries: []store.TimeEntry{{
			ID: "e", AssignmentID: "a", WorkDate: "2026-04-02", Minutes: 480,
		}},
		Rates: []store.Rate{
			{PersonID: "h", RoleID: "pg", FromDate: "2026-01-01", HourlyYen: 3000},
			{PersonID: "h", RoleID: "pg", FromDate: "2026-02-01", HourlyYen: 5000},
		},
	}
	got := Build(b, time.Time{})
	if got.ActualCostYen != 40000 {
		t.Fatalf("actual cost = %d, want 40000 (latest rate 5000, not 3000)", got.ActualCostYen)
	}
	if got.RemainingCostYen != 360000 {
		t.Fatalf("remaining = %d", got.RemainingCostYen)
	}
	if got.GrossMargin == nil || *got.GrossMargin != 600000 {
		t.Fatalf("gross = %v", got.GrossMargin)
	}
	if !got.ActualKnown || got.UnsetMinutes != 0 {
		t.Fatalf("known actual: %+v", got)
	}
	if !got.DerivedBudget || got.BudgetYen != 400000 {
		t.Fatalf("budget should fall back to planned cost: %+v", got)
	}
	if got.PlannedMinutes != 4800 || got.ActualMinutes != 480 {
		t.Fatalf("minutes planned=%d actual=%d", got.PlannedMinutes, got.ActualMinutes)
	}
}

func TestBuildLeavesUnsetRateOutOfYen(t *testing.T) {
	b := store.Bundle{
		Project: store.Project{ID: "p", Status: "active"},
		Lines:   []store.ProcessLine{{ID: "l", ProcessName: "製造", PlannedMinutes: 480}},
		Assignments: []store.Assignment{{
			ID: "a", ProjectProcessID: "l", PersonID: "h", RoleID: "pg",
			RateKind: store.RateKindHourly, PlannedCostYen: 40000,
		}},
		Entries: []store.TimeEntry{{ID: "e", AssignmentID: "a", WorkDate: "2026-04-02", Minutes: 60}},
	}
	got := Build(b, time.Time{})
	if got.ActualKnown {
		t.Fatal("expected unknown rate")
	}
	if got.ActualCostYen != 0 || got.UnsetMinutes != 60 {
		t.Fatalf("cost=%d unset=%d", got.ActualCostYen, got.UnsetMinutes)
	}
}

func TestBuildTaskActualAndOverPlan(t *testing.T) {
	b := store.Bundle{
		Project: store.Project{ID: "p", Status: "active"},
		Lines:   []store.ProcessLine{{ID: "l", ProcessName: "製造", PlannedMinutes: 960}},
		Tasks: []store.Task{
			{ID: "t1", ProjectProcessID: "l", ProcessName: "製造", Name: "画面", PlannedMinutes: 480},
			{ID: "t2", ProjectProcessID: "l", ProcessName: "製造", Name: "API", PlannedMinutes: 480},
		},
		Assignments: []store.Assignment{{
			ID: "a", ProjectProcessID: "l", PersonID: "h", RoleID: "pg",
			RateKind: store.RateKindHourly, PlannedCostYen: 80000,
		}},
		Entries: []store.TimeEntry{
			{ID: "e1", AssignmentID: "a", TaskID: "t1", WorkDate: "2026-04-01", Minutes: 600},
			{ID: "e2", AssignmentID: "a", TaskID: "t2", WorkDate: "2026-04-02", Minutes: 120},
		},
		Rates: []store.Rate{{PersonID: "h", RoleID: "pg", FromDate: "2026-01-01", HourlyYen: 5000}},
	}
	got := Build(b, time.Time{})
	if got.ActualMinutes != 720 {
		t.Fatalf("actual minutes = %d", got.ActualMinutes)
	}
	if len(got.Tasks) != 2 {
		t.Fatalf("tasks = %d", len(got.Tasks))
	}
	if !got.Tasks[0].OverPlan || got.Tasks[0].ActualMinutes != 600 {
		t.Fatalf("task1: %+v", got.Tasks[0])
	}
	if got.Tasks[1].OverPlan || got.Tasks[1].ActualMinutes != 120 {
		t.Fatalf("task2: %+v", got.Tasks[1])
	}
	if got.Processes[0].OverPlan {
		t.Fatalf("process over plan = true, actual=%d planned=%d", got.Processes[0].ActualMinutes, got.Processes[0].PlannedMinutes)
	}
}

func TestBuildHourlyDespiteMonthlyRate(t *testing.T) {
	b := store.Bundle{
		Project: store.Project{ID: "p", Status: "active"},
		Lines:   []store.ProcessLine{{ID: "l", ProcessName: "製造", PlannedMinutes: 480}},
		Assignments: []store.Assignment{{
			ID: "a", ProjectProcessID: "l", PersonID: "h", RoleID: "pg",
			StartDate: "2026-04-01", EndDate: "2026-04-30",
			RateKind: store.RateKindHourly, PlannedCostYen: 0,
		}},
		Entries: []store.TimeEntry{{
			ID: "e", AssignmentID: "a", WorkDate: "2026-04-02", Minutes: 60,
		}},
		Rates: []store.Rate{{
			PersonID: "h", RoleID: "pg", FromDate: "2026-01-01",
			HourlyYen: 5000, MonthlyYen: 400000,
		}},
	}
	got := Build(b, time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC))
	if got.ActualCostYen != 5000 {
		t.Fatalf("hourly actual = %d, want 5000 (not monthly)", got.ActualCostYen)
	}
}

func TestBuildMonthlyActualIgnoresHours(t *testing.T) {
	order := int64(1000000)
	b := store.Bundle{
		Project: store.Project{ID: "p", OrderAmount: &order, Status: "active"},
		Lines:   []store.ProcessLine{{ID: "l", ProcessName: "製造", PlannedMinutes: 4800}},
		Assignments: []store.Assignment{{
			ID: "a", ProjectProcessID: "l", PersonID: "h", PersonName: "山田",
			RoleID: "pg", RoleName: "PG", StartDate: "2026-04-01", EndDate: "2026-06-30",
			RateKind: store.RateKindMonthly, PlannedCostYen: 300000,
		}},
		Entries: []store.TimeEntry{{
			ID: "e", AssignmentID: "a", WorkDate: "2026-04-10", Minutes: 480,
		}},
		Rates: []store.Rate{{
			PersonID: "h", RoleID: "pg", FromDate: "2026-01-01",
			HourlyYen: 5000, MonthlyYen: 100000,
		}},
	}
	got := Build(b, time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC))
	if got.ActualCostYen != 200000 {
		t.Fatalf("monthly actual = %d, want 200000 (Apr+May)", got.ActualCostYen)
	}
	if got.ActualMinutes != 480 {
		t.Fatalf("minutes still counted = %d", got.ActualMinutes)
	}
	if got.RemainingCostYen != 100000 {
		t.Fatalf("remaining = %d", got.RemainingCostYen)
	}
	if got.UnsetMinutes != 0 || !got.ActualKnown {
		t.Fatalf("known: %+v", got)
	}
}

func TestHourlyAtPicksNewestNotCheapest(t *testing.T) {
	idx := indexRates([]store.Rate{
		{PersonID: "h", RoleID: "pg", FromDate: "2026-01-01", HourlyYen: 3000},
		{PersonID: "h", RoleID: "pg", FromDate: "2026-02-01", HourlyYen: 9000},
	})
	yen, ok := hourlyAt(idx, "h", "pg", "2026-03-01")
	if !ok || yen != 9000 {
		t.Fatalf("got %d %v", yen, ok)
	}
	if _, ok := hourlyAt(idx, "h", "pg", "2025-12-31"); ok {
		t.Fatal("rate before the first from_date should be missing")
	}
}
