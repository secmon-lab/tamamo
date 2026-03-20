package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
	scenarioSvc "github.com/secmon-lab/tamamo/pkg/service/scenario"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

// LoadScenario loads and validates a scenario from path.
func (u *UseCase) LoadScenario(ctx context.Context, scenarioPath string) (*scenario.Scenario, error) {
	s, err := scenarioSvc.Load(ctx, scenarioPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to load scenario",
			goerr.V("path", scenarioPath),
			goerr.T(errutil.TagNotFound),
		)
	}

	if err := s.Validate(); err != nil {
		return nil, goerr.Wrap(err, "scenario validation failed",
			goerr.T(errutil.TagValidation),
		)
	}

	return s, nil
}
