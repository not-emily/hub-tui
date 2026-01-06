package app

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/config"
	"github.com/pxp/hub-tui/internal/ui/chat"
	"github.com/pxp/hub-tui/internal/ui/components"
	"github.com/pxp/hub-tui/internal/ui/login"
	"github.com/pxp/hub-tui/internal/ui/modal"
	"github.com/pxp/hub-tui/internal/ui/status"
)

const quitHintDuration = 2 * time.Second

// AppState represents the current application state.
type AppState int

const (
	StateLogin AppState = iota
	StateMain
)

// Cache holds cached data from hub-core.
type Cache struct {
	Assistants []client.Assistant
	Workflows  []client.Workflow
	Modules    []client.Module
	LastUpdate time.Time
}

// Context represents the current conversation context.
type Context struct {
	Type   string // "hub", "assistant", "workflow", etc.
	Target string // Name of target (empty for hub)
}

// TaskState tracks running and completed workflow tasks.
type TaskState struct {
	Running   []Run
	Completed []Run
	Failed    []Run
}

// Model is the root Bubble Tea model for hub-tui.
type Model struct {
	config       *config.Config
	client       *client.Client
	program      *tea.Program // Reference for sending messages from goroutines
	cache        Cache
	context      Context   // Current conversation context
	tasks        TaskState // Workflow task tracking
	width        int
	height       int
	state        AppState
	quitting     bool
	ctrlCPressed bool
	cancelAsk    context.CancelFunc // Cancel function for streaming request

	// Sub-components
	login     login.Model
	chat      chat.Model
	statusBar status.Model
	modal     modal.State
}

// New creates a new app model with the given config.
func New(cfg *config.Config) Model {
	needsServerURL := cfg.ServerURL == ""
	needsLogin := needsServerURL || cfg.Token == "" || client.IsTokenExpired(cfg.Token)

	m := Model{
		config:    cfg,
		chat:      chat.New(),
		statusBar: status.New(),
		modal:     modal.NewState(),
	}

	if needsLogin {
		m.state = StateLogin
		m.login = login.New(needsServerURL, cfg.ServerURL)
	} else {
		m.state = StateMain
		m.client = client.New(cfg.ServerURL)
		m.client.SetToken(cfg.Token)
	}

	return m
}

