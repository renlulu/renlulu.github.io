package backend_test

import (
	"strings"
	"testing"

	"github.com/renlulu/hearx/backend/internal/translator"
)

func TestSplitIntoChunks_SmallText(t *testing.T) {
	text := "hello world this is a test"
	chunks := translator.SplitIntoChunks(text, 100, 20)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != text {
		t.Errorf("expected %q, got %q", text, chunks[0].Text)
	}
}

func TestSplitIntoChunks_LargeText(t *testing.T) {
	// Generate 500 words
	words := make([]string, 500)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")

	chunks := translator.SplitIntoChunks(text, 200, 50)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify chunks cover all words
	lastEnd := chunks[len(chunks)-1].EndIndex
	if lastEnd != 500 {
		t.Errorf("last chunk should end at 500, got %d", lastEnd)
	}

	// Verify overlap
	if chunks[1].StartIndex >= chunks[0].EndIndex {
		t.Error("chunks should overlap")
	}
}

func TestSplitIntoChunks_EmptyText(t *testing.T) {
	chunks := translator.SplitIntoChunks("", 100, 20)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for empty text, got %d", len(chunks))
	}
}

func TestMergeTranslations(t *testing.T) {
	translations := []string{"第一段翻译", "第二段翻译", "第三段翻译"}
	result := translator.MergeTranslations(translations)

	if !strings.Contains(result, "第一段翻译") {
		t.Error("result should contain first translation")
	}
	if !strings.Contains(result, "第三段翻译") {
		t.Error("result should contain last translation")
	}
}

func TestMergeTranslations_Single(t *testing.T) {
	result := translator.MergeTranslations([]string{"唯一的翻译"})
	if result != "唯一的翻译" {
		t.Errorf("expected exact single translation, got %q", result)
	}
}

func TestMergeTranslations_Empty(t *testing.T) {
	result := translator.MergeTranslations(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
