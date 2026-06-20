package task

import (
	"sort"
	"testing"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

// Hierarchy: root -> child -> leaf, plus a sibling "other".
func sampleGroups() []domain.Group {
	return []domain.Group{
		{ID: "root", Enabled: true},
		{ID: "child", ParentID: "root", Enabled: true},
		{ID: "leaf", ParentID: "child", Enabled: true},
		{ID: "other", Enabled: true},
	}
}

func TestChainEnabled(t *testing.T) {
	groups := sampleGroups()
	byID := ByID(groups)

	if !ChainEnabled("leaf", byID) {
		t.Fatal("leaf should be enabled when whole chain is enabled")
	}
	if ChainEnabled("", byID) != true {
		t.Fatal("ungrouped (empty) should be enabled")
	}

	// Disable the middle group → leaf becomes ineffective, sibling unaffected.
	byID["child"] = domain.Group{ID: "child", ParentID: "root", Enabled: false}
	if ChainEnabled("leaf", byID) {
		t.Fatal("leaf should be disabled when an ancestor is disabled")
	}
	if !ChainEnabled("other", byID) {
		t.Fatal("sibling subtree should remain enabled")
	}

	// Disable the root → everything beneath is ineffective.
	byID["child"] = domain.Group{ID: "child", ParentID: "root", Enabled: true}
	byID["root"] = domain.Group{ID: "root", Enabled: false}
	if ChainEnabled("leaf", byID) {
		t.Fatal("disabling root should cascade to leaf")
	}
}

func TestWouldCycle(t *testing.T) {
	byID := ByID(sampleGroups())
	if !WouldCycle("root", "root", byID) {
		t.Fatal("a group cannot be its own parent")
	}
	if !WouldCycle("root", "leaf", byID) {
		t.Fatal("reparenting root under its own descendant must be a cycle")
	}
	if WouldCycle("other", "root", byID) {
		t.Fatal("moving an unrelated group under root is fine")
	}
	if WouldCycle("leaf", "other", byID) {
		t.Fatal("moving leaf under a sibling subtree is fine")
	}
}

func TestDescendantIDs(t *testing.T) {
	got := DescendantIDs("root", sampleGroups())
	sort.Strings(got)
	want := []string{"child", "leaf"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("descendants = %v, want %v", got, want)
	}
	if d := DescendantIDs("leaf", sampleGroups()); len(d) != 0 {
		t.Fatalf("leaf has no descendants, got %v", d)
	}
}

func TestBuildForest(t *testing.T) {
	roots := BuildForest(sampleGroups())
	if len(roots) != 2 { // root and other
		t.Fatalf("want 2 roots, got %d", len(roots))
	}
	var rootNode *TreeNode
	for _, r := range roots {
		if r.Group.ID == "root" {
			rootNode = r
		}
	}
	if rootNode == nil || len(rootNode.Children) != 1 || rootNode.Children[0].Group.ID != "child" {
		t.Fatalf("root subtree malformed: %+v", rootNode)
	}
	if len(rootNode.Children[0].Children) != 1 || rootNode.Children[0].Children[0].Group.ID != "leaf" {
		t.Fatal("leaf not nested under child")
	}
}
