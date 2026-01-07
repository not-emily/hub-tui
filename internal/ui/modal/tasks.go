package modal

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/components"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// TaskRun represents a task run for the modal.
type TaskRun struct {
	ID             string
	Workflow       string
	Status         string
	StartedAt      time.Time
	EndedAt        time.Time
	Error          string
	Result         *client.RunResult
	NeedsAttention bool
}

// isRunSuccess returns true if the run completed successfully.
func isRunSuccess(r client.Run) bool {
	if r.Status == "failed" {
		return false
	}
	if r.Result != nil && !r.Result.Success {
		return false
	}
	return true
}

// formatRunOutput extracts a readable output string from the run result.
func formatRunOutput(result *client.RunResult) string {
	if result == nil {
		return ""
	}

	var outputs []string
	for _, step := range result.Steps {
		if step.Error != "" {
			outputs = append(outputs, fmt.Sprintf("[%s] Error: %s", step.StepName, step.Error))
		} else if step.Output != nil {
			// Try to format the output nicely
			switch v := step.Output.(type) {
			case string:
				outputs = append(outputs, fmt.Sprintf("[%s] %s", step.StepName, v))
			case map[string]interface{}:
				if msg, ok := v["message"].(string); ok {
					outputs = append(outputs, fmt.Sprintf("[%s] %s", step.StepName, msg))
				} else {
					// JSON encode it
					b, _ := json.MarshalIndent(v, "", "  ")
					outputs = append(outputs, fmt.Sprintf("[%s]\n%s", step.StepName, string(b)))
				}
			default:
				b, _ := json.MarshalIndent(v, "", "  ")
				outputs = append(outputs, fmt.Sprintf("[%s]\n%s", step.StepName, string(b)))
			}
		}
	}

	return strings.Join(outputs, "\n")
}

// TasksModal displays running, completed, and failed tasks.
type TasksModal struct {
	client           *client.Client
	needsAttention   []TaskRun // All-time runs needing attention
	running          []TaskRun // Today's running
	completed        []TaskRun // Today's completed (needs_attention=false)
	failed           []TaskRun // Today's failed (needs_attention=false)
	allRuns          []TaskRun // Combined list for navigation
	selected         int
	loading          bool
	loadingDetail    bool   // Loading full run details
	error            string // Error loading task list
	detailError string    // Error loading task details
	view        tasksView
	detailRun   *TaskRun // Run being viewed in detail
	confirm     *components.Confirmation

	// Pagination state
	completedPage    int
	completedTotal   int // Total completed items
	failedPage       int
	failedTotal      int // Total failed items

	// History view state
	history         []TaskRun       // Current page of history
	historyPage     int             // Current page (0-indexed)
	historyTotal    int             // Total history items from API
	historyHasMore  bool            // Whether more pages are available
	historyCursors  map[int]string  // Cursor for each page (page number -> cursor)
	previousView    tasksView       // View to return to from detail
}

const itemsPerPage = 5
const historyItemsPerPage = 15

type tasksView int

const (
	viewTasksList tasksView = iota
	viewTaskDetail
	viewTasksHistory
)

// NewTasksModal creates a new tasks modal that fetches fresh data from the API.
func NewTasksModal(c *client.Client) *TasksModal {
	return &TasksModal{
		client:  c,
		loading: true,
		view:    viewTasksList,
		confirm: components.NewConfirmation(),
	}
}

func (m *TasksModal) buildAllRuns() {
	m.allRuns = nil
	m.allRuns = append(m.allRuns, m.needsAttention...)
	m.allRuns = append(m.allRuns, m.running...)
	// Only include visible page of completed/failed
	m.allRuns = append(m.allRuns, m.getCompletedPage()...)
	m.allRuns = append(m.allRuns, m.getFailedPage()...)
}

func (m *TasksModal) getCompletedPage() []TaskRun {
	start := m.completedPage * itemsPerPage
	end := start + itemsPerPage
	if start >= len(m.completed) {
		return nil
	}
	if end > len(m.completed) {
		end = len(m.completed)
	}
	return m.completed[start:end]
}