// SetProgram sets the tea.Program reference for sending messages.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	if m.state == StateMain {
		// Verify connection with health check
		return m.doHealthCheck()
	}
	return nil
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SetProgramMsg:
		m.program = msg.Program
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.login.SetSize(msg.Width, msg.Height)
		m.statusBar.SetWidth(msg.Width)
		m.modal.SetWidth(msg.Width)
		// Chat gets height minus status bar
		m.chat.SetSize(msg.Width, msg.Height-1)
		return m, nil

	case tea.KeyMsg:
		// Global key handling
		if IsQuit(msg) {
			// Cancel any ongoing streaming
			if m.cancelAsk != nil {
				m.cancelAsk()
			}
			if m.ctrlCPressed {
				m.quitting = true
				return m, tea.Quit
			}
			m.ctrlCPressed = true
			m.login.SetCtrlCPressed(true)
			m.statusBar.SetCtrlCPressed(true)
			return m, tea.Tick(quitHintDuration, func(time.Time) tea.Msg {
				return QuitHintExpiredMsg{}
			})
		}

		if IsRedraw(msg) {
			return m, tea.ClearScreen
		}

		// Reset Ctrl+C state on any other key
		m.ctrlCPressed = false
		m.login.SetCtrlCPressed(false)
		m.statusBar.SetCtrlCPressed(false)

		// Route to modal if open
		if m.modal.IsOpen() {
			handled, cmd := m.modal.Update(msg)
			if handled {
				return m, cmd
			}
		}

		// Route to current state handler
		switch m.state {
		case StateLogin:
			return m.updateLogin(msg)
		case StateMain:
			return m.updateMain(msg)
		}

	case QuitHintExpiredMsg:
		m.ctrlCPressed = false
		m.login.SetCtrlCPressed(false)
		m.statusBar.SetCtrlCPressed(false)
		return m, nil

	case LoginResultMsg:
		return m.handleLoginResult(msg)

	case HealthCheckMsg:
		return m.handleHealthCheck(msg)

	case StreamChunkMsg:
		m.chat.AppendToLastMessage(msg.Content)
		return m, nil

	case StreamDoneMsg:
		m.chat.FinishLastMessage()
		m.cancelAsk = nil
		if msg.Error != nil {
			// Could show error to user here
		}
		return m, nil

	case RouteMsg:
		m.context.Type = msg.Type
		m.context.Target = msg.Target
		m.statusBar.SetContext(msg.Type, msg.Target)
		m.chat.SetInContext(msg.Type == "assistant" && msg.Target != "")
		return m, nil

	case AskNeedsInputMsg:
		// Add placeholder text to the pending hub message
		m.chat.AppendToLastMessage("Please complete the form below.")
		m.chat.FinishLastMessage()

		// Open parameter form modal
		formModal := modal.NewParamFormModal(msg.Target, msg.Schema)
		cmd := m.modal.Open(formModal)
		return m, cmd

	case AskExecutedMsg:
		// Replace placeholder with success message
		if msg.Result != nil && msg.Result.Message != "" {
			m.chat.ReplaceLastMessageContent(msg.Result.Message)
		} else {
			m.chat.ReplaceLastMessageContent("Done.")
		}
		return m, nil

	case AskErrorMsg:
		// Replace placeholder with error message
		if msg.Error != nil {
			m.chat.ReplaceLastMessageContent("Error: " + msg.Error.Message)
		} else {
			m.chat.ReplaceLastMessageContent("An error occurred.")
		}
		return m, nil

	case modal.ParamFormSubmitMsg:
		// Close modal and submit structured params
		m.modal.Close()
		return m, m.doAskWithParams(msg.Target, msg.Params)

	case modal.ParamFormCancelMsg:
		// User cancelled - close modal and replace placeholder
		m.modal.Close()
		m.chat.ReplaceLastMessageContent("Form cancelled.")
		return m, nil

	case CacheRefreshMsg:
		return m.handleCacheRefresh(msg)

	case AuthExpiredMsg:
		return m.handleAuthExpired()

	case modal.ModulesLoadedMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.ModuleToggledMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.WorkflowsLoadedMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.IntegrationsLoadedMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.IntegrationConfiguredMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.IntegrationTestedMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.LLMDataLoadedMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.LLMAvailableProvidersMsg:
		if msg.Err != nil && client.IsAuthError(msg.Err) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.LLMProviderSavedMsg:
		if msg.Err != nil && client.IsAuthError(msg.Err) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.LLMProviderDeletedMsg:
		if msg.Err != nil && client.IsAuthError(msg.Err) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.LLMErrorMsg:
		if msg.Err != nil && client.IsAuthError(msg.Err) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.LLMModelsLoadedMsg:
		if msg.Err != nil && client.IsAuthError(msg.Err) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.LLMProfileSavedMsg:
		if msg.Err != nil && client.IsAuthError(msg.Err) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.LLMProfileDeletedMsg:
		if msg.Err != nil && client.IsAuthError(msg.Err) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.LLMProfileTestedMsg:
		if msg.Err != nil && client.IsAuthError(msg.Err) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.LLMProfileDefaultSetMsg:
		if msg.Err != nil && client.IsAuthError(msg.Err) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.TasksLoadedMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.TaskDetailLoadedMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.HistoryLoadedMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case modal.TaskDismissedMsg:
		if msg.Error != nil && client.IsAuthError(msg.Error) {
			return m.handleAuthExpired()
		}
		var cmds []tea.Cmd
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			cmds = append(cmds, cmd)
		}
		// Fetch task status immediately to update status bar counts
		if msg.Error == nil {
			cmds = append(cmds, m.doFetchTaskStatus())
		}
		return m, tea.Batch(cmds...)

	case components.ConfirmationExpiredMsg:
		if m.modal.IsOpen() {
			_, cmd := m.modal.UpdateMsg(msg)
			return m, cmd
		}

	case WorkflowStartedMsg:
		return m.handleWorkflowStarted(msg)

	case WorkflowErrorMsg:
		m.chat.AddSystemMessage("Failed to start workflow " + msg.Name + ": " + msg.Error)
		return m, nil

	case PollTasksMsg:
		return m.handlePollTasks()

	case TaskStatusMsg:
		return m.handleTaskStatus(msg)

	case TaskCancelledMsg:
		if msg.Error != nil {
			m.chat.AddSystemMessage("Failed to cancel task: " + msg.Error.Error())
		}
		return m, nil
	}

	return m, nil
}

