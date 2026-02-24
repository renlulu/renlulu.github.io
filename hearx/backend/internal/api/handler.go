package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/renlulu/hearx/backend/internal/jobstore"
	"github.com/renlulu/hearx/backend/internal/model"
	"github.com/renlulu/hearx/backend/internal/pipeline"
	"github.com/renlulu/hearx/backend/internal/skillpack"
)

type Handler struct {
	store    *jobstore.Store
	pipeline *pipeline.Pipeline
	loader   *skillpack.Loader
}

func NewHandler(store *jobstore.Store, pipe *pipeline.Pipeline, loader *skillpack.Loader) *Handler {
	return &Handler{store: store, pipeline: pipe, loader: loader}
}

type CreateJobRequest struct {
	SourceURL  string   `json:"source_url"`
	SkillPacks []string `json:"skill_packs"`
}

type CreateJobResponse struct {
	ID string `json:"id"`
}

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.SourceURL == "" {
		http.Error(w, `{"error":"source_url is required"}`, http.StatusBadRequest)
		return
	}

	job := &model.Job{
		ID:         uuid.New().String(),
		SourceURL:  req.SourceURL,
		SkillPacks: req.SkillPacks,
		Status:     model.StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	h.store.Create(job)
	go h.pipeline.Run(r.Context(), job)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(CreateJobResponse{ID: job.ID})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := h.store.Get(id)
	if err != nil {
		http.Error(w, `{"error":"job not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (h *Handler) ListSkills(w http.ResponseWriter, r *http.Request) {
	packs := h.loader.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(packs)
}
