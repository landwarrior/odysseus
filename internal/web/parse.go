package web

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"odysseus/internal/calc"
)

// formText は r のフォームから key の値を読み、前後の空白を除きます。
func formText(r *http.Request, key string) string {
	return strings.TrimSpace(r.FormValue(key))
}

// validStatus は status が planning、active、closed のどれかなら真です。
func validStatus(status string) bool {
	return status == "planning" || status == "active" || status == "closed"
}

// normalizeDate は raw を YYYY-MM-DD に整えます。スラッシュ区切りも受けます。
// required が真で raw が空なら、エラーメッセージを返します。正常時のメッセージは空です。
func normalizeDate(raw string, required bool) (string, string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		if required {
			return "", "日付を入力してください。"
		}
		return "", ""
	}
	s = strings.ReplaceAll(s, "/", "-")
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", "日付は YYYY-MM-DD で入力してください。"
	}
	return t.Format("2006-01-02"), ""
}

// normalizeMonth は raw が YYYY-MM なら、その文字列と true を返します。
func normalizeMonth(raw string) (string, bool) {
	t, err := time.Parse("2006-01", strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	return t.Format("2006-01"), true
}

// dateOrder は end が start より前なら、startLabel と endLabel を使ったエラーメッセージを返します。
// どちらかが空、または順序が正しければ空文字です。
func dateOrder(start, end, startLabel, endLabel string) string {
	if start != "" && end != "" && end < start {
		return endLabel + "は" + startLabel + "以降にしてください。"
	}
	return ""
}

// limitRunes は s が n 文字を超えていれば、label を含むエラーメッセージを返します。
func limitRunes(s, label string, n int) string {
	if utf8.RuneCountInString(s) > n {
		return label + "が長すぎます。"
	}
	return ""
}

// parseOptionalHours は raw を分にします。空なら 0 と空メッセージです。
func parseOptionalHours(raw string) (int64, string) {
	minutes, err := calc.ParseHours(raw)
	if errors.Is(err, calc.ErrEmpty) {
		return 0, ""
	}
	if err != nil {
		return 0, "時間は 0 以上の数で入力してください。"
	}
	return minutes, ""
}

// parseOptionalPersonDays は raw（人日。0.1 刻み）を分にします。空なら 0 と空メッセージです。
// standardMinutes は 1 人日の分数です。
func parseOptionalPersonDays(raw string, standardMinutes int64) (int64, string) {
	minutes, err := calc.ParsePersonDays(raw, standardMinutes)
	if errors.Is(err, calc.ErrEmpty) {
		return 0, ""
	}
	if err != nil {
		return 0, "見積工数は 0.1 人日単位で入力してください。"
	}
	return minutes, ""
}

// parseRequiredHours は raw を、0 より大きい分にします。空や 0 はエラーです。
func parseRequiredHours(raw string) (int64, string) {
	minutes, err := calc.ParseHours(raw)
	if err != nil || minutes <= 0 {
		return 0, "予定時間を入力してください。"
	}
	if minutes > 100000*60 {
		return 0, "予定時間が大きすぎます。"
	}
	return minutes, ""
}

// passwordIssue は password の長さを見ます。required が偽で空なら問題なしです。
// 短い、または bcrypt が扱える 72 バイトを超えるときはメッセージを返します。
func passwordIssue(password string, required bool) string {
	if password == "" {
		if required {
			return "パスワードを入力してください。"
		}
		return ""
	}
	if utf8.RuneCountInString(password) < 8 {
		return "パスワードは 8 文字以上にしてください。"
	}
	if len(password) > 72 {
		return "パスワードが長すぎます。"
	}
	return ""
}

// checkbox は r のフォームで key のチェックが付いていれば真です。外していると項目自体が届きません。
func checkbox(r *http.Request, key string) bool {
	return r.FormValue(key) == "1"
}
