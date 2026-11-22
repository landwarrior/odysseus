package web

import (
	"testing"

	"odysseus/internal/store"
)

func TestUniqueAssigneesInDecade(t *testing.T) {
	assigns := []store.Assignment{
		{ID: "1", PersonID: "a", StartDate: "2026-11-01", EndDate: "2026-11-30"},
		{ID: "2", PersonID: "a", StartDate: "2026-11-01", EndDate: "2026-11-15"}, // 同一人物・別工程
		{ID: "3", PersonID: "b", StartDate: "2026-11-01", EndDate: "2026-11-05"},
		{ID: "4", PersonID: "c", StartDate: "2026-12-01", EndDate: "2026-12-10"}, // 別旬
		{ID: "5", PersonID: "d", StartDate: "2026-11-01", EndDate: "2026-11-30"}, // 0 円（OJT）
	}
	billable := map[string]bool{
		"1": true, "2": true, "3": true, "4": true, "5": false,
	}
	if got := uniqueAssigneesInDecade(assigns, billable, "2026-11", 1); got != 2 {
		t.Fatalf("early Nov unique = %d, want 2 (d excluded)", got)
	}
	if got := uniqueAssigneesInDecade(assigns, billable, "2026-11", 3); got != 1 {
		t.Fatalf("late Nov unique = %d, want 1 (only a)", got)
	}
	if got := uniqueAssigneesInDecade(assigns, billable, "2026-12", 1); got != 1 {
		t.Fatalf("Dec unique = %d, want 1", got)
	}
}

func TestCapacityLoadClass(t *testing.T) {
	if got := capacityLoadClass(0, 100); got != "" {
		t.Fatalf("empty load: %q", got)
	}
	if got := capacityLoadClass(79, 100); got != "" {
		t.Fatalf("under 80%%: %q", got)
	}
	if got := capacityLoadClass(80, 100); got != "cap-warn" {
		t.Fatalf("80%%: %q", got)
	}
	if got := capacityLoadClass(99, 100); got != "cap-warn" {
		t.Fatalf("under capacity: %q", got)
	}
	if got := capacityLoadClass(100, 100); got != "cap-over" {
		t.Fatalf("at capacity: %q", got)
	}
	if got := capacityLoadClass(10, 0); got != "cap-over" {
		t.Fatalf("zero capacity with load: %q", got)
	}
	if got := capacityLoadClass(0, 0); got != "" {
		t.Fatalf("zero both: %q", got)
	}
}
