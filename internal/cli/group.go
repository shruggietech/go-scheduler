package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/task"
)

func newGroupCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "group", Short: "Organize tasks into nested groups"}
	cmd.AddCommand(groupAdd(), groupList(), groupEnable(), groupDisable(), groupRm())
	return cmd
}

func groupAdd() *cobra.Command {
	var parent string
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a group (optionally under --parent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			ctx, cancel := reqCtx()
			defer cancel()
			g, err := newClient().CreateGroup(ctx, server.GroupCreateRequest{Name: a[0], ParentID: parent})
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(g)
			}
			fmt.Fprintf(os.Stdout, "created group %s (%s)\n", g.ID, g.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&parent, "parent", "", "parent group ID")
	return cmd
}

func groupList() *cobra.Command {
	var tree bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List groups",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, cancel := reqCtx()
			defer cancel()
			cl := newClient()
			if tree {
				forest, err := cl.GroupTree(ctx)
				if err != nil {
					return err
				}
				if jsonOut {
					return printJSON(forest)
				}
				for _, n := range forest {
					printTree(n, 0)
				}
				return nil
			}
			groups, err := cl.ListGroups(ctx)
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(groups)
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tNAME\tPARENT\tENABLED")
			for _, g := range groups {
				parent := g.ParentID
				if parent == "" {
					parent = "-"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%t\n", g.ID, g.Name, parent, g.Enabled)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().BoolVar(&tree, "tree", false, "show as a hierarchy")
	return cmd
}

func printTree(n *task.TreeNode, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	state := ""
	if !n.Group.Enabled {
		state = " (disabled)"
	}
	fmt.Fprintf(os.Stdout, "%s- %s [%s]%s\n", indent, n.Group.Name, n.Group.ID, state)
	for _, c := range n.Children {
		printTree(c, depth+1)
	}
}

func groupEnable() *cobra.Command {
	return &cobra.Command{
		Use: "enable <id>", Short: "Enable a group (and its subtree)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error { return groupToggle(a[0], true) },
	}
}

func groupDisable() *cobra.Command {
	return &cobra.Command{
		Use: "disable <id>", Short: "Disable a group (cascades to its subtree)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error { return groupToggle(a[0], false) },
	}
}

func groupToggle(id string, enabled bool) error {
	ctx, cancel := reqCtx()
	defer cancel()
	if err := newClient().SetGroupEnabled(ctx, id, enabled); err != nil {
		return err
	}
	state := "disabled"
	if enabled {
		state = "enabled"
	}
	fmt.Fprintf(os.Stdout, "group %s %s\n", id, state)
	return nil
}

func groupRm() *cobra.Command {
	return &cobra.Command{
		Use: "rm <id>", Short: "Delete a group (children cascade; tasks are ungrouped)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			ctx, cancel := reqCtx()
			defer cancel()
			if err := newClient().DeleteGroup(ctx, a[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "deleted group %s\n", a[0])
			return nil
		},
	}
}
