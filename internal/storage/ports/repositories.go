// Package ports defines storage repository interfaces (ports).
// Implementations live in adapters (e.g. internal/storage/postgres).
// Domain and application layers depend only on these interfaces.
package ports

import (
	"context"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/asset"
	"cyberos/internal/domain/capability"
	"cyberos/internal/domain/finding"
	"cyberos/internal/domain/principal"
	"cyberos/internal/domain/project"
	"cyberos/internal/domain/scope"
	"cyberos/internal/domain/task"
)

// ProjectRepository persists and retrieves projects.
type ProjectRepository interface {
	Create(ctx context.Context, p *project.Project) error
	GetByID(ctx context.Context, id contracts.ProjectID) (*project.Project, error)
	Update(ctx context.Context, p *project.Project) error
	List(ctx context.Context, limit, offset int) ([]*project.Project, error)
}

// AssetRepository persists and retrieves assets.
type AssetRepository interface {
	Create(ctx context.Context, a *asset.Asset) error
	GetByID(ctx context.Context, id contracts.AssetID) (*asset.Asset, error)
	ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*asset.Asset, error)
}

// ScopeRepository persists and retrieves scopes.
type ScopeRepository interface {
	Create(ctx context.Context, s *scope.Scope) error
	GetByID(ctx context.Context, id contracts.ScopeID) (*scope.Scope, error)
	ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*scope.Scope, error)
	ListAll(ctx context.Context, limit, offset int) ([]*scope.Scope, error)
	Update(ctx context.Context, s *scope.Scope) error
}

// TaskRepository persists and retrieves tasks.
type TaskRepository interface {
	Create(ctx context.Context, t *task.Task) error
	GetByID(ctx context.Context, id contracts.TaskID) (*task.Task, error)
	Update(ctx context.Context, t *task.Task) error
	ListByScope(ctx context.Context, scopeID contracts.ScopeID, limit, offset int) ([]*task.Task, error)
	ListByState(ctx context.Context, state contracts.TaskState, limit, offset int) ([]*task.Task, error)
	ListDueScheduled(ctx context.Context, now time.Time, limit int) ([]*task.Task, error)
	List(ctx context.Context, limit, offset int) ([]*task.Task, error)
}

// PrincipalRepository persists and retrieves principals.
type PrincipalRepository interface {
	Create(ctx context.Context, p *principal.Principal) error
	GetByID(ctx context.Context, id contracts.PrincipalID) (*principal.Principal, error)
	List(ctx context.Context, limit, offset int) ([]*principal.Principal, error)
}

// CapabilityRepository persists and retrieves capabilities.
type CapabilityRepository interface {
	Create(ctx context.Context, c *capability.Capability) error
	GetByID(ctx context.Context, id contracts.CapabilityID) (*capability.Capability, error)
	List(ctx context.Context, limit, offset int) ([]*capability.Capability, error)
}

// AssessmentRecord is a persistence-friendly assessment snapshot (not authorization).
type AssessmentRecord struct {
	ID          string
	ProjectID   string
	ScopeID     string
	PrincipalID string
	Title       string
	Target      string
	State       string
	StepsJSON   []byte
	ResultsJSON []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AssessmentRepository persists assessment orchestration state.
type AssessmentRepository interface {
	Create(ctx context.Context, r AssessmentRecord) error
	Update(ctx context.Context, r AssessmentRecord) error
	GetByID(ctx context.Context, id string) (AssessmentRecord, error)
	List(ctx context.Context, limit, offset int) ([]AssessmentRecord, error)
}

// Repositories aggregates all repository ports for convenient wiring.
type Repositories struct {
	Projects     ProjectRepository
	Assets       AssetRepository
	Scopes       ScopeRepository
	Tasks        TaskRepository
	Principals   PrincipalRepository
	Capabilities CapabilityRepository
	Assessments  AssessmentRepository
	Findings     FindingRepository
	RiskEvals    RiskEvaluationRepository
}

// FindingRepository persists security findings (Phase 7A). Persistence only — not authorization.
type FindingRepository interface {
	Upsert(ctx context.Context, f *finding.Finding) error
	GetByID(ctx context.Context, id contracts.FindingID) (*finding.Finding, error)
	GetByFingerprint(ctx context.Context, projectID contracts.ProjectID, fingerprint string) (*finding.Finding, error)
	ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*finding.Finding, error)
	ListByAssessment(ctx context.Context, assessmentID string, limit, offset int) ([]*finding.Finding, error)
}

// RiskEvaluationRecord is a persistence DTO for riskprio.Evaluation (not Control Plane RiskResult).
type RiskEvaluationRecord struct {
	ID               string
	ProjectID        contracts.ProjectID
	AssessmentID     string
	FindingID        contracts.FindingID
	AssetID          contracts.AssetID
	Severity         contracts.FindingSeverity
	Confidence       contracts.FindingConfidence
	AssetCriticality contracts.AssetCriticality
	Exposure         contracts.ExposureContext
	Score            int
	Priority         contracts.RiskPriority
	Factors          map[string]string
	ModelVersion     string
	CalculatedAt     time.Time
}

// RiskEvaluationRepository persists risk prioritization evaluations (Phase 7A).
type RiskEvaluationRepository interface {
	Upsert(ctx context.Context, r RiskEvaluationRecord) error
	GetByID(ctx context.Context, id string) (RiskEvaluationRecord, error)
	GetByFinding(ctx context.Context, findingID contracts.FindingID) (RiskEvaluationRecord, error)
	ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]RiskEvaluationRecord, error)
	ListByAssessment(ctx context.Context, assessmentID string, limit, offset int) ([]RiskEvaluationRecord, error)
}
