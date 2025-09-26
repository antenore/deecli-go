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
	"github.com/antenore/deecli/internal/debug"
	"github.com/antenore/deecli/internal/chat/input"
	"github.com/antenore/deecli/internal/chat/keydetect"
	"github.com/antenore/deecli/internal/chat/messages"
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
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Custom message types for tea
// cancelApiMsg for cancelling ongoing API requests
type cancelApiMsg struct{}


// NewModel represents a clean chat interface using proper Bubbletea components
type NewModel struct {
	viewport         viewport.Model
	sidebarViewport  viewport.Model  // Separate viewport for sidebar
	textarea         textarea.Model  // Replace string input with textarea
	fileContext      *files.FileContext
	apiClient        *api.Service
	completionEngine *CompletionEngine
	configManager    *config.Manager // Config manager for proper config integration
	commandHandler   *commands.Handler // Command handler for chat commands
	renderer         *ui.Renderer    // UI renderer for message formatting
	layoutManager    *ui.Layout      // Layout manager for UI layout calculations
	sidebar          *ui.Sidebar     // Sidebar for files display
	aiOperations     *ai.Operations  // AI operations handler
	viewportManager  *viewportmgr.Manager // Viewport and message manager
	spinner          *ui.Spinner     // Visual thinking indicator
	width            int
	height           int
	ready            bool
	helpVisible      bool
	filesWidgetVisible bool
	isLoading        bool
	loadingMsg       string
	focusMode        string // "input", "viewport", or "sidebar" - tracks which component has focus
	keyDetector      *keydetect.Detector // Key detection handler
	messageManager   *messages.Manager // Message storage and formatting
	messages         []string // Keep track of all messages for full scrollback
	apiMessages      []api.Message // Keep chat history for API context
	sessionManager   *sessions.Manager
	currentSession   *sessions.Session
	sessionLoader    *sessions.Loader
	inputManager     *input.Manager // Input and history management
	apiCancel        context.CancelFunc // Function to cancel ongoing API request
	fileTracker      *tracker.FileTracker // Track files mentioned in AI responses

	// Streaming support
	streamingEnabled bool                // Whether to use streaming API
	streamingManager *streaming.Manager // Streaming operations manager
	streamReader     api.StreamReader   // Current stream reader
	streamContent    string             // Accumulated stream content

	// API response handling - now managed by apiHandler
	apiResponseHandler *apiHandler.Handler      // Handles API response processing

	// Function calling support - now managed by toolsManager
	toolsManager       *toolsManager.Manager    // Manages all tool execution and approval

	// Keep these for backward compatibility during migration
	toolsRegistry      *tools.Registry           // Registry of available tools
	toolsExecutor      *tools.Executor           // Executor for running tools
	permissionManager  *permissions.Manager     // Permission manager for tools
	approvalHandler    *ui.ApprovalHandler      // Approval UI handler
	// Tool-related fields now managed by toolsManager
}

// initializeComponents creates common components needed by both constructors
func initializeComponents(width, height int, client *api.Service, configManager *config.Manager) (*files.FileContext, *CompletionEngine, *ui.Renderer, *ui.Layout, *ui.Sidebar, *ai.Operations, *history.Manager, []string) {
	// Initialize history manager and load existing history
	historyMgr, err := history.NewManager()
	var historyData []string
	if err == nil && historyMgr != nil {
		historyData, _ = historyMgr.Load()
	}

	fileCtx := files.NewFileContext()

	// Initialize file watcher with configuration
	var debounceMs int = 100 // Default debounce time
	if configManager != nil {
		debounceMs = configManager.GetAutoReloadDebounce()
	}
	watcher, err := files.NewWatcher(time.Duration(debounceMs) * time.Millisecond)
	if err == nil && watcher.IsSupported() {
		fileCtx.SetWatcher(watcher)
	} else if err != nil {
		// Log warning but continue
		fmt.Printf("⚠️ File watching not supported: %v\n", err)
		fmt.Printf("   Use /reload command to manually reload modified files\n")
	}

	completionEngine := NewCompletionEngine()
	renderer := ui.NewRenderer(configManager)
	layoutManager := ui.NewLayout(configManager)
	sidebar := ui.NewSidebar()
	aiOperations := ai.NewOperations(client, fileCtx, configManager)

	return fileCtx, completionEngine, renderer, layoutManager, sidebar, aiOperations, historyMgr, historyData
}

// createTextarea creates and configures a textarea component
func createTextarea(width int) textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Enter to send, Ctrl+Enter for new line)"
	ta.ShowLineNumbers = false
	ta.SetHeight(3)
	ta.SetWidth(width - 4)
	ta.CharLimit = 0
	ta.Focus()
	ta.Prompt = "┃ "

	// Customize KeyMap: Use default fallback key for newlines
	// Will be updated by updateTextareaKeymap() if config is available
	keyMap := textarea.DefaultKeyMap
	keyMap.InsertNewline.SetKeys("ctrl+j") // Default fallback
	ta.KeyMap = keyMap

	return ta
}

func newChatModel() *NewModel {
	return newChatModelInternal(nil, "", "", 0, 0)
}

func newChatModelWithConfig(configManager *config.Manager, apiKey, model string, temperature float64, maxTokens int) *NewModel {
	return newChatModelInternal(configManager, apiKey, model, temperature, maxTokens)
}

// createAPIClient creates API client with fallback to environment variables
func createAPIClient(configManager *config.Manager, apiKey, model string, temperature float64, maxTokens int) *api.Service {
	if apiKey != "" {
		return api.NewDeepSeekService(apiKey, model, temperature, maxTokens)
	}

	// Use environment variable fallback for simple constructor
	envApiKey := os.Getenv("DEEPSEEK_API_KEY")
	if envApiKey != "" {
		return api.NewDeepSeekService(envApiKey, "deepseek-chat", 0.1, 2048)
	}
	return nil
}