func (m *TasksModal) getFailedPage() []TaskRun {
	start := m.failedPage * itemsPerPage
	end := start + itemsPerPage
	if start >= len(m.failed) {
		return nil
	}
	if end > len(m.failed) {
		end = len(m.failed)
	}
	return m.failed[start:end]
}

// getSelectedSection returns which section the cursor is currently in.
func (m *TasksModal) getSelectedSection() string {
	idx := 0

	// Needs Attention section
	if m.selected < idx+len(m.needsAttention) {
		return "attention"
	}
	idx += len(m.needsAttention)

	// Running section
	if m.selected < idx+len(m.running) {
		return "running"
	}
	idx += len(m.running)

	// Completed section (paginated)
	completedPage := m.getCompletedPage()
	if m.selected < idx+len(completedPage) {
		return "completed"
	}
	idx += len(completedPage)

	// Failed section (paginated)
	return "failed"
}

// getSectionStartIndex returns the index where a section starts in allRuns.
func (m *TasksModal) getSectionStartIndex(section string) int {
	idx := 0
	if section == "attention" {
		return idx
	}
	idx += len(m.needsAttention)
	if section == "running" {
		return idx
	}
	idx += len(m.running)
	if section == "completed" {
		return idx
	}
	idx += len(m.getCompletedPage())
	return idx // failed
}

// TasksLoadedMsg is sent when tasks are loaded.
type TasksLoadedMsg struct {
	NeedsAttention []TaskRun
	Running        []TaskRun
	Completed      []TaskRun
	Failed         []TaskRun
	Error          error
}

// TaskDetailLoadedMsg is sent when full run details are loaded.
type TaskDetailLoadedMsg struct {
	Run   *TaskRun
	Error error
}

// TaskCancelRequestMsg is sent when a cancel is requested.
type TaskCancelRequestMsg struct {
	RunID string
}

// TaskDismissedMsg is sent when a task is dismissed.
type TaskDismissedMsg struct {
	RunID string
	Error error
}

// HistoryLoadedMsg is sent when history is loaded.
type HistoryLoadedMsg struct {
	Runs       []TaskRun
	Total      int
	HasMore    bool
	NextCursor string
	Page       int // Which page was loaded
	Error      error
}

// Init initializes the modal.
func (m *TasksModal) Init() tea.Cmd {
	// If we already have state, no need to load
	if !m.loading {
		return nil
	}
	return m.loadTasks()
}

func (m *TasksModal) loadTasks() tea.Cmd {
	return func() tea.Msg {
		today := time.Now().Format("2006-01-02")

		// Fetch all runs needing attention (any date)
		needsAttentionFilter := true
		attentionResp, err := m.client.ListRuns(&client.RunsFilter{
			NeedsAttention: &needsAttentionFilter,
		})
		if err != nil {
			return TasksLoadedMsg{Error: err}
		}

		// Fetch today's runs
		todayResp, err := m.client.ListRuns(&client.RunsFilter{
			Since: today,
		})
		if err != nil {
			return TasksLoadedMsg{Error: err}
		}

		// Build needs attention list
		var needsAttention []TaskRun
		attentionIDs := make(map[string]bool)
		for _, r := range attentionResp.Runs {
			attentionIDs[r.ID] = true
			needsAttention = append(needsAttention, clientRunToTaskRun(r))
		}

		// Build today's lists, excluding items already in needs attention
		var running, completed, failed []TaskRun
		for _, r := range todayResp.Runs {
			if attentionIDs[r.ID] {
				continue // Already in needs attention section
			}
			tr := clientRunToTaskRun(r)
			if r.Status == "running" {
				running = append(running, tr)
			} else if isRunSuccess(r) {
				completed = append(completed, tr)
			} else {
				failed = append(failed, tr)
			}
		}

		// Sort each category by most recent first
		sortByMostRecent(needsAttention)
		sortByMostRecent(running)
		sortByMostRecent(completed)
		sortByMostRecent(failed)

		return TasksLoadedMsg{
			NeedsAttention: needsAttention,
			Running:        running,
			Completed:      completed,
			Failed:         failed,
		}
	}
}

func clientRunToTaskRun(r client.Run) TaskRun {
	return TaskRun{
		ID:             r.ID,
		Workflow:       r.Workflow,
		Status:         r.Status,
		StartedAt:      r.StartedAt,
		EndedAt:        r.EndedAt,
		Error:          r.Error,
		Result:         r.Result,
		NeedsAttention: r.NeedsAttention,
	}
}

