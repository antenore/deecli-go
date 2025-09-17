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

	"github.com/antenore/deecli/internal/ai"
	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/chat/commands"
	"github.com/antenore/deecli/internal/chat/input"
	"github.com/antenore/deecli/internal/chat/keydetect"
	"github.com/antenore/deecli/internal/chat/tracker"
	"github.com/antenore/deecli/internal/chat/ui"
	viewportmgr "github.com/antenore/deecli/internal/chat/viewport"
	"github.com/antenore/deecli/internal/config"
	"github.com/antenore/deecli/internal/editor"
	"github.com/antenore/deecli/internal/files"
	"github.com/antenore/deecli/internal/history"
	"github.com/antenore/deecli/internal/sessions"
	"github.com/antenore/deecli/internal/utils"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	width            int
	height           int
	ready            bool
	helpVisible      bool
	filesWidgetVisible bool
	isLoading        bool
	loadingMsg       string
	focusMode        string // "input", "viewport", or "sidebar" - tracks which component has focus
	keyDetector      *keydetect.Detector // Key detection handler
	messages         []string // Keep track of all messages for full scrollback
	apiMessages      []api.Message // Keep chat history for API context
	sessionManager   *sessions.Manager
	currentSession   *sessions.Session
	sessionLoader    *sessions.Loader
	inputManager     *input.Manager // Input and history management
	apiCancel        context.CancelFunc // Function to cancel ongoing API request
	fileTracker      *tracker.FileTracker // Track files mentioned in AI responses

	// Streaming support
	streamingEnabled bool             // Whether to use streaming API
	streamReader     api.StreamReader // Current stream reader
	streamContent    string           // Accumulated stream content
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
	completionEngine := NewCompletionEngine()
	renderer := ui.NewRenderer(configManager)
	layoutManager := ui.NewLayout(configManager)
	sidebar := ui.NewSidebar()
	aiOperations := ai.NewOperations(client, fileCtx)

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
		width:            width,
		height:           height,
		focusMode:        "input", // Start with input focused
		messages:         []string{}, // Initialize message history
		apiMessages:      []api.Message{}, // Initialize API message history
		sessionManager:   sessionMgr,
		currentSession:   currentSession,
		fileTracker:      tracker.NewFileTracker(), // Initialize file tracker
		streamingEnabled: true, // Enable streaming by default
	}

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

	return chatModel
}

// getHistoryManager returns the history manager from input manager
func (m *NewModel) getHistoryManager() *history.Manager {
	if m.inputManager != nil {
		return m.inputManager.GetHistoryManager()
	}
	return nil
}

// createCommandDependencies creates Dependencies struct for command handlers
func (m *NewModel) createCommandDependencies() commands.Dependencies {
	var inputHistory []string
	if m.inputManager != nil {
		inputHistory = m.inputManager.GetInputHistory()
	}

	return commands.Dependencies{
		FileContext:      m.fileContext,
		APIClient:        m.apiClient,
		ConfigManager:    m.configManager,
		SessionManager:   m.sessionManager,
		CurrentSession:   m.currentSession,
		HistoryManager:   m.getHistoryManager(),
		FileTracker:      m.fileTracker,
		Messages:         m.messages,
		APIMessages:      m.apiMessages,
		InputHistory:     inputHistory,
		HelpVisible:      m.helpVisible,
		MessageLogger:    m.addMessage,
		SetLoading:       m.setLoading,
		SetCancel:        m.setCancel,
		RefreshUI:        m.refreshViewport,
		ShowHistory:      m.showHistoryFromInputManager,
		AnalyzeFiles:     m.analyzeFiles,
		ExplainFiles:     m.explainFiles,
		ImproveFiles:     m.improveFiles,
		GenerateEditSuggestions: m.generateEditSuggestions,
		SetHelpVisible:   m.setHelpVisible,
		SetKeyDetection:  m.keyDetector.SetDetection,
	}
}

// Helper methods for command dependencies
func (m *NewModel) showHistoryFromInputManager() {
	if m.inputManager != nil {
		m.inputManager.ShowHistory()
	}
}

func (m *NewModel) setLoading(loading bool, message string) {
	m.isLoading = loading
	m.loadingMsg = message
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
		m.isLoading = false
		m.loadingMsg = ""
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

	case ai.StreamStartedMsg:
		// Stream has started, save the reader and start reading chunks
		m.streamReader = msg.Stream
		m.streamContent = ""
		m.isLoading = true  // Set loading flag for streaming
		// Add initial placeholder assistant message - this will be updated during streaming
		m.messages = append(m.messages, m.renderer.FormatMessage("assistant", ""))
		m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
		m.viewport.GotoBottom()
		// Start reading the first chunk
		return m, ai.ReadNextChunk(msg.Stream, m.streamContent)

	case ai.StreamChunkMsg:
		// Handle incoming chunk
		if msg.Err != nil {
			m.handleStreamComplete(m.streamContent, msg.Err)
			return m, nil
		}

		// Append chunk content
		m.streamContent += msg.Content

		// Update the display with accumulated content
		m.updateStreamingDisplay(m.streamContent)

		// Continue reading next chunk
		if m.streamReader != nil {
			return m, ai.ReadNextChunk(m.streamReader, m.streamContent)
		}

	case ai.StreamCompleteMsg:
		// Stream completed
		m.handleStreamComplete(msg.TotalContent, msg.Err)

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
		// Handle key detection mode first (highest priority)
		if m.keyDetector.IsDetecting() {
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
		case "ctrl+w":
			// Improved focus cycling
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

		// Handle viewport scrolling when viewport has focus
		if m.focusMode == "viewport" {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end":
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
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
				}
			}

			// Handle Tab completion BEFORE textarea gets the key
			if msg.String() == "tab" && m.inputManager != nil {
				input := m.textarea.Value()
				m.inputManager.HandleTabCompletion(input)
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
								contextPrompt = m.fileContext.BuildContextPrompt()
							}

							m.textarea.Reset()
							if m.inputManager != nil {
								m.inputManager.ClearCompletions()
							}
							m.isLoading = true
							m.loadingMsg = "Thinking..."
							m.refreshViewport()

							return m, m.callAPI(contextPrompt, input)
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
	header := m.layoutManager.RenderHeader(filesCount, m.focusMode)

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

	// Combine all parts: header + main content + footer
	// This ensures header stays fixed at top while viewport scrolls
	return fmt.Sprintf("%s\n%s\n%s", header, mainContent, footer)
}

