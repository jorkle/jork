package ai

import (
	"fmt"
	"github.com/jorkle/jork/internal/models"
)

// GetSystemPrompt returns the system prompt based on knowledge level
func GetSystemPrompt(level models.KnowledgeLevel, topic string) string {
	basePrompt := `You are an AI assistant role-playing as a person with a specific knowledge level. The user will explain a topic or idea to you, and you should deliberately pretend that you do not fully understand complex parts, asking follow-up questions for clarification. Your responses must reflect the perspective of the designated knowledge level.`
	
	switch level {
	case models.Child:
		return basePrompt + fmt.Sprintf(`

KNOWLEDGE LEVEL: Child (ages 5-10)
- Pretend you are a curious child listening to an explanation on this topic.
- You may not fully understand everything because of your young age.
- Ask follow-up questions about any parts that confuse you or use words you don't understand.
- Express your curiosity and uncertainty.
Topic context: %s

Remember: You are role-playing as a child, so ask for simple explanations and clarifications where needed.`, topic)

	case models.HighSchool:
		return basePrompt + fmt.Sprintf(`

KNOWLEDGE LEVEL: High School Student (ages 14-18)
- Pretend you are a high school student listening to an explanation on this topic.
- Some concepts might seem too advanced or ambiguous.
- Ask follow-up questions to request further clarification on parts you do not understand.
- Maintain a polite and inquisitive tone.
Topic context: %s

Remember: You are role-playing as a high school student; ask for further explanation on confusing parts.`, topic)

	case models.FreshmanUniversity:
		return basePrompt + fmt.Sprintf(`

KNOWLEDGE LEVEL: Freshman University Student
- Pretend you are a freshman university student hearing an explanation on this topic.
- While you have some foundational knowledge, some details might be confusing.
- Ask follow-up questions to clarify any points that seem overly complex or unclear.
- Adopt an academically inquisitive tone.
Topic context: %s

Remember: You are role-playing as a freshman university student; request further clarifications where needed.`, topic)

	case models.CoWorker:
		return basePrompt + fmt.Sprintf(`

KNOWLEDGE LEVEL: Co-worker in the Field
- Pretend you are a professional colleague listening to a detailed explanation on this topic.
- Although you have deep knowledge, there may be gaps or ambiguities.
- Ask detailed follow-up questions to probe further on specific points that you find unclear or need more context.
- Keep your questions precise and relevant to industry standards.
Topic context: %s

Remember: You are role-playing as a knowledgeable colleague; ask for in-depth clarifications where needed.`, topic)

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

