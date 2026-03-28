package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AlexiaChen/issue-kanban-mcp/internal/apiclient"
	"github.com/AlexiaChen/issue-kanban-mcp/internal/queue"
)

type viewState int

const (
	viewLoading viewState = iota
	viewProjectList
	viewKanbanBoard
	viewCreateQueue
	viewCreateTask
	viewEditTask
	viewConfirmDelete
)

type (
	projectsLoadedMsg struct{ queues []apiclient.QueueWithStats }
	issuesLoadedMsg  struct{ tasks []queue.Task }
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
	taskIdx       int    // row index within the focused kanban column
	kanbanColIdx  int    // which kanban column is focused: 0=pending, 1=doing, 2=finished
	inputs        []textinput.Model
	descInput     textarea.Model // multi-line description field for task forms
	focusIdx      int
	formMode      string
	editingTask   *queue.Task
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

func (a App) loadProjectsCmd() tea.Cmd {
	return func() tea.Msg {
		queues, err := a.client.ListProjects(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return projectsLoadedMsg{queues}
	}
}

func (a App) loadIssuesCmd() tea.Cmd {
	if a.currentQueue == nil {
		return nil
	}
	queueID := a.currentQueue.ID
	return func() tea.Msg {
		tasks, err := a.client.ListIssues(context.Background(), queueID, "")
		if err != nil {
			return errMsg{err}
		}
		return issuesLoadedMsg{tasks}
	}
}

// Init kicks off queue loading and the auto-refresh ticker.
func (a App) Init() tea.Cmd {
	return tea.Batch(a.loadProjectsCmd(), tickCmd())
}

// Update handles messages and returns the updated model and next command.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case projectsLoadedMsg:
		a.queues = msg.queues
		if a.state == viewLoading {
			a.state = viewProjectList
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

	case issuesLoadedMsg:
		a.tasks = msg.tasks
		return a, nil

	case actionDoneMsg:
		a.isError = false
		switch a.state {
		case viewProjectList:
			return a, a.loadProjectsCmd()
		case viewKanbanBoard:
			return a, tea.Batch(a.loadIssuesCmd(), a.loadProjectsCmd())
		}
		return a, nil

	case errMsg:
		a.statusMsg = msg.err.Error()
		a.isError = true
		return a, nil

	case tickMsg:
		var cmd tea.Cmd
		switch a.state {
		case viewProjectList:
			cmd = a.loadProjectsCmd()
		case viewKanbanBoard:
			// Refresh both tasks and queue stats so the header counters stay current.
			cmd = tea.Batch(a.loadIssuesCmd(), a.loadProjectsCmd())
		}
		return a, tea.Batch(cmd, tickCmd())

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	// Forward all other messages (e.g. cursor blink) to active text inputs.
	if a.state == viewCreateQueue || a.state == viewCreateTask || a.state == viewEditTask {
		var cmds []tea.Cmd
		for i := range a.inputs {
			var cmd tea.Cmd
			a.inputs[i], cmd = a.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		if a.isIssueForm() {
			var cmd tea.Cmd
			a.descInput, cmd = a.descInput.Update(msg)
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)
	}

	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.state {
	case viewProjectList:
		return a.handleProjectListKey(msg)
	case viewKanbanBoard:
		return a.handleIssueListKey(msg)
	case viewCreateQueue, viewCreateTask, viewEditTask:
		return a.handleFormKey(msg)
	case viewConfirmDelete:
		return a.handleConfirmKey(msg)
	}
	return a, nil
}

func (a App) handleProjectListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			a.state = viewKanbanBoard
			a.taskIdx = 0
			a.statusMsg = ""
			return a, a.loadIssuesCmd()
		}
	case "n":
		a.formMode = "queue"
		a.state = viewCreateQueue
		a.inputs = makeProjectInputs()
		a.focusIdx = 0
		a.statusMsg = ""
		cmd := a.inputs[0].Focus()
		return a, cmd
	case "d":
		if len(a.queues) > 0 && a.queueIdx < len(a.queues) {
			q := a.queues[a.queueIdx]
			a.pendingDelete = pendingDeleteInfo{kind: "project", id: q.ID, name: q.Name}
			a.state = viewConfirmDelete
		}
	case "R":
		a.statusMsg = ""
		return a, a.loadProjectsCmd()
	case "q", "ctrl+c":
		return a, tea.Quit
	}
	return a, nil
}

