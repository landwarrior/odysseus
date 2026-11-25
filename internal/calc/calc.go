// Package calc は予実の金額と工数の計算です。データベースには依存しません。
package calc

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrEmpty    = errors.New("empty")    // 入力が空
	ErrFormat   = errors.New("format")   // 数値として読めない
	ErrNegative = errors.New("negative") // 負の数
)

// CostYen は minutes（分）と hourlyYen（円/時）から金額を求めます。
// 計算は minutes × hourlyYen ÷ 60 で、円未満は四捨五入です。
// minutes が 0 以下、または hourlyYen が 0 未満のときは 0 です。0 円単価は実績原価 0 になります。
func CostYen(minutes, hourlyYen int64) int64 {
	if minutes <= 0 || hourlyYen < 0 {
		return 0
	}
	if hourlyYen == 0 {
		return 0
	}
	num := minutes * hourlyYen
	q := num / 60
	if num%60 >= 30 {
		q++
	}
	return q
}

// MonthlyCost は startDate〜endDate（YYYY-MM-DD）に掛かる暦月数 × monthlyYen です。
// 開始月と終了月を含みます。日付が不正、終了が開始より前、単価が 0 未満なら 0 です。0 円単価は 0 です。
func MonthlyCost(startDate, endDate string, monthlyYen int64) int64 {
	if monthlyYen < 0 {
		return 0
	}
	if monthlyYen == 0 {
		return 0
	}
	start, err := time.ParseInLocation("2006-01-02", startDate, time.UTC)
	if err != nil {
		return 0
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, time.UTC)
	if err != nil || end.Before(start) {
		return 0
	}
	months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1
	return int64(months) * monthlyYen
}

// AssignmentPlannedCost は互換用です。準委任の予定原価は案件横断の按分（MonthlyPersonCosts）で求め、アサインには 0 を保存します。
func AssignmentPlannedCost(startDate, endDate string, monthlyYen int64) int64 {
	return MonthlyCost(startDate, endDate, monthlyYen)
}

// AccruedMonthlyCost は startDate〜endDate のうち、asOfDate までに掛かる暦月分の月間原価です。
// 按分前の月額合計用。asOfDate が開始より前なら 0、終了より後なら期間全体と同じです。
func AccruedMonthlyCost(startDate, endDate, asOfDate string, monthlyYen int64) int64 {
	if monthlyYen < 0 {
		return 0
	}
	if monthlyYen == 0 {
		return 0
	}
	start, err := time.ParseInLocation("2006-01-02", startDate, time.UTC)
	if err != nil {
		return 0
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, time.UTC)
	if err != nil || end.Before(start) {
		return 0
	}
	asOf, err := time.ParseInLocation("2006-01-02", asOfDate, time.UTC)
	if err != nil || asOf.Before(start) {
		return 0
	}
	through := end
	if asOf.Before(end) {
		through = asOf
	}
	return MonthlyCost(startDate, through.Format("2006-01-02"), monthlyYen)
}

// MonthlyStake は準委任アサイン1件分の按分入力です。
// HasRate が偽のときはそのアサインの月間単価が無く、該当月の実績原価は未確定になります。
type MonthlyStake struct {
	PersonID   string
	ProjectID  string
	StartDate  string
	EndDate    string
	MonthlyYen int64
	HasRate    bool
}

// PersonProjectCost は準委任按分後の、人材×案件の予定・実績原価です。
type PersonProjectCost struct {
	PlannedYen int64
	ActualYen  int64
}

// AllocateByWeights は totalYen を weights（キー→重み）の比で分けます。
// 重みが 0 以下のキーは無視します。端数は最大剰余法で、合計が totalYen と一致します。
// totalYen が 0 以下、または有効な重みが無ければ空の map です。
func AllocateByWeights(totalYen int64, weights map[string]int64) map[string]int64 {
	out := map[string]int64{}
	if totalYen <= 0 {
		return out
	}
	type part struct {
		id       string
		weight   int64
		base     int64
		fraction int64
	}
	var parts []part
	var sumWeight int64
	for id, w := range weights {
		if w <= 0 {
			continue
		}
		parts = append(parts, part{id: id, weight: w})
		sumWeight += w
	}
	if sumWeight <= 0 || len(parts) == 0 {
		return out
	}
	var assigned int64
	for i := range parts {
		num := totalYen * parts[i].weight
		parts[i].base = num / sumWeight
		parts[i].fraction = num % sumWeight
		assigned += parts[i].base
	}
	remain := totalYen - assigned
	sort.SliceStable(parts, func(i, j int) bool {
		if parts[i].fraction != parts[j].fraction {
			return parts[i].fraction > parts[j].fraction
		}
		return parts[i].id < parts[j].id
	})
	for i := int64(0); i < remain && int(i) < len(parts); i++ {
		parts[i].base++
	}
	for _, p := range parts {
		out[p.id] = p.base
	}
	return out
}