// renderFilesSidebar creates the files sidebar content
func (m *NewModel) renderFilesSidebar() string {
	return m.sidebar.RenderFilesSidebar(m.fileContext)
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
	// Update renderer with current viewport dimensions
	if m.renderer != nil {
		m.renderer.SetViewportWidth(m.viewport.Width, m.filesWidgetVisible)
	}

	// Save to session database
	if m.sessionManager != nil && m.currentSession != nil && role != "system" {
		m.sessionManager.SaveMessage(m.currentSession.ID, role, content)
	}

	// Store in API format for conversation context (exclude system messages)
	if role != "system" {
		m.apiMessages = append(m.apiMessages, api.Message{
			Role:    role,
			Content: content,
		})
		// Sync with AI operations
		if m.aiOperations != nil {
			m.aiOperations.SetAPIMessages(m.apiMessages)
		}
	}

	// Use renderer to format the message
	var formattedContent string
	if m.renderer != nil {
		formattedContent = m.renderer.FormatMessage(role, content)
	} else {
		// Fallback if renderer is not available
		formattedContent = fmt.Sprintf("%s: %s", role, content)
	}
	
	// Add to message history
	m.messages = append(m.messages, formattedContent)
	
	// Rebuild full content from all messages
	fullContent := strings.Join(m.messages, "\n\n")
	m.viewport.SetContent(fullContent)
	m.viewport.GotoBottom()
}

func (m *NewModel) refreshViewport() {
	// Rebuild viewport from message history
	if m.isLoading {
		// Add loading indicator temporarily
		loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
		loadingMsg := loadingStyle.Render("🔄 " + m.loadingMsg)

		// Add hint about cancellation
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		hintMsg := hintStyle.Render("Press Esc to cancel")

		// Show all messages plus loading indicator
		allContent := strings.Join(m.messages, "\n\n")
		if allContent != "" {
			m.viewport.SetContent(allContent + "\n\n" + loadingMsg + "\n" + hintMsg)
		} else {
			m.viewport.SetContent(loadingMsg + "\n" + hintMsg)
		}
		m.viewport.GotoBottom()
	} else {
		// Just show all messages
		fullContent := strings.Join(m.messages, "\n\n")
		m.viewport.SetContent(fullContent)
	}
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

	// Use streaming if enabled
	if m.streamingEnabled {
		cmd := m.aiOperations.CallAPIStream(contextPrompt, userInput)
		// Store the cancel function
		m.apiCancel = m.aiOperations.GetAPICancel()
		return cmd
	}

	// Fall back to non-streaming
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
	m.isLoading = false
	m.loadingMsg = ""
	m.apiCancel = nil
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
		} else {
			// Fallback to generic error message
			m.addMessage("system", fmt.Sprintf("❌ Error: %v", err))
		}
	} else {
		m.addMessage("assistant", response)
		// Track files mentioned in the AI response
		if m.fileTracker != nil {
			m.fileTracker.ExtractFilesFromResponse(response)
		}
	}
	m.viewport.GotoBottom()
}

// updateStreamingDisplay updates the display with streaming content
// Following the official Bubbletea chat example pattern
func (m *NewModel) updateStreamingDisplay(content string) {
	// Only update if we're in loading/streaming mode and have messages
	if !m.isLoading || len(m.messages) == 0 {
		return
	}

	// Update the last message (which should be our streaming assistant message)
	// This follows the exact pattern from the official Bubbletea chat example
	lastIdx := len(m.messages) - 1
	m.messages[lastIdx] = m.renderer.FormatMessage("assistant", content)

	// Follow official pattern: SetContent + GotoBottom
	m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
	m.viewport.GotoBottom()
}

// handleStreamComplete handles the completion of a stream
func (m *NewModel) handleStreamComplete(content string, err error) {
	m.isLoading = false
	m.loadingMsg = ""
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
			m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
		}

		// Track files mentioned in the AI response
		if m.fileTracker != nil {
			m.fileTracker.ExtractFilesFromResponse(content)
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

	m.messages = messages
	m.apiMessages = apiMessages
	// Sync with AI operations
	if m.aiOperations != nil {
		m.aiOperations.SetAPIMessages(m.apiMessages)
	}

	return nil
}

