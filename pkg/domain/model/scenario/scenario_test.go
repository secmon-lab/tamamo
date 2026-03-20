package scenario_test

import (
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

func validScenario() *scenario.Scenario {
	return &scenario.Scenario{
		Meta: scenario.Meta{
			Name:            "Test Portal",
			Description:     "Test scenario",
			ServerSignature: "nginx/1.24.0",
			Headers:         map[string]string{"X-Powered-By": "Express"},
			Theme:           "test-theme",
		},
		Pages: []scenario.Page{
			{Path: "/login", HTMLFile: "pages/login.html", ContentType: "text/html"},
		},
		Routes: []scenario.Route{
			{Path: "/login", Method: "GET", StatusCode: 200, Headers: map[string]string{"Content-Type": "text/html"}, Body: "<html></html>"},
		},
	}
}

func TestScenarioValidate(t *testing.T) {
	t.Run("valid scenario passes", func(t *testing.T) {
		s := validScenario()
		gt.NoError(t, s.Validate())
	})

	t.Run("missing name fails", func(t *testing.T) {
		s := validScenario()
		s.Meta.Name = ""
		err := s.Validate()
		gt.Error(t, err)
		gt.True(t, goerr.HasTag(err, errutil.TagValidation))
	})

	t.Run("missing server signature fails", func(t *testing.T) {
		s := validScenario()
		s.Meta.ServerSignature = ""
		err := s.Validate()
		gt.Error(t, err)
		gt.True(t, goerr.HasTag(err, errutil.TagValidation))
	})

	t.Run("no pages fails", func(t *testing.T) {
		s := validScenario()
		s.Pages = nil
		err := s.Validate()
		gt.Error(t, err)
		gt.True(t, goerr.HasTag(err, errutil.TagValidation))
	})

	t.Run("no routes fails", func(t *testing.T) {
		s := validScenario()
		s.Routes = nil
		err := s.Validate()
		gt.Error(t, err)
		gt.True(t, goerr.HasTag(err, errutil.TagValidation))
	})

	t.Run("invalid page fails", func(t *testing.T) {
		s := validScenario()
		s.Pages = []scenario.Page{{Path: "", HTMLFile: "test.html"}}
		err := s.Validate()
		gt.Error(t, err)
		gt.True(t, goerr.HasTag(err, errutil.TagValidation))
	})

	t.Run("invalid route fails", func(t *testing.T) {
		s := validScenario()
		s.Routes = []scenario.Route{{Path: "/test", Method: "", StatusCode: 200}}
		err := s.Validate()
		gt.Error(t, err)
		gt.True(t, goerr.HasTag(err, errutil.TagValidation))
	})

	t.Run("multiple errors are collected", func(t *testing.T) {
		s := &scenario.Scenario{}
		err := s.Validate()
		gt.Error(t, err)

		// Should contain errors for: name, server_signature, pages, routes
		unwrapped := err.(interface{ Unwrap() []error }).Unwrap()
		gt.N(t, len(unwrapped)).Equal(4)
	})
}

func TestPageValidate(t *testing.T) {
	t.Run("missing path fails", func(t *testing.T) {
		p := &scenario.Page{HTMLFile: "test.html"}
		err := p.Validate()
		gt.Error(t, err)
	})

	t.Run("missing html_file fails", func(t *testing.T) {
		p := &scenario.Page{Path: "/test"}
		err := p.Validate()
		gt.Error(t, err)
	})
}

func TestRouteValidate(t *testing.T) {
	t.Run("missing path fails", func(t *testing.T) {
		r := &scenario.Route{Method: "GET", StatusCode: 200}
		err := r.Validate()
		gt.Error(t, err)
	})

	t.Run("missing method fails", func(t *testing.T) {
		r := &scenario.Route{Path: "/test", StatusCode: 200}
		err := r.Validate()
		gt.Error(t, err)
	})

	t.Run("zero status code fails", func(t *testing.T) {
		r := &scenario.Route{Path: "/test", Method: "GET"}
		err := r.Validate()
		gt.Error(t, err)
	})
}
