package interfaces

import (
	"context"

	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
)

// Generator abstracts LLM-based scenario generation.
type Generator interface {
	Generate(ctx context.Context) (*scenario.Scenario, error)
}
