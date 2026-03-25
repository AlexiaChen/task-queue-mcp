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
		Use:   "task-queue-cli",
		Short: "CLI for the task-queue-mcp server",
		Long:  "A command-line interface for managing task queues via the task-queue-mcp REST API.",
	}
	root.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:9292", "server base URL")

	clientFn := func(cmd *cobra.Command) *apiclient.Client {
		url, _ := cmd.Root().PersistentFlags().GetString("server")
		return apiclient.New(url)
	}

	root.AddCommand(newQueuesCmd(clientFn))
	root.AddCommand(newTasksCmd(clientFn))
	return root
}

// ---- queues ----

func newQueuesCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queues",
		Short: "Manage queues",
	}
	cmd.AddCommand(newQueuesListCmd(clientFn))
	cmd.AddCommand(newQueuesCreateCmd(clientFn))
	cmd.AddCommand(newQueuesDeleteCmd(clientFn))
	cmd.AddCommand(newQueuesStatsCmd(clientFn))
	return cmd
}

func newQueuesListCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all queues",
		RunE: func(cmd *cobra.Command, args []string) error {
			queues, err := clientFn(cmd).ListQueues(context.Background())
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

func newQueuesCreateCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var name, desc string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			q, err := clientFn(cmd).CreateQueue(context.Background(), queue.CreateQueueInput{
				Name:        name,
				Description: desc,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created queue %d: %s\n", q.ID, q.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "queue name (required)")
	cmd.Flags().StringVar(&desc, "desc", "", "queue description")
	return cmd
}

func newQueuesDeleteCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "Delete queue %d? [y/N]: ", id)
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
						fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
						return nil
					}
				}
			}
			if err := clientFn(cmd).DeleteQueue(context.Background(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted queue %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func newQueuesStatsCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "stats <id>",
		Short: "Show stats for a queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			stats, err := clientFn(cmd).GetQueueStats(context.Background(), id)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Queue %d stats:\n", id)
			fmt.Fprintf(out, "  Total:    %d\n", stats.Total)
			fmt.Fprintf(out, "  Pending:  %d\n", stats.Pending)
			fmt.Fprintf(out, "  Doing:    %d\n", stats.Doing)
			fmt.Fprintf(out, "  Finished: %d\n", stats.Finished)
			return nil
		},
	}
}

// ---- tasks ----

func newTasksCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Manage tasks",
	}
	cmd.AddCommand(newTasksListCmd(clientFn))
	cmd.AddCommand(newTasksGetCmd(clientFn))
	cmd.AddCommand(newTasksCreateCmd(clientFn))
	cmd.AddCommand(newTasksStartCmd(clientFn))
	cmd.AddCommand(newTasksFinishCmd(clientFn))
	cmd.AddCommand(newTasksResetCmd(clientFn))
	cmd.AddCommand(newTasksDeleteCmd(clientFn))
	cmd.AddCommand(newTasksPrioritizeCmd(clientFn))
	return cmd
}

func newTasksListCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var statusFilter string
	cmd := &cobra.Command{
		Use:   "list <queue-id>",
		Short: "List tasks in a queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queueID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid queue-id: %w", err)
			}
			tasks, err := clientFn(cmd).ListTasks(context.Background(), queueID, statusFilter)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tPRIORITY\tTITLE\tDESCRIPTION")
			for _, t := range tasks {
				fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\n",
					t.ID, colorStatus(t.Status), t.Priority, t.Title, t.Description)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&statusFilter, "status", "", "filter by status: pending, doing, finished")
	return cmd
}

func newTasksGetCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get task details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			t, err := clientFn(cmd).GetTask(context.Background(), id)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "ID:          %d\n", t.ID)
			fmt.Fprintf(out, "Queue ID:    %d\n", t.QueueID)
			fmt.Fprintf(out, "Title:       %s\n", t.Title)
			fmt.Fprintf(out, "Description: %s\n", t.Description)
			fmt.Fprintf(out, "Status:      %s\n", colorStatus(t.Status))
			fmt.Fprintf(out, "Priority:    %d\n", t.Priority)
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

func newTasksCreateCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var title, desc string
	var priority int
	cmd := &cobra.Command{
		Use:   "create <queue-id>",
		Short: "Create a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queueID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid queue-id: %w", err)
			}
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			t, err := clientFn(cmd).CreateTask(context.Background(), queue.CreateTaskInput{
				QueueID:     queueID,
				Title:       title,
				Description: desc,
				Priority:    priority,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created task %d: %s\n", t.ID, t.Title)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "task title (required)")
	cmd.Flags().StringVar(&desc, "desc", "", "task description")
	cmd.Flags().IntVar(&priority, "priority", 0, "task priority")
	return cmd
}

func newTasksStartCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "start <id>",
		Short: "Start a task (pending → doing)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			t, err := clientFn(cmd).StartTask(context.Background(), id)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Task %d started: %s\n", t.ID, t.Title)
			return nil
		},
	}
}

func newTasksFinishCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "finish <id>",
		Short: "Finish a task (doing → finished)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			t, err := clientFn(cmd).FinishTask(context.Background(), id)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Task %d finished: %s\n", t.ID, t.Title)
			return nil
		},
	}
}

func newTasksResetCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "reset <id>",
		Short: "Reset a task to pending",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			t, err := clientFn(cmd).UpdateTask(context.Background(), id, queue.StatusPending)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Task %d reset to pending: %s\n", t.ID, t.Title)
			return nil
		},
	}
}

func newTasksDeleteCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "Delete task %d? [y/N]: ", id)
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
						fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
						return nil
					}
				}
			}
			if err := clientFn(cmd).DeleteTask(context.Background(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted task %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func newTasksPrioritizeCmd(clientFn func(*cobra.Command) *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "prioritize <id>",
		Short: "Move a task to the front of its queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			t, err := clientFn(cmd).PrioritizeTask(context.Background(), id)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Task %d moved to front: %s\n", t.ID, t.Title)
			return nil
		},
	}
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
