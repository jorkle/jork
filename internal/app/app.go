package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jorkle/jork/internal/ai"
	"github.com/jorkle/jork/internal/audio"
	"github.com/jorkle/jork/internal/config"
	"github.com/jorkle/jork/internal/models"
)

// App represents the main application
type App struct {
	config       *config.Config
	claudeClient *ai.ClaudeClient
	ttsClient    *ai.TTSClient
	sttClient    *ai.STTClient
	recorder     *audio.Recorder
	player       *audio.Player
	state        *models.AppState
}

// NewApp creates a new application instance
func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize AI clients
	claudeClient := ai.NewClaudeClient(cfg.AnthropicAPIKey, cfg.ClaudeModel)
	ttsClient := ai.NewTTSClient(cfg.OpenAIAPIKey, cfg.OpenAITTSModel, cfg.OpenAITTSVoice)
	sttClient := ai.NewSTTClient(cfg.OpenAIAPIKey, cfg.OpenAISTTModel)

	// Initialize audio components
	recorder, err := audio.NewRecorder(cfg.SampleRate, 1) // mono
	if err != nil {
		return nil, fmt.Errorf("failed to create audio recorder: %w", err)
	}

	player := audio.NewPlayer()

	// Initialize app state
	state := &models.AppState{
		CurrentMode:     cfg.DefaultMode,
		KnowledgeLevel:  cfg.DefaultKnowledgeLevel,
		IsRecording:     false,
		IsPlaying:       false,
		IsProcessing:    false,
		ConversationLog: make([]models.ConversationEntry, 0),
	}

	return &App{
		config:       cfg,
		claudeClient: claudeClient,
		ttsClient:    ttsClient,
		sttClient:    sttClient,
		recorder:     recorder,
		player:       player,
		state:        state,
	}, nil
}

// Run starts the application
func (a *App) Run() error {
	// Validate API keys
	if err := a.claudeClient.ValidateAPIKey(); err != nil {
		return fmt.Errorf("invalid Anthropic API key: %w", err)
	}

	if err := a.ttsClient.ValidateAPIKey(); err != nil {
		return fmt.Errorf("invalid OpenAI TTS API key: %w", err)
	}

	if err := a.sttClient.ValidateAPIKey(); err != nil {
		return fmt.Errorf("invalid OpenAI STT API key: %w", err)
	}

	// Create and run the Bubbletea program
	model := NewModel(a)
	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run program: %w", err)
	}

	return nil
}

