//go:build ignore
// +build ignore

// Copyright 2025 Antenore Gatta
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chat

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/antenore/deecli/internal/ai"
	"github.com/antenore/deecli/internal/api"
	apiHandler "github.com/antenore/deecli/internal/chat/api"
	"github.com/antenore/deecli/internal/chat/commands"
	"github.com/antenore/deecli/internal/chat/input"
	"github.com/antenore/deecli/internal/chat/keydetect"
	"github.com/antenore/deecli/internal/chat/messages"
	"github.com/antenore/deecli/internal/chat/state"
	"github.com/antenore/deecli/internal/chat/streaming"
	toolsManager "github.com/antenore/deecli/internal/chat/tools"
	"github.com/antenore/deecli/internal/chat/tracker"
	"github.com/antenore/deecli/internal/chat/ui"
	viewportmgr "github.com/antenore/deecli/internal/chat/viewport"
	"github.com/antenore/deecli/internal/config"
	"github.com/antenore/deecli/internal/editor"
	"github.com/antenore/deecli/internal/files"
	"github.com/antenore/deecli/internal/history"
	"github.com/antenore/deecli/internal/permissions"
	"github.com/antenore/deecli/internal/sessions"
	"github.com/antenore/deecli/internal/tools"
	"github.com/antenore/deecli/internal/tools/functions"
	"github.com/antenore/deecli/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel represents the new, streamlined chat interface that coordinates between components
type NewModel struct {
	// Core components
	fileContext      *files.FileContext
	apiClient        *api.Service
	configManager    *config.Manager
	completionEngine *CompletionEngine
	
	// Extracted managers
	stateManager     *state.Manager
	toolsManager     *toolsManager.Manager  
	apiHandler       *apiHandler.Handler
	
	// UI and interaction
	commandHandler   *commands.Handler
	renderer         *ui.Renderer
	sidebar          *ui.Sidebar
	aiOperations     *ai.Operations
	viewportManager  *viewportmgr.Manager
	spinner          *ui.Spinner
	keyDetector      *keydetect.Detector
	messageManager   *messages.Manager
	inputManager     *input.Manager
	streamingManager *streaming.Manager
	
	// Session and persistence
	sessionManager   *sessions.Manager
	currentSession   *sessions.Session
	sessionLoader    *sessions.Loader
	fileTracker      *tracker.FileTracker
	
	// Core state (minimal)
	messages         []string
	apiMessages      []api.Message
	streamReader     api.StreamReader
	streamContent    string
	streamingEnabled bool
}

// NewNewModel creates a new refactored model with proper dependency injection
func NewNewModel(configManager *config.Manager, apiKey, model string, temperature float64, maxTokens int) *NewModel {
	// Create API client
	client := createAPIClient(configManager, apiKey, model, temperature, maxTokens)
	
	// Get terminal dimensions
	width, height := utils.GetTerminalSize()
	
	// Initialize file context
	fileCtx := files.NewFileContext()
	setupFileWatcher(fileCtx, configManager)
	
	// Initialize session management
	sessionMgr, currentSession := initializeSessionManagement(configManager)
	
	// Initialize core components
	completionEngine := NewCompletionEngine()
	renderer := ui.NewRenderer(configManager)
	layoutManager := ui.NewLayout(configManager)
	sidebar := ui.NewSidebar()
	aiOperations := ai.NewOperations(client, fileCtx, configManager)
	spinner := ui.NewDefaultSpinner()
	fileTracker := tracker.NewFileTracker()
	streamingManager := streaming.NewManager()
	
	// Initialize history and input management
	historyMgr, historyData := initializeHistory()
	
	// Initialize extracted managers
	stateManager := state.NewManager(state.Dependencies{
		LayoutManager: layoutManager,
		InputManager: input.NewManager(historyData, historyMgr, completionEngine,
			func(role, content string) {}, // Will be set later
			func() {},                     // Will be set later
		),
	})
	
	stateManager.InitializeViewports(width, height)
	stateManager.CreateTextarea(width, configManager)
	
	// Initialize tools management
	toolsManager := initializeToolsManager(configManager)
	
	// Initialize API handler
	apiHandler := apiHandler.NewHandler(apiHandler.Dependencies{
		FileTracker: fileTracker,
	})
	
	// Create the model
	model := &NewModel{
		fileContext:      fileCtx,
		apiClient:        client,
		configManager:    configManager,
		completionEngine: completionEngine,
		stateManager:     stateManager,
		toolsManager:     toolsManager,
		apiHandler:       apiHandler,
		renderer:         renderer,
		sidebar:          sidebar,
		aiOperations:     aiOperations,
		spinner:          spinner,
		sessionManager:   sessionMgr,
		currentSession:   currentSession,
		fileTracker:      fileTracker,
		streamingManager: streamingManager,
		messages:         []string{},
		apiMessages:      []api.Message{},
		streamingEnabled: true,
	}
	
	// Initialize remaining components that need model reference
	model.initializeDependentComponents(historyData, historyMgr)
	
	// Set up initial UI state
	model.setupInitialState()
	
	return model
}

