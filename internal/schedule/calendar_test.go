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
	// 2026-11 上旬: 日〜火始まり。1=日,2=月,...,10=火。平日は 2-6,9,10 = 7 日。
	days, err := BusinessDays("2026-11", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if days != 7 {
		t.Fatalf("days = %d", days)
	}
}

func TestColorForSort(t *testing.T) {
	if ColorForSort(1) != "cyan" || ColorForSort(5) != "cyan" {
		t.Fatal(ColorForSort(1), ColorForSort(5))
	}
}
