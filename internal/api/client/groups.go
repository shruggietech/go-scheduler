package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/task"
)

// CreateGroup creates a group.
func (c *Client) CreateGroup(ctx context.Context, req server.GroupCreateRequest) (domain.Group, error) {
	var g domain.Group
	err := c.do(ctx, http.MethodPost, "/v1/groups", req, &g)
	return g, err
}

// ListGroups lists all groups (flat).
func (c *Client) ListGroups(ctx context.Context) ([]domain.Group, error) {
	var out struct {
		Groups []domain.Group `json:"groups"`
	}
	err := c.do(ctx, http.MethodGet, "/v1/groups", nil, &out)
	return out.Groups, err
}

// GroupTree returns the group hierarchy as a forest.
func (c *Client) GroupTree(ctx context.Context) ([]*task.TreeNode, error) {
	var out struct {
		Tree []*task.TreeNode `json:"tree"`
	}
	q := url.Values{"tree": {"true"}}
	err := c.do(ctx, http.MethodGet, "/v1/groups?"+q.Encode(), nil, &out)
	return out.Tree, err
}

// SetGroupEnabled enables or disables a group (cascades to its subtree).
func (c *Client) SetGroupEnabled(ctx context.Context, id string, enabled bool) error {
	action := "disable"
	if enabled {
		action = "enable"
	}
	return c.do(ctx, http.MethodPost, "/v1/groups/"+id+"/"+action, nil, nil)
}

// DeleteGroup deletes a group (its children cascade; tasks are ungrouped).
func (c *Client) DeleteGroup(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/groups/"+id, nil, nil)
}
