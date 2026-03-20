package scenario_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	scenarioSvc "github.com/secmon-lab/tamamo/pkg/service/scenario"
)

func setupTestScenario(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Write scenario.json
	scenarioJSON := `{
		"name": "Test Admin Portal",
		"description": "Test scenario",
		"server_signature": "Apache/2.4.57",
		"headers": {"X-Powered-By": "PHP/8.2"},
		"theme": "test"
	}`
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "scenario.json"), []byte(scenarioJSON), 0o600))

	// Write routes.json
	routesJSON := `{
		"routes": [
			{
				"path": "/",
				"method": "GET",
				"status_code": 302,
				"headers": {"Location": "/login"},
				"body": ""
			},
			{
				"path": "/login",
				"method": "GET",
				"status_code": 200,
				"headers": {"Content-Type": "text/html"},
				"body_file": "pages/login.html"
			},
			{
				"path": "/api/auth/login",
				"method": "POST",
				"status_code": 200,
				"headers": {"Content-Type": "application/json"},
				"body": "{\"success\": true}"
			}
		]
	}`
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "routes.json"), []byte(routesJSON), 0o600))

	// Write pages
	gt.NoError(t, os.MkdirAll(filepath.Join(dir, "pages"), 0o750))
	loginHTML := `<html><body><h1>Login</h1></body></html>`
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "pages", "login.html"), []byte(loginHTML), 0o600))

	return dir
}

func TestLoadFromDir(t *testing.T) {
	dir := setupTestScenario(t)

	s, err := scenarioSvc.Load(context.Background(), dir)
	gt.NoError(t, err)
	gt.Equal(t, s.Meta.Name, "Test Admin Portal")
	gt.Equal(t, s.Meta.ServerSignature, "Apache/2.4.57")
	gt.Equal(t, s.Meta.Headers["X-Powered-By"], "PHP/8.2")

	// Routes should be loaded
	gt.Equal(t, len(s.Routes), 3)
	gt.Equal(t, s.Routes[0].Path, "/")
	gt.Equal(t, s.Routes[0].StatusCode, 302)
	gt.Equal(t, s.Routes[1].Path, "/login")

	// body_file should be resolved
	gt.S(t, s.Routes[1].Body).Contains("<h1>Login</h1>")

	// Pages should be collected from body_file references
	gt.Equal(t, len(s.Pages), 1)
	gt.Equal(t, s.Pages[0].Path, "/login")
	gt.Equal(t, s.Pages[0].HTMLFile, "pages/login.html")
}

func TestLoadNonExistent(t *testing.T) {
	_, err := scenarioSvc.Load(context.Background(), "/nonexistent/path")
	gt.Error(t, err)
}

func TestLoadMissingScenarioJSON(t *testing.T) {
	dir := t.TempDir()
	_, err := scenarioSvc.Load(context.Background(), dir)
	gt.Error(t, err)
}

func TestSaveAndLoad(t *testing.T) {
	dir := setupTestScenario(t)

	// Load existing scenario
	s, err := scenarioSvc.Load(context.Background(), dir)
	gt.NoError(t, err)

	// Save to new directory
	newDir := t.TempDir()
	gt.NoError(t, scenarioSvc.Save(context.Background(), newDir, s))

	// Load from new directory
	reloaded, err := scenarioSvc.Load(context.Background(), newDir)
	gt.NoError(t, err)
	gt.Equal(t, reloaded.Meta.Name, s.Meta.Name)
	gt.Equal(t, reloaded.Meta.ServerSignature, s.Meta.ServerSignature)
}
