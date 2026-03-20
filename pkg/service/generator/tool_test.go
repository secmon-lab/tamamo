package generator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/tamamo/pkg/service/generator"
)

func TestWriteFileToolSet(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	ts := generator.NewWriteFileToolSetForTest(dir)

	t.Run("specs returns write_file tool", func(t *testing.T) {
		specs, err := ts.Specs(ctx)
		gt.NoError(t, err)
		gt.Equal(t, len(specs), 1)
		gt.Equal(t, specs[0].Name, "write_file")
	})

	t.Run("writes file successfully", func(t *testing.T) {
		result, err := ts.Run(ctx, "write_file", map[string]any{
			"path":    "test.txt",
			"content": "hello world",
		})
		gt.NoError(t, err)
		gt.Equal(t, result["status"], "ok")

		content, err := os.ReadFile(filepath.Join(dir, "test.txt"))
		gt.NoError(t, err)
		gt.Equal(t, string(content), "hello world")
	})

	t.Run("creates subdirectories", func(t *testing.T) {
		result, err := ts.Run(ctx, "write_file", map[string]any{
			"path":    "pages/login.html",
			"content": "<html></html>",
		})
		gt.NoError(t, err)
		gt.Equal(t, result["status"], "ok")

		content, err := os.ReadFile(filepath.Join(dir, "pages", "login.html"))
		gt.NoError(t, err)
		gt.Equal(t, string(content), "<html></html>")
	})

	t.Run("rejects directory traversal", func(t *testing.T) {
		_, err := ts.Run(ctx, "write_file", map[string]any{
			"path":    "../escape.txt",
			"content": "should fail",
		})
		gt.Error(t, err)
	})

	t.Run("rejects absolute path", func(t *testing.T) {
		_, err := ts.Run(ctx, "write_file", map[string]any{
			"path":    "/etc/passwd",
			"content": "should fail",
		})
		gt.Error(t, err)
	})

	t.Run("rejects unknown tool", func(t *testing.T) {
		_, err := ts.Run(ctx, "unknown_tool", map[string]any{})
		gt.Error(t, err)
	})
}