func (m *TasksModal) loadHistory(page int) tea.Cmd {
	// Get cursor for this page (empty string for page 0)
	cursor := ""
	if page > 0 {
		cursor = m.historyCursors[page]
	}

	return func() tea.Msg {
		filter := &client.RunsFilter{
			Limit: historyItemsPerPage,
		}
		if cursor != "" {
			filter.Cursor = cursor
		}

		resp, err := m.client.ListRuns(filter)
		if err != nil {
			return HistoryLoadedMsg{Error: err, Page: page}
		}

		var runs []TaskRun
		for _, run := range resp.Runs {
			runs = append(runs, clientRunToTaskRun(run))
		}

		return HistoryLoadedMsg{
			Runs:       runs,
			Total:      resp.Pagination.Total,
			HasMore:    resp.Pagination.HasMore,
			NextCursor: resp.Pagination.NextCursor,
			Page:       page,
		}
	}
}

func (m *TasksModal) loadTaskDetail(runID string) tea.Cmd {
	return func() tea.Msg {
		// Retry up to 3 times with a short delay to handle race conditions
		// where the run just completed but hub-core hasn't finished writing
		var run *client.Run
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			run, err = m.client.GetRun(runID)
			if err == nil {
				break
			}
			// If not found, wait a bit and retry (race condition with hub-core)
			if attempt < 2 {
				time.Sleep(300 * time.Millisecond)
			}
		}
		if err != nil {
			return TaskDetailLoadedMsg{Error: err}
		}

		tr := &TaskRun{
			ID:             run.ID,
			Workflow:       run.Workflow,
			Status:         run.Status,
			StartedAt:      run.StartedAt,
			EndedAt:        run.EndedAt,
			Error:          run.Error,
			Result:         run.Result,
			NeedsAttention: run.NeedsAttention,
		}
		return TaskDetailLoadedMsg{Run: tr}
	}
}