func (a App) handleIssueListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	groups := a.groupedIssues()
	currentCol := groups[a.kanbanColIdx]

	switch msg.String() {
	case "j", "down":
		if a.taskIdx < len(currentCol)-1 {
			a.taskIdx++
		}
	case "k", "up":
		if a.taskIdx > 0 {
			a.taskIdx--
		}
	case "h", "left":
		if a.kanbanColIdx > 0 {
			a.kanbanColIdx--
			a.taskIdx = 0
		}
	case "l", "right":
		if a.kanbanColIdx < 2 {
			a.kanbanColIdx++
			a.taskIdx = 0
		}
	case "n":
		a.formMode = "task"
		a.state = viewCreateTask
		a.inputs = makeIssueInputs()
		a.descInput = newDescInput(a.effectiveWidth(), a.height)
		a.descInput.SetValue("")
		a.focusIdx = 0
		a.statusMsg = ""
		cmd := a.inputs[0].Focus()
		return a, cmd
	case "d":
		t := a.selectedKanbanIssue()
		if t != nil {
			a.pendingDelete = pendingDeleteInfo{kind: "issue", id: t.ID, name: t.Title}
			a.state = viewConfirmDelete
		}
	case "e":
		t := a.selectedKanbanIssue()
		if t != nil && t.Status == queue.StatusPending {
			a.editingTask = t
			a.formMode = "edit"
			a.state = viewEditTask
			a.inputs = makeEditIssueInputs(*t)
			a.descInput = newDescInput(a.effectiveWidth(), a.height)
			a.descInput.SetValue(t.Description)
			a.focusIdx = 0
			a.statusMsg = ""
			cmd := a.inputs[0].Focus()
			return a, cmd
		}
	case "p":
		t := a.selectedKanbanIssue()
		if t != nil && t.Status == queue.StatusPending {
			return a, a.doPrioritizeIssue(t.ID)
		}
	case "R":
		a.statusMsg = ""
		return a, tea.Batch(a.loadIssuesCmd(), a.loadProjectsCmd())
	case "q", "esc":
		a.state = viewProjectList
		a.currentQueue = nil
		a.statusMsg = ""
		return a, a.loadProjectsCmd()
	}
	return a, nil
}

func (a App) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlS:
		return a.submitForm()
	case tea.KeyEsc:
		if a.state == viewCreateQueue {
			a.state = viewProjectList
		} else if a.state == viewEditTask {
			a.state = viewKanbanBoard
			a.editingTask = nil
		} else {
			a.state = viewKanbanBoard
		}
		a.statusMsg = ""
		return a, nil
	case tea.KeyTab:
		if a.isIssueForm() {
			// Blur current field (inline to avoid value-receiver copy loss).
			if a.focusIdx == 1 {
				a.descInput.Blur()
			} else {
				a.inputs[a.taskInputIdx()].Blur()
			}
			a.focusIdx = (a.focusIdx + 1) % 3
			// Focus next field.
			if a.focusIdx == 1 {
				cmd := a.descInput.Focus()
				return a, cmd
			}
			cmd := a.inputs[a.taskInputIdx()].Focus()
			return a, cmd
		}
		a.inputs[a.focusIdx].Blur()
		a.focusIdx = (a.focusIdx + 1) % len(a.inputs)
		cmd := a.inputs[a.focusIdx].Focus()
		return a, cmd
	case tea.KeyShiftTab:
		if a.isIssueForm() {
			if a.focusIdx == 1 {
				a.descInput.Blur()
			} else {
				a.inputs[a.taskInputIdx()].Blur()
			}
			a.focusIdx = (a.focusIdx - 1 + 3) % 3
			if a.focusIdx == 1 {
				cmd := a.descInput.Focus()
				return a, cmd
			}
			cmd := a.inputs[a.taskInputIdx()].Focus()
			return a, cmd
		}
		a.inputs[a.focusIdx].Blur()
		a.focusIdx = (a.focusIdx - 1 + len(a.inputs)) % len(a.inputs)
		cmd := a.inputs[a.focusIdx].Focus()
		return a, cmd
	default:
		if a.isIssueForm() && a.focusIdx == 1 {
			var cmd tea.Cmd
			a.descInput, cmd = a.descInput.Update(msg)
			return a, cmd
		}
		var cmd tea.Cmd
		idx := a.taskInputIdx()
		if !a.isIssueForm() {
			idx = a.focusIdx
		}
		a.inputs[idx], cmd = a.inputs[idx].Update(msg)
		return a, cmd
	}
}

