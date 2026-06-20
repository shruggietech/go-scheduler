package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/store"
)

// GroupCreateRequest is the body for POST /v1/groups.
type GroupCreateRequest struct {
	Name     string `json:"name"`
	ParentID string `json:"parent_id,omitempty"`
}

// GroupUpdateRequest is the body for PATCH /v1/groups/{id}. Provide Name to
// rename and/or Parent to reparent (nil leaves the parent unchanged).
type GroupUpdateRequest struct {
	Name   string  `json:"name,omitempty"`
	Parent *string `json:"parent_id,omitempty"`
}

func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req GroupCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "body", "invalid JSON")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, CodeValidation, "name", "name is required")
		return
	}
	g := &domain.Group{Name: req.Name, ParentID: req.ParentID, Enabled: true}
	if err := s.store.CreateGroup(g); err != nil {
		s.groupErr(w, err)
		return
	}
	s.reload()
	writeJSON(w, http.StatusCreated, g)
}

func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("tree") == "true" {
		forest, err := s.store.GroupTree()
		if err != nil {
			s.internal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tree": forest})
		return
	}
	groups, err := s.store.ListGroups()
	if err != nil {
		s.internal(w, err)
		return
	}
	if groups == nil {
		groups = []domain.Group{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": groups})
}

func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	g, err := s.store.GetGroup(r.PathValue("id"))
	if err != nil {
		s.notFoundOr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req GroupUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "body", "invalid JSON")
		return
	}
	if _, err := s.store.GetGroup(id); err != nil {
		s.notFoundOr(w, err)
		return
	}
	if req.Name != "" {
		if err := s.store.RenameGroup(id, req.Name); err != nil {
			s.internal(w, err)
			return
		}
	}
	if req.Parent != nil {
		if err := s.store.SetGroupParent(id, *req.Parent); err != nil {
			s.groupErr(w, err)
			return
		}
	}
	s.reload()
	g, err := s.store.GetGroup(id)
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteGroup(r.PathValue("id")); err != nil {
		s.notFoundOr(w, err)
		return
	}
	s.reload()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleEnableGroup(w http.ResponseWriter, r *http.Request) {
	s.setGroupEnabled(w, r, true)
}

func (s *Server) handleDisableGroup(w http.ResponseWriter, r *http.Request) {
	s.setGroupEnabled(w, r, false)
}

func (s *Server) setGroupEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	if err := s.store.SetGroupEnabled(r.PathValue("id"), enabled); err != nil {
		s.notFoundOr(w, err)
		return
	}
	s.reload() // cascade affects which tasks are eligible to run
	w.WriteHeader(http.StatusNoContent)
}

// groupErr maps group-specific errors (cycle, bad parent) to validation.
func (s *Server) groupErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrCycle) {
		writeError(w, http.StatusBadRequest, CodeValidation, "parent_id", "would create a group cycle")
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "", "not found")
		return
	}
	// A non-existent parent is a validation problem, not a 500.
	writeError(w, http.StatusBadRequest, CodeValidation, "parent_id", err.Error())
}
