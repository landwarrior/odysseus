package schedule

import "testing"

func TestIterMonths(t *testing.T) {
	got, err := IterMonths("2026-11", "2026-12")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "2026-11" || got[1] != "2026-12" {
		t.Fatalf("%v", got)
	}
}

func TestBusinessDays(t *testing.T) {
	// 2026-11 上旬: 1=日,2=月,...,10=火。平日は 2-6,9,10 = 7 日。
	days, err := BusinessDays("2026-11", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if days != 7 {
		t.Fatalf("days = %d", days)
	}
	days, err = BusinessDays("2026-11", 1, []string{"2026-11-03"})
	if err != nil {
		t.Fatal(err)
	}
	if days != 6 {
		t.Fatalf("with holiday = %d", days)
	}
}

func TestColorForSort(t *testing.T) {
	if ColorForSort(1) != "cyan" || ColorForSort(5) != "rose" || ColorForSort(7) != "cyan" {
		t.Fatal(ColorForSort(1), ColorForSort(5), ColorForSort(7))
	}
}

func TestOverlapsDecade(t *testing.T) {
	if !OverlapsDecade("2026-11", 1, "2026-11-01", "2026-11-05") {
		t.Fatal("should overlap early November")
	}
	if OverlapsDecade("2026-11", 3, "2026-11-01", "2026-11-05") {
		t.Fatal("should not overlap late November")
	}
	if !OverlapsDecade("2026-11", 2, "2026-11-10", "2026-11-15") {
		t.Fatal("should overlap mid November")
	}
	if OverlapsDecade("2026-11", 1, "", "") {
		t.Fatal("empty range is not active")
	}
	if !OverlapsDecade("2026-12", 1, "2026-11-01", "") {
		t.Fatal("open end should keep overlapping")
	}
}

func TestNormalizeColor(t *testing.T) {
	if NormalizeColor("orange") != "orange" {
		t.Fatal(NormalizeColor("orange"))
	}
	if NormalizeColor("rose") != "rose" {
		t.Fatal(NormalizeColor("rose"))
	}
	if NormalizeColor("yellow") != "yellow" {
		t.Fatal(NormalizeColor("yellow"))
	}
	if NormalizeColor("nope") != "cyan" {
		t.Fatal(NormalizeColor("nope"))
	}
}