// newChatModelInternal is the consolidated constructor implementation
func newChatModelInternal(configManager *config.Manager, apiKey, model string, temperature float64, maxTokens int) *NewModel {
	client := createAPIClient(configManager, apiKey, model, temperature, maxTokens)

	// Initialize session manager (only for config-based constructor)
	var sessionMgr *sessions.Manager
	var currentSession *sessions.Session
	if configManager != nil {
		var err error
		sessionMgr, err = sessions.NewManager()
		if err == nil && sessionMgr != nil {
			currentSession, _ = sessionMgr.GetCurrentSession()
		}
	}

	// Get terminal size
	width, height := utils.GetTerminalSize()

	// Initialize textarea for multi-line input
	ta := createTextarea(width)

	// Initialize common components
	fileCtx, completionEngine, renderer, layoutManager, sidebar, aiOperations, historyMgr, historyData := initializeComponents(width, height, client, configManager)

	chatModel := &NewModel{
		textarea:         ta,
		fileContext:      fileCtx,
		apiClient:        client,
		completionEngine: completionEngine,
		configManager:    configManager,
		renderer:         renderer,
		layoutManager:    layoutManager,
		sidebar:          sidebar,
		aiOperations:     aiOperations,
		spinner:          ui.NewDefaultSpinner(), // Initialize visual thinking indicator
		width:            width,
		height:           height,
		focusMode:        "input", // Start with input focused
		messages:         []string{}, // Initialize message history
		apiMessages:      []api.Message{}, // Initialize API message history
		sessionManager:   sessionMgr,
		currentSession:   currentSession,
		fileTracker:      tracker.NewFileTracker(), // Initialize file tracker
		streamingEnabled: true, // Enable streaming by default
		streamingManager: streaming.NewManager(), // Initialize streaming manager
	}

	// Initialize function calling support
	if configManager != nil {
		// Register all built-in tools
		if err := functions.RegisterAll(); err != nil {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "Warning: Failed to register tools: %v\n", err)
		}

		// Initialize tools components
		chatModel.toolsRegistry = tools.DefaultRegistry
		chatModel.approvalHandler = ui.NewApprovalHandler()
		chatModel.permissionManager = permissions.NewManager(configManager, chatModel.approvalHandler)
		chatModel.toolsExecutor = tools.NewExecutor(chatModel.toolsRegistry, chatModel.permissionManager)

		// Initialize the integrated tools manager
		chatModel.toolsManager = toolsManager.NewManager(toolsManager.Dependencies{
			ToolsRegistry:     chatModel.toolsRegistry,
			ToolsExecutor:     chatModel.toolsExecutor,
			PermissionManager: chatModel.permissionManager,
			ApprovalHandler:   chatModel.approvalHandler,
		})

		// Initialize the integrated API response handler
		chatModel.apiResponseHandler = apiHandler.NewHandler(apiHandler.Dependencies{
			FileTracker: chatModel.fileTracker,
		})

		// Set available tools in AI operations
		if chatModel.toolsRegistry != nil {
			availableTools := chatModel.toolsRegistry.GetAPITools()
			chatModel.aiOperations.SetAvailableTools(availableTools)
		}
	}

	// Initialize message manager
	chatModel.messageManager = messages.NewManager(messages.Dependencies{
		Renderer:       chatModel.renderer,
		Spinner:        chatModel.spinner,
		SessionManager: chatModel.sessionManager,
		CurrentSession: chatModel.currentSession,
		AIOperations:   chatModel.aiOperations,
	})

	// Initialize input manager
	chatModel.inputManager = input.NewManager(
		historyData,
		historyMgr,
		chatModel.completionEngine,
		chatModel.addMessage,
		chatModel.refreshViewport,
	)

	// Initialize key detector (only if config is provided)
	if configManager != nil {
		chatModel.keyDetector = keydetect.New(keydetect.Dependencies{
			ConfigManager:  configManager,
			MessageLogger:  chatModel.addMessage,
			RefreshView:    chatModel.refreshViewport,
			LayoutManager:  chatModel.layoutManager,
			UpdateKeymap:   func() { chatModel.keyDetector.UpdateTextareaKeymap(&chatModel.textarea) },
		})

		// Update textarea keymap with configured newline key
		chatModel.keyDetector.UpdateTextareaKeymap(&chatModel.textarea)
	}

	// Initialize command handler with dependencies
	chatModel.commandHandler = commands.NewHandler(chatModel.createCommandDependencies())

	// Initialize session loader with dependencies (only if session exists)
	if sessionMgr != nil && currentSession != nil {
		chatModel.sessionLoader = sessions.NewLoader(&sessions.LoaderDependencies{
			SessionManager:       sessionMgr,
			CurrentSession:       currentSession,
			Renderer:            chatModel.renderer,
			Viewport:            &chatModel.viewport,
			ViewportWidth:       chatModel.viewport.Width,
			FilesWidgetVisible:  chatModel.filesWidgetVisible,
			FormatInitialContent: func() string {
				if chatModel.viewportManager != nil {
					return chatModel.viewportManager.FormatInitialContent()
				}
				return "Welcome to DeeCLI"
			},
		})
	}

	// Enable auto-reload if configured and supported
	if configManager != nil && configManager.GetAutoReloadFiles() && fileCtx.IsAutoReloadSupported() {
		// Create a context for the watcher (it will live for the lifetime of the app)
		ctx := context.Background()

		// Set up auto-reload with notification callback
		if err := fileCtx.EnableAutoReload(ctx, func(results []files.ReloadResult) {
			changedCount := 0
			for _, result := range results {
				if result.Status == "changed" {
					changedCount++
				}
			}

			// Show auto-reload notification if configured
			if configManager.GetShowReloadNotices() && changedCount > 0 {
				chatModel.addMessage("system", fmt.Sprintf("📁 Auto-reloaded %d modified file(s)", changedCount))

				// Update sidebar if visible
				if chatModel.filesWidgetVisible {
					chatModel.sidebarViewport.SetContent(chatModel.renderFilesSidebar())
				}

				// Refresh viewport to show the message
				chatModel.refreshViewport()
			}
		}); err != nil {
			// Auto-reload setup failed, but continue
			chatModel.addMessage("system", fmt.Sprintf("⚠️ Auto-reload setup failed: %v", err))
		}
	} else if configManager != nil && !fileCtx.IsAutoReloadSupported() {
		// Show platform limitation message once
		chatModel.addMessage("system",
			"ℹ️ File auto-reload is not available on this platform.\n" +
			"   Use /reload command to manually reload modified files.")
	}

	return chatModel
}

