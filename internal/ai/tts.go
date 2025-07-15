package ai

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sashabaranov/go-openai"
)

// TTSClient handles text-to-speech conversion using OpenAI
type TTSClient struct {
	client *openai.Client
	model  string
	voice  string
}

// NewTTSClient creates a new TTS client
func NewTTSClient(apiKey, model, voice string) *TTSClient {
	return &TTSClient{
		client: openai.NewClient(apiKey),
		model:  model,
		voice:  voice,
	}
}

// TextToSpeech converts text to audio and saves it to a file
func (t *TTSClient) TextToSpeech(text string, outputPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create the TTS request
	req := openai.CreateSpeechRequest{
		Model: openai.SpeechModel(t.model),
		Input: text,
		Voice: openai.VoiceAlloy, // Default voice, can be made configurable
	}

	// Override with configured voice if available
	switch t.voice {
	case "alloy":
		req.Voice = openai.VoiceAlloy
	case "echo":
		req.Voice = openai.VoiceEcho
	case "fable":
		req.Voice = openai.VoiceFable
	case "onyx":
		req.Voice = openai.VoiceOnyx
	case "nova":
		req.Voice = openai.VoiceNova
	case "shimmer":
		req.Voice = openai.VoiceShimmer
	default:
		req.Voice = openai.VoiceAlloy
	}

	// Make the request
	response, err := t.client.CreateSpeech(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create speech: %w", err)
	}
	defer response.Close()

	// Ensure the output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Copy the audio data to the file
	_, err = io.Copy(file, response)
	if err != nil {
		return fmt.Errorf("failed to write audio data: %w", err)
	}

	return nil
}

// ValidateAPIKey checks if the OpenAI API key is valid
func (t *TTSClient) ValidateAPIKey() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to make a simple request to validate the API key
	req := openai.CreateSpeechRequest{
		Model: openai.SpeechModel(t.model),
		Input: "test",
		Voice: openai.VoiceAlloy,
	}

	response, err := t.client.CreateSpeech(ctx, req)
	if err != nil {
		return fmt.Errorf("invalid OpenAI API key or TTS access: %w", err)
	}
	defer response.Close()

	return nil
}

// GetAvailableVoices returns the list of available voices
func (t *TTSClient) GetAvailableVoices() []string {
	return []string{"alloy", "echo", "fable", "onyx", "nova", "shimmer"}
}

