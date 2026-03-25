package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"task-queue-mcp/internal/apiclient"
	"task-queue-mcp/internal/queue"
)

type viewState int

const (
	viewLoading viewState = iota
	viewQueueList
	viewTaskList
	viewCreateQueue
	viewCreateTask
	viewConfirmDelete
)

type (
	queuesLoadedMsg struct{ queues []apiclient.QueueWithStats }
	tasksLoadedMsg  struct{ tasks []queue.Task }
	actionDoneMsg   struct{}
	errMsg          struct{ err error }
	tickMsg         time.Time
)

type pendingDeleteInfo struct {
	kind string
	id   int64
	name string
}

// App is the main bubbletea model for the TUI.
type App struct {
	client        *apiclient.Client
	state         viewState
	queues        []apiclient.QueueWithStats
	queueIdx      int
	tasks         []queue.Task
	taskIdx       int
	inputs        []textinput.Model
	focusIdx      int
	formMode      string
	pendingDelete pendingDeleteInfo
	statusMsg     string
	isError       bool
	width         int
	height        int
	currentQueue  *apiclient.QueueWithStats
}

// NewApp creates a new App model.
func NewApp(client *apiclient.Client) App {
	return App{
		client: client,
		state:  viewLoading,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (a App) loadQueuesCmd() tea.Cmd {
	return func() tea.Msg {
		queues, err := a.client.ListQueues(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return queuesLoadedMsg{queues}
	}
}

func (a App) loadTasksCmd() tea.Cmd {
	if a.currentQueue == nil {
		return nil
	}
	queueID := a.currentQueue.ID
	return func() tea.Msg {
		tasks, err := a.client.ListTasks(context.Background(), queueID, "")
		if err != nil {
			return errMsg{err}
		}
		return tasksLoadedMsg{tasks}
	}
}

// Init kicks off queue loading and the auto-refresh ticker.
func (a App) Init() tea.Cmd {
	return tea.Batch(a.loadQueuesCmd(), tickCmd())
}

// Update handles messages and returns the updated model and next command.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case queuesLoadedMsg:
		a.queues = msg.queues
		if a.state == viewLoading {
			a.state = viewQueueList
		}
		// Refresh the currentQueue pointer into the new slice.
		if a.currentQueue != nil {
			for i := range a.queues {
				if a.queues[i].ID == a.currentQueue.ID {
					a.currentQueue = &a.queues[i]
					break
				}
			}
		}
		return a, nil

	case tasksLoadedMsg:
		a.tasks = msg.tasks
		return a, nil

	case actionDoneMsg:
		a.isError = false
		switch a.state {
		case viewQueueList:
			return a, a.loadQueuesCmd()
		case viewTaskList:
			return a, tea.Batch(a.loadTasksCmd(), a.loadQueuesCmd())
		}
		return a, nil

	case errMsg:
		a.statusMsg = msg.err.Error()
		a.isError = true
		return a, nil

	case tickMsg:
		var cmd tea.Cmd
		switch a.state {
		case viewQueueList:
			cmd = a.loadQueuesCmd()
		case viewTaskList:
			cmd = a.loadTasksCmd()
		}
		return a, tea.Batch(cmd, tickCmd())

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	// Forward all other messages (e.g. cursor blink) to active text inputs.
	if a.state == viewCreateQueue || a.state == viewCreateTask {
		var cmds []tea.Cmd
		for i := range a.inputs {
			var cmd tea.Cmd
			a.inputs[i], cmd = a.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)
	}

	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.state {
	case viewQueueList:
		return a.handleQueueListKey(msg)
	case viewTaskList:
		return a.handleTaskListKey(msg)
	case viewCreateQueue, viewCreateTask:
		return a.handleFormKey(msg)
	case viewConfirmDelete:
		return a.handleConfirmKey(msg)
	}
	return a, nil
}

func (a App) handleQueueListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if a.queueIdx < len(a.queues)-1 {
			a.queueIdx++
		}
	case "k", "up":
		if a.queueIdx > 0 {
			a.queueIdx--
		}
	case "enter":
		if len(a.queues) > 0 && a.queueIdx < len(a.queues) {
			q := a.queues[a.queueIdx]
			a.currentQueue = &q
			a.state = viewTaskList
			a.taskIdx = 0
			a.statusMsg = ""
			return a, a.loadTasksCmd()
		}
	case "n":
		a.formMode = "queue"
		a.state = viewCreateQueue
		a.inputs = makeQueueInputs()
		a.focusIdx = 0
		a.statusMsg = ""
		cmd := a.inputs[0].Focus()
		return a, cmd
	case "d":
		if len(a.queues) > 0 && a.queueIdx < len(a.queues) {
			q := a.queues[a.queueIdx]
			a.pendingDelete = pendingDeleteInfo{kind: "queue", id: q.ID, name: q.Name}
			a.state = viewConfirmDelete
		}
	case "R":
		a.statusMsg = ""
		return a, a.loadQueuesCmd()
	case "q", "ctrl+c":
		return a, tea.Quit
	}
	return a, nil
}