// CalendarMonths は startDate〜endDate（YYYY-MM-DD）に掛かる暦月を YYYY-MM で返します。
// 日付が不正、または終了が開始より前なら nil です。
func CalendarMonths(startDate, endDate string) []string {
	start, err := time.ParseInLocation("2006-01-02", startDate, time.UTC)
	if err != nil {
		return nil
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, time.UTC)
	if err != nil || end.Before(start) {
		return nil
	}
	cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	var out []string
	for !cur.After(last) {
		out = append(out, cur.Format("2006-01"))
		cur = cur.AddDate(0, 1, 0)
	}
	return out
}

// monthOverlaps は startDate〜endDate が ym（YYYY-MM）と重なるとき true です。
func monthOverlaps(startDate, endDate, ym string) bool {
	start, err := time.ParseInLocation("2006-01-02", startDate, time.UTC)
	if err != nil {
		return false
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, time.UTC)
	if err != nil || end.Before(start) {
		return false
	}
	monthStart, err := time.ParseInLocation("2006-01", ym, time.UTC)
	if err != nil {
		return false
	}
	monthEnd := monthStart.AddDate(0, 1, -1)
	return !end.Before(monthStart) && !start.After(monthEnd)
}

// MonthlyPersonCosts は準委任の予定・実績原価を、人×月で1回だけ計上し受注金額比で案件へ按分します。
// orders は案件 ID → 受注金額（nil は母数から外し、その案件への按分は 0）。
// asOfDate は実績の計上上限日（YYYY-MM-DD）。
// rateMissing が真のときは、どれかの月で月間単価が取れず実績原価は未確定です。
func MonthlyPersonCosts(stakes []MonthlyStake, orders map[string]*int64, asOfDate string) (byPersonProject map[string]PersonProjectCost, rateMissing bool) {
	byPersonProject = map[string]PersonProjectCost{}
	if len(stakes) == 0 {
		return byPersonProject, false
	}
	people := map[string]struct{}{}
	var rangeStart, rangeEnd string
	for _, s := range stakes {
		people[s.PersonID] = struct{}{}
		if rangeStart == "" || (s.StartDate != "" && s.StartDate < rangeStart) {
			rangeStart = s.StartDate
		}
		if rangeEnd == "" || (s.EndDate != "" && s.EndDate > rangeEnd) {
			rangeEnd = s.EndDate
		}
	}
	months := CalendarMonths(rangeStart, rangeEnd)
	asOfMonths := map[string]bool{}
	if asOfDate != "" {
		if _, err := time.ParseInLocation("2006-01-02", asOfDate, time.UTC); err == nil {
			for _, ym := range CalendarMonths(rangeStart, asOfDate) {
				asOfMonths[ym] = true
			}
		}
	}
	for personID := range people {
		for _, ym := range months {
			var monthlyYen int64
			var sawStake, sawRate bool
			weights := map[string]int64{}
			for _, s := range stakes {
				if s.PersonID != personID || !monthOverlaps(s.StartDate, s.EndDate, ym) {
					continue
				}
				sawStake = true
				if s.HasRate {
					sawRate = true
					if s.MonthlyYen > monthlyYen {
						monthlyYen = s.MonthlyYen
					}
				}
				if amt, ok := orders[s.ProjectID]; ok && amt != nil && *amt > 0 {
					weights[s.ProjectID] = *amt
				}
			}
			if !sawStake {
				continue
			}
			if !sawRate {
				rateMissing = true
				continue
			}
			shares := AllocateByWeights(monthlyYen, weights)
			for projectID, yen := range shares {
				key := personID + "\x00" + projectID
				cur := byPersonProject[key]
				cur.PlannedYen += yen
				if asOfMonths[ym] {
					cur.ActualYen += yen
				}
				byPersonProject[key] = cur
			}
		}
	}
	return byPersonProject, rateMissing
}

// PersonProjectKey は MonthlyPersonCosts の map キーです。
func PersonProjectKey(personID, projectID string) string {
	return personID + "\x00" + projectID
}

// RemainingCost は planned（予定原価）から actual（実績原価）を引いた残りです。
// actual が planned 以上なら 0 を返します。
func RemainingCost(planned, actual int64) int64 {
	if actual >= planned {
		return 0
	}
	return planned - actual
}

// GrossMargin は orderAmount（受注金額）から actualCost（実績原価）と remainingCost（残予定原価）を引いた粗利です。
func GrossMargin(orderAmount, actualCost, remainingCost int64) int64 {
	return orderAmount - actualCost - remainingCost
}

// ParseHours は raw（画面の時間。0.1 刻み）を分にします。
// raw が空なら ErrEmpty、数値でなければ ErrFormat、負なら ErrNegative です。
// 0.1 時間未満の端数は受け付けません。
func ParseHours(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, ErrEmpty
	}
	if strings.ContainsAny(s, ",，") {
		return 0, ErrFormat
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, ErrFormat
	}
	if f < 0 {
		return 0, ErrNegative
	}
	tenths := math.Round(f * 10)
	if math.Abs(f*10-tenths) > 1e-9 {
		return 0, ErrFormat
	}
	// 0.1 時間 = 6 分。
	return int64(tenths) * 6, nil
}