// Update handles input.
func (m *TasksModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case TasksLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.needsAttention = msg.NeedsAttention
			m.running = msg.Running
			m.completed = msg.Completed
			m.failed = msg.Failed
			m.completedPage = 0
			m.failedPage = 0
			m.completedTotal = len(msg.Completed)
			m.failedTotal = len(msg.Failed)
			m.buildAllRuns()
			m.error = ""
		}
		return m, nil

	case TaskDetailLoadedMsg:
		m.loadingDetail = false
		if msg.Error != nil {
			// Show error in detail view, don't hide the whole list
			m.detailError = msg.Error.Error()
		} else if msg.Run != nil {
			m.detailRun = msg.Run
			m.detailError = ""
		}
		return m, nil

	case TaskDismissedMsg:
		// Clear pending dismiss state
		m.confirm.Clear()
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			// Reload tasks to reflect the dismiss
			return m, m.loadTasks()
		}
		return m, nil

	case components.ConfirmationExpiredMsg:
		m.confirm.HandleExpired(msg)
		return m, nil

	case HistoryLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.history = msg.Runs
			m.historyPage = msg.Page
			m.historyTotal = msg.Total
			m.historyHasMore = msg.HasMore
			m.selected = 0
			// Save cursor for the next page
			if msg.HasMore && msg.NextCursor != "" {
				if m.historyCursors == nil {
					m.historyCursors = make(map[int]string)
				}
				m.historyCursors[msg.Page+1] = msg.NextCursor
			}
			m.error = ""
		}
		return m, nil

	case tea.KeyMsg:
		if m.view == viewTaskDetail {
			return m.updateDetail(msg)
		}
		if m.view == viewTasksHistory {
			return m.updateHistory(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m *TasksModal) updateList(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.confirm.Clear()
		return nil, nil // Close modal
	case "up", "k":
		m.confirm.Clear()
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		m.confirm.Clear()
		if m.selected < len(m.allRuns)-1 {
			m.selected++
		}
	case "enter":
		m.confirm.Clear()
		if len(m.allRuns) > 0 && m.selected < len(m.allRuns) {
			run := m.allRuns[m.selected]
			m.detailRun = &run // Show basic info immediately
			m.previousView = viewTasksList
			m.view = viewTaskDetail
			m.loadingDetail = true
			// Fetch full details from API
			return m, m.loadTaskDetail(run.ID)
		}
	case "c":
		m.confirm.Clear()
		// Cancel selected running task
		if len(m.allRuns) > 0 && m.selected < len(m.allRuns) {
			run := m.allRuns[m.selected]
			if run.Status == "running" {
				return m, m.cancelTask(run.ID)
			}
		}
	case "d":
		// Dismiss selected task that needs attention
		if len(m.allRuns) > 0 && m.selected < len(m.allRuns) {
			run := m.allRuns[m.selected]
			if run.NeedsAttention {
				if execute, cmd := m.confirm.Check("dismiss", run.ID); execute {
					return m, m.dismissTask(run.ID)
				} else if cmd != nil {
					return m, cmd
				}
			}
		}
	case "n":
		// Next page - only for the section where cursor is
		m.confirm.Clear()
		section := m.getSelectedSection()
		changed := false
		if section == "completed" && m.completedTotal > itemsPerPage {
			maxPage := (m.completedTotal - 1) / itemsPerPage
			if m.completedPage < maxPage {
				m.completedPage++
				changed = true
			}
		} else if section == "failed" && m.failedTotal > itemsPerPage {
			maxPage := (m.failedTotal - 1) / itemsPerPage
			if m.failedPage < maxPage {
				m.failedPage++
				changed = true
			}
		}
		if changed {
			m.buildAllRuns()
			// Keep selection at start of the paginated section
			m.selected = m.getSectionStartIndex(section)
		}
	case "p":
		// Previous page - only for the section where cursor is
		m.confirm.Clear()
		section := m.getSelectedSection()
		changed := false
		if section == "completed" && m.completedPage > 0 {
			m.completedPage--
			changed = true
		} else if section == "failed" && m.failedPage > 0 {
			m.failedPage--
			changed = true
		}
		if changed {
			m.buildAllRuns()
			// Keep selection at start of the paginated section
			m.selected = m.getSectionStartIndex(section)
		}
	case "h":
		// Switch to history view
		m.confirm.Clear()
		m.view = viewTasksHistory
		m.loading = true
		m.selected = 0
		m.history = nil
		m.historyPage = 0
		m.historyCursors = make(map[int]string)
		return m, m.loadHistory(0)
	}
	return m, nil
}

