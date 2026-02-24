package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicProvider struct {
	client *anthropic.Client
}

func NewAnthropic(apiKey string) *AnthropicProvider {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicProvider{client: &client}
}

func (a *AnthropicProvider) Chat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
		},
		Temperature: anthropic.Float(0.3),
	})
	if err != nil {
		return "", fmt.Errorf("anthropic chat: %w", err)
	}
	for _, block := range resp.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("anthropic returned no text content")
}
