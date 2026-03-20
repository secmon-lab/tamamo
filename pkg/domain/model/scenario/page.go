package scenario

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

// Page represents an HTML page served by the honeypot.
type Page struct {
	Path        string `json:"path"`
	HTMLFile    string `json:"html_file"`
	ContentType string `json:"content_type"`
}

// Validate checks the page for required fields.
func (p *Page) Validate() error {
	if p.Path == "" {
		return goerr.New("page missing required field",
			goerr.V("field", "path"),
			goerr.T(errutil.TagValidation),
		)
	}
	if p.HTMLFile == "" {
		return goerr.New("page missing required field",
			goerr.V("field", "html_file"),
			goerr.T(errutil.TagValidation),
		)
	}
	return nil
}