// createCommandDependencies creates Dependencies struct for command handlers
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
		ToolsRegistry:    m.toolsRegistry,
		Messages:         m.messages,
		APIMessages:      m.apiMessages,
		InputHistory:     inputHistory,
		HelpVisible:      m.helpVisible,
		MessageLogger:    m.addMessage,
		SetLoading:       m.setLoading,
		SetCancel:        m.setCancel,
		RefreshUI:        m.refreshViewport,
		ShowHistory: func() {
			if m.inputManager != nil {
				m.inputManager.ShowHistory()
			}
		},
		AnalyzeFiles:     m.analyzeFiles,
		ExplainFiles:     m.explainFiles,
		ImproveFiles:     m.improveFiles,
		GenerateEditSuggestions: m.generateEditSuggestions,
		SetHelpVisible:   m.setHelpVisible,
		SetKeyDetection:  m.keyDetector.SetDetection,
	}
}

func (m *NewModel) setLoading(loading bool, message string) tea.Cmd {
	m.isLoading = loading
	m.loadingMsg = message

	// Control spinner animation
	if loading {
		return m.spinner.Start()
	} else {
		m.spinner.Stop()
		return nil
	}
}

func (m *NewModel) setCancel(cancel context.CancelFunc) {
	m.apiCancel = cancel
}

func (m *NewModel) setHelpVisible(visible bool) {
	m.helpVisible = visible
	if m.helpVisible {
		if m.viewportManager != nil {
			m.viewport.SetContent(m.viewportManager.HelpContent())
		} else {
			m.viewport.SetContent("Help not available")
		}
	} else {
		m.refreshViewport()
	}
}


func (m NewModel) Init() tea.Cmd {
	return nil
}