// Initialize components that need references to the model
func (m *NewModel) initializeDependentComponents(historyData []string, historyMgr *history.Manager) {
	// Initialize message manager
	m.messageManager = messages.NewManager(messages.Dependencies{
		Renderer:       m.renderer,
		Spinner:        m.spinner,
		SessionManager: m.sessionManager,
		CurrentSession: m.currentSession,
		AIOperations:   m.aiOperations,
	})
	
	// Initialize input manager with proper callbacks
	m.inputManager = input.NewManager(
		historyData,
		historyMgr,
		m.completionEngine,
		m.addMessage,
		m.refreshViewport,
	)
	
	// Update state manager with input manager
	m.stateManager = state.NewManager(state.Dependencies{
		LayoutManager: m.stateManager.(*state.Manager).GetLayoutManager(), // Get existing layout manager
		InputManager:  m.inputManager,
	})
	
	// Initialize key detector
	if m.configManager != nil {
		m.keyDetector = keydetect.New(keydetect.Dependencies{
			ConfigManager:  m.configManager,
			MessageLogger:  m.addMessage,
			RefreshView:    m.refreshViewport,
			LayoutManager:  m.stateManager.(*state.Manager).GetLayoutManager(),
			UpdateKeymap:   func() { m.keyDetector.UpdateTextareaKeymap(m.stateManager.GetTextarea()) },
		})
	}
	
	// Initialize command handler
	m.commandHandler = commands.NewHandler(m.createCommandDependencies())
	
	// Initialize viewport manager
	m.viewportManager = viewportmgr.NewManager(viewportmgr.Dependencies{
		Viewport:         m.stateManager.GetViewport(),
		Renderer:         m.renderer,
		LayoutManager:    m.stateManager.(*state.Manager).GetLayoutManager(),
		ConfigManager:    m.configManager,
		SessionManager:   m.sessionManager,
		CurrentSession:   m.currentSession,
		Spinner:          m.spinner,
		Messages:         &m.messages,
		APIMessages:      &m.apiMessages,
		FilesWidgetVisible: func() *bool { 
			visible := m.stateManager.IsFilesWidgetVisible()
			return &visible
		}(),
		IsLoading:        func() *bool { 
			loading := m.stateManager.IsLoading()
			return &loading
		}(),
		LoadingMsg:       func() *string {
			msg := m.stateManager.GetLoadingMessage()
			return &msg
		}(),
	})
	
	// Initialize session loader
	if m.sessionManager != nil && m.currentSession != nil {
		m.sessionLoader = sessions.NewLoader(&sessions.LoaderDependencies{
			SessionManager:       m.sessionManager,
			CurrentSession:       m.currentSession,
			Renderer:            m.renderer,
			Viewport:            m.stateManager.GetViewport(),
			ViewportWidth:       m.stateManager.GetViewport().Width,
			FilesWidgetVisible:  m.stateManager.IsFilesWidgetVisible(),
			FormatInitialContent: func() string {
				if m.viewportManager != nil {
					return m.viewportManager.FormatInitialContent()
				}
				return "Welcome to DeeCLI"
			},
		})
	}
}

