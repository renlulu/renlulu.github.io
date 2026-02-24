package model

import "time"

type JobStatus string

const (
	StatusPending      JobStatus = "pending"
	StatusFetching     JobStatus = "fetching"
	StatusTranscribing JobStatus = "transcribing"
	StatusTranslating  JobStatus = "translating"
	StatusCompleted    JobStatus = "completed"
	StatusFailed       JobStatus = "failed"
)

type Job struct {
	ID         string    `json:"id"`
	SourceURL  string    `json:"source_url"`
	SkillPacks []string  `json:"skill_packs"`
	Status     JobStatus `json:"status"`
	Error      string    `json:"error,omitempty"`
	Result     *Result   `json:"result,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Result struct {
	Transcript []TranscriptSegment `json:"transcript"`
	Markdown   string              `json:"markdown"`
	SRT        string              `json:"srt"`
	Summary    string              `json:"summary"`
}

type TranscriptSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type SkillPack struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description" json:"description"`
	Glossary    map[string]string `yaml:"glossary" json:"glossary"`
	Prompt      string            `yaml:"prompt" json:"prompt"`
	Entities    []EntityRule      `yaml:"entities" json:"entities"`
}

type EntityRule struct {
	Pattern     string `yaml:"pattern" json:"pattern"`
	Category    string `yaml:"category" json:"category"`
	Translation string `yaml:"translation" json:"translation"`
}