func (m Model) updateLogin(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for form submission
	if m.login.IsSubmit(msg) {
		if err := m.login.Validate(); err != "" {
			m.login.SetError(err)
			return m, nil
		}

		m.login.SetConnecting()

		// Create client with the server URL
		serverURL := m.login.ServerURL()
		if serverURL == "" {
			serverURL = m.config.ServerURL
		}
		m.client = client.New(serverURL)

		return m, m.doLogin(m.login.Username(), m.login.Password())
	}

	// Update login form
	var cmd tea.Cmd
	m.login, cmd = m.login.Update(msg)
	return m, cmd
}

func (m Model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle autocomplete navigation when visible
	if m.chat.IsAutocompleteVisible() {
		switch msg.String() {
		case "up":
			m.chat.AutocompleteUp()
			return m, nil
		case "down":
			m.chat.AutocompleteDown()
			return m, nil
		case "enter", "tab":
			m.chat.CompleteInput()
			return m, nil
		case "esc":
			m.chat.HideAutocomplete()
			return m, nil
		}
	}

	// Handle Tab to show autocomplete
	if msg.String() == "tab" && !m.chat.IsStreaming() {
		prefix, partial := m.chat.GetInputPrefix()
		suggestions := m.getSuggestions(prefix, partial)
		if len(suggestions) > 0 {
			m.chat.ShowAutocomplete(prefix, partial, suggestions)
		}
		return m, nil
	}

	// Handle Enter to send message
	if msg.String() == "enter" && !m.chat.IsStreaming() {
		input := m.chat.InputValue()
		if input != "" {
			// Check for slash command
			if cmd := chat.ParseCommand(input); cmd != nil {
				m.chat.ClearInput()
				return m.handleCommand(cmd)
			}

			// Check for # workflow trigger
			if len(input) > 1 && input[0] == '#' {
				workflowName := input[1:]
				m.chat.ClearInput()
				m.chat.AddSystemMessage("Starting workflow: " + workflowName)
				return m, m.doRunWorkflow(workflowName)
			}

			m.chat.AddUserMessage(input)
			m.chat.ClearInput()
			m.chat.AddHubMessage()

			// Route based on @ prefix and current target
			startsWithAt := len(input) > 0 && input[0] == '@'

			if startsWithAt {
				// @ prefix: always route through /ask (let hub-core decide)
				return m, m.doAsk(input)
			} else if m.context.Type == "assistant" && m.context.Target != "" {
				// No @ prefix but in assistant context: send directly to assistant
				return m, m.doAssistantChat(m.context.Target, input)
			} else {
				// No @ prefix, no assistant context: send to /ask
				return m, m.doAsk(input)
			}
		}
		return m, nil
	}

	// Hide autocomplete on any other key
	if m.chat.IsAutocompleteVisible() {
		m.chat.HideAutocomplete()
	}

	// Update chat
	var cmd tea.Cmd
	m.chat, cmd = m.chat.Update(msg)
	return m, cmd
}

