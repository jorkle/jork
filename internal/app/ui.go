package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jorkle/jork/internal/models"
)

// UIState represents the current UI state
type UIState int

const (
	MainMenu UIState = iota
	ModeSelection
	KnowledgeLevelSelection
	Conversation
	Recording
	Processing
	Settings     // NEW: Settings menu state
	SettingsEdit // NEW: Settings edit dialog state
	APIKeyInput  // NEW: API Key input dialog state
	APIKeyVerifying // NEW: API Key verifying state
)

// Model represents the Bubbletea model
type Model struct {
	app             *App
	uiState         UIState
	textInput       string
	cursor          int
	selectedMode    int
	selectedLevel   int
	selectedSetting int
	message         string
	error           string
	lastResponse    string
	recording       bool
	recordingTime   time.Duration
	width           int
	height          int
	isSamplingVoice bool // NEW: flag for TTS voice sample playback
	editTitle       string
	editOptions     []string
	openaiKeyInput  string // NEW: for OpenAI API key input
	openaiKeyError  string // NEW: for displaying API key error
}

// NewModel creates a new Bubbletea model
func NewModel(app *App) *Model {
	return &Model{
		app:           app,
		uiState:       MainMenu,
		textInput:     "",
		cursor:        0,
		selectedMode:  int(app.state.CurrentMode),
		selectedLevel: int(app.state.KnowledgeLevel),
		width:         80,
		height:        24,
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case recordingTickMsg:
		if m.recording {
			m.recordingTime = msg.duration
			return m, m.tickRecording()
		}
		return m, nil

	case processingDoneMsg:
		m.uiState = Conversation
		m.lastResponse = msg.response
		m.error = msg.error
		return m, nil

	case RecordingStartedMsg:
		m.recording = true
		m.recordingTime = 0
		m.uiState = Recording
		return m, m.tickRecording()

	case RecordingStoppedMsg:
		m.recording = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.uiState = Conversation
		} else {
			m.uiState = Processing
			return m, m.processVoiceInput(msg.AudioData.(*models.AudioData))
		}
		return m, nil

	case ProcessingCompletedMsg:
		m.uiState = Conversation
		m.lastResponse = msg.Response
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.error = ""
		}
		return m, nil
	case APIKeyValidationDoneMsg:
		if msg.err != nil {
			m.openaiKeyError = "Validation failed: " + msg.err.Error()
			m.uiState = APIKeyInput
		} else {
			m.app.config.OpenAIAPIKey = m.openaiKeyInput
			m.uiState = MainMenu
		}
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.uiState {
	case MainMenu:
		return m.handleMainMenuKeys(msg)
	case ModeSelection:
		return m.handleModeSelectionKeys(msg)
	case KnowledgeLevelSelection:
		return m.handleKnowledgeLevelKeys(msg)
	case Conversation:
		return m.handleConversationKeys(msg)
	case Recording:
		return m.handleRecordingKeys(msg)
	case Processing:
		return m.handleProcessingKeys(msg)
	case Settings:
		return m.handleSettingsKeys(msg)
	case SettingsEdit:
		return m.handleSettingsEditKeys(msg)
	case APIKeyInput:
		return m.handleAPIKeyInputKeys(msg)
	case APIKeyVerifying:
		return m.handleAPIKeyVerifyingKeys(msg)
	default:
		return m, nil
	}
}

// handleMainMenuKeys handles main menu navigation
func (m *Model) handleMainMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "1":
		m.uiState = ModeSelection
		return m, nil
	case "2":
		m.uiState = KnowledgeLevelSelection
		return m, nil
	case "3":
		m.uiState = Conversation
		return m, nil
	case "4":
		// Show conversation history
		m.message = m.formatConversationHistory()
		return m, nil
	case "5":
		m.uiState = Settings
		return m, nil
	}
	return m, nil
}

// handleModeSelectionKeys handles mode selection
func (m *Model) handleModeSelectionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.uiState = MainMenu
		return m, nil
	case "up", "k":
		if m.selectedMode > 0 {
			m.selectedMode--
		}
		return m, nil
	case "down", "j":
		if m.selectedMode < 3 {
			m.selectedMode++
		}
		return m, nil
	case "enter":
		m.app.SetMode(models.CommunicationMode(m.selectedMode))
		m.uiState = MainMenu
		return m, nil
	}
	return m, nil
}

