package asr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/renlulu/hearx/backend/internal/model"
)

type WhisperProvider struct {
	client *openai.Client
}

func NewWhisper(apiKey string) *WhisperProvider {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &WhisperProvider{client: &client}
}

// verboseJSON is the raw verbose_json response from Whisper API
type verboseJSON struct {
	Text     string `json:"text"`
	Segments []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	} `json:"segments"`
}

func (w *WhisperProvider) Transcribe(ctx context.Context, wavPath string) ([]model.TranscriptSegment, error) {
	f, err := os.Open(wavPath)
	if err != nil {
		return nil, fmt.Errorf("opening wav file: %w", err)
	}
	defer f.Close()

	resp, err := w.client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		File:                   f,
		Model:                  openai.AudioModelWhisper1,
		ResponseFormat:         openai.AudioResponseFormatVerboseJSON,
		TimestampGranularities: []string{"segment"},
	})
	if err != nil {
		return nil, fmt.Errorf("whisper transcription: %w", err)
	}

	// Parse verbose_json from raw response to get segments
	var verbose verboseJSON
	if err := json.Unmarshal([]byte(resp.RawJSON()), &verbose); err != nil {
		// Fallback: return single segment with full text
		return []model.TranscriptSegment{{Text: resp.Text}}, nil
	}

	if len(verbose.Segments) == 0 {
		return []model.TranscriptSegment{{Text: resp.Text}}, nil
	}

	segments := make([]model.TranscriptSegment, 0, len(verbose.Segments))
	for _, seg := range verbose.Segments {
		segments = append(segments, model.TranscriptSegment{
			Start: seg.Start,
			End:   seg.End,
			Text:  seg.Text,
		})
	}
	return segments, nil
}