func (m Model) getSuggestions(prefix chat.InputPrefix, partial string) []string {
	var items []string

	switch prefix {
	case chat.PrefixAssistant:
		for _, a := range m.cache.Assistants {
			items = append(items, a.Name)
		}
	case chat.PrefixWorkflow:
		for _, w := range m.cache.Workflows {
			items = append(items, w.Name)
		}
	case chat.PrefixCommand:
		items = chat.KnownCommands
	default:
		return nil
	}

	return chat.FilterSuggestions(items, partial)
}

func (m Model) handleCommand(cmd *chat.Command) (tea.Model, tea.Cmd) {
	switch cmd.Name {
	case "exit":
		m.quitting = true
		return m, tea.Quit

	case "clear":
		m.chat.ClearMessages()
		return m, nil

	case "hub":
		m.context.Type = "hub"
		m.context.Target = ""
		m.statusBar.SetContext("hub", "")
		m.chat.SetInContext(false)
		m.chat.AddSystemMessage("Returned to hub context.")
		return m, nil

	case "help":
		return m, m.modal.Open(modal.NewHelpModal())

	case "refresh":
		m.chat.AddSystemMessage("Refreshing cache...")
		return m, m.doRefreshCache()

	case "settings":
		return m, m.modal.Open(modal.NewSettingsModal(m.config, m.statusBar.IsConnected()))

	case "modules":
		return m, m.modal.Open(modal.NewModulesModal(m.client))

	case "workflows":
		return m, m.modal.Open(modal.NewWorkflowsModal(m.client))

	case "integrations":
		return m, m.modal.Open(modal.NewIntegrationsModal(m.client))

	case "tasks":
		// Convert app.Run to modal.TaskRun
		var running, completed, failed []modal.TaskRun
		for _, r := range m.tasks.Running {
			running = append(running, appRunToModalRun(r))
		}
		for _, r := range m.tasks.Completed {
			completed = append(completed, appRunToModalRun(r))
		}
		for _, r := range m.tasks.Failed {
			failed = append(failed, appRunToModalRun(r))
		}
		return m, m.modal.Open(modal.NewTasksModalWithState(m.client, running, completed, failed))

	default:
		if !chat.IsValidCommand(cmd.Name) {
			m.chat.AddSystemMessage("Unknown command: /" + cmd.Name + ". Type /help for available commands.")
		}
		return m, nil
	}
}

func (m Model) handleLoginResult(msg LoginResultMsg) (tea.Model, tea.Cmd) {
	if !msg.Success {
		m.login.SetError(msg.Error)
		return m, nil
	}

	// Store token and server URL in config
	m.config.ServerURL = m.client.BaseURL()
	m.config.Token = msg.Token
	m.config.TokenExp = msg.ExpiresAt
	if err := m.config.Save(); err != nil {
		m.login.SetError("Failed to save config: " + err.Error())
		return m, nil
	}

	// Set token on client
	m.client.SetToken(msg.Token)

	// Transition to main state
	m.state = StateMain
	m.statusBar.SetServerURL(m.client.BaseURL())
	m.statusBar.SetState(status.StateConnecting)

	// Set chat size
	m.chat.SetSize(m.width, m.height-1)
	m.chat.FocusInput()

	return m, m.doHealthCheck()
}

func (m Model) handleHealthCheck(msg HealthCheckMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		m.statusBar.SetState(status.StateConnected)
		// Trigger cache refresh and task loading after successful connection
		return m, tea.Batch(
			m.doRefreshCache(),
			m.doFetchTaskStatus(),
		)
	}
	m.statusBar.SetState(status.StateDisconnected)
	// If we were in login, show the error
	if m.state == StateLogin {
		m.login.SetError(msg.Error)
	}
	return m, nil
}