// FormatHours は minutes（分。負でもよい）を時間の文字列にします（0.1 刻み）。
// その文字列を ParseHours に渡すと、同じ 0.1 時間刻みの分に戻ります。
func FormatHours(minutes int64) string {
	if minutes == 0 {
		return "0"
	}
	sign := ""
	if minutes < 0 {
		sign = "-"
		minutes = -minutes
	}
	tenths := (minutes*10 + 30) / 60
	if tenths%10 == 0 {
		return sign + strconv.FormatInt(tenths/10, 10)
	}
	return fmt.Sprintf("%s%d.%d", sign, tenths/10, tenths%10)
}

// PersonDayTenths は minutes を 0.1 人日の個数にし、四捨五入します。
// standardMinutes は 1 人日の分数です。0 以下のときは 480 分として計算します。
// 負の minutes なら負の刻み数を返します。
func PersonDayTenths(minutes, standardMinutes int64) int64 {
	if standardMinutes <= 0 {
		standardMinutes = 480
	}
	sign := int64(1)
	if minutes < 0 {
		sign = -1
		minutes = -minutes
	}
	return sign * ((minutes*10 + standardMinutes/2) / standardMinutes)
}

// FormatPersonDayTenths は 0.1 人日の個数を "n.n" 形式の文字列にします。
func FormatPersonDayTenths(tenths int64) string {
	sign := ""
	if tenths < 0 {
		sign = "-"
		tenths = -tenths
	}
	return fmt.Sprintf("%s%d.%d", sign, tenths/10, tenths%10)
}

// FormatPersonDays は minutes を人日の文字列にし、小数第1位で四捨五入します。
// standardMinutes は 1 人日の分数です。0 以下のときは 480 分として計算します。
func FormatPersonDays(minutes, standardMinutes int64) string {
	return FormatPersonDayTenths(PersonDayTenths(minutes, standardMinutes))
}

// SnapToPersonDays は minutes を、画面の人日表示と同じ 0.1 人日刻みの分数に揃えます。
// 表示が 1.8 なら保存も 1.8 人日ぶんの分になり、割当と差分がずれません。
func SnapToPersonDays(minutes, standardMinutes int64) int64 {
	if standardMinutes <= 0 {
		standardMinutes = 480
	}
	tenths := PersonDayTenths(minutes, standardMinutes)
	if tenths < 0 {
		return -((-tenths) * standardMinutes / 10)
	}
	return tenths * standardMinutes / 10
}

// PersonDayDiffTenths は見積と割当の差分を、画面の人日（0.1 刻み）同士の引き算で求めます。
// 分のまま引いてから四捨五入すると、同じ表示値同士でも -0.1 になることがあるためです。
func PersonDayDiffTenths(estimateMinutes, allocatedMinutes, standardMinutes int64) int64 {
	return PersonDayTenths(estimateMinutes, standardMinutes) - PersonDayTenths(allocatedMinutes, standardMinutes)
}

// ParsePersonDays は raw（画面の人日。0.1 刻み）を分にします。
// standardMinutes は 1 人日の分数です。0 以下のときは 480 分として計算します。
// raw が空なら ErrEmpty、数値でなければ ErrFormat、負なら ErrNegative です。
// 0.1 人日未満の端数は受け付けません。
func ParsePersonDays(raw string, standardMinutes int64) (int64, error) {
	if standardMinutes <= 0 {
		standardMinutes = 480
	}
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, ErrEmpty
	}
	if strings.ContainsAny(s, ",，") {
		return 0, ErrFormat
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, ErrFormat
	}
	if f < 0 {
		return 0, ErrNegative
	}
	tenths := math.Round(f * 10)
	if math.Abs(f*10-tenths) > 1e-9 {
		return 0, ErrFormat
	}
	return int64(tenths) * standardMinutes / 10, nil
}

// ParseYen は raw（画面の金額。カンマや円記号を含んでよい）を円にします。
// raw が空なら nil です。読めないときや負のときは ErrFormat です。
func ParseYen(raw string) (*int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, nil
	}
	s = strings.NewReplacer(",", "", "，", "", "¥", "", "￥", "", "円", "", " ", "", "\u3000", "").Replace(s)
	if s == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 || n > 9999999999 {
		return nil, ErrFormat
	}
	return &n, nil
}

// FormatComma は n（整数。負でもよい）を 1,234 の形にします。金額の入力欄用です。
func FormatComma(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	lead := len(s) % 3
	if lead == 0 {
		lead = 3
	}
	var b strings.Builder
	b.WriteString(sign)
	b.WriteString(s[:lead])
	for i := lead; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// FormatYen は n（円。負でもよい）を ¥1,234 の形にします。
func FormatYen(n int64) string {
	if n < 0 {
		return "-¥" + FormatComma(-n)
	}
	return "¥" + FormatComma(n)
}