func (a App) handleTaskListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if a.taskIdx < len(a.tasks)-1 {
			a.taskIdx++
		}
	case "k", "up":
		if a.taskIdx > 0 {
			a.taskIdx--
		}
	case "n":
		a.formMode = "task"
		a.state = viewCreateTask
		a.inputs = makeTaskInputs()
		a.focusIdx = 0
		a.statusMsg = ""
		cmd := a.inputs[0].Focus()
		return a, cmd
	case "d":
		if len(a.tasks) > 0 && a.taskIdx < len(a.tasks) {
			t := a.tasks[a.taskIdx]
			a.pendingDelete = pendingDeleteInfo{kind: "task", id: t.ID, name: t.Title}
			a.state = viewConfirmDelete
		}
	case "s":
		if len(a.tasks) > 0 && a.taskIdx < len(a.tasks) {
			if a.tasks[a.taskIdx].Status == queue.StatusPending {
				return a, a.doStartTask(a.tasks[a.taskIdx].ID)
			}
		}
	case "f":
		if len(a.tasks) > 0 && a.taskIdx < len(a.tasks) {
			if a.tasks[a.taskIdx].Status == queue.StatusDoing {
				return a, a.doFinishTask(a.tasks[a.taskIdx].ID)
			}
		}
	case "r":
		if len(a.tasks) > 0 && a.taskIdx < len(a.tasks) {
			if a.tasks[a.taskIdx].Status == queue.StatusFinished {
				return a, a.doResetTask(a.tasks[a.taskIdx].ID)
			}
		}
	case "p":
		if len(a.tasks) > 0 && a.taskIdx < len(a.tasks) {
			if a.tasks[a.taskIdx].Status == queue.StatusPending {
				return a, a.doPrioritizeTask(a.tasks[a.taskIdx].ID)
			}
		}
	case "R":
		a.statusMsg = ""
		return a, a.loadTasksCmd()
	case "q", "esc":
		a.state = viewQueueList
		a.currentQueue = nil
		a.statusMsg = ""
		return a, a.loadQueuesCmd()
	}
	return a, nil
}

func (a App) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlS:
		return a.submitForm()
	case tea.KeyEsc:
		if a.state == viewCreateQueue {
			a.state = viewQueueList
		} else {
			a.state = viewTaskList
		}
		a.statusMsg = ""
		return a, nil
	case tea.KeyTab:
		a.inputs[a.focusIdx].Blur()
		a.focusIdx = (a.focusIdx + 1) % len(a.inputs)
		cmd := a.inputs[a.focusIdx].Focus()
		return a, cmd
	case tea.KeyShiftTab:
		a.inputs[a.focusIdx].Blur()
		a.focusIdx = (a.focusIdx - 1 + len(a.inputs)) % len(a.inputs)
		cmd := a.inputs[a.focusIdx].Focus()
		return a, cmd
	default:
		var cmd tea.Cmd
		a.inputs[a.focusIdx], cmd = a.inputs[a.focusIdx].Update(msg)
		return a, cmd
	}
}

func (a App) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		return a.doDelete()
	case "n", "esc":
		if a.pendingDelete.kind == "queue" {
			a.state = viewQueueList
		} else {
			a.state = viewTaskList
		}
	}
	return a, nil
}

