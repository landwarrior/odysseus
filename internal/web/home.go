package web

import (
	"net/http"

	"odysseus/internal/report"
	"odysseus/internal/store"
)

// homePage はホームです。管理者には Rows、メンバーには Assigned を出します。
type homePage struct {
	base
	Rows     []homeRow
	Assigned []store.Project
}

// homeRow は進行中案件1件と、その予実です。
type homeRow struct {
	Project store.Project
	Report  report.Report
}

// handleHome は管理者に進行中案件の予実、メンバーには自分の実績入力先を出します。
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	me := personFrom(r)
	page := homePage{base: s.page(w, r, "ホーム")}
	if me.IsAdmin {
		bundles, err := s.store.LoadBundlesByStatus(r.Context(), "active")
		if err != nil {
			s.fail(w, err)
			return
		}
		for _, bundle := range bundles {
			page.Rows = append(page.Rows, homeRow{Project: bundle.Project, Report: report.Build(bundle)})
		}
	} else {
		assigned, err := s.store.ListAssignedProjects(r.Context(), me.ID)
		if err != nil {
			s.fail(w, err)
			return
		}
		page.Assigned = assigned
	}
	s.render(w, http.StatusOK, "home", page)
}