// isIssueForm reports whether the current form uses a textarea for description.
func (a App) isIssueForm() bool {
	return a.formMode == "task" || a.formMode == "edit"
}

// taskInputIdx maps focusIdx to the correct a.inputs index for task forms.
// Task forms: focusIdx 0→inputs[0] (title), 1→descInput (not in inputs), 2→inputs[1] (priority).
func (a App) taskInputIdx() int {
	if a.focusIdx == 2 {
		return 1
	}
	return 0
}

func (a App) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		return a.doDelete()
	case "n", "esc":
		if a.pendingDelete.kind == "project" {
			a.state = viewProjectList
		} else {
			a.state = viewKanbanBoard
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
		a.state = viewProjectList
		a.statusMsg = ""
		return a, a.doCreateProject(input)
	}

	if a.state == viewCreateTask {
		title := strings.TrimSpace(a.inputs[0].Value())
		if title == "" {
			a.statusMsg = "Title is required"
			a.isError = true
			return a, nil
		}
		if a.currentQueue == nil {
			a.statusMsg = "No project selected"
			a.isError = true
			return a, nil
		}
		desc := strings.TrimSpace(a.descInput.Value())
		priorityStr := strings.TrimSpace(a.inputs[1].Value())
		priority := queue.PriorityLow
		if priorityStr != "" {
			p, err := queue.ParsePriority(priorityStr)
			if err != nil {
				a.statusMsg = "Priority must be low, medium, or high"
				a.isError = true
				return a, nil
			}
			priority = p
		}
		input := queue.CreateTaskInput{
			ProjectID:   a.currentQueue.ID,
			Title:       title,
			Description: desc,
			Priority:    priority,
		}
		a.state = viewKanbanBoard
		a.statusMsg = ""
		return a, a.doCreateIssue(input)
	}

	if a.state == viewEditTask {
		if a.editingTask == nil {
			a.state = viewKanbanBoard
			return a, nil
		}
		title := strings.TrimSpace(a.inputs[0].Value())
		if title == "" {
			a.statusMsg = "Title is required"
			a.isError = true
			return a, nil
		}
		desc := strings.TrimSpace(a.descInput.Value())
		priorityStr := strings.TrimSpace(a.inputs[1].Value())
		priority := queue.PriorityLow
		if priorityStr != "" {
			p, err := queue.ParsePriority(priorityStr)
			if err != nil {
				a.statusMsg = "Priority must be low, medium, or high"
				a.isError = true
				return a, nil
			}
			priority = p
		}
		id := a.editingTask.ID
		a.state = viewKanbanBoard
		a.editingTask = nil
		a.statusMsg = ""
		return a, a.doEditIssue(id, title, desc, priority)
	}
	return a, nil
}

// --- async commands ---