func (m Model) handleCacheRefresh(msg CacheRefreshMsg) (tea.Model, tea.Cmd) {
	if !msg.Success {
		m.chat.AddSystemMessage("Cache refresh failed: " + msg.Error)
		return m, nil
	}

	// Update cache with fresh data
	m.cache.LastUpdate = time.Now()

	// Convert names back to full structs (we only pass names in the message)
	m.cache.Assistants = make([]client.Assistant, len(msg.Assistants))
	for i, name := range msg.Assistants {
		m.cache.Assistants[i] = client.Assistant{Name: name}
	}
	m.cache.Workflows = make([]client.Workflow, len(msg.Workflows))
	for i, name := range msg.Workflows {
		m.cache.Workflows[i] = client.Workflow{Name: name}
	}
	m.cache.Modules = make([]client.Module, len(msg.Modules))
	for i, name := range msg.Modules {
		m.cache.Modules[i] = client.Module{Name: name}
	}

	return m, nil
}

func (m Model) handleAuthExpired() (tea.Model, tea.Cmd) {
	// Clear token from config
	m.config.Token = ""
	m.config.TokenExp = ""
	_ = m.config.Save() // Best effort save

	// Close any open modal
	m.modal.Close()

	// Reset to login state
	m.state = StateLogin
	m.login = login.New(false, m.config.ServerURL)
	m.login.SetSize(m.width, m.height)
	m.login.SetError("Session expired. Please log in again.")

	m.statusBar.SetState(status.StateDisconnected)

	return m, nil
}

func (m Model) doLogin(username, password string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.Login(username, password)
		if err != nil {
			return LoginResultMsg{Success: false, Error: err.Error()}
		}
		return LoginResultMsg{
			Success:   true,
			Token:     resp.Token,
			ExpiresAt: resp.ExpiresAt,
		}
	}
}

func (m Model) doHealthCheck() tea.Cmd {
	return func() tea.Msg {
		if err := m.client.Health(); err != nil {
			return HealthCheckMsg{Success: false, Error: err.Error()}
		}
		return HealthCheckMsg{Success: true}
	}
}

func (m Model) doRefreshCache() tea.Cmd {
	return func() tea.Msg {
		var assistantNames, workflowNames, moduleNames []string

		// Fetch assistants
		assistants, err := m.client.ListAssistants()
		if err != nil {
			if client.IsAuthError(err) {
				return AuthExpiredMsg{}
			}
			return CacheRefreshMsg{Success: false, Error: "assistants: " + err.Error()}
		}
		for _, a := range assistants {
			assistantNames = append(assistantNames, a.Name)
		}

		// Fetch workflows
		workflows, err := m.client.ListWorkflows()
		if err != nil {
			if client.IsAuthError(err) {
				return AuthExpiredMsg{}
			}
			return CacheRefreshMsg{Success: false, Error: "workflows: " + err.Error()}
		}
		for _, w := range workflows {
			workflowNames = append(workflowNames, w.Name)
		}

		// Fetch modules
		modules, err := m.client.ListModules()
		if err != nil {
			if client.IsAuthError(err) {
				return AuthExpiredMsg{}
			}
			return CacheRefreshMsg{Success: false, Error: "modules: " + err.Error()}
		}
		for _, m := range modules {
			moduleNames = append(moduleNames, m.Name)
		}

		return CacheRefreshMsg{
			Success:    true,
			Assistants: assistantNames,
			Workflows:  workflowNames,
			Modules:    moduleNames,
		}
	}
}

func (m *Model) doAsk(message string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelAsk = cancel

	return func() tea.Msg {
		callbacks := client.AskCallbacks{
			OnRoute: func(route client.RouteInfo) {
				if m.program != nil {
					m.program.Send(RouteMsg{Type: route.Type, Target: route.Target})
				}
			},
			OnChunk: func(chunk string) {
				if m.program != nil {
					m.program.Send(StreamChunkMsg{Content: chunk})
				}
			},
		}

		resp, err := m.client.Ask(ctx, message, callbacks)
		if err != nil {
			return StreamDoneMsg{Error: err}
		}

		// Handle status-based response
		switch resp.Status {
		case client.StatusNeedsInput:
			return AskNeedsInputMsg{
				Target: resp.Target,
				Schema: resp.Schema,
			}
		case client.StatusExecuted:
			return AskExecutedMsg{
				Target: resp.Target,
				Result: resp.Result,
			}
		case client.StatusError:
			return AskErrorMsg{
				Target: resp.Target,
				Error:  resp.Error,
			}
		default:
			// Legacy response format (assistant chat, etc.) - no status field
			return StreamDoneMsg{Error: nil}
		}
	}
}

