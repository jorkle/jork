package models

import "time"

// CommunicationMode represents the four different communication modes
type CommunicationMode int

const (
	TextToVoice CommunicationMode = iota
	VoiceToText
	TextToText
	VoiceToVoice
)

func (m CommunicationMode) String() string {
	switch m {
	case TextToVoice:
		return "Text → Voice"
	case VoiceToText:
		return "Voice → Text"
	case TextToText:
		return "Text → Text"
	case VoiceToVoice:
		return "Voice → Voice"
	default:
		return "Unknown"
	}
}

// KnowledgeLevel represents the AI's knowledge level setting
type KnowledgeLevel int

const (
	Child KnowledgeLevel = iota
	HighSchool
	FreshmanUniversity
	CoWorker
)

func (k KnowledgeLevel) String() string {
	switch k {
	case Child:
		return "Child"
	case HighSchool:
		return "High School Student"
	case FreshmanUniversity:
		return "Freshman University Student"
	case CoWorker:
		return "Co-worker in the Field"
	default:
		return "Unknown"
	}
}

func (k KnowledgeLevel) Description() string {
	switch k {
	case Child:
		return "Explain concepts in very simple terms, like talking to a curious child"
	case HighSchool:
		return "Use high school level vocabulary and concepts"
	case FreshmanUniversity:
		return "Assume basic university-level understanding in the field"
	case CoWorker:
		return "Communicate as if talking to a knowledgeable colleague"
	default:
		return "Unknown level"
	}
}

// AppState represents the current state of the application
type AppState struct {
	CurrentMode     CommunicationMode
	KnowledgeLevel  KnowledgeLevel
	IsRecording     bool
	IsPlaying       bool
	IsProcessing    bool
	LastMessage     string
	LastResponse    string
	ConversationLog []ConversationEntry
}

// ConversationEntry represents a single exchange in the conversation
type ConversationEntry struct {
	Timestamp    time.Time
	UserInput    string
	AIResponse   string
	Mode         CommunicationMode
	KnowledgeLevel KnowledgeLevel
	IsVoiceInput bool
	IsVoiceOutput bool
}

// ClaudeRequest represents a structured request to Claude API
type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
	System    string    `json:"system,omitempty"`
}

// ClaudeResponse represents Claude's structured response
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AudioData represents audio data for recording/playback
type AudioData struct {
	Data       []float32
	SampleRate int
	Duration   time.Duration
}

