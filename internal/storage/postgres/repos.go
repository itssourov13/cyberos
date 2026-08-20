package postgres

import "cyberos/internal/storage/ports"

// NewRepositories constructs all PostgreSQL repository implementations.
func NewRepositories(db *DB) *ports.Repositories {
	return &ports.Repositories{
		Projects:     NewProjectRepo(db),
		Assets:       NewAssetRepo(db),
		Scopes:       NewScopeRepo(db),
		Tasks:        NewTaskRepo(db),
		Principals:   NewPrincipalRepo(db),
		Capabilities: NewCapabilityRepo(db),
		Assessments:  NewAssessmentRepo(db),
	}
}
