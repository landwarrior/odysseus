package web

import (
	"errors"
	"net/http"

	"odysseus/internal/store"
)

// rolePage は役割の追加と、登録済み役割の編集です。
type rolePage struct {
	base
	Roles []roleView
}

// roleView は一覧の1行です。Position は上から数えた位置です。
type roleView struct {
	store.Role
	Position int
	First    bool
	Last     bool
}

// processPage は工程の追加と、登録済み工程の編集です。
type processPage struct {
	base
	Processes []processView
}

// processView は一覧の1行です。Position は上から数えた位置です。
type processView struct {
	store.Process
	Position int
	First    bool
	Last     bool
}

// handleRoles は役割の追加欄と、上から順に並んだ登録済み役割を出します。
func (s *Server) handleRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := s.store.ListRoles(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	views := make([]roleView, len(roles))
	for i, role := range roles {
		views[i] = roleView{
			Role: role, Position: i + 1,
			First: i == 0, Last: i == len(roles)-1,
		}
	}
	s.render(w, http.StatusOK, "roles", rolePage{base: s.page(w, r, "役割"), Roles: views})
}

// handleRoleCreate はフォームの役割を、一覧の一番下に追加します。
func (s *Server) handleRoleCreate(w http.ResponseWriter, r *http.Request) {
	role, msg := readRole(r)
	if msg != "" {
		s.redirectWarn(w, r, "/roles", msg)
		return
	}
	existing, err := s.store.ListRoles(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	var maxOrder int64
	for _, item := range existing {
		if item.SortOrder > maxOrder {
			maxOrder = item.SortOrder
		}
	}
	role.SortOrder = maxOrder + 10
	if _, err := s.store.InsertRole(r.Context(), role); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.redirectWarn(w, r, "/roles", "同じ名前の役割が既にあります。")
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/roles", "登録しました。")
}

// handleRoleUpdate はパスの役割の名前と有効フラグを更新します。並び順は変えません。
func (s *Server) handleRoleUpdate(w http.ResponseWriter, r *http.Request) {
	role, msg := readRole(r)
	if msg != "" {
		s.redirectWarn(w, r, "/roles", msg)
		return
	}
	current, err := s.findRole(r, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	role.ID = current.ID
	role.SortOrder = current.SortOrder
	if err := s.store.UpdateRole(r.Context(), role); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.redirectWarn(w, r, "/roles", "同じ名前の役割が既にあります。")
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/roles", "登録しました。")
}

// handleRoleDelete はパスの役割を、参照が無ければ削除します。
func (s *Server) handleRoleDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteRole(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, store.ErrInUse) {
			s.redirectWarn(w, r, "/roles", "単価かアサインで使われているため削除できません。無効にしてください。")
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/roles", "削除しました。")
}

// handleRoleMove はパスの役割を、direction が up なら上へ、down なら下へ一つ動かします。
func (s *Server) handleRoleMove(w http.ResponseWriter, r *http.Request) {
	direction := formText(r, "direction")
	moved, err := s.store.MoveRole(r.Context(), r.PathValue("id"), direction)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "見つかりません。", http.StatusNotFound)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	if !moved {
		s.redirect(w, r, "/roles", "")
		return
	}
	s.redirect(w, r, "/roles", "並び順を変更しました。")
}

// readRole は r のフォームから、役割の名前と有効フラグを読みます。並び順は含みません。
func readRole(r *http.Request) (store.Role, string) {
	role := store.Role{Name: formText(r, "name"), Active: checkbox(r, "active")}
	if role.Name == "" {
		return role, "名前を入力してください。"
	}
	if msg := limitRunes(role.Name, "名前", 32); msg != "" {
		return role, msg
	}
	return role, ""
}

// findRole は id の役割を、現在の並び順のまま返します。
func (s *Server) findRole(r *http.Request, id string) (store.Role, error) {
	roles, err := s.store.ListRoles(r.Context())
	if err != nil {
		return store.Role{}, err
	}
	for _, role := range roles {
		if role.ID == id {
			return role, nil
		}
	}
	return store.Role{}, store.ErrNotFound
}

// handleProcesses は工程の追加欄と、上から順に並んだ登録済み工程を出します。
func (s *Server) handleProcesses(w http.ResponseWriter, r *http.Request) {
	processes, err := s.store.ListProcesses(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	views := make([]processView, len(processes))
	for i, process := range processes {
		views[i] = processView{
			Process: process, Position: i + 1,
			First: i == 0, Last: i == len(processes)-1,
		}
	}
	s.render(w, http.StatusOK, "processes", processPage{base: s.page(w, r, "工程"), Processes: views})
}

// handleProcessCreate はフォームの工程を、一覧の一番下に追加します。
func (s *Server) handleProcessCreate(w http.ResponseWriter, r *http.Request) {
	process, msg := readProcess(r)
	if msg != "" {
		s.redirectWarn(w, r, "/processes", msg)
		return
	}
	existing, err := s.store.ListProcesses(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	var maxOrder int64
	for _, item := range existing {
		if item.SortOrder > maxOrder {
			maxOrder = item.SortOrder
		}
	}
	process.SortOrder = maxOrder + 10
	if _, err := s.store.InsertProcess(r.Context(), process); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.redirectWarn(w, r, "/processes", "同じ名前の工程が既にあります。")
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/processes", "登録しました。")
}

// handleProcessUpdate はパスの工程の名前と有効フラグを更新します。並び順は変えません。
func (s *Server) handleProcessUpdate(w http.ResponseWriter, r *http.Request) {
	process, msg := readProcess(r)
	if msg != "" {
		s.redirectWarn(w, r, "/processes", msg)
		return
	}
	current, err := s.findProcess(r, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	process.ID = current.ID
	process.SortOrder = current.SortOrder
	if err := s.store.UpdateProcess(r.Context(), process); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.redirectWarn(w, r, "/processes", "同じ名前の工程が既にあります。")
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/processes", "登録しました。")
}

// handleProcessMove はパスの工程を、direction が up なら上へ、down なら下へ一つ動かします。
func (s *Server) handleProcessMove(w http.ResponseWriter, r *http.Request) {
	direction := formText(r, "direction")
	moved, err := s.store.MoveProcess(r.Context(), r.PathValue("id"), direction)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "見つかりません。", http.StatusNotFound)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	if !moved {
		s.redirect(w, r, "/processes", "")
		return
	}
	s.redirect(w, r, "/processes", "並び順を変更しました。")
}

// handleProcessDelete はパスの工程を、案件で使われていなければ削除します。
func (s *Server) handleProcessDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteProcess(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, store.ErrInUse) {
			s.redirectWarn(w, r, "/processes", "プロジェクトで使われているため削除できません。無効にしてください。")
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "見つかりません。", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	s.redirect(w, r, "/processes", "削除しました。")
}

// readProcess は r のフォームから、工程の名前と有効フラグを読みます。並び順は含みません。
func readProcess(r *http.Request) (store.Process, string) {
	process := store.Process{Name: formText(r, "name"), Active: checkbox(r, "active")}
	if process.Name == "" {
		return process, "名前を入力してください。"
	}
	if msg := limitRunes(process.Name, "名前", 32); msg != "" {
		return process, msg
	}
	return process, ""
}

// findProcess は id の工程を、現在の並び順のまま返します。
func (s *Server) findProcess(r *http.Request, id string) (store.Process, error) {
	processes, err := s.store.ListProcesses(r.Context())
	if err != nil {
		return store.Process{}, err
	}
	for _, process := range processes {
		if process.ID == id {
			return process, nil
		}
	}
	return store.Process{}, store.ErrNotFound
}
