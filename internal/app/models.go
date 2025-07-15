package app

import (
	"time"
	
	"github.com/jorkle/jork/internal/models"
)

// UIModel represents the UI state and data
type UIModel struct {
	// Current UI state
	CurrentView string
	
	// Input handling
	TextInput     string
	CursorPos     int
	
	// Selection states
	SelectedMode  int
	SelectedLevel int
	
	// Status and messages
	StatusMessage string
	ErrorMessage  string
	LastResponse  string
	
	// Recording state
	IsRecording   bool
	RecordingTime time.Duration
	
	// Processing state
	IsProcessing bool
	
	// Display dimensions
	Width  int
	Height int
	
	// Conversation history for display
	ConversationHistory []models.ConversationEntry
}

// NewUIModel creates a new UI model with default values
func NewUIModel() *UIModel {
	return &UIModel{
		CurrentView:   "main_menu",
		TextInput:     "",
		CursorPos:     0,
		SelectedMode:  0,
		SelectedLevel: 3, // Default to CoWorker level
		Width:         80,
		Height:        24,
		ConversationHistory: make([]models.ConversationEntry, 0),
	}
}

// Reset clears the UI model state
func (m *UIModel) Reset() {
	m.TextInput = ""
	m.CursorPos = 0
	m.StatusMessage = ""
	m.ErrorMessage = ""
	m.IsRecording = false
	m.RecordingTime = 0
	m.IsProcessing = false
}

// SetError sets an error message
func (m *UIModel) SetError(err error) {
	if err != nil {
		m.ErrorMessage = err.Error()
	} else {
		m.ErrorMessage = ""
	}
}

// ClearError clears the error message
func (m *UIModel) ClearError() {
	m.ErrorMessage = ""
}

// SetStatus sets a status message
func (m *UIModel) SetStatus(message string) {
	m.StatusMessage = message
}

// ClearStatus clears the status message
func (m *UIModel) ClearStatus() {
	m.StatusMessage = ""
}