func (m *NewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle spinner animation
	if spinnerCmd := m.spinner.Update(msg); spinnerCmd != nil {
		cmds = append(cmds, spinnerCmd)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		if !m.ready {
			// Initialize viewports with proper size and positioning
			m.viewport = viewport.New(m.width, 10) // Initial size, will be set by layout
			m.viewport.YPosition = 1  // Start after header line

			// Initialize sidebar viewport
			m.sidebarViewport = viewport.New(25, 10) // Initial size, will be set by layout
			m.sidebarViewport.YPosition = 1  // Start after header line

			// Set proper layout (this will correct the sizes and positions)
			m.layout()

			// Initialize viewport manager now that viewport is properly set up
			if m.viewportManager == nil {
				m.viewportManager = viewportmgr.NewManager(viewportmgr.Dependencies{
					Viewport:         &m.viewport,
					Renderer:         m.renderer,
					LayoutManager:    m.layoutManager,
					ConfigManager:    m.configManager,
					SessionManager:   m.sessionManager,
					CurrentSession:   m.currentSession,
					Spinner:          m.spinner,
					Messages:         &m.messages,
					APIMessages:      &m.apiMessages,
					FilesWidgetVisible: &m.filesWidgetVisible,
					IsLoading:        &m.isLoading,
					LoadingMsg:       &m.loadingMsg,
				})
			}

			// Add welcome message to history
			if m.viewportManager != nil {
				m.messages = append(m.messages, m.viewportManager.FormatInitialContent())
			} else {
				m.messages = append(m.messages, "Welcome to DeeCLI")
			}
			m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
			m.ready = true
		} else {
			// Update viewport width and recalculate layout
			m.viewport.Width = m.width
			m.layout()
		}

	case cancelApiMsg:
		if cmd := m.setLoading(false, ""); cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.apiCancel = nil
		// Close stream reader if active
		if m.streamReader != nil {
			m.streamReader.Close()
			m.streamReader = nil
			m.streamContent = ""
		}
		m.addMessage("system", "🚫 Request cancelled")
		m.viewport.GotoBottom()

	case ai.APIResponseMsg:
		m.handleAPIResponse(msg.Response, msg.Err)

	case ai.ToolCallsResponseMsg:
		if cmd := m.handleToolCallsResponse(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case ToolExecutionCompleteMsg:
		if cmd := m.handleToolExecutionComplete(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case toolsManager.CreateApprovalDialogMsg:
		// Create and show approval dialog
		m.toolsManager.CreateApprovalDialog(msg.ApprovalRequest, m.width, m.height)
		m.refreshViewport()

	case toolsManager.RequestToolApprovalMsg:
		// Request approval for next tool in queue
		if cmd := m.requestToolApproval(msg.ToolCall); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case toolsManager.TriggerFollowupMsg:
		// Trigger follow-up API call after tool execution
		if m.aiOperations != nil {
			m.toolsManager.SetSuppressToolCalls(true)
			if cmd := m.setLoading(true, "Continuing..."); cmd != nil {
				follow := m.aiOperations.CallAPIWithToolsNoChoice("", "")
				m.apiCancel = m.aiOperations.GetAPICancel()
				cmds = append(cmds, cmd, follow)
			} else {
				follow := m.aiOperations.CallAPIWithToolsNoChoice("", "")
				m.apiCancel = m.aiOperations.GetAPICancel()
				cmds = append(cmds, follow)
			}
		}

	case ai.StreamStartedMsg:
		// Use streaming manager to handle stream start
		if cmd := m.setLoading(true, "Thinking..."); cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.refreshViewport()
		// Let streaming manager handle the stream start
		nextCmd := m.streamingManager.StartStream(msg, m.renderer, &m.messages)
		return m, nextCmd

	case ai.StreamChunkMsg:
		// Use streaming manager to handle chunk processing
		nextCmd, extraCmds := m.streamingManager.HandleChunk(msg, m.spinner, &m.isLoading, m.setLoading)
		if extraCmds != nil {
			cmds = append(cmds, extraCmds...)
		}
		if nextCmd != nil {
			cmds = append(cmds, nextCmd)
		}

		// Update display with current streaming content
		m.streamingManager.UpdateDisplay(m.streamingManager.GetStreamContent(), m.renderer, &m.messages, &m.viewport)

		// Keep message manager in sync
		if m.messageManager != nil {
			m.messageManager.SetMessages(m.messages)
		}

		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}

	case ai.StreamChunkWithToolsMsg:
		// Similar to StreamChunkMsg but tracks tool calls
		// Continue streaming with accumulated tool calls
		nextCmd := ai.ReadNextChunkWithTools(
			m.streamingManager.GetStream(),
			m.streamingManager.GetStreamContent(),
			msg.ToolCalls,
		)
		if nextCmd != nil {
			cmds = append(cmds, nextCmd)
		}

		// Update display with current streaming content
		if msg.Content != "" {
			m.streamingManager.AppendContent(msg.Content)
			m.streamingManager.UpdateDisplay(m.streamingManager.GetStreamContent(), m.renderer, &m.messages, &m.viewport)
		}

	case ai.ToolCallsStreamMsg:
		// Streaming completed with tool calls
		completionMsg := ai.StreamCompleteMsg{
			TotalContent: msg.TotalContent,
			Err:          nil,
		}
		m.streamingManager.CompleteStream(completionMsg)
		if cmd := m.setLoading(false, ""); cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Handle the tool calls
		if len(msg.ToolCalls) > 0 {
			// Convert to non-streaming format and handle
			toolMsg := ai.ToolCallsResponseMsg{
				ToolCalls: msg.ToolCalls,
				Response:  nil, // No full response in streaming
			}
			if cmd := m.handleToolCallsResponse(toolMsg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case ai.StreamCompleteMsg:
		// Use streaming manager to handle completion
		completionCmd := m.streamingManager.CompleteStream(msg)
		return m, completionCmd

	case streaming.StreamCompleteInternalMsg:
		// Handle streaming completion from streaming manager
		m.handleStreamCompleteInternal(msg)

	case editor.EditorFinishedMsg:
		if msg.Error != nil {
			m.addMessage("system", fmt.Sprintf("❌ Editor error: %v", msg.Error))
		} else {
			m.addMessage("system", "✓ Editor closed")
			
			// Auto-reload any files that are currently loaded to pick up changes
			if len(m.fileContext.Files) > 0 {
				results, err := m.fileContext.ReloadFiles(nil) // Reload all loaded files
				if err != nil {
					m.addMessage("system", fmt.Sprintf("⚠️ Failed to auto-reload files: %v", err))
				} else if len(results) > 0 {
					changedCount := 0
					for _, result := range results {
						if result.Status == "changed" {
							changedCount++
						}
					}
					if changedCount > 0 {
						m.addMessage("system", fmt.Sprintf("🔄 Auto-reloaded %d file(s), %d changed", len(results), changedCount))
					}
					
					// Update sidebar if visible
					if m.filesWidgetVisible {
						m.sidebarViewport.SetContent(m.renderFilesSidebar())
					}
				}
			}
		}
		m.refreshViewport()

	case tea.KeyMsg:
		// Handle tool approval dialog first (highest priority)
		if m.toolsManager.IsShowingApproval() && m.toolsManager.GetApprovalDialog() != nil {
			done, response := m.toolsManager.GetApprovalDialog().Update(msg.String())
			if done && response != nil {
				m.toolsManager.SetShowingApproval(false)
				m.toolsManager.ClearApprovalDialog()
				return m, m.executeApprovedTool(*response)
			}
			return m, nil // Dialog is still active
		}

		// Handle key detection mode (second priority)
		if m.keyDetector != nil && m.keyDetector.IsDetecting() {
			return m, m.keyDetector.HandleDetection(msg.String())
		}

		// First handle global keys that work regardless of focus
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Cancel ongoing API request if one is active
			if m.isLoading && m.apiCancel != nil {
				m.apiCancel()
				m.apiCancel = nil
				return m, func() tea.Msg { return cancelApiMsg{} }
			}
		case "f1":
			m.helpVisible = !m.helpVisible
			if m.helpVisible {
				if m.viewportManager != nil {
					m.viewport.SetContent(m.viewportManager.HelpContent())
				} else {
					m.viewport.SetContent("Help not available")
				}
			} else {
				m.refreshViewport()
			}
			return m, nil
		case "f2":
			m.filesWidgetVisible = !m.filesWidgetVisible
			if m.filesWidgetVisible {
				m.sidebarViewport.SetContent(m.renderFilesSidebar())
				m.sidebarViewport.GotoTop()
			}
			m.layout()
			return m, nil
		case "f3":
			// Toggle raw code mode for easy copying
			if m.renderer != nil {
				isRaw := m.renderer.ToggleRawCodeMode()
				// Note: This only affects NEW messages, not existing ones
				// Show status message
				statusMsg := "Code blocks: FORMATTED (with borders) - new messages only"
				if isRaw {
					statusMsg = "Code blocks: RAW (copy-friendly) - new messages only"
				}
				m.addSystemMessage(statusMsg)
			}
			return m, nil
		// Removed ctrl+w interception - now it naturally deletes words in textarea
		}

		// Handle viewport scrolling when viewport has focus
		if m.focusMode == "viewport" {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end":
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			case "tab":
				// Continue focus cycle from viewport
				if m.filesWidgetVisible {
					m.focusMode = "sidebar"
					m.sidebarViewport.GotoTop()
				} else {
					m.focusMode = "input"
					m.textarea.Focus()
				}
				return m, nil
			case "enter", "esc":
				m.focusMode = "input"
				m.textarea.Focus()
				return m, nil
			}
		}

		// Handle sidebar scrolling
		if m.focusMode == "sidebar" {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end":
				m.sidebarViewport, cmd = m.sidebarViewport.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			case "tab":
				// Complete focus cycle - back to input
				m.focusMode = "input"
				m.textarea.Focus()
				return m, nil
			case "enter", "esc":
				m.focusMode = "input"
				m.textarea.Focus()
				return m, nil
			}
		}

		// Input mode - handle special keys first, then let textarea handle the rest
		if m.focusMode == "input" {
			// Handle completion navigation with arrow keys
			if m.inputManager != nil {
				completions, _, showCompletions := m.inputManager.GetCompletionState()
				if showCompletions && len(completions) > 0 {
					switch msg.String() {
					case "down", "ctrl+n":
						if m.inputManager.HandleCompletionNavigation("down", &m.textarea) {
							return m, nil
						}
					case "up", "ctrl+p":
						if m.inputManager.HandleCompletionNavigation("up", &m.textarea) {
							return m, nil
						}
					case "tab", "enter":
						if m.inputManager.AcceptCompletion(&m.textarea) {
							// If this was Enter, don't let it fall through to send message
							if msg.String() == "enter" {
								return m, nil
							}
						}
					case "esc":
						m.inputManager.ClearCompletions()
						return m, nil
					}
				} else {
					// No completions shown - handle arrow keys for history navigation
					// Only use arrows for history if input is single-line
					currentInput := m.textarea.Value()
					isMultiLine := strings.Contains(currentInput, "\n")

					if !isMultiLine && m.inputManager != nil {
						switch msg.String() {
						case "up":
							if m.inputManager.HandleHistoryBack(&m.textarea) {
								return m, nil
							}
						case "down":
							if m.inputManager.HandleHistoryForward(&m.textarea) {
								return m, nil
							}
						}
					}
				}
			}

			// Smart Tab: completion if available, focus switch otherwise
			// (Tab for accepting completions is already handled above when completions are shown)
			if msg.String() == "tab" && m.inputManager != nil {
				input := m.textarea.Value()

				// Try to show completions
				if m.inputManager.HandleTabCompletion(input) {
					// Completions are now showing
					return m, nil
				}

				// No completions available, use Tab for focus switching
				switch m.focusMode {
				case "input":
					m.focusMode = "viewport"
					m.textarea.Blur()
				case "viewport":
					if m.filesWidgetVisible {
						m.focusMode = "sidebar"
						m.sidebarViewport.GotoTop()
					} else {
						m.focusMode = "input"
						m.textarea.Focus()
					}
				case "sidebar":
					m.focusMode = "input"
					m.textarea.Focus()
				default:
					m.focusMode = "input"
					m.textarea.Focus()
				}
				return m, nil
			}
			
			// Handle history navigation BEFORE textarea gets the keys
			historyHandled := false
			if m.configManager != nil && m.inputManager != nil {
				historyBackKey := m.configManager.GetHistoryBackKey()
				historyForwardKey := m.configManager.GetHistoryForwardKey()

				switch msg.String() {
				case historyBackKey: // Previous history (default ctrl+p)
					historyHandled = true
					if m.inputManager.HandleHistoryBack(&m.textarea) {
						return m, nil
					}

				case historyForwardKey: // Next history (default ctrl+n)
					historyHandled = true
					if m.inputManager.HandleHistoryForward(&m.textarea) {
						return m, nil
					}
				}
			}

			// Clear completions on any other key (except history navigation)
			if msg.String() != "tab" && !historyHandled && m.inputManager != nil {
				m.inputManager.ClearCompletions()
			}

			// Let textarea handle non-tab, non-history keys
			if !historyHandled {
				m.textarea, cmd = m.textarea.Update(msg)
			}
			cmds = append(cmds, cmd)

			// After textarea processes key, check if it was Enter (for submission)
			if msg.String() == "enter" {
				input := strings.TrimSpace(m.textarea.Value())
				if input != "" {
					// Add to history via input manager
					if m.inputManager != nil {
						m.inputManager.AddToHistory(input)
					}

					if strings.HasPrefix(input, "/") {
						// Handle chat commands
						cmd := m.handleCommand(input)
						m.textarea.Reset()
						if m.inputManager != nil {
							m.inputManager.ClearCompletions()
						}
						return m, cmd
					} else {
						// Add user message
						m.addMessage("user", input)

						// Send to API
						if m.apiClient != nil {
							contextPrompt := ""
							if len(m.fileContext.Files) > 0 {
								// Get config for smart context management
								maxContextSize := 100000 // Default
								if m.configManager != nil {
									cfg := m.configManager.Get()
									if cfg != nil && cfg.MaxContextSize > 0 {
										maxContextSize = cfg.MaxContextSize
									}
								}

								// Estimate if we need truncation (leave buffer for user input and API overhead)
								inputSize := len(input)
								bufferSize := inputSize + 10000 // Reserve 10KB for API overhead and user input
								contextBudget := maxContextSize - bufferSize

								if contextBudget > 5000 { // Only use truncation if we have reasonable budget
									contextPrompt = m.fileContext.BuildContextPromptWithLimit(contextBudget)
								} else {
									// Very tight budget, use minimal context
									contextPrompt = fmt.Sprintf("Files loaded: %d (content truncated due to size limits)\n",
										len(m.fileContext.Files))
								}
							}

							m.textarea.Reset()
							if m.inputManager != nil {
								m.inputManager.ClearCompletions()
							}
							if cmd := m.setLoading(true, "Thinking..."); cmd != nil {
								cmds = append(cmds, cmd)
							}
							m.refreshViewport()

							cmds = append(cmds, m.callAPI(contextPrompt, input))
							return m, tea.Batch(cmds...)
						} else {
							m.addMessage("system", "Please set DEEPSEEK_API_KEY environment variable")
							m.textarea.Reset()
						}
					}
				}
			}

			return m, tea.Batch(cmds...)
		}
	}
	// End of switch statement handling

	// If we have an error type message, handle it
	if err, ok := msg.(error); ok {
		m.addMessage("system", fmt.Sprintf("❌ Unexpected error: %v", err))
	}

	return m, tea.Batch(cmds...)
}

