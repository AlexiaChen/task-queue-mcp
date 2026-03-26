package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"task-queue-mcp/internal/apiclient"
	"task-queue-mcp/internal/queue"
)

// ANSI colour codes for status badges.
const (
	ansiReset  = "\033[0m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiGreen  = "\033[32m"
	ansiRed    = "\033[31m"
)

func colorStatus(status queue.TaskStatus) string {
	switch status {
	case queue.StatusPending:
		return ansiYellow + "pending" + ansiReset
	case queue.StatusDoing:
		return ansiCyan + "doing" + ansiReset
	case queue.StatusFinished:
		return ansiGreen + "finished" + ansiReset
	default:
		return string(status)
	}
}

// newRootCmd builds the root cobra command tree.
func newRootCmd() *cobra.Command {
	var serverURL string

	root := &cobra.Command{
		Use:   "issue-kanban-cli",
		Short: "CLI for the issue-kanban-mcp server",
		Long:  "A command-line interface for managing issue kanbans via the issue-kanban-mcp REST API.",
	}
	root.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:9292", "server base URL")

	clientFn := func(cmd *cobra.Command) *apiclient.Client {
		url, _ := cmd.Root().PersistentFlags().GetString("server")
		return apiclient.New(url)
	}

	root.AddCommand(newProjectsCmd(clientFn))
	root.AddCommand(newIssuesCmd(clientFn))
	return root
}

// ---- projects ----

func newProjectsCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Manage projects",
	}
	cmd.AddCommand(newProjectsListCmd(clientFn))
	cmd.AddCommand(newProjectsCreateCmd(clientFn))
	cmd.AddCommand(newProjectsDeleteCmd(clientFn))
	cmd.AddCommand(newProjectsStatsCmd(clientFn))
	return cmd
}

func newProjectsListCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			queues, err := clientFn(cmd).ListProjects(context.Background())
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tPENDING\tDOING\tFINISHED\tTOTAL")
			for _, q := range queues {
				fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d\t%d\n",
					q.ID, q.Name, q.Description,
					q.Stats.Pending, q.Stats.Doing, q.Stats.Finished, q.Stats.Total)
			}
			return w.Flush()
		},
	}
}

func newProjectsCreateCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var name, desc string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			q, err := clientFn(cmd).CreateProject(context.Background(), queue.CreateQueueInput{
				Name:        name,
				Description: desc,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created project %d: %s\n", q.ID, q.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "project name (required)")
	cmd.Flags().StringVar(&desc, "desc", "", "project description")
	return cmd
}

func newProjectsDeleteCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "Delete project %d? [y/N]: ", id)
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
						fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
						return nil
					}
				}
			}
			if err := clientFn(cmd).DeleteProject(context.Background(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted project %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func newProjectsStatsCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "stats <id>",
		Short: "Show stats for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			stats, err := clientFn(cmd).GetProjectStats(context.Background(), id)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Project %d stats:\n", id)
			fmt.Fprintf(out, "  Total:    %d\n", stats.Total)
			fmt.Fprintf(out, "  Pending:  %d\n", stats.Pending)
			fmt.Fprintf(out, "  Doing:    %d\n", stats.Doing)
			fmt.Fprintf(out, "  Finished: %d\n", stats.Finished)
			return nil
		},
	}
}

// ---- issues ----

func newIssuesCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issues",
		Short: "Manage issues",
	}
	cmd.AddCommand(newIssuesListCmd(clientFn))
	cmd.AddCommand(newIssuesGetCmd(clientFn))
	cmd.AddCommand(newIssuesCreateCmd(clientFn))
	cmd.AddCommand(newIssuesEditCmd(clientFn))
	cmd.AddCommand(newIssuesDeleteCmd(clientFn))
	cmd.AddCommand(newIssuesPrioritizeCmd(clientFn))
	return cmd
}

func newIssuesListCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var statusFilter string
	cmd := &cobra.Command{
		Use:   "list <project-id>",
		Short: "List issues in a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queueID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid project-id: %w", err)
			}
			tasks, err := clientFn(cmd).ListIssues(context.Background(), queueID, statusFilter)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tPRIORITY\tTITLE\tDESCRIPTION")
			for _, t := range tasks {
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					t.ID, colorStatus(t.Status), t.Priority.String(), t.Title, t.Description)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&statusFilter, "status", "", "filter by status: pending, doing, finished")
	return cmd
}

func newIssuesGetCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get issue details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			t, err := clientFn(cmd).GetIssue(context.Background(), id)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "ID:          %d\n", t.ID)
			fmt.Fprintf(out, "Project ID:  %d\n", t.ProjectID)
			fmt.Fprintf(out, "Title:       %s\n", t.Title)
			fmt.Fprintf(out, "Description: %s\n", t.Description)
			fmt.Fprintf(out, "Status:      %s\n", colorStatus(t.Status))
			fmt.Fprintf(out, "Priority:    %s\n", t.Priority.String())
			fmt.Fprintf(out, "Position:    %d\n", t.Position)
			fmt.Fprintf(out, "Created:     %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Fprintf(out, "Updated:     %s\n", t.UpdatedAt.Format("2006-01-02 15:04:05"))
			if t.StartedAt != nil {
				fmt.Fprintf(out, "Started:     %s\n", t.StartedAt.Format("2006-01-02 15:04:05"))
			}
			if t.FinishedAt != nil {
				fmt.Fprintf(out, "Finished:    %s\n", t.FinishedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}
}

func newIssuesCreateCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var title, desc, priorityStr string
	cmd := &cobra.Command{
		Use:   "create <project-id>",
		Short: "Create an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queueID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid project-id: %w", err)
			}
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			prio, err := queue.ParsePriority(priorityStr)
			if err != nil {
				return fmt.Errorf("invalid priority: %w", err)
			}
			t, err := clientFn(cmd).CreateIssue(context.Background(), queue.CreateTaskInput{
				ProjectID:   queueID,
				Title:       title,
				Description: desc,
				Priority:    prio,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created issue %d: %s\n", t.ID, t.Title)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "issue title (required)")
	cmd.Flags().StringVar(&desc, "desc", "", "issue description")
	cmd.Flags().StringVar(&priorityStr, "priority", "low", "issue priority: low, medium, or high")
	return cmd
}

func newIssuesEditCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var title, desc, priorityStr string
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a pending issue's title, description, or priority",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			var titlePtr, descPtr *string
			var priorityPtr *queue.Priority
			if cmd.Flags().Changed("title") {
				titlePtr = &title
			}
			if cmd.Flags().Changed("desc") {
				descPtr = &desc
			}
			if cmd.Flags().Changed("priority") {
				prio, err := queue.ParsePriority(priorityStr)
				if err != nil {
					return fmt.Errorf("invalid priority: %w", err)
				}
				priorityPtr = &prio
			}
			if titlePtr == nil && descPtr == nil && priorityPtr == nil {
				return fmt.Errorf("provide at least one of --title, --desc, --priority")
			}
			t, err := clientFn(cmd).EditIssue(context.Background(), id, titlePtr, descPtr, priorityPtr)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Issue %d updated: %s\n", t.ID, t.Title)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&desc, "desc", "", "new description")
	cmd.Flags().StringVar(&priorityStr, "priority", "low", "new priority: low, medium, or high")
	return cmd
}

func newIssuesDeleteCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "Delete issue %d? [y/N]: ", id)
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
						fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
						return nil
					}
				}
			}
			if err := clientFn(cmd).DeleteIssue(context.Background(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted issue %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func newIssuesPrioritizeCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "prioritize <id>",
		Short: "Move an issue to the front of its project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			t, err := clientFn(cmd).PrioritizeIssue(context.Background(), id)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Issue %d moved to front: %s\n", t.ID, t.Title)
			return nil
		},
	}
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