// handleKnowledgeLevelKeys handles knowledge level selection
func (m *Model) handleKnowledgeLevelKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.uiState = MainMenu
		return m, nil
	case "up", "k":
		if m.selectedLevel > 0 {
			m.selectedLevel--
		}
		return m, nil
	case "down", "j":
		if m.selectedLevel < 3 {
			m.selectedLevel++
		}
		return m, nil
	case "enter":
		m.app.SetKnowledgeLevel(models.KnowledgeLevel(m.selectedLevel))
		m.uiState = MainMenu
		return m, nil
	}
	return m, nil
}

// handleConversationKeys handles conversation input
func (m *Model) handleConversationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.uiState = MainMenu
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		return m.handleConversationSubmit()
	case "ctrl+r":
		return m.handleVoiceInput()
	case "backspace":
		if len(m.textInput) > 0 {
			m.textInput = m.textInput[:len(m.textInput)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.textInput += msg.String()
		}
		return m, nil
	}
}

// handleConversationSubmit handles text input submission
func (m *Model) handleConversationSubmit() (tea.Model, tea.Cmd) {
	if strings.TrimSpace(m.textInput) == "" {
		return m, nil
	}

	input := strings.TrimSpace(m.textInput)
	m.textInput = ""
	m.uiState = Processing
	m.error = ""

	return m, ProcessTextCmd(m.app, input)
}

// handleVoiceInput handles voice input
func (m *Model) handleVoiceInput() (tea.Model, tea.Cmd) {
	mode := m.app.state.CurrentMode
	if mode != models.VoiceToText && mode != models.VoiceToVoice {
		m.error = "Voice input not supported in current mode"
		return m, nil
	}

	return m, StartRecordingCmd(m.app)
}

// handleRecordingKeys handles recording state
func (m *Model) handleRecordingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "space":
		return m.stopRecording()
	case "q", "esc":
		m.app.StopRecording()
		m.recording = false
		m.uiState = Conversation
		return m, nil
	}
	return m, nil
}

// handleProcessingKeys handles processing state
func (m *Model) handleProcessingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.uiState = Conversation
		return m, nil
	}
	return m, nil
}

// stopRecording stops recording and processes the audio
func (m *Model) stopRecording() (tea.Model, tea.Cmd) {
	return m, StopRecordingCmd(m.app)
}

// View renders the UI
func (m *Model) View() string {
	switch m.uiState {
	case MainMenu:
		return m.renderMainMenu()
	case ModeSelection:
		return m.renderModeSelection()
	case KnowledgeLevelSelection:
		return m.renderKnowledgeLevelSelection()
	case Conversation:
		return m.renderConversation()
	case Recording:
		return m.renderRecording()
	case Processing:
		return m.renderProcessing()
	case Settings:
		return m.renderSettings()
	case SettingsEdit:
		return m.renderSettingsEdit()
	case APIKeyInput:
		return m.renderAPIKeyInput()
	case APIKeyVerifying:
		return m.renderAPIKeyVerifying()
	default:
		return "Unknown state"
	}
}

// renderMainMenu renders the main menu
func (m *Model) renderMainMenu() string {
	title := titleStyle.Render("JORK - AI Communication Assistant")

	state := m.app.GetState()
	status := fmt.Sprintf("Mode: %s | Knowledge Level: %s",
		state.CurrentMode.String(),
		state.KnowledgeLevel.String())

	menu := `
1. Select Communication Mode
2. Select Knowledge Level
3. Start Conversation
4. View Conversation History
5. Settings

Press 'q' to quit`

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		statusStyle.Render(status),
		"",
		menuStyle.Render(menu),
	)
}

// renderModeSelection renders the mode selection screen
func (m *Model) renderModeSelection() string {
	title := titleStyle.Render("Select Communication Mode")

	modes := []string{
		"Text → Voice",
		"Voice → Text",
		"Text → Text",
		"Voice → Voice",
	}

	var items []string
	for i, mode := range modes {
		if i == m.selectedMode {
			items = append(items, selectedStyle.Render("> "+mode))
		} else {
			items = append(items, "  "+mode)
		}
	}

	help := helpStyle.Render("↑/↓ to navigate, Enter to select, Esc to go back")

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		strings.Join(items, "\n"),
		"",
		help,
	)
}

// renderKnowledgeLevelSelection renders the knowledge level selection screen
func (m *Model) renderKnowledgeLevelSelection() string {
	title := titleStyle.Render("Select Knowledge Level")

	levels := []string{
		"Child",
		"High School Student",
		"Freshman University Student",
		"Co-worker in the Field",
	}

	var items []string
	for i, level := range levels {
		if i == m.selectedLevel {
			items = append(items, selectedStyle.Render("> "+level))
		} else {
			items = append(items, "  "+level)
		}
	}

	help := helpStyle.Render("↑/↓ to navigate, Enter to select, Esc to go back")

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		strings.Join(items, "\n"),
		"",
		help,
	)
}

