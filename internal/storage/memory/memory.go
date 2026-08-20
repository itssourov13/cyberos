// Package memory provides in-process repository implementations for tests
// and non-PG local runs. Not an authorization source.
package memory

import (
	"context"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/project"
	"cyberos/internal/domain/scope"
	"cyberos/internal/domain/task"
	"cyberos/internal/storage/ports"
)

// Repos is an in-memory ports.Repositories aggregate.
type Repos struct {
	Projects    *ProjectStore
	Scopes      *ScopeStore
	Tasks       *TaskStore
	Assessments *AssessmentStore
}

// NewRepos constructs empty memory repositories.
func NewRepos() *Repos {
	return &Repos{
		Projects:    NewProjectStore(),
		Scopes:      NewScopeStore(),
		Tasks:       NewTaskStore(),
		Assessments: NewAssessmentStore(),
	}
}

// AsPorts returns a ports.Repositories view.
func (r *Repos) AsPorts() *ports.Repositories {
	return &ports.Repositories{
		Projects:    r.Projects,
		Scopes:      r.Scopes,
		Tasks:       r.Tasks,
		Assessments: r.Assessments,
	}
}

type ProjectStore struct {
	mu    sync.RWMutex
	items map[contracts.ProjectID]*project.Project
}

func NewProjectStore() *ProjectStore {
	return &ProjectStore{items: map[contracts.ProjectID]*project.Project{}}
}

func (s *ProjectStore) Create(_ context.Context, p *project.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[p.ID]; ok {
		return contracts.ErrConflict
	}
	cp := *p
	s.items[p.ID] = &cp
	return nil
}
func (s *ProjectStore) GetByID(_ context.Context, id contracts.ProjectID) (*project.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (s *ProjectStore) Update(_ context.Context, p *project.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[p.ID]; !ok {
		return contracts.ErrNotFound
	}
	cp := *p
	s.items[p.ID] = &cp
	return nil
}
func (s *ProjectStore) List(_ context.Context, limit, offset int) ([]*project.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*project.Project
	i := 0
	for _, p := range s.items {
		if i < offset {
			i++
			continue
		}
		cp := *p
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
		i++
	}
	return out, nil
}

type ScopeStore struct {
	mu    sync.RWMutex
	items map[contracts.ScopeID]*scope.Scope
}

func NewScopeStore() *ScopeStore {
	return &ScopeStore{items: map[contracts.ScopeID]*scope.Scope{}}
}

func (s *ScopeStore) Create(_ context.Context, sc *scope.Scope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[sc.ID]; ok {
		return contracts.ErrConflict
	}
	cp := *sc
	s.items[sc.ID] = &cp
	return nil
}
func (s *ScopeStore) GetByID(_ context.Context, id contracts.ScopeID) (*scope.Scope, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *sc
	return &cp, nil
}
func (s *ScopeStore) Update(_ context.Context, sc *scope.Scope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[sc.ID]; !ok {
		return contracts.ErrNotFound
	}
	cp := *sc
	s.items[sc.ID] = &cp
	return nil
}
func (s *ScopeStore) ListByProject(_ context.Context, projectID contracts.ProjectID, limit, offset int) ([]*scope.Scope, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*scope.Scope
	for _, sc := range s.items {
		if sc.ProjectID != projectID {
			continue
		}
		cp := *sc
		out = append(out, &cp)
	}
	return out, nil
}
func (s *ScopeStore) ListAll(_ context.Context, limit, offset int) ([]*scope.Scope, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*scope.Scope
	for _, sc := range s.items {
		cp := *sc
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

type TaskStore struct {
	mu    sync.RWMutex
	items map[contracts.TaskID]*task.Task
}

func NewTaskStore() *TaskStore {
	return &TaskStore{items: map[contracts.TaskID]*task.Task{}}
}

func (s *TaskStore) Create(_ context.Context, t *task.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[t.ID]; ok {
		return contracts.ErrConflict
	}
	cp := *t
	s.items[t.ID] = &cp
	return nil
}
func (s *TaskStore) GetByID(_ context.Context, id contracts.TaskID) (*task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *t
	return &cp, nil
}
func (s *TaskStore) Update(_ context.Context, t *task.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[t.ID]; !ok {
		return contracts.ErrNotFound
	}
	cp := *t
	s.items[t.ID] = &cp
	return nil
}
func (s *TaskStore) ListByScope(_ context.Context, scopeID contracts.ScopeID, limit, offset int) ([]*task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*task.Task
	for _, t := range s.items {
		if t.ScopeID != scopeID {
			continue
		}
		cp := *t
		out = append(out, &cp)
	}
	return out, nil
}
func (s *TaskStore) ListByState(_ context.Context, state contracts.TaskState, limit, offset int) ([]*task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*task.Task
	for _, t := range s.items {
		if t.State != state {
			continue
		}
		cp := *t
		out = append(out, &cp)
	}
	return out, nil
}
func (s *TaskStore) ListDueScheduled(_ context.Context, now time.Time, limit int) ([]*task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*task.Task
	for _, t := range s.items {
		if t.State == contracts.TaskStateScheduled && t.IsDue(now) {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (s *TaskStore) List(_ context.Context, limit, offset int) ([]*task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*task.Task
	for _, t := range s.items {
		cp := *t
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

type AssessmentStore struct {
	mu    sync.RWMutex
	items map[string]ports.AssessmentRecord
}

func NewAssessmentStore() *AssessmentStore {
	return &AssessmentStore{items: map[string]ports.AssessmentRecord{}}
}

func (s *AssessmentStore) Create(_ context.Context, r ports.AssessmentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[r.ID]; ok {
		return contracts.ErrConflict
	}
	s.items[r.ID] = r
	return nil
}
func (s *AssessmentStore) Update(_ context.Context, r ports.AssessmentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[r.ID]; !ok {
		return contracts.ErrNotFound
	}
	s.items[r.ID] = r
	return nil
}
func (s *AssessmentStore) GetByID(_ context.Context, id string) (ports.AssessmentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.items[id]
	if !ok {
		return ports.AssessmentRecord{}, contracts.ErrNotFound
	}
	return r, nil
}
func (s *AssessmentStore) List(_ context.Context, limit, offset int) ([]ports.AssessmentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []ports.AssessmentRecord
	for _, r := range s.items {
		out = append(out, r)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
