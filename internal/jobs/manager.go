package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/rdvid/huginn/internal/models"
	"github.com/rdvid/huginn/internal/pipeline"
)

// Manager is a lightweight in-memory async job store.
type Manager struct {
	mu      sync.RWMutex
	jobs    map[string]*models.Job
	runner  *pipeline.Runner
	workers chan struct{}
}

// NewManager creates a job manager with bounded worker slots.
func NewManager(runner *pipeline.Runner, maxWorkers int) *Manager {
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	return &Manager{
		jobs:    make(map[string]*models.Job),
		runner:  runner,
		workers: make(chan struct{}, maxWorkers),
	}
}

// Submit enqueues an async search job.
func (m *Manager) Submit(ctx context.Context, req models.SearchRequest) (*models.Job, error) {
	now := time.Now().UTC()
	job := &models.Job{
		ID:        uuid.New().String(),
		Status:    models.JobPending,
		Request:   req,
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	go m.execute(ctx, job.ID)
	return cloneJob(job), nil
}

func (m *Manager) execute(parent context.Context, id string) {
	m.workers <- struct{}{}
	defer func() { <-m.workers }()

	m.setStatus(id, models.JobRunning, "", nil)

	m.mu.RLock()
	job := m.jobs[id]
	req := job.Request
	m.mu.RUnlock()

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	results, err := m.runner.Run(ctx, req)
	if err != nil {
		m.setStatus(id, models.JobFailed, err.Error(), nil)
		return
	}
	m.setStatus(id, models.JobCompleted, "", results)
}

func (m *Manager) setStatus(id, status, errMsg string, results []models.Business) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return
	}
	job.Status = status
	job.Error = errMsg
	job.UpdatedAt = time.Now().UTC()
	if results != nil {
		job.Results = results
	}
}

// Get returns a job by id.
func (m *Manager) Get(id string) (*models.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	return cloneJob(job), nil
}

// List returns all jobs (newest first by CreatedAt is not guaranteed; map iteration).
func (m *Manager) List() []*models.Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*models.Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, cloneJob(j))
	}
	return out
}

// Results returns completed job results.
func (m *Manager) Results(id string) ([]models.Business, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	if job.Status != models.JobCompleted {
		return nil, fmt.Errorf("job status is %s", job.Status)
	}
	out := make([]models.Business, len(job.Results))
	copy(out, job.Results)
	return out, nil
}

func cloneJob(j *models.Job) *models.Job {
	c := *j
	if j.Results != nil {
		c.Results = make([]models.Business, len(j.Results))
		copy(c.Results, j.Results)
	}
	return &c
}
