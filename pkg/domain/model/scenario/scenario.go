package scenario

import (
	"errors"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

// Scenario is the top-level container for a honeypot scenario.
type Scenario struct {
	Meta   Meta    `json:"meta"`
	Pages  []Page  `json:"pages"`
	Routes []Route `json:"routes"`
}

// Meta holds scenario metadata including server signature and headers.
type Meta struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	ServerSignature string            `json:"server_signature"`
	Headers         map[string]string `json:"headers"`
	Theme           string            `json:"theme"`
}

// Validate checks the scenario for required fields and consistency.
// All validation errors are collected and returned together.
func (s *Scenario) Validate() error {
	var errs []error

	if s.Meta.Name == "" {
		errs = append(errs, goerr.New("scenario missing required field",
			goerr.V("field", "meta.name"),
			goerr.T(errutil.TagValidation),
		))
	}
	if s.Meta.ServerSignature == "" {
		errs = append(errs, goerr.New("scenario missing required field",
			goerr.V("field", "meta.server_signature"),
			goerr.T(errutil.TagValidation),
		))
	}
	if len(s.Pages) == 0 {
		errs = append(errs, goerr.New("scenario must have at least one page",
			goerr.T(errutil.TagValidation),
		))
	}
	if len(s.Routes) == 0 {
		errs = append(errs, goerr.New("scenario must have at least one route",
			goerr.T(errutil.TagValidation),
		))
	}

	for i, p := range s.Pages {
		if err := p.Validate(); err != nil {
			errs = append(errs, goerr.Wrap(err, "invalid page",
				goerr.V("index", i),
				goerr.T(errutil.TagValidation),
			))
		}
	}
	for i, r := range s.Routes {
		if err := r.Validate(); err != nil {
			errs = append(errs, goerr.Wrap(err, "invalid route",
				goerr.V("index", i),
				goerr.T(errutil.TagValidation),
			))
		}
	}

	return errors.Join(errs...)
}