// Setup initial UI state
func (m *NewModel) setupInitialState() {
	// Add welcome message
	if m.viewportManager != nil {
		m.messages = append(m.messages, m.viewportManager.FormatInitialContent())
	} else {
		m.messages = append(m.messages, "Welcome to DeeCLI")
	}
	
	content := strings.Join(m.messages, "\n\n")
	m.stateManager.SetViewportContent(content)
}

// Init implements the tea.Model interface
func (m *NewModel) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model interface - now focuses only on coordination
func (m *NewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle spinner animation
	if spinnerCmd := m.spinner.Update(msg); spinnerCmd != nil {
		cmds = append(cmds, spinnerCmd)
	}
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowResize(msg, cmds)
		
	case tea.KeyMsg:
		return m.handleKeyInput(msg, cmds)
		
	case ai.APIResponseMsg:
		m.handleAPIResponse(msg.Response, msg.Err)
		
	case ai.ToolCallsResponseMsg:
		if cmd := m.toolsManager.HandleToolCallsResponse(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
		
	case toolsManager.ToolExecutionCompleteMsg:
		if cmd := m.handleToolExecutionComplete(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
		
	case toolsManager.CreateApprovalDialogMsg:
		m.toolsManager.CreateApprovalDialog(msg.ApprovalRequest, m.stateManager.GetDimensions())
		
	case toolsManager.RequestToolApprovalMsg:
		// Handle next tool approval request
		return m, m.toolsManager.requestToolApproval(msg.ToolCall)
		
	case toolsManager.TriggerFollowupMsg:
		return m, m.triggerFollowupCall()
		
	case ai.StreamStartedMsg, ai.StreamChunkMsg, ai.StreamCompleteMsg:
		return m.handleStreamingMessages(msg, cmds)
		
	case editor.EditorFinishedMsg:
		m.handleEditorFinished(msg)
		
	default:
		if err, ok := msg.(error); ok {
			m.addMessage("system", fmt.Sprintf("❌ Unexpected error: %v", err))
		}
	}
	
	return m, tea.Batch(cmds...)
}

// View implements the tea.Model interface
func (m *NewModel) View() string {
	if !m.stateManager.IsReady() {
		return "\n  Initializing..."
	}
	
	// Handle approval dialog display
	if m.toolsManager.IsShowingApproval() {
		dialog := m.toolsManager.GetApprovalDialog()
		if dialog != nil {
			// Show header + dialog
			filesCount := len(m.fileContext.Files)
			header := m.renderer.(*ui.Renderer).GetLayoutManager().RenderHeader(
				filesCount, m.stateManager.GetFocusMode(), m.fileContext, m.renderer,
			)
			return fmt.Sprintf("%s\n%s", header, m.toolsManager.GetApprovalDialogView())
		}
	}
	
	// Normal view composition
	return m.renderNormalView()
}

// Helper methods for Update coordination

func (m *NewModel) handleWindowResize(msg tea.WindowSizeMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	m.stateManager.HandleResize(msg.Width, msg.Height)
	return m, tea.Batch(cmds...)
}

func (m *NewModel) handleKeyInput(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	// Handle tool approval dialog first
	if m.toolsManager.IsShowingApproval() {
		done, response := m.toolsManager.UpdateApprovalDialog(msg.String())
		if done && response != nil {
			m.toolsManager.SetShowingApproval(false)
			m.toolsManager.ClearApprovalDialog()
			return m, m.toolsManager.ExecuteApprovedTool(*response)
		}
		return m, nil
	}
	
	// Handle key detection mode
	if m.keyDetector != nil && m.keyDetector.IsDetecting() {
		return m, m.keyDetector.HandleDetection(msg.String())
	}
	
	// Handle global keys
	if cmd := m.handleGlobalKeys(msg.String()); cmd != nil {
		return m, cmd
	}
	
	// Handle focus-specific keys
	return m.handleFocusKeys(msg, cmds)
}

func (m *NewModel) handleGlobalKeys(key string) tea.Cmd {
	switch key {
	case "ctrl+c":
		return tea.Quit
	case "esc":
		return m.handleEscape()
	case "f1":
		return m.handleF1Toggle()
	case "f2":
		return m.handleF2Toggle()
	case "f3":
		return m.handleF3Toggle()
	}
	return nil
}

// Create command dependencies
func (m *NewModel) createCommandDependencies() commands.Dependencies {
	var inputHistory []string
	var historyManager *history.Manager

	if m.inputManager != nil {
		inputHistory = m.inputManager.GetInputHistory()
		historyManager = m.inputManager.GetHistoryManager()
	}

	return commands.Dependencies{
		FileContext:      m.fileContext,
		APIClient:        m.apiClient,
		ConfigManager:    m.configManager,
		SessionManager:   m.sessionManager,
		CurrentSession:   m.currentSession,
		HistoryManager:   historyManager,
		FileTracker:      m.fileTracker,
		ToolsRegistry:    m.toolsManager.(*toolsManager.Manager).GetToolsRegistry(),
		Messages:         m.messages,
		APIMessages:      m.apiMessages,
		InputHistory:     inputHistory,
		HelpVisible:      m.stateManager.IsHelpVisible(),
		MessageLogger:    m.addMessage,
		SetLoading:       m.setLoading,
		SetCancel:        m.setCancel,
		RefreshUI:        m.refreshViewport,
		ShowHistory:      func() { m.inputManager.ShowHistory() },
		AnalyzeFiles:     m.analyzeFiles,
		ExplainFiles:     m.explainFiles,
		ImproveFiles:     m.improveFiles,
		GenerateEditSuggestions: m.generateEditSuggestions,
		SetHelpVisible:   m.setHelpVisible,
		SetKeyDetection:  m.keyDetector.SetDetection,
	}
}

// Message and viewport management
func (m *NewModel) addMessage(role, content string) {
	viewportWrapper := messages.NewViewportWrapper(m.stateManager.GetViewport())
	m.messageManager.AddMessage(role, content, viewportWrapper, m.stateManager.IsFilesWidgetVisible())
	m.messages = m.messageManager.GetMessages()
	m.apiMessages = m.messageManager.GetAPIMessages()
}

func (m *NewModel) refreshViewport() {
	viewportWrapper := messages.NewViewportWrapper(m.stateManager.GetViewport())
	m.messageManager.RefreshViewport(viewportWrapper, m.stateManager.IsLoading(), m.stateManager.GetLoadingMessage())
}

// State management helpers
func (m *NewModel) setLoading(loading bool, message string) tea.Cmd {
	m.stateManager.SetLoading(loading, message)
	if loading {
		return m.spinner.Start()
	} else {
		m.spinner.Stop()
		return nil
	}
}

func (m *NewModel) setCancel(cancel context.CancelFunc) {
	m.stateManager.SetAPICancel(cancel)
}

func (m *NewModel) setHelpVisible(visible bool) {
	m.stateManager.SetHelpVisible(visible)
	if visible && m.viewportManager != nil {
		m.stateManager.SetViewportContent(m.viewportManager.HelpContent())
	} else {
		m.refreshViewport()
	}
}

// API response handling
func (m *NewModel) handleAPIResponse(response string, err error) {
	m.setLoading(false, "")
	m.stateManager.ClearAPICancel()
	
	result := m.apiHandler.HandleResponse(
		response, err, 
		m.toolsManager.ShouldSuppressToolCalls(), 
		m.fileContext,
	)
	
	if result.ShouldSuppress {
		m.toolsManager.ClearSuppressToolCalls()
	}
	
	if !result.Success {
		m.addMessage("system", result.ErrorMessage)
	} else if len(result.ToolCalls) > 0 {
		toolMsg := ai.ToolCallsResponseMsg{ToolCalls: result.ToolCalls}
		m.toolsManager.HandleToolCallsResponse(toolMsg)
	} else if result.AssistantContent != "" {
		m.addMessage("assistant", result.AssistantContent)
	}
	
	m.stateManager.GoToBottom()
}

// Tool execution handling
func (m *NewModel) handleToolExecutionComplete(msg toolsManager.ToolExecutionCompleteMsg) tea.Cmd {
	cmd, success := m.toolsManager.HandleToolExecutionComplete(msg, m.aiOperations)
	
	if !success {
		if msg.Error != nil {
			m.addMessage("system", fmt.Sprintf("❌ Tool execution failed: %v", msg.Error))
		} else if msg.Result != nil && !msg.Result.Success {
			m.addMessage("system", fmt.Sprintf("❌ Tool execution failed: %s", msg.Result.Error))
		} else {
			m.addMessage("system", "❌ Tool execution returned no result")
		}
		return nil
	}
	
	// Show successful tool output
	m.addMessage("system", fmt.Sprintf("🔧 %s result:\n\n%s", msg.ToolCall.Function.Name, msg.Result.Output))
	
	// Update API messages with tool call and result
	m.apiMessages = append(m.apiMessages, api.Message{
		Role:      "assistant",
		Content:   "",
		ToolCalls: []api.ToolCall{msg.ToolCall},
	})
	
	m.apiMessages = append(m.apiMessages, api.Message{
		Role:       "tool",
		Content:    msg.Result.Output,
		ToolCallID: msg.ToolCall.ID,
	})
	
	// Sync conversation history
	m.aiOperations.SetAPIMessages(m.apiMessages)
	
	return cmd
}

// Streaming handling
func (m *NewModel) handleStreamingMessages(msg tea.Msg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	// Delegate to streaming manager
	switch msg := msg.(type) {
	case ai.StreamStartedMsg:
		if cmd := m.setLoading(true, "Thinking..."); cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.refreshViewport()
		nextCmd := m.streamingManager.StartStream(msg, m.renderer, &m.messages)
		return m, nextCmd
		
	case ai.StreamChunkMsg:
		nextCmd, extraCmds := m.streamingManager.HandleChunk(msg, m.spinner, func() *bool {
			loading := m.stateManager.IsLoading()
			return &loading
		}(), m.setLoading)
		
		if extraCmds != nil {
			cmds = append(cmds, extraCmds...)
		}
		if nextCmd != nil {
			cmds = append(cmds, nextCmd)
		}
		
		m.streamingManager.UpdateDisplay(m.streamingManager.GetStreamContent(), m.renderer, &m.messages, m.stateManager.GetViewport())
		
		if m.messageManager != nil {
			m.messageManager.SetMessages(m.messages)
		}
		
		return m, tea.Batch(cmds...)
		
	case ai.StreamCompleteMsg:
		return m, m.streamingManager.CompleteStream(msg)
	}
	
	return m, tea.Batch(cmds...)
}

// Additional helper methods would continue here...
// (The rest would follow the same pattern of coordination rather than direct implementation)

// Helper initialization functions

func initializeHistory() (*history.Manager, []string) {
	historyMgr, err := history.NewManager()
	var historyData []string
	if err == nil && historyMgr != nil {
		historyData, _ = historyMgr.Load()
	}
	return historyMgr, historyData
}

func initializeSessionManagement(configManager *config.Manager) (*sessions.Manager, *sessions.Session) {
	if configManager == nil {
		return nil, nil
	}
	
	sessionMgr, err := sessions.NewManager()
	if err != nil {
		return nil, nil
	}
	
	currentSession, _ := sessionMgr.GetCurrentSession()
	return sessionMgr, currentSession
}

func initializeToolsManager(configManager *config.Manager) *toolsManager.Manager {
	if configManager == nil {
		return nil
	}
	
	// Register all built-in tools
	if err := functions.RegisterAll(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to register tools: %v\n", err)
	}
	
	registry := tools.DefaultRegistry
	approvalHandler := ui.NewApprovalHandler()
	permissionManager := permissions.NewManager(configManager, approvalHandler)
	executor := tools.NewExecutor(registry, permissionManager)
	
	return toolsManager.NewManager(toolsManager.Dependencies{
		ToolsRegistry:     registry,
		ToolsExecutor:     executor,
		PermissionManager: permissionManager,
		ApprovalHandler:   approvalHandler,
	})
}

func setupFileWatcher(fileCtx *files.FileContext, configManager *config.Manager) {
	if configManager == nil {
		return
	}
	
	debounceMs := 100
	if configManager != nil {
		debounceMs = configManager.GetAutoReloadDebounce()
	}
	
	watcher, err := files.NewWatcher(time.Duration(debounceMs) * time.Millisecond)
	if err == nil && watcher.IsSupported() {
		fileCtx.SetWatcher(watcher)
	}
}