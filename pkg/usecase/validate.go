package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	scenarioSvc "github.com/secmon-lab/tamamo/pkg/service/scenario"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

// Validate checks a scenario for correctness.
func (u *UseCase) Validate(ctx context.Context, scenarioPath string) error {
	s, err := scenarioSvc.Load(ctx, scenarioPath)
	if err != nil {
		return goerr.Wrap(err, "failed to load scenario for validation",
			goerr.V("path", scenarioPath),
			goerr.T(errutil.TagNotFound),
		)
	}

	if err := s.Validate(); err != nil {
		return goerr.Wrap(err, "scenario validation failed",
			goerr.T(errutil.TagValidation),
		)
	}

	u.logger.Info("scenario validation passed",
		"name", s.Meta.Name,
		"pages", len(s.Pages),
		"routes", len(s.Routes),
	)

	return nil
}
