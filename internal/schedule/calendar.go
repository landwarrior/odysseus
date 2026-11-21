// Package schedule は旬単位の表示期間と営業日の計算です。データベースは知りません。
package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DecadeLabels は旬番号から表示名です。
var DecadeLabels = map[int]string{
	1: "上旬",
	2: "中旬",
	3: "下旬",
}

// PhaseColors は工程マスタで選べる背景色です。
var PhaseColors = []string{"cyan", "orange", "green", "lavender", "rose", "yellow"}

// PhaseColorLabels は背景色の表示名です。
var PhaseColorLabels = map[string]string{
	"cyan":     "水色",
	"orange":   "橙色",
	"green":    "緑色",
	"lavender": "藤色",
	"rose":     "桃色",
	"yellow":   "黄色",
}

// NormalizeColor は value を許容色に揃えます。不正なら水色です。
func NormalizeColor(value string) string {
	for _, c := range PhaseColors {
		if value == c {
			return value
		}
	}
	return "cyan"
}

// ColorForSort は sortOrder（工程マスタの並び）から背景色名を返します。色未設定時の仮です。
func ColorForSort(sortOrder int64) string {
	if sortOrder <= 0 {
		return PhaseColors[0]
	}
	return PhaseColors[int((sortOrder-1)%int64(len(PhaseColors)))]
}

// ParseYM は YYYY-MM を年と月に分けます。
func ParseYM(ym string) (int, int, error) {
	parts := strings.Split(ym, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("月の形式が不正です")
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("月の形式が不正です")
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("月の形式が不正です")
	}
	if month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("月の形式が不正です")
	}
	return year, month, nil
}

// FormatYM は年と月から YYYY-MM を作ります。
func FormatYM(year, month int) string {
	return fmt.Sprintf("%04d-%02d", year, month)
}

// NormalizeYM は value（YYYY-MM または YYYY/MM）を YYYY-MM に揃えます。
func NormalizeYM(value string) (string, error) {
	text := strings.TrimSpace(strings.ReplaceAll(value, "/", "-"))
	if text == "" {
		return "", fmt.Errorf("月が未入力です")
	}
	year, month, err := ParseYM(text)
	if err != nil {
		return "", err
	}
	return FormatYM(year, month), nil
}

// DecadeOf は day が属する旬（1=上旬、2=中旬、3=下旬）を返します。
func DecadeOf(day time.Time) int {
	if day.Day() <= 10 {
		return 1
	}
	if day.Day() <= 20 {
		return 2
	}
	return 3
}

// CurrentPeriod は today の年月と旬です。today が零値なら現在時刻を使います。
func CurrentPeriod(today time.Time) (string, int) {
	if today.IsZero() {
		today = time.Now()
	}
	return FormatYM(today.Year(), int(today.Month())), DecadeOf(today)
}

// IterMonths は fromYM から toYM までの YYYY-MM を順に返します。
func IterMonths(fromYM, toYM string) ([]string, error) {
	year, month, err := ParseYM(fromYM)
	if err != nil {
		return nil, err
	}
	endYear, endMonth, err := ParseYM(toYM)
	if err != nil {
		return nil, err
	}
	if year > endYear || (year == endYear && month > endMonth) {
		return nil, fmt.Errorf("表示期間の開始が終了より後です")
	}
	var result []string
	for year < endYear || (year == endYear && month <= endMonth) {
		result = append(result, FormatYM(year, month))
		month++
		if month > 12 {
			month = 1
			year++
		}
		if len(result) > 36 {
			return nil, fmt.Errorf("表示期間は 36 か月以内にしてください")
		}
	}
	return result, nil
}

// AddMonths は ym に delta か月足した YYYY-MM を返します。
func AddMonths(ym string, delta int) (string, error) {
	year, month, err := ParseYM(ym)
	if err != nil {
		return "", err
	}
	t := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, delta, 0)
	return FormatYM(t.Year(), int(t.Month())), nil
}

// DecadeDayRange は year/month の decade 旬の開始日と終了日（その月の日）を返します。
func DecadeDayRange(year, month, decade int) (int, int, error) {
	lastDay := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	switch decade {
	case 1:
		return 1, min(10, lastDay), nil
	case 2:
		return 11, min(20, lastDay), nil
	case 3:
		return 21, lastDay, nil
	default:
		return 0, 0, fmt.Errorf("旬が不正です")
	}
}

// OverlapsDecade は startDate〜endDate（YYYY-MM-DD、どちらか空可）が ym の decade 旬と重なるかです。
// 両方空なら false です。開始だけならその日以降、終了だけならその日以前とみなします。
func OverlapsDecade(ym string, decade int, startDate, endDate string) bool {
	startDate = strings.TrimSpace(startDate)
	endDate = strings.TrimSpace(endDate)
	if startDate == "" && endDate == "" {
		return false
	}
	year, month, err := ParseYM(ym)
	if err != nil {
		return false
	}
	startDay, endDay, err := DecadeDayRange(year, month, decade)
	if err != nil {
		return false
	}
	decadeStart := time.Date(year, time.Month(month), startDay, 0, 0, 0, 0, time.UTC)
	decadeEnd := time.Date(year, time.Month(month), endDay, 0, 0, 0, 0, time.UTC)
	rangeStart := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	if startDate != "" {
		t, err := time.ParseInLocation("2006-01-02", startDate, time.UTC)
		if err != nil {
			return false
		}
		rangeStart = t
	}
	if endDate != "" {
		t, err := time.ParseInLocation("2006-01-02", endDate, time.UTC)
		if err != nil {
			return false
		}
		rangeEnd = t
	}
	if rangeStart.After(rangeEnd) {
		return false
	}
	return !rangeStart.After(decadeEnd) && !rangeEnd.Before(decadeStart)
}

// BusinessDays は ym の decade 旬にある平日数です。holidays は YYYY-MM-DD の祝日です。
func BusinessDays(ym string, decade int, holidays []string) (int, error) {
	year, month, err := ParseYM(ym)
	if err != nil {
		return 0, err
	}
	startDay, endDay, err := DecadeDayRange(year, month, decade)
	if err != nil {
		return 0, err
	}
	holidaySet := make(map[string]struct{}, len(holidays))
	for _, h := range holidays {
		holidaySet[h] = struct{}{}
	}
	count := 0
	for day := startDay; day <= endDay; day++ {
		d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		if _, ok := holidaySet[d.Format("2006-01-02")]; ok {
			continue
		}
		count++
	}
	return count, nil
}