func (a App) submitForm() (tea.Model, tea.Cmd) {
	if a.state == viewCreateQueue {
		name := strings.TrimSpace(a.inputs[0].Value())
		if name == "" {
			a.statusMsg = "Name is required"
			a.isError = true
			return a, nil
		}
		desc := strings.TrimSpace(a.inputs[1].Value())
		input := queue.CreateQueueInput{Name: name, Description: desc}
		a.state = viewQueueList
		a.statusMsg = ""
		return a, a.doCreateQueue(input)
	}

	if a.state == viewCreateTask {
		title := strings.TrimSpace(a.inputs[0].Value())
		if title == "" {
			a.statusMsg = "Title is required"
			a.isError = true
			return a, nil
		}
		if a.currentQueue == nil {
			a.statusMsg = "No queue selected"
			a.isError = true
			return a, nil
		}
		desc := strings.TrimSpace(a.inputs[1].Value())
		priorityStr := strings.TrimSpace(a.inputs[2].Value())
		priority := 0
		if priorityStr != "" {
			p, err := strconv.Atoi(priorityStr)
			if err != nil {
				a.statusMsg = "Priority must be a number"
				a.isError = true
				return a, nil
			}
			priority = p
		}
		input := queue.CreateTaskInput{
			QueueID:     a.currentQueue.ID,
			Title:       title,
			Description: desc,
			Priority:    priority,
		}
		a.state = viewTaskList
		a.statusMsg = ""
		return a, a.doCreateTask(input)
	}
	return a, nil
}

// --- async commands ---

