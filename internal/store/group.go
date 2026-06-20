package store

import (
	"fmt"
	"time"

	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/task"
)

// ErrCycle is returned when an operation would create a group cycle.
var ErrCycle = fmt.Errorf("store: operation would create a group cycle")

// groupsByID loads all groups indexed by ID.
func (s *Store) groupsByID() (map[string]domain.Group, []domain.Group, error) {
	groups, err := s.ListGroups()
	if err != nil {
		return nil, nil, err
	}
	return task.ByID(groups), groups, nil
}

// GroupChainEnabled reports whether groupID and all of its ancestors are
// enabled. An empty groupID (ungrouped) is enabled.
func (s *Store) GroupChainEnabled(groupID string) (bool, error) {
	if groupID == "" {
		return true, nil
	}
	byID, _, err := s.groupsByID()
	if err != nil {
		return false, err
	}
	return task.ChainEnabled(groupID, byID), nil
}

// ValidateParent ensures parentID exists and that assigning it to groupID would
// not create a cycle. groupID may be empty for a not-yet-created group.
func (s *Store) ValidateParent(groupID, parentID string) error {
	if parentID == "" {
		return nil
	}
	byID, _, err := s.groupsByID()
	if err != nil {
		return err
	}
	if _, ok := byID[parentID]; !ok {
		return fmt.Errorf("store: parent group %q does not exist", parentID)
	}
	if groupID != "" && task.WouldCycle(groupID, parentID, byID) {
		return ErrCycle
	}
	return nil
}

// SetGroupParent reparents a group, rejecting cycles.
func (s *Store) SetGroupParent(groupID, parentID string) error {
	if err := s.ValidateParent(groupID, parentID); err != nil {
		return err
	}
	res, err := s.db.Exec(`UPDATE groups SET parent_id=?, updated_at=? WHERE id=?`,
		nullStr(parentID), fmtTime(time.Now().UTC()), groupID)
	return affected(res, err, "set group parent")
}

// RenameGroup updates a group's name.
func (s *Store) RenameGroup(id, name string) error {
	res, err := s.db.Exec(`UPDATE groups SET name=?, updated_at=? WHERE id=?`,
		name, fmtTime(time.Now().UTC()), id)
	return affected(res, err, "rename group")
}

// GroupTree returns the group hierarchy as a forest.
func (s *Store) GroupTree() ([]*task.TreeNode, error) {
	_, groups, err := s.groupsByID()
	if err != nil {
		return nil, err
	}
	return task.BuildForest(groups), nil
}