// doAskWithParams sends structured params to the API.
func (m *Model) doAskWithParams(target string, params map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		req := client.AskRequest{
			Target: target,
			Params: params,
		}

		resp, err := m.client.AskDirect(req)
		if err != nil {
			return AskErrorMsg{
				Target: target,
				Error: &client.AskError{
					Code:    "request_failed",
					Message: err.Error(),
				},
			}
		}

		switch resp.Status {
		case client.StatusNeedsInput:
			// Validation errors - reopen form with errors
			return AskNeedsInputMsg{
				Target: resp.Target,
				Schema: resp.Schema,
			}
		case client.StatusExecuted:
			return AskExecutedMsg{
				Target: resp.Target,
				Result: resp.Result,
			}
		case client.StatusError:
			return AskErrorMsg{
				Target: resp.Target,
				Error:  resp.Error,
			}
		default:
			return AskErrorMsg{
				Target: target,
				Error: &client.AskError{
					Code:    "unknown_status",
					Message: "Unexpected response status: " + resp.Status,
				},
			}
		}
	}
}

func (m *Model) doAssistantChat(assistant, message string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelAsk = cancel

	return func() tea.Msg {
		callbacks := client.AssistantChatCallbacks{
			OnAssistant: func(info client.AssistantInfo) {
				// Confirm we're talking to the right assistant
				if m.program != nil {
					m.program.Send(RouteMsg{Type: "assistant", Target: info.Name})
				}
			},
			OnChunk: func(chunk string) {
				if m.program != nil {
					m.program.Send(StreamChunkMsg{Content: chunk})
				}
			},
		}

		_, err := m.client.AssistantChat(ctx, assistant, message, callbacks)
		return StreamDoneMsg{Error: err}
	}
}

func (m Model) doRunWorkflow(name string) tea.Cmd {
	return func() tea.Msg {
		runID, err := m.client.RunWorkflow(name)
		if err != nil {
			if client.IsAuthError(err) {
				return AuthExpiredMsg{}
			}
			return WorkflowErrorMsg{Name: name, Error: err.Error()}
		}
		return WorkflowStartedMsg{Name: name, RunID: runID}
	}
}

func (m Model) handleWorkflowStarted(msg WorkflowStartedMsg) (tea.Model, tea.Cmd) {
	// Add to running tasks
	m.tasks.Running = append(m.tasks.Running, Run{
		ID:        msg.RunID,
		Workflow:  msg.Name,
		Status:    "running",
		StartedAt: time.Now().Format(time.RFC3339),
	})

	// Update status bar
	m.updateTaskCounts()

	// Start polling if this is the first running task
	if len(m.tasks.Running) == 1 {
		return m, m.pollTasks()
	}
	return m, nil
}

func (m Model) handlePollTasks() (tea.Model, tea.Cmd) {
	// Only poll if there are running tasks
	if len(m.tasks.Running) == 0 {
		return m, nil
	}

	return m, tea.Batch(
		m.doFetchTaskStatus(),
		m.pollTasks(),
	)
}

func (m Model) pollTasks() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return PollTasksMsg{}
	})
}