func (m *TasksModal) updateDetail(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Return to the view we came from (list or history)
		if m.previousView == viewTasksHistory {
			m.view = viewTasksHistory
		} else {
			m.view = viewTasksList
		}
		m.detailRun = nil
		m.detailError = ""
		m.confirm.Clear()
	case "r":
		m.confirm.Clear()
		// Refresh details
		if m.detailRun != nil && !m.loadingDetail {
			m.loadingDetail = true
			m.detailError = ""
			return m, m.loadTaskDetail(m.detailRun.ID)
		}
	case "c":
		m.confirm.Clear()
		// Cancel if running
		if m.detailRun != nil && m.detailRun.Status == "running" {
			return m, m.cancelTask(m.detailRun.ID)
		}
	case "d":
		// Dismiss if needs attention
		if m.detailRun != nil && m.detailRun.NeedsAttention {
			runID := m.detailRun.ID
			if execute, cmd := m.confirm.Check("dismiss", runID); execute {
				// Return to the view we came from
				if m.previousView == viewTasksHistory {
					m.view = viewTasksHistory
				} else {
					m.view = viewTasksList
				}
				m.detailRun = nil
				return m, m.dismissTask(runID)
			} else if cmd != nil {
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m *TasksModal) updateHistory(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Return to main list view
		m.view = viewTasksList
		m.selected = 0
		m.confirm.Clear()
	case "up", "k":
		m.confirm.Clear()
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		m.confirm.Clear()
		if m.selected < len(m.history)-1 {
			m.selected++
		}
	case "enter":
		m.confirm.Clear()
		if len(m.history) > 0 && m.selected < len(m.history) {
			run := m.history[m.selected]
			m.detailRun = &run
			m.previousView = viewTasksHistory
			m.view = viewTaskDetail
			m.loadingDetail = true
			return m, m.loadTaskDetail(run.ID)
		}
	case "n":
		// Next page if available
		m.confirm.Clear()
		if m.historyHasMore && !m.loading {
			// Check if we have cursor for next page
			nextPage := m.historyPage + 1
			if _, hasCursor := m.historyCursors[nextPage]; hasCursor {
				m.loading = true
				return m, m.loadHistory(nextPage)
			}
		}
	case "p":
		// Previous page
		m.confirm.Clear()
		if m.historyPage > 0 && !m.loading {
			m.loading = true
			return m, m.loadHistory(m.historyPage - 1)
		}
	case "d":
		// Dismiss selected task that needs attention
		if len(m.history) > 0 && m.selected < len(m.history) {
			run := m.history[m.selected]
			if run.NeedsAttention {
				if execute, cmd := m.confirm.Check("dismiss", run.ID); execute {
					return m, m.dismissTask(run.ID)
				} else if cmd != nil {
					return m, cmd
				}
			}
		}
	}
	return m, nil
}

// Title returns the modal title.
func (m *TasksModal) Title() string {
	if m.view == viewTaskDetail && m.detailRun != nil {
		return "Task: " + m.detailRun.Workflow
	}
	if m.view == viewTasksHistory {
		return "Task History"
	}
	return "Tasks"
}

// View renders the modal content.
func (m *TasksModal) View() string {
	if m.view == viewTaskDetail {
		return m.viewDetail()
	}
	if m.view == viewTasksHistory {
		return m.viewHistory()
	}
	return m.viewList()
}

func (m *TasksModal) viewList() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Loading tasks...")
	}

	if m.error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			errorStyle.Render("Error: "+m.error),
			"",
			hintStyle.Render("[r] Retry"),
		)
	}

	if len(m.allRuns) == 0 {
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			hintStyle.Render("No tasks today."),
			"",
			hintStyle.Render("[h] History"),
		)
	}

	var lines []string
	runIndex := 0 // Track index across all sections for selection

	headerStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	attentionStyle := lipgloss.NewStyle().Foreground(theme.Warning).Bold(true)
	timeStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	pageStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	runningIndicator := lipgloss.NewStyle().Foreground(theme.Warning).Render("●")
	completedIndicator := lipgloss.NewStyle().Foreground(theme.Success).Render("✓")
	failedIndicator := lipgloss.NewStyle().Foreground(theme.Error).Render("✗")
	attentionIndicator := lipgloss.NewStyle().Foreground(theme.Warning).Bold(true).Render("⚠")

	// Needs Attention section (all-time)
	if len(m.needsAttention) > 0 {
		lines = append(lines, headerStyle.Render("Needs Attention:"))
		for _, r := range m.needsAttention {
			name := attentionStyle.Render(r.Workflow)
			if runIndex == m.selected {
				name = selectedStyle.Render(r.Workflow)
			}
			name += " " + attentionIndicator

			// Show status indicator based on run status
			var indicator string
			var timeText string
			switch r.Status {
			case "running":
				indicator = runningIndicator
				timeText = "Started " + formatElapsed(r.StartedAt)
			case "completed":
				indicator = completedIndicator
				timeText = "Completed " + formatElapsed(r.EndedAt)
			default:
				indicator = failedIndicator
				timeText = "Failed " + formatElapsed(r.EndedAt)
			}
			line := fmt.Sprintf("  %s %s    %s", indicator, name, timeStyle.Render(timeText))
			lines = append(lines, line)
			runIndex++
		}
		lines = append(lines, "")
	}

	// Running section (today)
	if len(m.running) > 0 {
		lines = append(lines, headerStyle.Render("Running:"))
		for _, r := range m.running {
			name := normalStyle.Render(r.Workflow)
			if runIndex == m.selected {
				name = selectedStyle.Render(r.Workflow)
			}
			elapsed := formatElapsed(r.StartedAt)
			line := fmt.Sprintf("  %s %s    %s", runningIndicator, name, timeStyle.Render("Started "+elapsed))
			lines = append(lines, line)
			runIndex++
		}
		lines = append(lines, "")
	}

	// Completed section (today, paginated)
	completedPage := m.getCompletedPage()
	if len(completedPage) > 0 || m.completedTotal > 0 {
		header := "Completed:"
		if m.completedTotal > itemsPerPage {
			totalPages := (m.completedTotal + itemsPerPage - 1) / itemsPerPage
			header += pageStyle.Render(fmt.Sprintf(" (page %d/%d)", m.completedPage+1, totalPages))
		}
		lines = append(lines, headerStyle.Render("Completed:")+pageStyle.Render(header[10:]))
		for _, r := range completedPage {
			name := normalStyle.Render(r.Workflow)
			if runIndex == m.selected {
				name = selectedStyle.Render(r.Workflow)
			}
			elapsed := formatElapsed(r.EndedAt)
			line := fmt.Sprintf("  %s %s    %s", completedIndicator, name, timeStyle.Render("Completed "+elapsed))
			lines = append(lines, line)
			runIndex++
		}
		lines = append(lines, "")
	}

	// Failed section (today, paginated)
	failedPage := m.getFailedPage()
	if len(failedPage) > 0 || m.failedTotal > 0 {
		header := "Failed:"
		if m.failedTotal > itemsPerPage {
			totalPages := (m.failedTotal + itemsPerPage - 1) / itemsPerPage
			header += pageStyle.Render(fmt.Sprintf(" (page %d/%d)", m.failedPage+1, totalPages))
		}
		lines = append(lines, headerStyle.Render("Failed:")+pageStyle.Render(header[7:]))
		for _, r := range failedPage {
			name := normalStyle.Render(r.Workflow)
			if runIndex == m.selected {
				name = selectedStyle.Render(r.Workflow)
			}
			elapsed := formatElapsed(r.EndedAt)
			errText := ""
			if r.Error != "" {
				errText = "\n      " + lipgloss.NewStyle().Foreground(theme.Error).Render(r.Error)
			}
			line := fmt.Sprintf("  %s %s    %s%s", failedIndicator, name, timeStyle.Render("Failed "+elapsed), errText)
			lines = append(lines, line)
			runIndex++
		}
		lines = append(lines, "")
	}

	// Hints
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	warningHintStyle := lipgloss.NewStyle().Foreground(theme.Warning)

	// Check if selected task needs attention for dismiss hint
	var selectedNeedsAttention bool
	if len(m.allRuns) > 0 && m.selected < len(m.allRuns) {
		selectedNeedsAttention = m.allRuns[m.selected].NeedsAttention
	}

	// Check for pending dismiss confirmation
	if m.confirm.IsPending("dismiss", "") {
		lines = append(lines, warningHintStyle.Render("Press d again to dismiss"))
	} else {
		hints := "[Enter] Details"
		if len(m.running) > 0 {
			hints += "  [c] Cancel"
		}
		if selectedNeedsAttention {
			hints += "  [d] Dismiss"
		}
		// Add pagination hints only if current section has multiple pages
		section := m.getSelectedSection()
		showPagination := false
		if section == "completed" && m.completedTotal > itemsPerPage {
			showPagination = true
		} else if section == "failed" && m.failedTotal > itemsPerPage {
			showPagination = true
		}
		if showPagination {
			hints += "  [n/p] Next/Prev page"
		}
		hints += "  [h] History"
		lines = append(lines, hintStyle.Render(hints))
	}

	return strings.Join(lines, "\n")
}

