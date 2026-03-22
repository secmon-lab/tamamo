package scenario

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

// Route defines an API endpoint served by the honeypot.
type Route struct {
	Path       string            `json:"path"`
	Method     string            `json:"method"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	BodyFile   string            `json:"body_file,omitempty"`
	Auth       *AuthStrategy     `json:"auth,omitempty"`

	// Hang makes the route wait indefinitely without sending a response.
	// The connection stays open until the client gives up (timeout).
	Hang bool `json:"hang,omitempty"`
}

// Validate checks the route for required fields.
func (r *Route) Validate() error {
	if r.Path == "" {
		return goerr.New("route missing required field",
			goerr.V("field", "path"),
			goerr.T(errutil.TagValidation),
		)
	}
	if r.Method == "" {
		return goerr.New("route missing required field",
			goerr.V("field", "method"),
			goerr.T(errutil.TagValidation),
		)
	}
	if r.StatusCode == 0 && !r.Hang {
		return goerr.New("route missing required field",
			goerr.V("field", "status_code"),
			goerr.T(errutil.TagValidation),
		)
	}
	return nil
}
