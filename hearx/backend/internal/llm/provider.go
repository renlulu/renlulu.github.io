package llm

import "context"

type Provider interface {
	Chat(ctx context.Context, systemPrompt, userMessage string) (string, error)
}
