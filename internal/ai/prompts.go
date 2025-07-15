package ai

import (
	"fmt"
	"github.com/jorkle/jork/internal/models"
)

// GetSystemPrompt returns the system prompt based on knowledge level
func GetSystemPrompt(level models.KnowledgeLevel, topic string) string {
	basePrompt := `You are an AI assistant helping to explain concepts and answer questions. Your responses should be tailored to the knowledge level of the person you're talking to.`
	
	switch level {
	case models.Child:
		return basePrompt + fmt.Sprintf(`

KNOWLEDGE LEVEL: Child (ages 5-10)
- Use very simple words and short sentences
- Explain things like you're talking to a curious child
- Use analogies to familiar things (toys, animals, family, etc.)
- Be patient and encouraging
- Avoid technical jargon completely
- Use concrete examples rather than abstract concepts
- Make it fun and engaging

Topic context: %s

Remember: A child is asking about this topic, so break it down into the simplest possible terms.`, topic)

	case models.HighSchool:
		return basePrompt + fmt.Sprintf(`

KNOWLEDGE LEVEL: High School Student (ages 14-18)
- Use high school level vocabulary
- Explain concepts clearly but you can use some technical terms if you explain them
- Reference things they might know from school subjects
- Be encouraging and educational
- You can assume basic math, science, and general knowledge
- Use examples from everyday life and popular culture

Topic context: %s

Remember: You're talking to a high school student, so use appropriate complexity and examples.`, topic)

	case models.FreshmanUniversity:
		return basePrompt + fmt.Sprintf(`

KNOWLEDGE LEVEL: Freshman University Student
- Use university-level vocabulary and concepts
- You can assume they have basic knowledge in the field being discussed
- Reference fundamental concepts they should know from introductory courses
- Be informative and somewhat academic in tone
- You can use technical terms but still explain complex ones
- Reference textbook concepts and academic examples

Topic context: %s

Remember: You're talking to a freshman university student in this field, so they have some foundational knowledge.`, topic)

	case models.CoWorker:
		return basePrompt + fmt.Sprintf(`

KNOWLEDGE LEVEL: Co-worker in the Field
- Use professional terminology freely
- Assume deep knowledge in the field
- Reference advanced concepts, tools, and methodologies
- Be direct and efficient in communication
- You can discuss complex topics without extensive explanation
- Reference industry standards, best practices, and current developments

Topic context: %s

Remember: You're talking to a knowledgeable colleague, so communicate at a professional level.`, topic)

	default:
		return basePrompt + "\n\nPlease provide helpful and appropriate responses."
	}
}

// GetConversationContext builds context from previous conversation entries
func GetConversationContext(entries []models.ConversationEntry, maxEntries int) []models.Message {
	if len(entries) == 0 {
		return nil
	}
	
	// Take the last maxEntries entries
	start := 0
	if len(entries) > maxEntries {
		start = len(entries) - maxEntries
	}
	
	messages := make([]models.Message, 0, len(entries[start:])*2)
	
	for _, entry := range entries[start:] {
		// Add user message
		messages = append(messages, models.Message{
			Role:    "user",
			Content: entry.UserInput,
		})
		
		// Add assistant response
		messages = append(messages, models.Message{
			Role:    "assistant",
			Content: entry.AIResponse,
		})
	}
	
	return messages
}

// FormatUserInput formats user input based on the communication mode
func FormatUserInput(input string, mode models.CommunicationMode) string {
	switch mode {
	case models.VoiceToText, models.VoiceToVoice:
		return fmt.Sprintf("[Voice Input] %s", input)
	case models.TextToText, models.TextToVoice:
		return input
	default:
		return input
	}
}

// GetModeInstructions returns additional instructions based on communication mode
func GetModeInstructions(mode models.CommunicationMode) string {
	switch mode {
	case models.TextToVoice, models.VoiceToVoice:
		return "\n\nIMPORTANT: Your response will be converted to speech, so:\n- Write in a natural, conversational tone\n- Avoid special characters, formatting, or symbols\n- Use words instead of abbreviations\n- Keep sentences clear and well-paced for speech"
	case models.VoiceToText, models.TextToText:
		return "\n\nYou can use normal text formatting in your response."
	default:
		return ""
	}
}

