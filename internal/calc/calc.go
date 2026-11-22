// Package calc は予実の金額と工数の計算です。データベースには依存しません。
package calc

import (
	"errors"
	"fmt"
	"math"
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

// AssignmentPlannedCost はアサインの予定原価です。月間単価 × 期間の暦月数です（準委任）。
// 月間単価が無いときは 0 です（請負の予定原価は工程予算側で見ます）。
func AssignmentPlannedCost(startDate, endDate string, monthlyYen int64) int64 {
	return MonthlyCost(startDate, endDate, monthlyYen)
}

// AccruedMonthlyCost は startDate〜endDate のうち、asOfDate までに掛かる暦月分の月間原価です（準委任の実績原価）。
// asOfDate が開始より前なら 0、終了より後なら期間全体と同じです。
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

// ParseHours は raw（画面の時間。1.5 のような小数可）を分にします。
// raw が空なら ErrEmpty、数値でなければ ErrFormat、負なら ErrNegative です。
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
	return int64(math.Round(f * 60)), nil
}

// FormatHours は minutes（分。負でもよい）を時間の文字列にします。
// その文字列を ParseHours に渡すと、同じ分に戻ります。
func FormatHours(minutes int64) string {
	if minutes == 0 {
		return "0"
	}
	sign := ""
	if minutes < 0 {
		sign = "-"
		minutes = -minutes
	}
	whole := minutes / 60
	rem := minutes % 60
	if rem == 0 {
		return sign + strconv.FormatInt(whole, 10)
	}
	hundredths := (rem*100 + 30) / 60
	if hundredths >= 100 {
		whole++
		hundredths -= 100
	}
	if hundredths == 0 {
		return fmt.Sprintf("%s%d", sign, whole)
	}
	if hundredths%10 == 0 {
		return fmt.Sprintf("%s%d.%d", sign, whole, hundredths/10)
	}
	return fmt.Sprintf("%s%d.%02d", sign, whole, hundredths)
}

// FormatPersonDays は minutes を人日の文字列にし、小数第1位で四捨五入します。
// standardMinutes は 1 人日の分数です。0 以下のときは 480 分として計算します。
func FormatPersonDays(minutes, standardMinutes int64) string {
	if standardMinutes <= 0 {
		standardMinutes = 480
	}
	sign := ""
	if minutes < 0 {
		sign = "-"
		minutes = -minutes
	}
	tenths := (minutes*10 + standardMinutes/2) / standardMinutes
	return fmt.Sprintf("%s%d.%d", sign, tenths/10, tenths%10)
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