func (m *TasksModal) viewHistory() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Loading history...")
	}

	if m.error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			errorStyle.Render("Error: "+m.error),
			"",
			hintStyle.Render("[Esc] Back  [r] Retry"),
		)
	}

	if len(m.history) == 0 {
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Foreground(theme.TextSecondary).Render("No task history."),
			"",
			hintStyle.Render("[Esc] Back"),
		)
	}

	var lines []string

	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	attentionStyle := lipgloss.NewStyle().Foreground(theme.Warning).Bold(true)
	timeStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	pageStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	runningIndicator := lipgloss.NewStyle().Foreground(theme.Warning).Render("●")
	completedIndicator := lipgloss.NewStyle().Foreground(theme.Success).Render("✓")
	failedIndicator := lipgloss.NewStyle().Foreground(theme.Error).Render("✗")
	attentionIndicator := lipgloss.NewStyle().Foreground(theme.Warning).Bold(true).Render("⚠")

	// Show range and total count
	startItem := m.historyPage*historyItemsPerPage + 1
	endItem := startItem + len(m.history) - 1
	countText := fmt.Sprintf("Showing %d-%d of %d tasks", startItem, endItem, m.historyTotal)
	lines = append(lines, pageStyle.Render(countText))
	lines = append(lines, "")

	// Render history list
	for i, r := range m.history {
		name := normalStyle.Render(r.Workflow)
		if r.NeedsAttention {
			name = attentionStyle.Render(r.Workflow)
		}
		if i == m.selected {
			name = selectedStyle.Render(r.Workflow)
		}

		// Add attention indicator after name if needed
		if r.NeedsAttention {
			name += " " + attentionIndicator
		}

		// Determine status indicator and time text
		var indicator string
		var timeText string
		switch r.Status {
		case "running":
			indicator = runningIndicator
			timeText = "Started " + formatElapsed(r.StartedAt)
		case "completed":
			indicator = completedIndicator
			timeText = "Completed " + formatElapsed(r.EndedAt)
		default:
			indicator = failedIndicator
			timeText = "Failed " + formatElapsed(r.EndedAt)
		}

		line := fmt.Sprintf("  %s %s    %s", indicator, name, timeStyle.Render(timeText))
		lines = append(lines, line)
	}

	lines = append(lines, "")

	// Hints
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	warningHintStyle := lipgloss.NewStyle().Foreground(theme.Warning)

	// Check if selected task needs attention for dismiss hint
	var selectedNeedsAttention bool
	if len(m.history) > 0 && m.selected < len(m.history) {
		selectedNeedsAttention = m.history[m.selected].NeedsAttention
	}

	if m.confirm.IsPending("dismiss", "") {
		lines = append(lines, warningHintStyle.Render("Press d again to dismiss"))
	} else {
		hints := "[Esc] Back  [Enter] Details"
		if selectedNeedsAttention {
			hints += "  [d] Dismiss"
		}
		// Show pagination hints if there are multiple pages
		hasNextPage := m.historyHasMore
		hasPrevPage := m.historyPage > 0
		if hasNextPage || hasPrevPage {
			hints += "  [n/p] Next/Prev page"
		}
		lines = append(lines, hintStyle.Render(hints))
	}

	return strings.Join(lines, "\n")
}

