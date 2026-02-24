package asr

import (
	"context"
	"fmt"

	restapi "github.com/deepgram/deepgram-go-sdk/pkg/api/listen/v1/rest"
	interfaces "github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces"
	restclient "github.com/deepgram/deepgram-go-sdk/pkg/client/listen/v1/rest"
	"github.com/renlulu/hearx/backend/internal/model"
)

type DeepgramProvider struct {
	apiKey string
}

func NewDeepgram(apiKey string) *DeepgramProvider {
	return &DeepgramProvider{apiKey: apiKey}
}

func (d *DeepgramProvider) Transcribe(ctx context.Context, wavPath string) ([]model.TranscriptSegment, error) {
	rc := restclient.New(d.apiKey, &interfaces.ClientOptions{})
	if rc == nil {
		return nil, fmt.Errorf("failed to create deepgram client")
	}
	c := restapi.New(rc)

	options := &interfaces.PreRecordedTranscriptionOptions{
		Model:       "nova-2",
		Language:    "en",
		Punctuate:   true,
		Utterances:  true,
		SmartFormat: true,
	}

	resp, err := c.FromFile(ctx, wavPath, options)
	if err != nil {
		return nil, fmt.Errorf("deepgram transcription: %w", err)
	}

	// Prefer utterances for better segment grouping
	if resp.Results.Utterances != nil && len(resp.Results.Utterances) > 0 {
		segments := make([]model.TranscriptSegment, 0, len(resp.Results.Utterances))
		for _, utt := range resp.Results.Utterances {
			segments = append(segments, model.TranscriptSegment{
				Start: utt.Start,
				End:   utt.End,
				Text:  utt.Transcript,
			})
		}
		return segments, nil
	}

	// Fallback to word-level from alternatives
	var segments []model.TranscriptSegment
	for _, ch := range resp.Results.Channels {
		for _, alt := range ch.Alternatives {
			for _, word := range alt.Words {
				segments = append(segments, model.TranscriptSegment{
					Start: word.Start,
					End:   word.End,
					Text:  word.Word,
				})
			}
		}
	}
	return segments, nil
}
