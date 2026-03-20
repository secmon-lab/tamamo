package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

// writeFileToolSet implements gollem.ToolSet for writing scenario files.
type writeFileToolSet struct {
	outputDir string
}

func newWriteFileToolSet(outputDir string) *writeFileToolSet {
	return &writeFileToolSet{outputDir: outputDir}
}

// Specs returns the tool specifications for the write_file tool.
func (t *writeFileToolSet) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "write_file",
			Description: "Write content to a file in the scenario directory",
			Parameters: map[string]*gollem.Parameter{
				"path": {
					Type:        gollem.TypeString,
					Description: "Relative file path within the scenario directory (e.g., pages/login.html, scenario.json, routes.json)",
					Required:    true,
				},
				"content": {
					Type:        gollem.TypeString,
					Description: "File content to write",
					Required:    true,
				},
			},
		},
	}, nil
}

// Run executes the write_file tool.
func (t *writeFileToolSet) Run(_ context.Context, name string, args map[string]any) (map[string]any, error) {
	if name != "write_file" {
		return nil, goerr.New("unknown tool",
			goerr.V("name", name),
			goerr.T(errutil.TagInternal),
		)
	}

	pathArg, ok := args["path"].(string)
	if !ok || pathArg == "" {
		return nil, goerr.New("path parameter is required",
			goerr.T(errutil.TagValidation),
		)
	}

	content, ok := args["content"].(string)
	if !ok {
		return nil, goerr.New("content parameter is required",
			goerr.T(errutil.TagValidation),
		)
	}

	// Sanitize path to prevent directory traversal
	cleanPath := filepath.Clean(pathArg)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		return nil, goerr.New("invalid file path: must be relative and within scenario directory",
			goerr.V("path", pathArg),
			goerr.T(errutil.TagValidation),
		)
	}

	target := filepath.Join(t.outputDir, cleanPath)
	// Verify the target is within the output directory
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to resolve absolute path",
			goerr.T(errutil.TagInternal),
		)
	}
	absOutput, err := filepath.Abs(t.outputDir)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to resolve output directory",
			goerr.T(errutil.TagInternal),
		)
	}
	if !strings.HasPrefix(absTarget, absOutput+string(os.PathSeparator)) && absTarget != absOutput {
		return nil, goerr.New("file path escapes scenario directory",
			goerr.V("path", pathArg),
			goerr.T(errutil.TagValidation),
		)
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return nil, goerr.Wrap(err, "failed to create directory",
			goerr.V("path", filepath.Dir(target)),
			goerr.T(errutil.TagInternal),
		)
	}

	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		return nil, goerr.Wrap(err, "failed to write file",
			goerr.V("path", target),
			goerr.T(errutil.TagInternal),
		)
	}

	return map[string]any{
		"status": "ok",
		"path":   cleanPath,
		"bytes":  len(content),
	}, nil
}
