package web

import (
	"net/http"
	"time"

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
// OverBudget が真のときは、どれかの工程で予算を超えています。
type homeRow struct {
	Project    store.Project
	Report     report.Report
	OverBudget bool
}

// handleHome は管理者に進行中案件すべての予実、メンバーには自分の実績入力先を出します。
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	me := personFrom(r)
	page := homePage{base: s.page(w, r, "ホーム")}
	if me.IsAdmin {
		bundles, err := s.store.LoadBundlesByStatus(r.Context(), "active")
		if err != nil {
			s.fail(w, err)
			return
		}
		peers, err := s.store.ExpandMonthlyPeers(r.Context(), bundles)
		if err != nil {
			s.fail(w, err)
			return
		}
		for _, bundle := range bundles {
			rep := report.BuildWithContext(bundle, peers, time.Time{})
			page.Rows = append(page.Rows, homeRow{
				Project: bundle.Project, Report: rep, OverBudget: reportOverBudget(rep),
			})
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

// reportOverBudget は rep の工程のどれかが予算超過なら真です。
func reportOverBudget(rep report.Report) bool {
	for _, process := range rep.Processes {
		if process.OverBudget {
			return true
		}
	}
	return false
}
