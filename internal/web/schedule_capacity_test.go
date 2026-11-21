package web

import "testing"

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