func (m *TasksModal) viewDetail() string {
	if m.detailRun == nil {
		return "No task selected"
	}

	r := m.detailRun
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)

	var statusStyle lipgloss.Style
	switch r.Status {
	case "running":
		statusStyle = lipgloss.NewStyle().Foreground(theme.Warning)
	case "completed":
		statusStyle = lipgloss.NewStyle().Foreground(theme.Success)
	default:
		statusStyle = lipgloss.NewStyle().Foreground(theme.Error)
	}

	var lines []string

	statusLine := labelStyle.Render("Status:    ") + statusStyle.Render(r.Status)
	if r.NeedsAttention {
		attentionStyle := lipgloss.NewStyle().Foreground(theme.Warning).Bold(true)
		statusLine += "  " + attentionStyle.Render("⚠ Needs Attention")
	}
	lines = append(lines, statusLine)
	lines = append(lines, labelStyle.Render("Started:   ")+valueStyle.Render(formatTime(r.StartedAt)))

	// Show loading indicator or error for fetching full details
	if m.loadingDetail {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Loading details..."))
	} else if m.detailError != "" {
		lines = append(lines, "")
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		lines = append(lines, errorStyle.Render("Could not load full details: "+m.detailError))
		lines = append(lines, labelStyle.Render("(Run may have been cleaned up by hub-core)"))
	}

	if !r.EndedAt.IsZero() {
		lines = append(lines, labelStyle.Render("Ended:     ")+valueStyle.Render(formatTime(r.EndedAt)))
		duration := r.EndedAt.Sub(r.StartedAt)
		lines = append(lines, labelStyle.Render("Duration:  ")+valueStyle.Render(formatDuration(duration)))
	}

	if r.Error != "" {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Error:"))
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.Error).Render("  "+r.Error))
	}

	output := formatRunOutput(r.Result)
	if output != "" {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Output:"))
		// Indent output lines
		for _, line := range strings.Split(output, "\n") {
			lines = append(lines, "  "+valueStyle.Render(line))
		}
	}

	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	warningHintStyle := lipgloss.NewStyle().Foreground(theme.Warning)

	// Check for pending dismiss confirmation
	if m.confirm.IsPending("dismiss", r.ID) {
		lines = append(lines, warningHintStyle.Render("Press d again to dismiss"))
	} else {
		hints := "[Esc] Back  [r] Refresh"
		if r.Status == "running" {
			hints += "  [c] Cancel"
		}
		if r.NeedsAttention {
			hints += "  [d] Dismiss"
		}
		lines = append(lines, hintStyle.Render(hints))
	}

	return strings.Join(lines, "\n")
}

