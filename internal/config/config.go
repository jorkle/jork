package config

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/jorkle/jork/internal/models"
)

// Config holds the application configuration
type Config struct {
	// API Configuration
	AnthropicAPIKey string
	OpenAIAPIKey    string
	
	// AI Model Configuration
	ClaudeModel     string
	OpenAITTSModel  string
	OpenAITTSVoice  string
	ConversationModel string
	TTSTargetModel    string
	TTSTargetVoice    string
	STTTargetModel    string
	ResponseVerbosity int
	SpeechSpeed       int
	AvailableModels   []string
	EncryptSettings   bool
	OpenAISTTModel  string
	
	// Audio Configuration
	SampleRate      int
	BufferSize      int
	InputDevice     string
	OutputDevice    string
	
	// Application Settings
	DefaultMode           models.CommunicationMode
	DefaultKnowledgeLevel models.KnowledgeLevel
	MaxConversationHistory int
	
	// File Paths
	ConfigDir   string
	LogFile     string
	AudioTempDir string
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "jork")
	
	return &Config{
		// API Configuration - will be loaded from environment
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		
		// AI Model Configuration
		ClaudeModel: func() string {
			if v := os.Getenv("OPENAI_MODEL"); v != "" {
				return v
			}
			return "gpt-4"
		}(),
		OpenAITTSModel:  "tts-1",
		OpenAITTSVoice:  "alloy",
		ConversationModel: "gpt-4",
		TTSTargetModel:    "tts-1",
		TTSTargetVoice:    "alloy",
		STTTargetModel:    "whisper-1",
		ResponseVerbosity: 2,
		SpeechSpeed:       2,
		AvailableModels:   []string{},
		EncryptSettings:   false,
		OpenAISTTModel:  "whisper-1",
		
		// Audio Configuration
		SampleRate:      44100,
		BufferSize:      1024,
		InputDevice:     "default",
		OutputDevice:    "default",
		
		// Application Settings
		DefaultMode:           models.TextToText,
		DefaultKnowledgeLevel: models.CoWorker,
		MaxConversationHistory: 50,
		
		// File Paths
		ConfigDir:    configDir,
		LogFile:      filepath.Join(configDir, "conversation.log"),
		AudioTempDir: filepath.Join(configDir, "audio_temp"),
	}
}

// Load loads configuration from environment variables and validates it
func Load() (*Config, error) {
	config := DefaultConfig()
	
	// Validate required API keys
	if config.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}
	
	// Create necessary directories
	if err := os.MkdirAll(config.ConfigDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	
	if err := os.MkdirAll(config.AudioTempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audio temp directory: %w", err)
	}
	
	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.OpenAIAPIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}
	
	if c.SampleRate <= 0 {
		return fmt.Errorf("sample rate must be positive")
	}
	
	if c.BufferSize <= 0 {
		return fmt.Errorf("buffer size must be positive")
	}
	
	return nil
}