// renderConversation renders the conversation interface
func (m *Model) renderConversation() string {
	state := m.app.GetState()
	title := titleStyle.Render("Conversation")

	status := fmt.Sprintf("Mode: %s | Knowledge Level: %s",
		state.CurrentMode.String(),
		state.KnowledgeLevel.String())

	var response string
	if m.lastResponse != "" {
		response = responseStyle.Render("AI: " + m.lastResponse)
	}

	var errorMsg string
	if m.error != "" {
		errorMsg = errorStyle.Render("Error: " + m.error)
	}

	input := inputStyle.Render("You: " + m.textInput + "█")

	var help string
	if state.CurrentMode == models.VoiceToText || state.CurrentMode == models.VoiceToVoice {
		help = helpStyle.Render("Type your message and press Enter, or press Ctrl+R for voice input. Esc to go back.")
	} else {
		help = helpStyle.Render("Type your message and press Enter. Esc to go back.")
	}

	parts := []string{title, "", statusStyle.Render(status), ""}

	if response != "" {
		parts = append(parts, response, "")
	}

	if errorMsg != "" {
		parts = append(parts, errorMsg, "")
	}

	parts = append(parts, input, "", help)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderRecording renders the recording interface
func (m *Model) renderRecording() string {
	title := titleStyle.Render("Recording...")

	duration := recordingStyle.Render(fmt.Sprintf("Duration: %.1fs", m.recordingTime.Seconds()))

	help := helpStyle.Render("Press Enter or Space to stop recording, Esc to cancel")

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		duration,
		"",
		help,
	)
}

// renderProcessing renders the processing interface
func (m *Model) renderProcessing() string {
	title := titleStyle.Render("Processing...")

	spinner := processingStyle.Render("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

	help := helpStyle.Render("Please wait...")

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		spinner,
		"",
		help,
	)
}

// renderStartupWizard renders the initial configuration wizard UI

// formatConversationHistory formats the conversation history for display
func (m *Model) formatConversationHistory() string {
	state := m.app.GetState()
	if len(state.ConversationLog) == 0 {
		return "No conversation history"
	}

	var history []string
	for _, entry := range state.ConversationLog {
		timestamp := entry.Timestamp.Format("15:04:05")
		history = append(history, fmt.Sprintf("[%s] You: %s", timestamp, entry.UserInput))
		history = append(history, fmt.Sprintf("[%s] AI: %s", timestamp, entry.AIResponse))
		history = append(history, "")
	}

	return strings.Join(history, "\n")
}

// Commands and messages

type recordingTickMsg struct {
	duration time.Duration
}

type processingDoneMsg struct {
	response string
	error    string
}

func (m *Model) tickRecording() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return recordingTickMsg{duration: m.recordingTime + 100*time.Millisecond}
	})
}

// APIKeyValidationDoneMsg indicates result of API key validation
type APIKeyValidationDoneMsg struct {
	err error
}

func ValidateAPIKeyCmd(app *App, apiKey string) tea.Cmd {
	return func() tea.Msg {
		// Update the client's API key before validation
		app.openaiClient.APIKey = apiKey
		err := app.openaiClient.ValidateAPIKey()
		return APIKeyValidationDoneMsg{err: err}
	}
}

// processVoiceInput creates a command to process voice input
func (m *Model) processVoiceInput(audioData *models.AudioData) tea.Cmd {
	return ProcessVoiceCmd(m.app, audioData)
}

// Add key handling for the Startup Wizard state
func (m *Model) handleStartupWizardKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Finish the setup wizard and return to the main menu
		m.uiState = MainMenu
		return m, nil
	case "esc", "q":
		m.uiState = MainMenu
		return m, nil
	default:
		return m, nil
	}
}

// Styles
func (m *Model) renderSettings() string {
	title := titleStyle.Render("Settings")

	// Prepare each setting as a string
	settings := []string{
		fmt.Sprintf("Conversation Model: %s", m.app.config.ConversationModel),
		fmt.Sprintf("TTS Model: %s", m.app.config.TTSTargetModel),
		fmt.Sprintf("TTS Voice: %s", m.app.config.TTSTargetVoice),
		fmt.Sprintf("STT Model: %s", m.app.config.STTTargetModel),
		fmt.Sprintf("Response Verbosity: %d", m.app.config.ResponseVerbosity),
		fmt.Sprintf("Speech Verbosity: %d", m.app.config.SpeechVerbosity),
	}
	encryptStr := "Off"
	if m.app.config.EncryptSettings {
		encryptStr = "On"
	}
	settings = append(settings, fmt.Sprintf("Encrypt Settings: %s", encryptStr))
	settings = append(settings, "OpenAI API Key: ****")

	// Render each setting, highlighting the selected one
	var renderedItems []string
	for i, setting := range settings {
		if i == m.selectedSetting {
			renderedItems = append(renderedItems, selectedStyle.Render("> "+setting))
		} else {
			renderedItems = append(renderedItems, "  "+setting)
		}
	}

	help := helpStyle.Render("↑/↓ to navigate, Enter to edit value, 'v' to sample TTS voice, Esc to return")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", strings.Join(renderedItems, "\n"), "", help)
}

