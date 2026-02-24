package asr

import (
	"context"

	"github.com/renlulu/hearx/backend/internal/model"
)

type Provider interface {
	Transcribe(ctx context.Context, wavPath string) ([]model.TranscriptSegment, error)
}
