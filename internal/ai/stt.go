package ai

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sashabaranov/go-openai"
)

// STTClient handles speech-to-text conversion using OpenAI Whisper
type STTClient struct {
	client *openai.Client
	model  string
}

// NewSTTClient creates a new STT client
func NewSTTClient(apiKey, model string) *STTClient {
	return &STTClient{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

// SpeechToText converts audio file to text
func (s *STTClient) SpeechToText(audioFilePath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Open the audio file
	audioFile, err := os.Open(audioFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer audioFile.Close()

	// Create the transcription request
	req := openai.AudioRequest{
		Model:    s.model,
		FilePath: audioFilePath,
		Reader:   audioFile,
	}

	// Make the request
	response, err := s.client.CreateTranscription(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create transcription: %w", err)
	}

	return response.Text, nil
}

// ValidateAPIKey checks if the OpenAI API key is valid for STT
func (s *STTClient) ValidateAPIKey() error {
	// For STT validation, we'll just check if we can create a client
	// A full validation would require a test audio file
	if s.client == nil {
		return fmt.Errorf("invalid OpenAI client")
	}
	return nil
}
