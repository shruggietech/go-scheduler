// Package task holds tree logic for task groups: cascading enabled-state
// resolution, descendant enumeration, and cycle detection. The functions are
// pure (operating over an in-memory view of groups) so they are testable
// without a database; the store loads groups and delegates here.
package task

import "github.com/shruggietech/go-scheduler/internal/domain"

// ByID indexes groups by their ID.
func ByID(groups []domain.Group) map[string]domain.Group {
	m := make(map[string]domain.Group, len(groups))
	for _, g := range groups {
		m[g.ID] = g
	}
	return m
}

// ChainEnabled reports whether groupID and all of its ancestors are enabled.
// A disabled group anywhere in the chain makes the whole subtree ineffective.
// An empty groupID (ungrouped) is always enabled. Unknown ancestors are treated
// as enabled (orphan tolerance); a cycle is treated as disabled (defensive).
func ChainEnabled(groupID string, byID map[string]domain.Group) bool {
	seen := map[string]bool{}
	for id := groupID; id != ""; {
		if seen[id] {
			return false
		}
		seen[id] = true
		g, ok := byID[id]
		if !ok {
			return true
		}
		if !g.Enabled {
			return false
		}
		id = g.ParentID
	}
	return true
}

// WouldCycle reports whether making newParent the parent of groupID would create
// a cycle — i.e. newParent is groupID itself or a descendant of groupID.
func WouldCycle(groupID, newParent string, byID map[string]domain.Group) bool {
	if newParent == "" {
		return false
	}
	seen := map[string]bool{}
	for id := newParent; id != ""; {
		if id == groupID {
			return true
		}
		if seen[id] {
			return false
		}
		seen[id] = true
		g, ok := byID[id]
		if !ok {
			return false
		}
		id = g.ParentID
	}
	return false
}

// DescendantIDs returns the IDs of every group beneath groupID (any depth).
func DescendantIDs(groupID string, groups []domain.Group) []string {
	children := map[string][]string{}
	for _, g := range groups {
		children[g.ParentID] = append(children[g.ParentID], g.ID)
	}
	var out []string
	var walk func(id string)
	walk = func(id string) {
		for _, c := range children[id] {
			out = append(out, c)
			walk(c)
		}
	}
	walk(groupID)
	return out
}

// TreeNode is a group with its children, for rendering hierarchies.
type TreeNode struct {
	Group    domain.Group `json:"group"`
	Children []*TreeNode  `json:"children,omitempty"`
}

// BuildForest assembles top-level groups (and their descendants) into a forest.
func BuildForest(groups []domain.Group) []*TreeNode {
	nodes := make(map[string]*TreeNode, len(groups))
	for _, g := range groups {
		nodes[g.ID] = &TreeNode{Group: g}
	}
	var roots []*TreeNode
	for _, g := range groups {
		n := nodes[g.ID]
		if parent, ok := nodes[g.ParentID]; ok && g.ParentID != "" {
			parent.Children = append(parent.Children, n)
		} else {
			roots = append(roots, n)
		}
	}
	return roots
}
