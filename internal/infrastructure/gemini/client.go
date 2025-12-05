package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewGeminiClient(apiKey string) (*GeminiClient, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	model := client.GenerativeModel("gemini-1.5-pro")
	model.SetTemperature(0.7)

	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

func (c *GeminiClient) Close() {
	c.client.Close()
}

func (c *GeminiClient) GenerateMatchExplanation(ctx context.Context, user1Traits, user2Traits map[string]interface{}) (string, error) {
	prompt := fmt.Sprintf(`
		Analyze the compatibility of two users based on their traits.
		User 1: %v
		User 2: %v
		
		Task: Write a short, engaging explanation (1-2 sentences) of why they are a good match. 
		Focus on complementarity (e.g., "Your calmness balances her energy").
		Language: Russian.
		Output: Just the explanation text.
	`, user1Traits, user2Traits)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		// Fallback to mock response if API is unavailable
		fmt.Printf("⚠️  [AI Wingman] Gemini API unavailable, using fallback explanation\n")
		return c.getMockExplanation(user1Traits, user2Traits), nil
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return c.getMockExplanation(user1Traits, user2Traits), nil
	}

	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			sb.WriteString(string(txt))
		}
	}

	return strings.TrimSpace(sb.String()), nil
}

func (c *GeminiClient) getMockExplanation(user1Traits, user2Traits map[string]interface{}) string {
	// Extract names if available
	name1 := "Вы"
	name2 := "ваш матч"
	if n, ok := user1Traits["Name"].(string); ok && n != "" {
		name1 = n
	}
	if n, ok := user2Traits["Name"].(string); ok && n != "" {
		name2 = n
	}

	mockExplanations := []string{
		fmt.Sprintf("Ваши общие интересы создают отличную основу для знакомства! %s и %s разделяют любовь к новым впечатлениям.", name1, name2),
		fmt.Sprintf("Вы дополняете друг друга: спокойствие %s гармонирует с энергичностью %s.", name1, name2),
		fmt.Sprintf("Ваши личности идеально сочетаются — %s и %s смогут найти общий язык!", name1, name2),
		"Ваша совместимость по интересам и характеру впечатляет! Это может стать началом чего-то особенного.",
	}

	// Return first mock explanation (or random in production)
	return mockExplanations[0]
}

func (c *GeminiClient) GenerateIcebreakers(ctx context.Context, user1Interests, user2Interests []string) ([]string, error) {
	prompt := fmt.Sprintf(`
		Generate 3 creative icebreaker messages for a dating app match.
		User 1 Interests: %v
		User 2 Interests: %v
		
		Task: Create 3 distinct opening lines that User 1 could send to User 2.
		Focus on shared interests or interesting contrasts.
		Language: Russian.
		Output: JSON array of strings. Example: ["Hi...", "Hello..."]
	`, user1Interests, user2Interests)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, fmt.Errorf("no content generated")
	}

	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			sb.WriteString(string(txt))
		}
	}

	responseText := strings.TrimSpace(sb.String())
	// Clean up markdown code blocks if present
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")

	var icebreakers []string
	if err := json.Unmarshal([]byte(responseText), &icebreakers); err != nil {
		// Fallback if JSON parsing fails - just return raw text split by newlines
		lines := strings.Split(responseText, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "[") && !strings.HasSuffix(line, "]") {
				icebreakers = append(icebreakers, line)
			}
		}
		if len(icebreakers) == 0 {
			// If manual parsing also failed or it was valid JSON but we failed to parse it initially
			return nil, fmt.Errorf("failed to parse icebreakers: %w", err)
		}
	}

	return icebreakers, nil
}