func (m NewModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Build header using layout manager
	filesCount := len(m.fileContext.Files)
	header := m.layoutManager.RenderHeader(filesCount, m.focusMode, m.fileContext, m.renderer)

	// Build main content area using layout manager
	chatContent := m.viewport.View()
	sidebarContent := m.sidebarViewport.View()
	mainContent := m.layoutManager.RenderMainContent(chatContent, sidebarContent, m.width, m.filesWidgetVisible, m.focusMode)

	// Build footer using layout manager
	inputArea := m.textarea.View()
	completions := []string{}
	completionIndex := 0
	if m.inputManager != nil {
		completions, completionIndex, _ = m.inputManager.GetCompletionState()
	}
	footer := m.layoutManager.RenderFooter(inputArea, completions, completionIndex, m.width)

	// Check if approval dialog should be shown
	if m.toolsManager.IsShowingApproval() && m.toolsManager.GetApprovalDialog() != nil {
		// Show only the approval dialog - don't show the main chat interface
		// This prevents duplication and focuses the user on the approval request
		dialogView := m.toolsManager.GetApprovalDialog().View()
		return fmt.Sprintf("%s\n%s", header, dialogView)
	}

	// Normal view when no approval dialog is shown
	baseView := fmt.Sprintf("%s\n%s\n%s", header, mainContent, footer)
	return baseView
}