func (m Model) doFetchTaskStatus() tea.Cmd {
	return func() tea.Msg {
		// Fetch today's tasks for status bar counts
		today := time.Now().Format("2006-01-02")
		response, err := m.client.ListRuns(&client.RunsFilter{
			Since: today,
		})
		if err != nil {
			if client.IsAuthError(err) {
				return AuthExpiredMsg{}
			}
			return TaskStatusMsg{Error: err}
		}

		// Convert client.Run to app.Run
		var appRuns []Run
		for _, r := range response.Runs {
			appRuns = append(appRuns, Run{
				ID:             r.ID,
				Workflow:       r.Workflow,
				Status:         r.Status,
				StartedAt:      r.StartedAt.Format(time.RFC3339),
				EndedAt:        r.EndedAt.Format(time.RFC3339),
				Error:          r.Error,
				Result:         convertClientResult(r.Result),
				NeedsAttention: r.NeedsAttention,
			})
		}

		return TaskStatusMsg{Runs: appRuns}
	}
}

func (m Model) handleTaskStatus(msg TaskStatusMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		// Could log error, but keep polling
		return m, nil
	}

	// Build map of API runs for quick lookup
	apiRuns := make(map[string]Run)
	for _, r := range msg.Runs {
		apiRuns[r.ID] = r
	}

	// Track which IDs we've already processed (to avoid duplicates)
	processedIDs := make(map[string]bool)

	// Build new task lists
	var newRunning, newCompleted, newFailed []Run

	// Process currently tracked running tasks - detect state transitions
	for _, tracked := range m.tasks.Running {
		processedIDs[tracked.ID] = true
		if apiRun, found := apiRuns[tracked.ID]; found {
			if apiRun.Status == "running" {
				newRunning = append(newRunning, apiRun)
			} else if isRunSuccess(apiRun) {
				newCompleted = append(newCompleted, apiRun)
				m.chat.AddSystemMessage("Workflow completed: " + apiRun.Workflow)
			} else {
				newFailed = append(newFailed, apiRun)
				errMsg := apiRun.Workflow
				if apiRun.Error != "" {
					errMsg += ": " + apiRun.Error
				} else if apiRun.Result != nil && apiRun.Result.Error != "" {
					errMsg += ": " + apiRun.Result.Error
				}
				m.chat.AddSystemMessage("Workflow failed: " + errMsg)
			}
		} else {
			// Task disappeared from API - assume completed
			m.chat.AddSystemMessage("Workflow completed: " + tracked.Workflow)
			tracked.Status = "completed"
			newCompleted = append(newCompleted, tracked)
		}
	}

	// Update existing completed tasks from API (syncs NeedsAttention, etc.)
	for _, tracked := range m.tasks.Completed {
		if processedIDs[tracked.ID] {
			continue // Already processed from running transition
		}
		processedIDs[tracked.ID] = true
		if apiRun, found := apiRuns[tracked.ID]; found {
			newCompleted = append(newCompleted, apiRun)
		} else {
			newCompleted = append(newCompleted, tracked)
		}
	}

	// Update existing failed tasks from API (syncs NeedsAttention, etc.)
	for _, tracked := range m.tasks.Failed {
		if processedIDs[tracked.ID] {
			continue // Already processed
		}
		processedIDs[tracked.ID] = true
		if apiRun, found := apiRuns[tracked.ID]; found {
			newFailed = append(newFailed, apiRun)
		} else {
			newFailed = append(newFailed, tracked)
		}
	}

	// Add any runs from API that we're not already tracking (startup/refresh case)
	for _, r := range msg.Runs {
		if processedIDs[r.ID] {
			continue
		}
		if r.Status == "running" {
			newRunning = append(newRunning, r)
		} else if isRunSuccess(r) {
			newCompleted = append(newCompleted, r)
		} else {
			newFailed = append(newFailed, r)
		}
	}

	m.tasks.Running = newRunning
	m.tasks.Completed = newCompleted
	m.tasks.Failed = newFailed

	// Update status bar
	m.updateTaskCounts()

	return m, nil
}