func (m *Model) handleSettingsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedSetting > 0 {
			m.selectedSetting--
		}
		return m, nil
	case "down", "j":
		if m.selectedSetting < 7 {
			m.selectedSetting++
		}
		return m, nil
	case "esc", "q":
		m.uiState = MainMenu
		return m, nil
	case "v":
		if !m.isSamplingVoice {
			m.isSamplingVoice = true
			go func() {
				_ = m.app.PlayAudioSample()
			}()
		} else {
			_ = m.app.StopAudio()
			m.isSamplingVoice = false
		}
		return m, nil
	case "enter":
		// If the selected setting is "Encrypt Settings", toggle its value.
		if m.selectedSetting == 6 {
			m.app.config.EncryptSettings = !m.app.config.EncryptSettings
		} else if m.selectedSetting == 7 {
			m.editTitle = "Enter OpenAI API Key"
			m.editOptions = []string{m.app.config.OpenAIAPIKey}
		} else {
			// For other settings, enter the editing dialog
			switch m.selectedSetting {
			case 0:
				m.editTitle = "Select Conversation Model"
				m.editOptions = []string{"claude-3-5-sonnet-20241022", "gpt-4"}
			case 1:
				m.editTitle = "Select TTS Model"
				m.editOptions = []string{"tts-1", "tts-2"}
			case 2:
				m.editTitle = "Select TTS Voice"
				m.editOptions = []string{"alloy", "echo", "fable", "onyx", "nova", "shimmer"}
			case 3:
				m.editTitle = "Select STT Model"
				m.editOptions = []string{"whisper-1", "whisper-2"}
			case 4:
				m.editTitle = "Select Response Verbosity"
				m.editOptions = []string{"1", "2", "3"}
			case 5:
				m.editTitle = "Select Speech Verbosity"
				m.editOptions = []string{"1", "2", "3"}
			}
		}
		m.cursor = 0
		m.uiState = SettingsEdit
		return m, nil
	default:
		return m, nil
	}
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1)

	menuStyle = lipgloss.NewStyle().
			MarginLeft(2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("86")).
			Padding(0, 1)

	responseStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(0, 1).
			MarginBottom(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	recordingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	processingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)
)

func (m *Model) handleSettingsEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "j":
		if m.cursor < len(m.editOptions)-1 {
			m.cursor++
		}
		return m, nil
	case "enter":
		switch m.selectedSetting {
		case 0:
			m.app.config.ConversationModel = m.editOptions[m.cursor]
		case 1:
			m.app.config.TTSTargetModel = m.editOptions[m.cursor]
		case 2:
			m.app.config.TTSTargetVoice = m.editOptions[m.cursor]
		case 3:
			m.app.config.STTTargetModel = m.editOptions[m.cursor]
		case 4:
			if val, err := strconv.Atoi(m.editOptions[m.cursor]); err == nil {
				m.app.config.ResponseVerbosity = val
			}
		case 5:
			if val, err := strconv.Atoi(m.editOptions[m.cursor]); err == nil {
				m.app.config.SpeechVerbosity = val
			}
		case 7:
			m.app.config.OpenAIAPIKey = m.editOptions[m.cursor]
			// Trigger health check after updating the API key
			if err := m.app.HealthCheck(); err != nil {
				m.error = "Health Check failed: " + err.Error()
			} else {
				m.error = "Health Check passed"
			}
		}
		m.uiState = Settings
		return m, nil
	case "esc", "q":
		m.uiState = Settings
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) renderSettingsEdit() string {
	title := titleStyle.Render(m.editTitle)
	var items []string
	for i, option := range m.editOptions {
		if i == m.cursor {
			items = append(items, selectedStyle.Render("> "+option))
		} else {
			items = append(items, "  "+option)
		}
	}
	help := helpStyle.Render("Use ↑/↓ to navigate, Enter to confirm, Esc to cancel")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", strings.Join(items, "\n"), "", help)
}