// renderFilesSidebar creates the files sidebar content
func (m *NewModel) renderFilesSidebar() string {
	return m.sidebar.RenderFilesSidebar(m.fileContext, m.configManager)
}


// layout calculates and sets proper dimensions for all components
func (m *NewModel) layout() {
	// Calculate viewport dimensions using layout manager
	hasCompletions := false
	if m.inputManager != nil {
		completions, _, showCompletions := m.inputManager.GetCompletionState()
		hasCompletions = showCompletions && len(completions) > 0
	}
	viewportHeight, yPosition := m.layoutManager.CalculateViewportDimensions(m.height, hasCompletions)

	// Update viewports with proper Y position
	m.viewport.Height = viewportHeight
	m.viewport.YPosition = yPosition  // Start after header

	m.sidebarViewport.Height = viewportHeight
	m.sidebarViewport.YPosition = yPosition  // Start after header

	// Update textarea width using layout manager
	textareaWidth := m.layoutManager.CalculateTextareaWidth(m.width, m.filesWidgetVisible)
	m.textarea.SetWidth(textareaWidth)
}

func (m *NewModel) addMessage(role, content string) {
	// Delegate to message manager
	viewportWrapper := messages.NewViewportWrapper(&m.viewport)
	m.messageManager.AddMessage(role, content, viewportWrapper, m.filesWidgetVisible)

	// Update local references for backward compatibility
	m.messages = m.messageManager.GetMessages()
	m.apiMessages = m.messageManager.GetAPIMessages()
}

func (m *NewModel) refreshViewport() {
	// Delegate to message manager
	viewportWrapper := messages.NewViewportWrapper(&m.viewport)
	m.messageManager.RefreshViewport(viewportWrapper, m.isLoading, m.loadingMsg)
}

// updateStreamingDisplay updates the display with streaming content
func (m *NewModel) updateStreamingDisplay(content string) {
	// Only update if we have messages (allow updates during entire streaming process)
	if len(m.messages) == 0 {
		return
	}

    // Update the last message (which should be our streaming assistant message)
    lastIdx := len(m.messages) - 1
    m.messages[lastIdx] = m.renderer.FormatMessage("assistant", content)
    // Keep message manager in sync so future refreshes don't lose content
    if m.messageManager != nil {
        m.messageManager.SetMessages(m.messages)
    }

	// Update viewport content
	m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
	m.viewport.GotoBottom()
}