func (a App) doCreateQueue(input queue.CreateQueueInput) tea.Cmd {
	return func() tea.Msg {
		if _, err := a.client.CreateQueue(context.Background(), input); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func (a App) doCreateTask(input queue.CreateTaskInput) tea.Cmd {
	return func() tea.Msg {
		if _, err := a.client.CreateTask(context.Background(), input); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func (a App) doDelete() (tea.Model, tea.Cmd) {
	kind := a.pendingDelete.kind
	id := a.pendingDelete.id
	if kind == "queue" {
		a.state = viewQueueList
	} else {
		a.state = viewTaskList
	}
	return a, func() tea.Msg {
		var err error
		if kind == "queue" {
			err = a.client.DeleteQueue(context.Background(), id)
		} else {
			err = a.client.DeleteTask(context.Background(), id)
		}
		if err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func (a App) doStartTask(id int64) tea.Cmd {
	return func() tea.Msg {
		if _, err := a.client.StartTask(context.Background(), id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func (a App) doFinishTask(id int64) tea.Cmd {
	return func() tea.Msg {
		if _, err := a.client.FinishTask(context.Background(), id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func (a App) doResetTask(id int64) tea.Cmd {
	return func() tea.Msg {
		if _, err := a.client.UpdateTask(context.Background(), id, queue.StatusPending); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func (a App) doPrioritizeTask(id int64) tea.Cmd {
	return func() tea.Msg {
		if _, err := a.client.PrioritizeTask(context.Background(), id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

// --- views ---

// View renders the TUI based on current state.
func (a App) View() string {
	switch a.state {
	case viewLoading:
		return "\n  " + helpStyle.Render("Loading...")
	case viewQueueList:
		return a.viewQueueList()
	case viewTaskList:
		return a.viewTaskList()
	case viewCreateQueue, viewCreateTask:
		return a.viewCreateForm()
	case viewConfirmDelete:
		return a.viewConfirmDelete()
	}
	return ""
}

func (a App) effectiveWidth() int {
	if a.width < 60 {
		return 80
	}
	return a.width
}

func (a App) viewQueueList() string {
	w := a.effectiveWidth()
	header := headerStyle.Width(w).Render(titleStyle.Render("📋 Task Queue Manager"))

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")

	if len(a.queues) == 0 {
		sb.WriteString(dimStyle.Render("  No queues yet. Press 'n' to create one."))
		sb.WriteString("\n")
	} else {
		for i, q := range a.queues {
			stats := fmt.Sprintf("  pending:%-3d  doing:%-3d  done:%-3d",
				q.Stats.Pending, q.Stats.Doing, q.Stats.Finished)
			line := fmt.Sprintf("  %-32s%s", q.Name, stats)
			if i == a.queueIdx {
				sb.WriteString(selectedItemStyle.Render(line))
			} else {
				sb.WriteString(normalItemStyle.Render(line))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	if msg := a.statusBar(); msg != "" {
		sb.WriteString(msg + "\n")
	}
	sb.WriteString(helpStyle.Render("  j/k: navigate  •  Enter: open  •  n: new  •  d: delete  •  R: refresh  •  q: quit"))
	return sb.String()
}

func (a App) viewTaskList() string {
	w := a.effectiveWidth()
	queueName := ""
	queueStats := ""
	if a.currentQueue != nil {
		queueName = a.currentQueue.Name
		queueStats = fmt.Sprintf("  [pending:%d  doing:%d  done:%d]",
			a.currentQueue.Stats.Pending, a.currentQueue.Stats.Doing, a.currentQueue.Stats.Finished)
	}
	header := headerStyle.Width(w).Render(titleStyle.Render("📋 " + queueName + queueStats))

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")

	if len(a.tasks) == 0 {
		sb.WriteString(dimStyle.Render("  No tasks yet. Press 'n' to create one."))
		sb.WriteString("\n")
	} else {
		for i, t := range a.tasks {
			badge := taskStatusLabel(t.Status)
			line := fmt.Sprintf("  %-11s  %-40s  pri:%-3d", badge, t.Title, t.Priority)
			if i == a.taskIdx {
				sb.WriteString(selectedItemStyle.Render(line))
			} else {
				sb.WriteString(normalItemStyle.Render(line))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	if msg := a.statusBar(); msg != "" {
		sb.WriteString(msg + "\n")
	}
	sb.WriteString(helpStyle.Render("  j/k: nav  •  n: new  •  s: start  •  f: finish  •  r: reset  •  p: prioritize  •  d: delete  •  Esc: back"))
	return sb.String()
}

func (a App) viewCreateForm() string {
	w := a.effectiveWidth()
	header := headerStyle.Width(w).Render(titleStyle.Render("📋 Task Queue Manager"))

	var formTitle string
	var labels []string
	if a.formMode == "queue" {
		formTitle = "Create New Queue"
		labels = []string{"Name:", "Description:"}
	} else {
		formTitle = "Create New Task"
		labels = []string{"Title:", "Description:", "Priority:"}
	}

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n  ")
	sb.WriteString(titleStyle.Render(formTitle))
	sb.WriteString("\n\n")

	for i, inp := range a.inputs {
		label := labelStyle.Render(fmt.Sprintf("  %-14s", labels[i]))
		sb.WriteString(label + inp.View() + "\n\n")
	}

	if a.statusMsg != "" {
		if a.isError {
			sb.WriteString("  " + errorStyle.Render(a.statusMsg) + "\n")
		} else {
			sb.WriteString("  " + successStyle.Render(a.statusMsg) + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("  Ctrl+S: submit  •  Esc: cancel  •  Tab/Shift+Tab: cycle fields"))
	return sb.String()
}

func (a App) viewConfirmDelete() string {
	w := a.effectiveWidth()
	header := headerStyle.Width(w).Render(titleStyle.Render("📋 Task Queue Manager"))

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("  Delete %s '%s'?\n\n", a.pendingDelete.kind, a.pendingDelete.name))
	sb.WriteString(helpStyle.Render("  y: yes, delete  •  n: cancel"))
	return sb.String()
}

func (a App) statusBar() string {
	if a.statusMsg == "" {
		return ""
	}
	if a.isError {
		return "  " + errorStyle.Render(a.statusMsg)
	}
	return "  " + successStyle.Render(a.statusMsg)
}

func taskStatusLabel(status queue.TaskStatus) string {
	switch status {
	case queue.StatusPending:
		return pendingBadge.Render("[pending]")
	case queue.StatusDoing:
		return doingBadge.Render("[doing]")
	case queue.StatusFinished:
		return finishedBadge.Render("[done]")
	default:
		return string(status)
	}
}

func makeQueueInputs() []textinput.Model {
	name := textinput.New()
	name.Placeholder = "Queue name (required)"
	name.CharLimit = 100

	desc := textinput.New()
	desc.Placeholder = "Description (optional)"
	desc.CharLimit = 255

	return []textinput.Model{name, desc}
}

func makeTaskInputs() []textinput.Model {
	title := textinput.New()
	title.Placeholder = "Task title (required)"
	title.CharLimit = 200

	desc := textinput.New()
	desc.Placeholder = "Description (optional)"
	desc.CharLimit = 500

	prio := textinput.New()
	prio.Placeholder = "0"
	prio.CharLimit = 5

	return []textinput.Model{title, desc, prio}
}
