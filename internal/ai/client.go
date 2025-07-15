package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jorkle/jork/internal/models"
)

 // OpenAIClient handles communication with the OpenAI API
type OpenAIClient struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
	BaseURL    string
}

// NewClaudeClient creates a new Claude API client
func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	return &OpenAIClient{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: "https://api.openai.com/v1/chat/completions",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateResponse sends a request to Claude and returns the response
func (c *OpenAIClient) GenerateResponse(
	userInput string,
	knowledgeLevel models.KnowledgeLevel,
	mode models.CommunicationMode,
	conversationHistory []models.ConversationEntry,
	topic string,
) (string, error) {
	// Build the system prompt
	systemPrompt := GetSystemPrompt(knowledgeLevel, topic)
	systemPrompt += GetModeInstructions(mode)

	// Build conversation context
	messages := GetConversationContext(conversationHistory, 10)

	// Add the current user input
	formattedInput := FormatUserInput(userInput, mode)
	messages = append(messages, models.Message{
		Role:    "user",
		Content: formattedInput,
	})
	var requestBody []byte
	var err error
	if strings.Contains(strings.ToLower(c.Model), "claude") {
		req := struct {
			Model               string           `json:"model"`
			Messages            []models.Message `json:"messages"`
			MaxCompletionTokens int              `json:"max_completion_tokens"`
		}{
			Model:               c.Model,
			Messages:            messages,
			MaxCompletionTokens: 1000,
		}
		requestBody, err = json.Marshal(req)
	} else {
		req := struct {
			Model               string           `json:"model"`
			Messages            []models.Message `json:"messages"`
			Temperature         float32          `json:"temperature"`
			MaxCompletionTokens int              `json:"max_completion_tokens"`
		}{
			Model:               c.Model,
			Messages:            messages,
			Temperature:         0.7,
			MaxCompletionTokens: 1000,
		}
		requestBody, err = json.Marshal(req)
	}
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Send the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var claudeResponse models.ClaudeResponse
	if err := json.Unmarshal(body, &claudeResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract the text content
	if len(claudeResponse.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return claudeResponse.Content[0].Text, nil
}

// ValidateAPIKey checks if the API key is valid by making a simple request
func (c *OpenAIClient) ValidateAPIKey() error {
	testMessages := []models.Message{
		{
			Role:    "user",
			Content: "Hello",
		},
	}
	
	var requestBody []byte
	var err error
	if strings.Contains(strings.ToLower(c.Model), "claude") {
		req := struct {
			Model               string           `json:"model"`
			Messages            []models.Message `json:"messages"`
			MaxCompletionTokens int              `json:"max_completion_tokens"`
		}{
			Model:               c.Model,
			Messages:            testMessages,
			MaxCompletionTokens: 10,
		}
		requestBody, err = json.Marshal(req)
	} else {
		req := struct {
			Model               string           `json:"model"`
			Messages            []models.Message `json:"messages"`
			Temperature         float32          `json:"temperature"`
			MaxCompletionTokens int              `json:"max_completion_tokens"`
		}{
			Model:               c.Model,
			Messages:            testMessages,
			Temperature:         0.7,
			MaxCompletionTokens: 10,
		}
		requestBody, err = json.Marshal(req)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal test request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send test request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API validation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *OpenAIClient) FetchAvailableModels() ([]string, error) {
	url := "https://api.openai.com/v1/models"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch models, status %d: %s", resp.StatusCode, string(body))
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		id := m.ID
		// Skip models with a date timestamp pattern (e.g. "2023-03-14")
		if matched, _ := regexp.MatchString(`\d{4}-\d{2}-\d{2}`, id); matched {
			continue
		}
		// Skip models not suitable for text-to-text communication (e.g. Whisper models, DALL·E variants, embeddings, TTS, moderation, realtime, research, image, transcribe, audio, instruct, davinci, babbage, codex, search)
		if strings.Contains(id, "whisper") ||
			strings.Contains(id, "dalle") ||
			strings.Contains(id, "dall-e") ||
			strings.Contains(id, "embedding") ||
			strings.Contains(id, "instruct") ||
			strings.Contains(id, "davinci") ||
			strings.Contains(id, "babbage") ||
			strings.Contains(id, "codex") ||
			strings.Contains(id, "tts") ||
			strings.Contains(id, "moderation") ||
			strings.Contains(id, "realtime") ||
			strings.Contains(id, "research") ||
			strings.Contains(id, "image") ||
			strings.Contains(id, "transcribe") ||
			strings.Contains(id, "audio") ||
			strings.Contains(id, "search") {
			continue
		}
		models = append(models, id)
	}
	return models, nil
}
