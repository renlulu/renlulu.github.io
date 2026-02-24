package translator

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/renlulu/hearx/backend/internal/llm"
	"github.com/renlulu/hearx/backend/internal/model"
	"github.com/renlulu/hearx/backend/internal/skillpack"
)

type Translator struct {
	llm    llm.Provider
	loader *skillpack.Loader
}

func New(llmProvider llm.Provider, loader *skillpack.Loader) *Translator {
	return &Translator{llm: llmProvider, loader: loader}
}

func (t *Translator) Translate(ctx context.Context, segments []model.TranscriptSegment, skillPacks []string) (string, error) {
	// Merge skill packs
	merged, err := t.loader.Merge(skillPacks)
	if err != nil {
		return "", fmt.Errorf("merging skill packs: %w", err)
	}

	systemPrompt := BuildSystemPrompt(merged)

	// Join all segments into full text
	var fullText strings.Builder
	for _, seg := range segments {
		fullText.WriteString(seg.Text)
		fullText.WriteString(" ")
	}

	// Split into chunks
	chunks := SplitIntoChunks(fullText.String(), defaultChunkSize, defaultOverlap)

	// Translate chunks concurrently
	translations := make([]string, len(chunks))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5)

	for i, chunk := range chunks {
		g.Go(func() error {
			result, err := t.llm.Chat(gctx, systemPrompt, chunk.Text)
			if err != nil {
				return fmt.Errorf("translating chunk %d: %w", i, err)
			}
			translations[i] = result
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return "", err
	}

	return MergeTranslations(translations), nil
}
