package report

import (
	"testing"

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
			PlannedMinutes: 4800, PlannedCostYen: 400000,
		}},
		Entries: []store.TimeEntry{{
			ID: "e", AssignmentID: "a", WorkDate: "2026-04-02", Minutes: 480,
		}},
		Rates: []store.Rate{
			{PersonID: "h", RoleID: "pg", FromDate: "2026-01-01", HourlyYen: 3000},
			{PersonID: "h", RoleID: "pg", FromDate: "2026-02-01", HourlyYen: 5000},
		},
	}
	got := Build(b)
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
			PlannedMinutes: 480, PlannedCostYen: 40000,
		}},
		Entries: []store.TimeEntry{{ID: "e", AssignmentID: "a", WorkDate: "2026-04-02", Minutes: 60}},
	}
	got := Build(b)
	if got.ActualKnown {
		t.Fatal("expected unknown rate")
	}
	if got.ActualCostYen != 0 || got.UnsetMinutes != 60 {
		t.Fatalf("cost=%d unset=%d", got.ActualCostYen, got.UnsetMinutes)
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
