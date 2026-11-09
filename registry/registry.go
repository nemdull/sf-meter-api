// Package registry wires all layers together.
// This is the only place that knows about every concrete type.
// All other packages depend only on interfaces.
package registry

import (
	"os"

	"github.com/nemdull/sf-meter-api/infrastructure"
	"github.com/nemdull/sf-meter-api/presentation"
	"github.com/nemdull/sf-meter-api/usecase"
)

// Registry holds application-wide configuration needed to wire dependencies.
type Registry struct {
	dsn string
}

// New creates a Registry. It reads DATABASE_URL from the environment;
// NewTechHandler falls back to the in-memory repository when it's unset.
func New() (*Registry, error) {
	return &Registry{dsn: os.Getenv("DATABASE_URL")}, nil
}

// NewTechHandler assembles the full dependency chain for /techs routes.
//
//	infrastructure.PostgresTechRepo (or MemoryTechRepo)
//	                 ← usecase.TechService
//	                 ← presentation.TechHandler
func (r *Registry) NewTechHandler() (*presentation.TechHandler, error) {
	var repo usecase.TechRepository
	if r.dsn != "" {
		pgRepo, err := infrastructure.NewPostgresTechRepo(r.dsn)
		if err != nil {
			return nil, err
		}
		repo = pgRepo
	} else {
		repo = infrastructure.NewMemoryTechRepo()
	}
	svc := usecase.NewTechService(repo)
	return presentation.NewTechHandler(svc), nil
}
