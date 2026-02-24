package pipeline

import (
	"context"
	"log"
	"time"

	"github.com/renlulu/hearx/backend/internal/asr"
	"github.com/renlulu/hearx/backend/internal/fetcher"
	"github.com/renlulu/hearx/backend/internal/jobstore"
	"github.com/renlulu/hearx/backend/internal/model"
	"github.com/renlulu/hearx/backend/internal/renderer"
	"github.com/renlulu/hearx/backend/internal/translator"
)

type Pipeline struct {
	fetcher    *fetcher.Fetcher
	asr        asr.Provider
	translator *translator.Translator
	renderer   *renderer.Renderer
	store      *jobstore.Store
}

func New(
	f *fetcher.Fetcher,
	asrProvider asr.Provider,
	t *translator.Translator,
	r *renderer.Renderer,
	s *jobstore.Store,
) *Pipeline {
	return &Pipeline{
		fetcher:    f,
		asr:        asrProvider,
		translator: t,
		renderer:   r,
		store:      s,
	}
}

func (p *Pipeline) Run(ctx context.Context, job *model.Job) {
	updateStatus := func(status model.JobStatus) {
		job.Status = status
		job.UpdatedAt = time.Now()
		p.store.Update(job)
		log.Printf("job %s: %s", job.ID, status)
	}

	fail := func(err error) {
		job.Error = err.Error()
		updateStatus(model.StatusFailed)
	}

	// Step 1: Fetch audio
	updateStatus(model.StatusFetching)
	wavPath, cleanup, err := p.fetcher.Fetch(ctx, job.SourceURL)
	if err != nil {
		fail(err)
		return
	}
	defer cleanup()

	// Step 2: ASR transcription
	updateStatus(model.StatusTranscribing)
	segments, err := p.asr.Transcribe(ctx, wavPath)
	if err != nil {
		fail(err)
		return
	}

	// Step 3: Translation
	updateStatus(model.StatusTranslating)
	translation, err := p.translator.Translate(ctx, segments, job.SkillPacks)
	if err != nil {
		fail(err)
		return
	}

	// Step 4: Render outputs
	markdown := p.renderer.RenderMarkdown(segments, translation)
	srt := p.renderer.RenderSRT(segments)
	summary, err := p.renderer.GenerateSummary(ctx, translation)
	if err != nil {
		log.Printf("job %s: summary generation failed (non-fatal): %v", job.ID, err)
		summary = ""
	}

	job.Result = &model.Result{
		Transcript: segments,
		Markdown:   markdown,
		SRT:        srt,
		Summary:    summary,
	}
	updateStatus(model.StatusCompleted)
}
