package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAIProvider struct {
	client *openai.Client
}

func NewOpenAI(apiKey string) *OpenAIProvider {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIProvider{client: &client}
}

func (o *OpenAIProvider) Chat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userMessage),
		},
		Temperature: openai.Float(0.3),
	})
	if err != nil {
		return "", fmt.Errorf("openai chat: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}
