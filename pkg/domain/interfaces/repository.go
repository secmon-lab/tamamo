package interfaces

import (
	"context"

	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
)

// Repository abstracts scenario read/write operations.
type Repository interface {
	Load(ctx context.Context, path string) (*scenario.Scenario, error)
	Save(ctx context.Context, path string, s *scenario.Scenario) error
}