// addSystemMessage adds a temporary system message to the viewport
func (m *NewModel) addSystemMessage(message string) {
	// Format as system message
	formatted := m.renderer.FormatMessage("system", message)

	// Add to messages temporarily (not saved to API messages)
	m.messages = append(m.messages, formatted)

	// Update viewport
	m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
	m.viewport.GotoBottom()
}

// Command handling and async functions (keeping the same logic)
func (m *NewModel) handleCommand(input string) tea.Cmd {
	// Update command handler with fresh dependencies before handling
	m.commandHandler = commands.NewHandler(m.createCommandDependencies())
	return m.commandHandler.Handle(input)
}




func (m *NewModel) callAPI(contextPrompt, userInput string) tea.Cmd {
	if m.aiOperations == nil {
		return func() tea.Msg {
			return ai.APIResponseMsg{Err: fmt.Errorf("AI operations not available")}
		}
	}

	// Check if context is too large for reliable streaming
	contextSize := len(contextPrompt) + len(userInput)

	// Get max context size and max tokens from config
	maxContextSize := 50000
	maxTokens := 2048
	if m.configManager != nil {
		cfg := m.configManager.Get()
		if cfg.MaxContextSize > 0 {
			maxContextSize = cfg.MaxContextSize
		}
		if cfg.MaxTokens > 0 {
			maxTokens = cfg.MaxTokens
		}
	}

	// Estimate tokens (rough approximation: 1 token ≈ 4 characters)
	contextTokens := contextSize / 4
	totalTokens := contextTokens + maxTokens

	// DeepSeek models have ~128K token limit, leave safety margin
	const modelTokenLimit = 120000
	if totalTokens > modelTokenLimit {
		return func() tea.Msg {
			return ai.APIResponseMsg{
				Err: fmt.Errorf("context + max_tokens (%d tokens) exceeds model limit (%d)\n\nContext: ~%d tokens, Max tokens: %d\n\nTry loading fewer files or reducing max_tokens setting",
					totalTokens, modelTokenLimit, contextTokens, maxTokens),
			}
		}
	}

    // Use streaming threshold based on configured MaxContextSize
    // Stream for any context under the configured limit
    streamingThreshold := maxContextSize

    // TEMPORARY WORKAROUND: Disable streaming when tools are available
    // DeepSeek appears to send tool calls as text content instead of proper function calls
    // This causes tool call markers to appear in chat instead of being processed
    hasTools := m.toolsRegistry != nil && len(m.toolsRegistry.GetAPITools()) > 0
    
    if hasTools {
        // Force non-streaming mode for tool-enabled conversations
        debug.Printf("[DEBUG] Disabling streaming due to tools being available\n")
        cmd := m.aiOperations.CallAPI(contextPrompt, userInput)
        m.apiCancel = m.aiOperations.GetAPICancel()
        return cmd
    }
    
    // Use streaming when enabled, no tools, and total context is under threshold
    if m.streamingEnabled && contextSize < streamingThreshold {
		cmd := m.aiOperations.CallAPIStream(contextPrompt, userInput)
		// Store the cancel function
		m.apiCancel = m.aiOperations.GetAPICancel()
		return cmd
	}

	// Use non-streaming for large contexts or when streaming is disabled
	cmd := m.aiOperations.CallAPI(contextPrompt, userInput)
	// Store the cancel function
	m.apiCancel = m.aiOperations.GetAPICancel()
	return cmd
}

func (m *NewModel) analyzeFiles() tea.Cmd {
	if m.aiOperations == nil {
		return func() tea.Msg {
			return ai.APIResponseMsg{Err: fmt.Errorf("AI operations not available")}
		}
	}
	return m.aiOperations.AnalyzeFiles()
}

func (m *NewModel) explainFiles() tea.Cmd {
	if m.aiOperations == nil {
		return func() tea.Msg {
			return ai.APIResponseMsg{Err: fmt.Errorf("AI operations not available")}
		}
	}
	return m.aiOperations.ExplainFiles()
}

func (m *NewModel) improveFiles() tea.Cmd {
	if m.aiOperations == nil {
		return func() tea.Msg {
			return ai.APIResponseMsg{Err: fmt.Errorf("AI operations not available")}
		}
	}
	return m.aiOperations.ImproveFiles()
}

func (m *NewModel) generateEditSuggestions() tea.Cmd {
	if m.aiOperations == nil {
		return func() tea.Msg {
			return ai.APIResponseMsg{Err: fmt.Errorf("AI operations not available")}
		}
	}
	cmd := m.aiOperations.GenerateEditSuggestions()
	// Store the cancel function
	m.apiCancel = m.aiOperations.GetAPICancel()
	return cmd
}








// handleAPIResponse handles API responses for both old and new message types
func (m *NewModel) handleAPIResponse(response string, err error) {
	m.setLoading(false, "")
	m.apiCancel = nil

	// Delegate to API response handler
	result := m.apiResponseHandler.HandleResponse(response, err, m.toolsManager.ShouldSuppressToolCalls(), m.fileContext)

	// Clear suppress flag if it was set
	if m.toolsManager.ShouldSuppressToolCalls() {
		m.toolsManager.ClearSuppressToolCalls()
		debug.Printf("[DEBUG] Suppressing tool call parsing for this response (tool_choice=none follow-up)\n")
	}

	if !result.Success {
		// Handle error result
		if result.ErrorMessage != "" {
			m.addMessage("system", result.ErrorMessage)
		}
	} else if result.AssistantContent != "" {
		// Handle successful response
		m.addMessage("assistant", result.AssistantContent)

		// Handle tool calls if present
		if len(result.ToolCalls) > 0 {
			toolMsg := ai.ToolCallsResponseMsg{
				ToolCalls: result.ToolCalls,
				Response:  nil,
			}
			m.handleToolCallsResponse(toolMsg)
		}
	}

	m.viewport.GotoBottom()
}