func (m *Model) updateTaskCounts() {
	// Count items needing attention across all categories
	needsAttention := 0
	for _, r := range m.tasks.Running {
		if r.NeedsAttention {
			needsAttention++
		}
	}
	for _, r := range m.tasks.Completed {
		if r.NeedsAttention {
			needsAttention++
		}
	}
	for _, r := range m.tasks.Failed {
		if r.NeedsAttention {
			needsAttention++
		}
	}
	m.statusBar.SetTaskCounts(len(m.tasks.Running), needsAttention)
}

// isRunSuccess returns true if the run completed successfully.
// A run is successful if status is "completed" AND result.success is true (or result is nil).
func isRunSuccess(r Run) bool {
	if r.Status == "failed" {
		return false
	}
	if r.Result != nil && !r.Result.Success {
		return false
	}
	return true
}

func (m Model) doCancelTask(runID string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.CancelRun(runID)
		if err != nil {
			if client.IsAuthError(err) {
				return AuthExpiredMsg{}
			}
		}
		return TaskCancelledMsg{RunID: runID, Error: err}
	}
}

// appRunToModalRun converts an app.Run to a modal.TaskRun.
func appRunToModalRun(r Run) modal.TaskRun {
	var startedAt, endedAt time.Time
	if r.StartedAt != "" {
		startedAt, _ = time.Parse(time.RFC3339, r.StartedAt)
	}
	if r.EndedAt != "" {
		endedAt, _ = time.Parse(time.RFC3339, r.EndedAt)
	}
	return modal.TaskRun{
		ID:             r.ID,
		Workflow:       r.Workflow,
		Status:         r.Status,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		Error:          r.Error,
		Result:         convertAppResultToClient(r.Result),
		NeedsAttention: r.NeedsAttention,
	}
}

// convertClientResult converts client.RunResult to app.RunResult.
func convertClientResult(cr *client.RunResult) *RunResult {
	if cr == nil {
		return nil
	}
	var steps []StepResult
	for _, s := range cr.Steps {
		steps = append(steps, StepResult{
			StepName: s.StepName,
			Success:  s.Success,
			Output:   s.Output,
			Error:    s.Error,
		})
	}
	return &RunResult{
		WorkflowName: cr.WorkflowName,
		Success:      cr.Success,
		Steps:        steps,
		Error:        cr.Error,
	}
}

// convertAppResultToClient converts app.RunResult to client.RunResult for modal.
func convertAppResultToClient(ar *RunResult) *client.RunResult {
	if ar == nil {
		return nil
	}
	var steps []client.StepResult
	for _, s := range ar.Steps {
		steps = append(steps, client.StepResult{
			StepName: s.StepName,
			Success:  s.Success,
			Output:   s.Output,
			Error:    s.Error,
		})
	}
	return &client.RunResult{
		WorkflowName: ar.WorkflowName,
		Success:      ar.Success,
		Steps:        steps,
		Error:        ar.Error,
	}
}

// View renders the UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	switch m.state {
	case StateLogin:
		return m.login.View()
	case StateMain:
		return m.renderMain()
	}

	return ""
}

func (m Model) renderMain() string {
	// Status bar at bottom
	statusBar := m.statusBar.View()

	// If modal is open, show: messages → modal → input → status bar
	if m.modal.IsOpen() {
		modalView := m.modal.View()
		modalHeight := lipgloss.Height(modalView)

		inputView := m.chat.ViewInputOnly()
		inputHeight := lipgloss.Height(inputView)
		statusHeight := lipgloss.Height(statusBar)

		// Calculate remaining height for messages
		// The -2 accounts for: chat's internal -1 plus 1 line spacer above modal
		messagesHeight := m.height - modalHeight - inputHeight - statusHeight - 2
		if messagesHeight < 0 {
			messagesHeight = 0
		}

		messagesView := m.chat.ViewMessagesOnly(messagesHeight)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			messagesView,
			"", // Spacer between messages and modal
			modalView,
			inputView,
			statusBar,
		)
	}

	// Normal view: chat + status bar
	chatView := m.chat.View()
	return lipgloss.JoinVertical(
		lipgloss.Left,
		chatView,
		statusBar,
	)
}
