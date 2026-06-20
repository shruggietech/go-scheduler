package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

func TestGroups_CreateNestAndCascadeRoutes(t *testing.T) {
	s := newTestServer(t)

	// Create a 3-level hierarchy via the API.
	mk := func(name, parent string) domain.Group {
		rec := doJSON(t, s, http.MethodPost, "/v1/groups", GroupCreateRequest{Name: name, ParentID: parent})
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %s: status %d body %s", name, rec.Code, rec.Body.String())
		}
		var g domain.Group
		_ = json.Unmarshal(rec.Body.Bytes(), &g)
		return g
	}
	backups := mk("Backups", "")
	database := mk("Database", backups.ID)
	mk("Nightly", database.ID)

	// Non-existent parent → 400 validation.
	if rec := doJSON(t, s, http.MethodPost, "/v1/groups", GroupCreateRequest{Name: "bad", ParentID: "nope"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad parent: status %d", rec.Code)
	}

	// Cycle: reparent Backups under its descendant Database → 400.
	parent := database.ID
	rec := doJSON(t, s, http.MethodPatch, "/v1/groups/"+backups.ID, GroupUpdateRequest{Parent: &parent})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("cycle reparent: status %d body %s", rec.Code, rec.Body.String())
	}
	var e APIError
	_ = json.Unmarshal(rec.Body.Bytes(), &e)
	if e.Error.Field != "parent_id" {
		t.Fatalf("expected parent_id field error, got %+v", e)
	}

	// Tree view returns the forest.
	tree := doJSON(t, s, http.MethodGet, "/v1/groups?tree=true", nil)
	if tree.Code != http.StatusOK {
		t.Fatalf("tree: status %d", tree.Code)
	}

	// Enable/disable routes work.
	if rec := doJSON(t, s, http.MethodPost, "/v1/groups/"+backups.ID+"/disable", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("disable: status %d", rec.Code)
	}
	got := doJSON(t, s, http.MethodGet, "/v1/groups/"+backups.ID, nil)
	var g domain.Group
	_ = json.Unmarshal(got.Body.Bytes(), &g)
	if g.Enabled {
		t.Fatal("group should be disabled")
	}

	// Delete cascades children (FK), leaving no groups.
	if rec := doJSON(t, s, http.MethodDelete, "/v1/groups/"+backups.ID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status %d", rec.Code)
	}
	list := doJSON(t, s, http.MethodGet, "/v1/groups", nil)
	var resp struct {
		Groups []domain.Group `json:"groups"`
	}
	_ = json.Unmarshal(list.Body.Bytes(), &resp)
	if len(resp.Groups) != 0 {
		t.Fatalf("expected cascade delete to remove all groups, got %d", len(resp.Groups))
	}
}
