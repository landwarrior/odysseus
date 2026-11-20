package web

import (
	"errors"
	"net/http"

	"odysseus/internal/store"
)

// holidaysPage は祝日・休業日マスタです。
type holidaysPage struct {
	base
	Holidays []store.Holiday
}

// handleHolidays は祝日一覧を出します。
func (s *Server) handleHolidays(w http.ResponseWriter, r *http.Request) {
	holidays, err := s.store.ListHolidays(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, http.StatusOK, "holidays", holidaysPage{
		base: s.page(w, r, "祝日・休業日"), Holidays: holidays,
	})
}

// handleHolidayCreate は祝日を追加します。同じ日付なら名称を上書きします。
func (s *Server) handleHolidayCreate(w http.ResponseWriter, r *http.Request) {
	date, msg := normalizeDate(r.FormValue("date"), true)
	if msg != "" {
		s.redirectWarn(w, r, "/holidays", msg)
		return
	}
	name := formText(r, "name")
	if m := limitRunes(name, "名称", 64); m != "" {
		s.redirectWarn(w, r, "/holidays", m)
		return
	}
	if err := s.store.UpsertHoliday(r.Context(), date, name); err != nil {
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/holidays", "登録しました。")
}

// handleHolidayUpdate はパスの日付の名称を更新します。
func (s *Server) handleHolidayUpdate(w http.ResponseWriter, r *http.Request) {
	date := r.PathValue("date")
	name := formText(r, "name")
	if m := limitRunes(name, "名称", 64); m != "" {
		s.redirectWarn(w, r, "/holidays", m)
		return
	}
	if err := s.store.UpdateHolidayName(r.Context(), date, name); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/holidays", "更新しました。")
}

// handleHolidayDelete はパスの日付の祝日を削除します。
func (s *Server) handleHolidayDelete(w http.ResponseWriter, r *http.Request) {
	date := r.PathValue("date")
	if err := s.store.DeleteHoliday(r.Context(), date); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/holidays", "削除しました。")
}