func (a App) doCreateProject(input queue.CreateQueueInput) tea.Cmd {
	return func() tea.Msg {
		if _, err := a.client.CreateProject(context.Background(), input); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func (a App) doCreateIssue(input queue.CreateTaskInput) tea.Cmd {
	return func() tea.Msg {
		if _, err := a.client.CreateIssue(context.Background(), input); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func (a App) doDelete() (tea.Model, tea.Cmd) {
	kind := a.pendingDelete.kind
	id := a.pendingDelete.id
	if kind == "project" {
		a.state = viewProjectList
	} else {
		a.state = viewKanbanBoard
	}
	return a, func() tea.Msg {
		var err error
		if kind == "project" {
			err = a.client.DeleteProject(context.Background(), id)
		} else {
			err = a.client.DeleteIssue(context.Background(), id)
		}
		if err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func (a App) doEditIssue(id int64, title, desc string, priority queue.Priority) tea.Cmd {
	return func() tea.Msg {
		if _, err := a.client.EditIssue(context.Background(), id, &title, &desc, &priority); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func (a App) doPrioritizeIssue(id int64) tea.Cmd {
	return func() tea.Msg {
		if _, err := a.client.PrioritizeIssue(context.Background(), id); err != nil {
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
	case viewProjectList:
		return a.viewProjectList()
	case viewKanbanBoard:
		return a.viewKanbanBoard()
	case viewCreateQueue, viewCreateTask, viewEditTask:
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

func (a App) viewProjectList() string {
	w := a.effectiveWidth()
	header := headerStyle.Width(w).Render(titleStyle.Render("📋 Issue Kanban Manager"))

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")

	if len(a.queues) == 0 {
		sb.WriteString(dimStyle.Render("  No projects yet. Press 'n' to create one."))
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

// groupedIssues groups a.tasks by status into [0]=pending, [1]=doing, [2]=finished.
func (a App) groupedIssues() [3][]queue.Task {
	var groups [3][]queue.Task
	for _, t := range a.tasks {
		switch t.Status {
		case queue.StatusPending:
			groups[0] = append(groups[0], t)
		case queue.StatusDoing:
			groups[1] = append(groups[1], t)
		case queue.StatusFinished:
			groups[2] = append(groups[2], t)
		}
	}
	return groups
}

// selectedKanbanIssue returns the currently focused task in the kanban board.
func (a App) selectedKanbanIssue() *queue.Task {
	groups := a.groupedIssues()
	col := groups[a.kanbanColIdx]
	if a.taskIdx >= 0 && a.taskIdx < len(col) {
		t := col[a.taskIdx]
		return &t
	}
	return nil
}

func (a App) viewKanbanBoard() string {
	w := a.effectiveWidth()
	queueName := ""
	queueStats := ""
	if a.currentQueue != nil {
		queueName = a.currentQueue.Name
		queueStats = fmt.Sprintf("  [pending:%d  doing:%d  done:%d]",
			a.currentQueue.Stats.Pending, a.currentQueue.Stats.Doing, a.currentQueue.Stats.Finished)
	}
	header := headerStyle.Width(w).Render(titleStyle.Render("📋 " + queueName + queueStats))

	groups := a.groupedIssues()
	colNames := [3]string{"📋 PENDING", "⚡ DOING", "✅ FINISHED"}
	colColors := [3]lipgloss.Color{"#FCD34D", "#67E8F9", "#6EE7B7"}
	focusedBorderColors := [3]lipgloss.Color{"#F59E0B", "#22D3EE", "#34D399"}

	colWidth := (w - 6) / 3

	var renderedCols [3]string
	for ci := 0; ci < 3; ci++ {
		group := groups[ci]
		isFocused := ci == a.kanbanColIdx

		borderColor := lipgloss.Color("#4B5563")
		if isFocused {
			borderColor = focusedBorderColors[ci]
		}

		// Column header
		colHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(colColors[ci]).
			Width(colWidth).
			Render(fmt.Sprintf("%s (%d)", colNames[ci], len(group)))

		var lines []string
		if len(group) == 0 {
			lines = append(lines, dimStyle.Width(colWidth).Render("  (empty)"))
		} else {
			for ri, t := range group {
				priLabel := t.Priority.String()
				maxTitle := colWidth - len(priLabel) - 3
				if maxTitle < 5 {
					maxTitle = 5
				}
				title := t.Title
				if len(title) > maxTitle {
					title = title[:maxTitle-1] + "…"
				}
				line := fmt.Sprintf("%-*s %s", maxTitle, title, priLabel)
				if isFocused && ri == a.taskIdx {
					lines = append(lines, selectedItemStyle.Width(colWidth).Render(line))
				} else {
					lines = append(lines, normalItemStyle.Width(colWidth).Render(line))
				}
			}
		}

		colContent := lipgloss.JoinVertical(lipgloss.Left, append([]string{colHeader}, lines...)...)
		renderedCols[ci] = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Width(colWidth).
			Padding(0, 1).
			Render(colContent)
	}

	kanban := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols[0], " ", renderedCols[1], " ", renderedCols[2])

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")
	sb.WriteString(kanban)
	sb.WriteString("\n\n")
	if msg := a.statusBar(); msg != "" {
		sb.WriteString(msg + "\n")
	}
	sb.WriteString(helpStyle.Render("  j/k: nav  •  h/l: columns  •  n: new  •  e: edit  •  p: prioritize  •  d: delete  •  R: refresh  •  Esc: back"))
	return sb.String()
}

func (a App) viewCreateForm() string {
	w := a.effectiveWidth()
	header := headerStyle.Width(w).Render(titleStyle.Render("📋 Issue Kanban Manager"))

	var formTitle string
	var labels []string
	if a.formMode == "queue" {
		formTitle = "Create New Project"
		labels = []string{"Name:", "Description:"}
	} else if a.formMode == "edit" {
		formTitle = "Edit Issue"
		labels = []string{"Title:", "Description:", "Priority:"}
	} else {
		formTitle = "Create New Issue"
		labels = []string{"Title:", "Description:", "Priority:"}
	}

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n  ")
	sb.WriteString(titleStyle.Render(formTitle))
	sb.WriteString("\n\n")

	for i, inp := range a.inputs {
		// For task forms, inputs[0]=title, inputs[1]=priority.
		// Description is handled as a textarea injected after title.
		var labelIdx int
		if a.isIssueForm() && i == 1 {
			labelIdx = 2 // "Priority:" is at index 2 of the 3-element labels slice
		} else {
			labelIdx = i
		}
		label := labelStyle.Render(fmt.Sprintf("  %-14s", labels[labelIdx]))
		sb.WriteString(label + inp.View() + "\n\n")
		if a.isIssueForm() && i == 0 {
			// render description textarea between title and priority
			descLabel := labelStyle.Render(fmt.Sprintf("  %-14s", labels[1]))
			sb.WriteString(descLabel + "\n")
			sb.WriteString(a.descInput.View() + "\n\n")
		}
	}

	if a.statusMsg != "" {
		if a.isError {
			sb.WriteString("  " + errorStyle.Render(a.statusMsg) + "\n")
		} else {
			sb.WriteString("  " + successStyle.Render(a.statusMsg) + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("  Ctrl+S: submit  •  Esc: cancel  •  Tab/Shift+Tab: cycle fields  •  Enter: newline in description"))
	return sb.String()
}

func (a App) viewConfirmDelete() string {
	w := a.effectiveWidth()
	header := headerStyle.Width(w).Render(titleStyle.Render("📋 Issue Kanban Manager"))

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

func issueStatusLabel(status queue.TaskStatus) string {
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

func makeProjectInputs() []textinput.Model {
	name := textinput.New()
	name.Placeholder = "Project name (required)"
	name.CharLimit = 100

	desc := textinput.New()
	desc.Placeholder = "Description (optional)"
	desc.CharLimit = 255

	return []textinput.Model{name, desc}
}

func makeIssueInputs() []textinput.Model {
	title := textinput.New()
	title.Placeholder = "Issue title (required)"
	title.CharLimit = 200

	prio := textinput.New()
	prio.Placeholder = "low / medium / high (default: low)"
	prio.CharLimit = 10

	return []textinput.Model{title, prio}
}

func makeEditIssueInputs(t queue.Task) []textinput.Model {
	title := textinput.New()
	title.Placeholder = "Issue title (required)"
	title.CharLimit = 200
	title.SetValue(t.Title)

	prio := textinput.New()
	prio.Placeholder = "low / medium / high (default: low)"
	prio.CharLimit = 10
	prio.SetValue(t.Priority.String())

	return []textinput.Model{title, prio}
}

func newDescInput(width, height int) textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Description (optional, multi-line)"
	ta.CharLimit = 0 // unlimited — prevents truncation when pasting or loading long descriptions
	ta.SetWidth(width - 20)
	// Use at least 10 rows, but scale with terminal height so large descriptions are editable.
	descHeight := height - 20
	if descHeight < 10 {
		descHeight = 10
	}
	ta.SetHeight(descHeight)
	ta.ShowLineNumbers = false
	return ta
}