// parseAndExtractToolCalls parses DeepSeek's tool call markup and extracts proper tool calls
func (m *NewModel) parseAndExtractToolCalls(content string) ([]api.ToolCall, string) {
	// Always use the integrated apiResponseHandler
	return m.apiResponseHandler.ParseAndExtractToolCalls(content)
}

// handleStreamCompleteInternal handles completion from streaming manager
func (m *NewModel) handleStreamCompleteInternal(msg streaming.StreamCompleteInternalMsg) {
	m.setLoading(false, "")
	m.apiCancel = nil

	// Clean up old streaming state
	m.streamReader = nil
	m.streamContent = ""

	if msg.Err != nil {
		// Handle error cases
		if apiErr, ok := msg.Err.(api.APIError); ok {
			if apiErr.Message != "request cancelled by user" {
				errorMsg := fmt.Sprintf("❌ %s", apiErr.UserMessage)
				if apiErr.StatusCode > 0 {
					errorMsg += fmt.Sprintf(" (HTTP %d)", apiErr.StatusCode)
				}
				m.addMessage("system", errorMsg)
			}
		} else if msg.Err != context.Canceled {
			m.addMessage("system", fmt.Sprintf("❌ Error: %v", msg.Err))
		}
	} else if msg.Content != "" {
		// Handle successful completion
		// If no message was added during streaming (no meaningful content), add it now
		if !msg.MessageAdded && msg.FinalContent != "" {
			m.addMessage("assistant", msg.FinalContent)
		}

		// Track files mentioned in response
		if m.fileTracker != nil {
			m.fileTracker.ExtractFilesFromResponseWithContext(msg.Content, m.fileContext.Files)
		}

		// Add to API messages for history
		m.apiMessages = append(m.apiMessages, api.Message{
			Role:    "assistant",
			Content: msg.Content,
		})
	}

	// Ensure viewport is up to date
	m.viewport.GotoBottom()
}

// Following the official Bubbletea chat example pattern
// handleStreamComplete handles the completion of a stream
func (m *NewModel) handleStreamComplete(content string, err error) {
	m.setLoading(false, "")
	m.apiCancel = nil
	m.streamReader = nil
	m.streamContent = ""

	if err != nil {
		// Check if it's an enhanced APIError
		if apiErr, ok := err.(api.APIError); ok {
			// Show user-friendly message, but don't show cancellation as error
			if apiErr.Message != "request cancelled by user" {
				errorMsg := fmt.Sprintf("❌ %s", apiErr.UserMessage)
				if apiErr.StatusCode > 0 {
					errorMsg += fmt.Sprintf(" (HTTP %d)", apiErr.StatusCode)
				}
				m.addMessage("system", errorMsg)
			}
		} else if err != context.Canceled {
			// Don't show error for context cancellation
			m.addMessage("system", fmt.Sprintf("❌ Error: %v", err))
		}
	} else {
		// Ensure final content is properly set (update the last message with final content)
		if len(m.messages) > 0 {
			lastIdx := len(m.messages) - 1
			m.messages[lastIdx] = m.renderer.FormatMessage("assistant", content)
			// Sync message manager before updating viewport
			if m.messageManager != nil {
				m.messageManager.SetMessages(m.messages)
			}
			m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
		}

		// Track files mentioned in the AI response
		if m.fileTracker != nil {
			m.fileTracker.ExtractFilesFromResponseWithContext(content, m.fileContext.Files)
		}
		// Add to API messages for history
		m.apiMessages = append(m.apiMessages, api.Message{
			Role:    "assistant",
			Content: content,
		})
	}
	m.viewport.GotoBottom()
}

func (m *NewModel) loadPreviousSession() error {
	if m.sessionLoader == nil {
		return fmt.Errorf("no session loader available")
	}

	messages, apiMessages, err := m.sessionLoader.LoadSession()
	if err != nil {
		return err
	}

	// Use message manager to set messages
	m.messageManager.SetMessages(messages)
	m.messageManager.SetAPIMessages(apiMessages)

	// Update local references for backward compatibility
	m.messages = messages
	m.apiMessages = apiMessages

	return nil
}

// handleToolCallsResponse handles AI responses that request tool executions
func (m *NewModel) handleToolCallsResponse(msg ai.ToolCallsResponseMsg) tea.Cmd {
	// Delegate to tools manager
	return m.toolsManager.HandleToolCallsResponse(msg)
}

// requestToolApproval shows approval dialog for a tool call
func (m *NewModel) requestToolApproval(toolCall api.ToolCall) tea.Cmd {
	// Delegate to tools manager for tool approval handling
	return m.toolsManager.HandleToolCallsResponse(ai.ToolCallsResponseMsg{ToolCalls: []api.ToolCall{toolCall}})
}

// executeApprovedTool executes a tool after user approval
func (m *NewModel) executeApprovedTool(response tools.ApprovalResponse) tea.Cmd {
	// Delegate to tools manager for tool execution
	return m.toolsManager.ExecuteApprovedTool(response)
}

// Use ToolExecutionCompleteMsg from tools manager
type ToolExecutionCompleteMsg = toolsManager.ToolExecutionCompleteMsg

// handleToolExecutionComplete handles the completion of tool execution
func (m *NewModel) handleToolExecutionComplete(msg ToolExecutionCompleteMsg) tea.Cmd {
	// Delegate to tools manager and handle success/failure
	cmd, success := m.toolsManager.HandleToolExecutionComplete(msg, m.aiOperations)
	if !success {
		// Tool execution failed, no further action needed
		return nil
	}
	// Return the command from tools manager (may trigger follow-up or next tool)
	return cmd
}
