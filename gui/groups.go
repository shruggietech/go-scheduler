package gui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/task"
)

// buildGroupsTab renders the group hierarchy as a tree with enable/disable and
// add/delete (FR-019/FR-020). T055 (GUI group tree) is satisfied here.
func (a *App) buildGroupsTab() fyne.CanvasObject {
	byID := map[string]domain.Group{}
	childIDs := map[string][]string{}

	rebuild := func(groups []domain.Group) {
		byID = task.ByID(groups)
		childIDs = map[string][]string{}
		for _, g := range groups {
			childIDs[g.ParentID] = append(childIDs[g.ParentID], g.ID)
		}
	}

	tree := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID { return childIDs[string(id)] },
		func(id widget.TreeNodeID) bool { return len(childIDs[string(id)]) > 0 },
		func(bool) fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.TreeNodeID, _ bool, o fyne.CanvasObject) {
			g := byID[string(id)]
			label := g.Name
			if !g.Enabled {
				label += "  (disabled)"
			}
			o.(*widget.Label).SetText(label)
		},
	)
	selected := ""
	tree.OnSelected = func(id widget.TreeNodeID) { selected = string(id) }

	refresh := func() {
		rebuild(a.model.Snapshot().Groups)
		tree.Refresh()
	}
	a.registerRefresher(refresh)

	addBtn := widget.NewButtonWithIcon("New Group", theme.ContentAddIcon(), func() {
		nameEntry := widget.NewEntry()
		parent := selected // selected group becomes parent if set
		parentNote := "top-level"
		if g, ok := byID[parent]; ok {
			parentNote = "under " + g.Name
		}
		items := []*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Parent", widget.NewLabel(parentNote)),
		}
		dialog.NewForm("New Group", "Create", "Cancel", items, func(ok bool) {
			if !ok || nameEntry.Text == "" {
				return
			}
			a.run(func(ctx context.Context) error {
				_, err := a.backend.CreateGroup(ctx, server.GroupCreateRequest{Name: nameEntry.Text, ParentID: parentIfExists(byID, parent)})
				return err
			})
		}, a.win).Show()
	})
	toggleBtn := widget.NewButton("Enable/Disable", func() {
		if g, ok := byID[selected]; ok {
			a.run(func(ctx context.Context) error { return a.backend.SetGroupEnabled(ctx, g.ID, !g.Enabled) })
		}
	})
	delBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		if g, ok := byID[selected]; ok {
			dialog.ShowConfirm("Delete group", "Delete "+g.Name+" (children cascade)?", func(yes bool) {
				if yes {
					a.run(func(ctx context.Context) error { return a.backend.DeleteGroup(ctx, g.ID) })
				}
			}, a.win)
		}
	})
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() { a.refreshAll() })

	toolbar := container.NewHBox(addBtn, toggleBtn, delBtn, refreshBtn)
	return container.NewBorder(toolbar, nil, nil, nil, tree)
}

func parentIfExists(byID map[string]domain.Group, id string) string {
	if _, ok := byID[id]; ok {
		return id
	}
	return ""
}
