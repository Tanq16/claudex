package taskCmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanq16/claude-usage/internal/model"
	"github.com/tanq16/claude-usage/internal/store"
	u "github.com/tanq16/claude-usage/internal/utils"
)

var addFlags struct {
	size string
}

var TaskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks for capacity planning",
}

var addCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new task",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, err := store.Open(store.DefaultPath())
		if err != nil {
			u.PrintFatal("Failed to open task store", err)
		}
		defer s.Close()

		size := model.TaskSize(strings.ToUpper(addFlags.size))
		if _, ok := model.SizeEstimates[size]; !ok {
			u.PrintFatal(fmt.Sprintf("Invalid size %q (use S, M, L, XL)", addFlags.size), nil)
		}

		task, err := s.AddTask(model.Task{
			Name:      strings.Join(args, " "),
			Size:      size,
			CreatedAt: time.Now(),
		})
		if err != nil {
			u.PrintFatal("Failed to add task", err)
		}

		est := model.SizeEstimates[size]
		u.PrintSuccess(fmt.Sprintf("Added task %s: %s [%s] (~%d turns, ~%.0f%% capacity)",
			task.ID, task.Name, task.Size, est.Messages, est.Percent5H))
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Run: func(cmd *cobra.Command, args []string) {
		s, err := store.Open(store.DefaultPath())
		if err != nil {
			u.PrintFatal("Failed to open task store", err)
		}
		defer s.Close()

		tasks, err := s.ListTasks()
		if err != nil {
			u.PrintFatal("Failed to list tasks", err)
		}

		if len(tasks) == 0 {
			u.PrintInfo("No tasks. Add one with: claude-usage task add \"name\" --size M")
			return
		}

		rows := make([][]string, len(tasks))
		for i, t := range tasks {
			status := "pending"
			if t.Done {
				status = "done"
			}
			rows[i] = []string{t.ID, string(t.Size), status, t.Name}
		}
		u.PrintTable([]string{"ID", "Size", "Status", "Name"}, rows)
	},
}

var doneCmd = &cobra.Command{
	Use:   "done [id]",
	Short: "Mark a task as done",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, err := store.Open(store.DefaultPath())
		if err != nil {
			u.PrintFatal("Failed to open task store", err)
		}
		defer s.Close()

		err = s.UpdateTask(args[0], func(t *model.Task) {
			t.Done = true
		})
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to mark task %s as done", args[0]), err)
		}
		u.PrintSuccess(fmt.Sprintf("Marked task %s as done", args[0]))
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove a task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, err := store.Open(store.DefaultPath())
		if err != nil {
			u.PrintFatal("Failed to open task store", err)
		}
		defer s.Close()

		if err := s.DeleteTask(args[0]); err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to remove task %s", args[0]), err)
		}
		u.PrintSuccess(fmt.Sprintf("Removed task %s", args[0]))
	},
}

func init() {
	addCmd.Flags().StringVarP(&addFlags.size, "size", "s", "M", "Task size (S, M, L, XL)")
	TaskCmd.AddCommand(addCmd, listCmd, doneCmd, removeCmd)
}
