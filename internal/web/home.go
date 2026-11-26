package web

import (
	"net/http"
	"time"

	"odysseus/internal/report"
	"odysseus/internal/store"
)

// homePage はホームです。
// Rows は管理者（全進行中）または責任者（担当の進行中）向けの予実一覧です。
// Assigned はメンバー／責任者向けの、自分のアサイン案件です。
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

// handleHome は管理者に進行中案件すべての予実、責任者に担当案件の予実、
// それ以外には自分の実績入力先を出します。責任者にはアサイン案件も併記します。
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	me := personFrom(r)
	page := homePage{base: s.page(w, r, "ホーム")}
	if me.IsAdmin {
		bundles, err := s.store.LoadBundlesByStatus(r.Context(), "active")
		if err != nil {
			s.fail(w, err)
			return
		}
		if err := s.appendHomeRows(r, &page, bundles); err != nil {
			s.fail(w, err)
			return
		}
	} else {
		managedIDs, err := s.store.ListManagedProjectIDs(r.Context(), me.ID)
		if err != nil {
			s.fail(w, err)
			return
		}
		if len(managedIDs) > 0 {
			bundles, err := s.store.LoadBundlesByIDs(r.Context(), managedIDs)
			if err != nil {
				s.fail(w, err)
				return
			}
			var active []store.Bundle
			for _, b := range bundles {
				if b.Project.Status == "active" {
					active = append(active, b)
				}
			}
			if err := s.appendHomeRows(r, &page, active); err != nil {
				s.fail(w, err)
				return
			}
		}
		assigned, err := s.store.ListAssignedProjects(r.Context(), me.ID)
		if err != nil {
			s.fail(w, err)
			return
		}
		page.Assigned = assigned
	}
	s.render(w, http.StatusOK, "home", page)
}

// appendHomeRows は bundles の予実行を page.Rows に足します。
func (s *Server) appendHomeRows(r *http.Request, page *homePage, bundles []store.Bundle) error {
	peers, err := s.store.ExpandMonthlyPeers(r.Context(), bundles)
	if err != nil {
		return err
	}
	for _, bundle := range bundles {
		rep := report.BuildWithContext(bundle, peers, time.Time{})
		page.Rows = append(page.Rows, homeRow{
			Project: bundle.Project, Report: rep, OverBudget: reportOverBudget(rep),
		})
	}
	return nil
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
