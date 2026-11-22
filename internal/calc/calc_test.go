package calc

import "testing"

func TestCostYen(t *testing.T) {
	cases := []struct {
		minutes int64
		hourly  int64
		want    int64
	}{
		{0, 5000, 0},
		{480, 0, 0},
		{1, 50, 1},
		{1, 100, 2},
		{29, 100, 48},
		{30, 100, 50},
		{31, 100, 52},
		{480, 5000, 40000},
		{4800, 5000, 400000},
	}
	for _, tc := range cases {
		if got := CostYen(tc.minutes, tc.hourly); got != tc.want {
			t.Fatalf("CostYen(%d, %d) = %d, want %d", tc.minutes, tc.hourly, got, tc.want)
		}
	}
}

func TestMonthlyAndAssignmentCost(t *testing.T) {
	if got := MonthlyCost("2026-04-01", "2026-04-30", 400000); got != 400000 {
		t.Fatalf("one month = %d", got)
	}
	if got := MonthlyCost("2026-04-15", "2026-06-10", 100000); got != 300000 {
		t.Fatalf("three months = %d", got)
	}
	if got := AssignmentPlannedCost("2026-04-01", "2026-04-30", 400000); got != 400000 {
		t.Fatalf("assignment planned = %d", got)
	}
	if got := AssignmentPlannedCost("2026-04-01", "2026-04-30", 0); got != 0 {
		t.Fatalf("no monthly = %d", got)
	}
	if got := AccruedMonthlyCost("2026-04-01", "2026-06-30", "2026-05-15", 100000); got != 200000 {
		t.Fatalf("accrued through May = %d", got)
	}
	if got := AccruedMonthlyCost("2026-04-01", "2026-06-30", "2026-03-31", 100000); got != 0 {
		t.Fatalf("before start = %d", got)
	}
	if got := AccruedMonthlyCost("2026-04-01", "2026-06-30", "2026-12-01", 100000); got != 300000 {
		t.Fatalf("after end = %d", got)
	}
}

func TestRemainingAndGross(t *testing.T) {
	if got := RemainingCost(400000, 40000); got != 360000 {
		t.Fatalf("remaining = %d", got)
	}
	if got := RemainingCost(100, 150); got != 0 {
		t.Fatalf("remaining over = %d", got)
	}
	if got := GrossMargin(1000000, 40000, 360000); got != 600000 {
		t.Fatalf("gross = %d", got)
	}
}

func TestParseHoursRoundTrip(t *testing.T) {
	for m := int64(0); m <= 180; m++ {
		got, err := ParseHours(FormatHours(m))
		if err != nil {
			t.Fatalf("parse %d (%s): %v", m, FormatHours(m), err)
		}
		if got != m {
			t.Fatalf("round trip %d via %s -> %d", m, FormatHours(m), got)
		}
	}
	if _, err := ParseHours(""); err != ErrEmpty {
		t.Fatalf("empty: %v", err)
	}
	if _, err := ParseHours("-1"); err != ErrNegative {
		t.Fatalf("negative: %v", err)
	}
	if _, err := ParseHours("abc"); err != ErrFormat {
		t.Fatalf("format: %v", err)
	}
}

func TestPersonDays(t *testing.T) {
	if got := FormatPersonDays(480, 480); got != "1.0" {
		t.Fatalf("1 day = %s", got)
	}
	if got := FormatPersonDays(240, 480); got != "0.5" {
		t.Fatalf("half = %s", got)
	}
	if got := FormatPersonDays(24, 480); got != "0.1" {
		t.Fatalf("round half up = %s", got)
	}
	// 旧・時間入力の 14 時間 = 840 分は表示 1.8。割当 1.8（864 分）との差分は表示上 0。
	if got := FormatPersonDays(840, 480); got != "1.8" {
		t.Fatalf("14h display = %s", got)
	}
	if got := SnapToPersonDays(840, 480); got != 864 {
		t.Fatalf("snap 840 = %d", got)
	}
	if got := PersonDayDiffTenths(840, 864, 480); got != 0 {
		t.Fatalf("display diff tenths = %d, want 0", got)
	}
	if got := FormatPersonDayTenths(PersonDayDiffTenths(840, 864, 480)); got != "0.0" {
		t.Fatalf("display diff = %s", got)
	}
	got, err := ParsePersonDays("1.5", 480)
	if err != nil || got != 720 {
		t.Fatalf("parse 1.5 = %d %v", got, err)
	}
	got, err = ParsePersonDays("0.1", 480)
	if err != nil || got != 48 {
		t.Fatalf("parse 0.1 = %d %v", got, err)
	}
	if _, err := ParsePersonDays("", 480); err != ErrEmpty {
		t.Fatalf("empty: %v", err)
	}
	if _, err := ParsePersonDays("0.15", 480); err != ErrFormat {
		t.Fatalf("not tenth: %v", err)
	}
}

func TestYen(t *testing.T) {
	if got := FormatComma(0); got != "0" {
		t.Fatal(got)
	}
	if got := FormatComma(400000); got != "400,000" {
		t.Fatal(got)
	}
	if got := FormatComma(1000000); got != "1,000,000" {
		t.Fatal(got)
	}
	if got := FormatYen(0); got != "¥0" {
		t.Fatal(got)
	}
	if got := FormatYen(400000); got != "¥400,000" {
		t.Fatal(got)
	}
	if got := FormatYen(1000000); got != "¥1,000,000" {
		t.Fatal(got)
	}
	if got := FormatYen(-360000); got != "-¥360,000" {
		t.Fatal(got)
	}
	n, err := ParseYen("￥1,000,000")
	if err != nil || n == nil || *n != 1000000 {
		t.Fatalf("parse yen: %v %v", n, err)
	}
	n, err = ParseYen("")
	if err != nil || n != nil {
		t.Fatalf("empty yen: %v %v", n, err)
	}
	if _, err := ParseYen("-1"); err != ErrFormat {
		t.Fatalf("bad yen: %v", err)
	}
}