// ProcessTextInput processes text input and returns AI response
func (a *App) ProcessTextInput(input string) (string, error) {
	a.state.IsProcessing = true
	defer func() { a.state.IsProcessing = false }()

	// Generate response using Claude
	response, err := a.claudeClient.GenerateResponse(
		input,
		a.state.KnowledgeLevel,
		a.state.CurrentMode,
		a.state.ConversationLog,
		"general", // topic - could be made configurable
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	// Log the conversation
	entry := models.ConversationEntry{
		Timestamp:      time.Now(),
		UserInput:      input,
		AIResponse:     response,
		Mode:           a.state.CurrentMode,
		KnowledgeLevel: a.state.KnowledgeLevel,
		IsVoiceInput:   a.state.CurrentMode == models.VoiceToText || a.state.CurrentMode == models.VoiceToVoice,
		IsVoiceOutput:  a.state.CurrentMode == models.TextToVoice || a.state.CurrentMode == models.VoiceToVoice,
	}

	a.state.ConversationLog = append(a.state.ConversationLog, entry)

	// Keep only the last N entries
	if len(a.state.ConversationLog) > a.config.MaxConversationHistory {
		a.state.ConversationLog = a.state.ConversationLog[len(a.state.ConversationLog)-a.config.MaxConversationHistory:]
	}

	a.state.LastMessage = input
	a.state.LastResponse = response

	return response, nil
}

// ProcessVoiceInput processes voice input and returns appropriate response
func (a *App) ProcessVoiceInput(audioData *models.AudioData) (string, error) {
	a.state.IsProcessing = true
	defer func() { a.state.IsProcessing = false }()

	// Save audio to temporary file for processing
	tempFile := filepath.Join(a.config.AudioTempDir, fmt.Sprintf("input_%d.wav", time.Now().Unix()))
	if err := a.recorder.SaveToWAV(audioData, tempFile); err != nil {
		return "", fmt.Errorf("failed to save audio: %w", err)
	}
	defer os.Remove(tempFile)

	// Convert speech to text using OpenAI Whisper
	transcription, err := a.sttClient.SpeechToText(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to transcribe audio: %w", err)
	}

	// Process the transcription as text
	return a.ProcessTextInput(transcription)
}

// GenerateVoiceResponse converts text response to speech
func (a *App) GenerateVoiceResponse(text string) (string, error) {
	a.state.IsProcessing = true
	defer func() { a.state.IsProcessing = false }()

	// Generate unique filename
	filename := filepath.Join(a.config.AudioTempDir, fmt.Sprintf("response_%d.mp3", time.Now().Unix()))

	// Convert text to speech
	if err := a.ttsClient.TextToSpeech(text, filename); err != nil {
		return "", fmt.Errorf("failed to generate speech: %w", err)
	}

	return filename, nil
}

// StartRecording starts audio recording
func (a *App) StartRecording() error {
	if a.state.IsRecording {
		return fmt.Errorf("already recording")
	}

	if err := a.recorder.StartRecording(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	a.state.IsRecording = true
	return nil
}

// StopRecording stops audio recording and returns the recorded data
func (a *App) StopRecording() (*models.AudioData, error) {
	if !a.state.IsRecording {
		return nil, fmt.Errorf("not currently recording")
	}

	audioData, err := a.recorder.StopRecording()
	if err != nil {
		return nil, fmt.Errorf("failed to stop recording: %w", err)
	}

	a.state.IsRecording = false
	return audioData, nil
}

// PlayAudio plays an audio file
func (a *App) PlayAudio(filename string) error {
	if a.state.IsPlaying {
		return fmt.Errorf("already playing audio")
	}

	// Determine file type and play accordingly
	ext := filepath.Ext(filename)
	switch ext {
	case ".mp3":
		if err := a.player.PlayMP3File(filename); err != nil {
			return fmt.Errorf("failed to play MP3: %w", err)
		}
	case ".wav":
		if err := a.player.PlayFile(filename); err != nil {
			return fmt.Errorf("failed to play WAV: %w", err)
		}
	default:
		return fmt.Errorf("unsupported audio format: %s", ext)
	}

	a.state.IsPlaying = true

	// Start a goroutine to monitor playback status
	go func() {
		a.player.WaitForPlayback()
		a.state.IsPlaying = false
	}()

	return nil
}

// StopAudio stops current audio playback
func (a *App) StopAudio() error {
	if !a.state.IsPlaying {
		return fmt.Errorf("no audio is currently playing")
	}

	if err := a.player.StopPlayback(); err != nil {
		return fmt.Errorf("failed to stop playback: %w", err)
	}

	a.state.IsPlaying = false
	return nil
}

// PlayAudioSample generates and plays a sample TTS audio using the current TTS settings.
func (a *App) PlayAudioSample() error {
	sampleText := "This is a sample voice from the selected TTS configuration."
	filename := filepath.Join(a.config.AudioTempDir, "sample_voice.mp3")
	if err := a.ttsClient.TextToSpeech(sampleText, filename); err != nil {
		return fmt.Errorf("failed to generate TTS sample: %w", err)
	}
	return a.player.PlayMP3File(filename)
}

// GenerateExplanationSample creates a sample explanation using the current knowledge level.
func (a *App) GenerateExplanationSample() (string, error) {
	prompt := fmt.Sprintf("Explain photosynthesis in a way suitable for %s.", a.state.KnowledgeLevel.String())
	return a.claudeClient.GenerateResponse(
		prompt,
		a.state.KnowledgeLevel,
		a.state.CurrentMode,
		a.state.ConversationLog,
		"general",
	)
}

// SetMode changes the communication mode
func (a *App) SetMode(mode models.CommunicationMode) {
	a.state.CurrentMode = mode
}

// SetKnowledgeLevel changes the knowledge level
func (a *App) SetKnowledgeLevel(level models.KnowledgeLevel) {
	a.state.KnowledgeLevel = level
}

// GetState returns the current application state
func (a *App) GetState() *models.AppState {
	return a.state
}

// Cleanup performs cleanup operations
func (a *App) Cleanup() error {
	// Stop any ongoing recording
	if a.state.IsRecording {
		a.recorder.StopRecording()
	}

	// Stop any ongoing playback
	if a.state.IsPlaying {
		a.player.StopPlayback()
	}

	// Close audio recorder
	if err := a.recorder.Close(); err != nil {
		log.Printf("Error closing recorder: %v", err)
	}

	// Clean up temporary audio files
	if err := a.cleanupTempFiles(); err != nil {
		log.Printf("Error cleaning up temp files: %v", err)
	}

	return nil
}

// cleanupTempFiles removes temporary audio files
func (a *App) cleanupTempFiles() error {
	return filepath.Walk(a.config.AudioTempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Remove files older than 1 hour
			if time.Since(info.ModTime()) > time.Hour {
				return os.Remove(path)
			}
		}

		return nil
	})
}