// cancelTask returns a command to reload tasks after cancelling.
func (m *TasksModal) cancelTask(runID string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.CancelRun(runID)
		if err != nil {
			return TasksLoadedMsg{Error: err}
		}

		// Reload tasks after cancel using the same logic as loadTasks
		today := time.Now().Format("2006-01-02")

		// Fetch all runs needing attention (any date)
		needsAttentionFilter := true
		attentionResp, err := m.client.ListRuns(&client.RunsFilter{
			NeedsAttention: &needsAttentionFilter,
		})
		if err != nil {
			return TasksLoadedMsg{Error: err}
		}

		// Fetch today's runs
		todayResp, err := m.client.ListRuns(&client.RunsFilter{
			Since: today,
		})
		if err != nil {
			return TasksLoadedMsg{Error: err}
		}

		// Build needs attention list
		var needsAttention []TaskRun
		attentionIDs := make(map[string]bool)
		for _, r := range attentionResp.Runs {
			attentionIDs[r.ID] = true
			needsAttention = append(needsAttention, clientRunToTaskRun(r))
		}

		// Build today's lists, excluding items already in needs attention
		var running, completed, failed []TaskRun
		for _, r := range todayResp.Runs {
			if attentionIDs[r.ID] {
				continue
			}
			tr := clientRunToTaskRun(r)
			if r.Status == "running" {
				running = append(running, tr)
			} else if isRunSuccess(r) {
				completed = append(completed, tr)
			} else {
				failed = append(failed, tr)
			}
		}

		sortByMostRecent(needsAttention)
		sortByMostRecent(running)
		sortByMostRecent(completed)
		sortByMostRecent(failed)

		return TasksLoadedMsg{
			NeedsAttention: needsAttention,
			Running:        running,
			Completed:      completed,
			Failed:         failed,
		}
	}
}

// dismissTask returns a command to dismiss a task that needs attention.
func (m *TasksModal) dismissTask(runID string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DismissRun(runID)
		return TaskDismissedMsg{RunID: runID, Error: err}
	}
}

// sortByMostRecent sorts tasks with needs_attention first, then by most recent.
func sortByMostRecent(tasks []TaskRun) {
	sort.Slice(tasks, func(i, j int) bool {
		ti, tj := tasks[i], tasks[j]

		// Needs attention items always come first
		if ti.NeedsAttention != tj.NeedsAttention {
			return ti.NeedsAttention
		}

		// Within same attention status, sort by time
		// For running tasks, sort by StartedAt; for completed/failed, sort by EndedAt
		if ti.Status == "running" {
			return ti.StartedAt.After(tj.StartedAt)
		}
		// Use EndedAt for completed/failed, fallback to StartedAt if zero
		endI, endJ := ti.EndedAt, tj.EndedAt
		if endI.IsZero() {
			endI = ti.StartedAt
		}
		if endJ.IsZero() {
			endJ = tj.StartedAt
		}
		return endI.After(endJ)
	})
}

// formatElapsed returns a human-readable elapsed time.
func formatElapsed(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	elapsed := time.Since(t)
	if elapsed < time.Minute {
		return "just now"
	} else if elapsed < time.Hour {
		mins := int(elapsed.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", mins)
	} else if elapsed < 24*time.Hour {
		hours := int(elapsed.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(elapsed.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// formatTime formats a time for display.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "< 1s"
	} else if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
