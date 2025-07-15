package app

import (
	"fmt"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jorkle/jork/internal/models"
)

// Command messages for the Bubbletea application

// RecordingStartedMsg indicates recording has started
type RecordingStartedMsg struct{}

// RecordingStoppedMsg indicates recording has stopped
type RecordingStoppedMsg struct {
	AudioData interface{} // Will contain *models.AudioData
	Error     error
}

// ProcessingStartedMsg indicates AI processing has started
type ProcessingStartedMsg struct{}

// ProcessingCompletedMsg indicates AI processing has completed
type ProcessingCompletedMsg struct {
	Response string
	Error    error
}

// AudioPlaybackStartedMsg indicates audio playback has started
type AudioPlaybackStartedMsg struct{}

// AudioPlaybackStoppedMsg indicates audio playback has stopped
type AudioPlaybackStoppedMsg struct {
	Error error
}

// StartRecordingCmd returns a command to start recording
func StartRecordingCmd(app *App) tea.Cmd {
	return func() tea.Msg {
		if err := app.StartRecording(); err != nil {
			return RecordingStartedMsg{} // Even if error, we tried to start
		}
		return RecordingStartedMsg{}
	}
}

// StopRecordingCmd returns a command to stop recording
func StopRecordingCmd(app *App) tea.Cmd {
	return func() tea.Msg {
		audioData, err := app.StopRecording()
		return RecordingStoppedMsg{
			AudioData: audioData,
			Error:     err,
		}
	}
}

// ProcessTextCmd returns a command to process text input
func ProcessTextCmd(app *App, input string) tea.Cmd {
	return func() tea.Msg {
		// Run health check before starting conversation
		if err := app.HealthCheck(); err != nil {
			return ProcessingCompletedMsg{
				Response: "",
				Error:    fmt.Errorf("Health check failed: %s", err.Error()),
			}
		}
		response, err := app.ProcessTextInput(input)
		
		// Handle voice output if needed
		if err == nil && (app.state.CurrentMode == models.TextToVoice || app.state.CurrentMode == models.VoiceToVoice) {
			if audioFile, audioErr := app.GenerateVoiceResponse(response); audioErr == nil {
				go app.PlayAudio(audioFile) // Play in background
			}
		}
		
		return ProcessingCompletedMsg{
			Response: response,
			Error:    err,
		}
	}
}

// ProcessVoiceCmd returns a command to process voice input
func ProcessVoiceCmd(app *App, audioData interface{}) tea.Cmd {
	return func() tea.Msg {
		// Type assertion to get the actual audio data
		if data, ok := audioData.(*models.AudioData); ok {
			response, err := app.ProcessVoiceInput(data)
			
			// Handle voice output if needed
			if err == nil && app.state.CurrentMode == models.VoiceToVoice {
				if audioFile, audioErr := app.GenerateVoiceResponse(response); audioErr == nil {
					go app.PlayAudio(audioFile) // Play in background
				}
			}
			
			msgResponse := response
			if app.state.CurrentMode == models.VoiceToVoice {
				msgResponse = "[Voice response played]"
			}
			return ProcessingCompletedMsg{
				Response: msgResponse,
				Error:    err,
			}
		}
		return ProcessingCompletedMsg{
			Response: "",
			Error:    fmt.Errorf("invalid audio data"),
		}
	}
}
