package jobstore

import (
	"fmt"
	"sync"

	"github.com/renlulu/hearx/backend/internal/model"
)

type Store struct {
	jobs sync.Map
}

func New() *Store {
	return &Store{}
}

func (s *Store) Create(job *model.Job) {
	s.jobs.Store(job.ID, job)
}

func (s *Store) Get(id string) (*model.Job, error) {
	v, ok := s.jobs.Load(id)
	if !ok {
		return nil, fmt.Errorf("job %q not found", id)
	}
	return v.(*model.Job), nil
}

func (s *Store) Update(job *model.Job) {
	s.jobs.Store(job.ID, job)
}
