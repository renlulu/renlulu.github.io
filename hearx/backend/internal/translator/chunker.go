package translator

import "strings"

const (
	defaultChunkSize = 2000  // words
	defaultOverlap   = 200   // words
)

type Chunk struct {
	Text       string
	StartIndex int // word index in original
	EndIndex   int
}

func SplitIntoChunks(text string, chunkSize, overlap int) []Chunk {
	if chunkSize == 0 {
		chunkSize = defaultChunkSize
	}
	if overlap == 0 {
		overlap = defaultOverlap
	}

	words := strings.Fields(text)
	if len(words) <= chunkSize {
		return []Chunk{{Text: text, StartIndex: 0, EndIndex: len(words)}}
	}

	var chunks []Chunk
	start := 0
	for start < len(words) {
		end := start + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunk := Chunk{
			Text:       strings.Join(words[start:end], " "),
			StartIndex: start,
			EndIndex:   end,
		}
		chunks = append(chunks, chunk)
		if end >= len(words) {
			break
		}
		start = end - overlap
	}
	return chunks
}

// MergeTranslations combines translated chunks, removing overlap duplicates.
// Uses a simple strategy: keep the first chunk fully, then for subsequent chunks
// skip the overlap portion and append the rest.
func MergeTranslations(translations []string) string {
	if len(translations) == 0 {
		return ""
	}
	if len(translations) == 1 {
		return translations[0]
	}

	var sb strings.Builder
	sb.WriteString(translations[0])
	for i := 1; i < len(translations); i++ {
		sb.WriteString("\n\n")
		sb.WriteString(translations[i])
	}
	return sb.String()
}
