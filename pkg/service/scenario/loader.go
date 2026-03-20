package scenario

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

// scenarioJSON represents the on-disk format of scenario.json.
type scenarioJSON struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	ServerSignature string            `json:"server_signature"`
	Headers         map[string]string `json:"headers"`
	Theme           string            `json:"theme"`
}

// routesJSON represents the on-disk format of routes.json.
type routesJSON struct {
	Routes []scenario.Route `json:"routes"`
}

// Load reads a scenario from a directory or ZIP file.
func Load(_ context.Context, path string) (*scenario.Scenario, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to stat scenario path",
			goerr.V("path", path),
			goerr.T(errutil.TagNotFound),
		)
	}

	if info.IsDir() {
		return loadFromDir(path)
	}

	if strings.HasSuffix(path, ".zip") {
		return loadFromZip(path)
	}

	return nil, goerr.New("unsupported scenario format: expected directory or .zip file",
		goerr.V("path", path),
		goerr.T(errutil.TagValidation),
	)
}

func loadFromDir(dir string) (*scenario.Scenario, error) {
	// Read scenario.json
	metaPath := filepath.Join(dir, "scenario.json")
	metaData, err := os.ReadFile(metaPath) // #nosec G304 -- path is constructed from user-specified scenario directory
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read scenario.json",
			goerr.V("path", metaPath),
			goerr.T(errutil.TagNotFound),
		)
	}

	var meta scenarioJSON
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, goerr.Wrap(err, "failed to parse scenario.json",
			goerr.V("path", metaPath),
			goerr.T(errutil.TagValidation),
		)
	}

	// Read routes.json
	routesPath := filepath.Join(dir, "routes.json")
	routesData, err := os.ReadFile(routesPath) // #nosec G304 -- path is constructed from user-specified scenario directory
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read routes.json",
			goerr.V("path", routesPath),
			goerr.T(errutil.TagNotFound),
		)
	}

	var routes routesJSON
	if err := json.Unmarshal(routesData, &routes); err != nil {
		return nil, goerr.Wrap(err, "failed to parse routes.json",
			goerr.V("path", routesPath),
			goerr.T(errutil.TagValidation),
		)
	}

	// Resolve body_file references and collect pages
	pages, err := resolveRoutes(dir, routes.Routes)
	if err != nil {
		return nil, err
	}

	s := &scenario.Scenario{
		Meta: scenario.Meta{
			Name:            meta.Name,
			Description:     meta.Description,
			ServerSignature: meta.ServerSignature,
			Headers:         meta.Headers,
			Theme:           meta.Theme,
		},
		Pages:  pages,
		Routes: routes.Routes,
	}

	return s, nil
}

// resolveRoutes reads body_file references and builds page list.
func resolveRoutes(dir string, routes []scenario.Route) ([]scenario.Page, error) {
	var pages []scenario.Page
	seen := make(map[string]bool)

	for i := range routes {
		r := &routes[i]
		if r.BodyFile == "" {
			continue
		}

		filePath := filepath.Join(dir, r.BodyFile)
		content, err := os.ReadFile(filePath) // #nosec G304 -- path is from scenario route definition
		if err != nil {
			return nil, goerr.Wrap(err, "failed to read body file",
				goerr.V("path", filePath),
				goerr.V("route", r.Path),
				goerr.T(errutil.TagNotFound),
			)
		}
		r.Body = string(content)

		if !seen[r.BodyFile] {
			seen[r.BodyFile] = true
			ct := r.Headers["Content-Type"]
			if ct == "" {
				ct = "text/html"
			}
			pages = append(pages, scenario.Page{
				Path:        r.Path,
				HTMLFile:    r.BodyFile,
				ContentType: ct,
			})
		}
	}

	return pages, nil
}

func loadFromZip(zipPath string) (*scenario.Scenario, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to open zip file",
			goerr.V("path", zipPath),
			goerr.T(errutil.TagNotFound),
		)
	}
	defer func() { _ = r.Close() }()

	// Extract to temp directory
	tmpDir, err := os.MkdirTemp("", "tamamo-scenario-*")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create temp directory",
			goerr.T(errutil.TagInternal),
		)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	for _, f := range r.File {
		name := filepath.Clean(f.Name)
		// Prevent zip slip
		target := filepath.Join(tmpDir, name)
		if !strings.HasPrefix(target, filepath.Clean(tmpDir)+string(os.PathSeparator)) {
			return nil, goerr.New("invalid file path in zip (zip slip attempt)",
				goerr.V("file", f.Name),
				goerr.T(errutil.TagValidation),
			)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o750); err != nil {
				return nil, goerr.Wrap(err, "failed to create directory from zip",
					goerr.V("path", target),
					goerr.T(errutil.TagInternal),
				)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return nil, goerr.Wrap(err, "failed to create parent directory",
				goerr.V("path", filepath.Dir(target)),
				goerr.T(errutil.TagInternal),
			)
		}

		src, err := f.Open()
		if err != nil {
			return nil, goerr.Wrap(err, "failed to open file in zip",
				goerr.V("file", f.Name),
				goerr.T(errutil.TagInternal),
			)
		}

		dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- target is validated against zip slip
		if err != nil {
			_ = src.Close()
			return nil, goerr.Wrap(err, "failed to create extracted file",
				goerr.V("path", target),
				goerr.T(errutil.TagInternal),
			)
		}

		// Limit extraction size to 50MB per file
		if _, err := io.Copy(dst, io.LimitReader(src, 50*1024*1024)); err != nil {
			_ = src.Close()
			_ = dst.Close()
			return nil, goerr.Wrap(err, "failed to extract file",
				goerr.V("file", f.Name),
				goerr.T(errutil.TagInternal),
			)
		}
		_ = src.Close()
		_ = dst.Close()
	}

	return loadFromDir(tmpDir)
}

// Save writes a scenario to a directory.
func Save(_ context.Context, dir string, s *scenario.Scenario) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return goerr.Wrap(err, "failed to create scenario directory",
			goerr.V("path", dir),
			goerr.T(errutil.TagInternal),
		)
	}

	// Write scenario.json
	meta := scenarioJSON{
		Name:            s.Meta.Name,
		Description:     s.Meta.Description,
		ServerSignature: s.Meta.ServerSignature,
		Headers:         s.Meta.Headers,
		Theme:           s.Meta.Theme,
	}
	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return goerr.Wrap(err, "failed to marshal scenario.json",
			goerr.T(errutil.TagInternal),
		)
	}
	if err := os.WriteFile(filepath.Join(dir, "scenario.json"), metaData, 0o600); err != nil {
		return goerr.Wrap(err, "failed to write scenario.json",
			goerr.T(errutil.TagInternal),
		)
	}

	// Write routes.json
	rt := routesJSON{Routes: s.Routes}
	routesData, err := json.MarshalIndent(rt, "", "  ")
	if err != nil {
		return goerr.Wrap(err, "failed to marshal routes.json",
			goerr.T(errutil.TagInternal),
		)
	}
	if err := os.WriteFile(filepath.Join(dir, "routes.json"), routesData, 0o600); err != nil {
		return goerr.Wrap(err, "failed to write routes.json",
			goerr.T(errutil.TagInternal),
		)
	}

	// Write page files
	for _, p := range s.Pages {
		pagePath := filepath.Join(dir, p.HTMLFile)
		if err := os.MkdirAll(filepath.Dir(pagePath), 0o750); err != nil {
			return goerr.Wrap(err, "failed to create page directory",
				goerr.V("path", filepath.Dir(pagePath)),
				goerr.T(errutil.TagInternal),
			)
		}
		// Find the body content from routes
		for _, r := range s.Routes {
			if r.BodyFile == p.HTMLFile {
				if err := os.WriteFile(pagePath, []byte(r.Body), 0o600); err != nil {
					return goerr.Wrap(err, "failed to write page file",
						goerr.V("path", pagePath),
						goerr.T(errutil.TagInternal),
					)
				}
				break
			}
		}
	}

	return nil
}

// SaveAsZip writes a scenario directory to a ZIP file.
func SaveAsZip(dir, zipPath string) error {
	f, err := os.Create(zipPath) // #nosec G304 -- path is from user-specified output
	if err != nil {
		return goerr.Wrap(err, "failed to create zip file",
			goerr.V("path", zipPath),
			goerr.T(errutil.TagInternal),
		)
	}
	defer func() { _ = f.Close() }()

	w := zip.NewWriter(f)
	defer func() { _ = w.Close() }()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return goerr.Wrap(err, "failed to walk directory",
				goerr.V("path", path),
				goerr.T(errutil.TagInternal),
			)
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return goerr.Wrap(err, "failed to compute relative path",
				goerr.V("path", path),
				goerr.T(errutil.TagInternal),
			)
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return goerr.Wrap(err, "failed to create zip header",
				goerr.V("file", rel),
				goerr.T(errutil.TagInternal),
			)
		}
		header.Name = filepath.ToSlash(rel)
		header.Method = zip.Deflate

		writer, err := w.CreateHeader(header)
		if err != nil {
			return goerr.Wrap(err, "failed to create zip entry",
				goerr.V("file", rel),
				goerr.T(errutil.TagInternal),
			)
		}

		src, err := os.Open(path) // #nosec G304 -- path is from controlled Walk within dir
		if err != nil {
			return goerr.Wrap(err, "failed to open file for zip",
				goerr.V("path", path),
				goerr.T(errutil.TagInternal),
			)
		}
		defer func() { _ = src.Close() }()

		written, err := io.Copy(writer, src)
		if err != nil {
			return goerr.Wrap(err, "failed to write file to zip",
				goerr.V("file", rel),
				goerr.T(errutil.TagInternal),
			)
		}
		_ = written

		return nil
	})
}

// ExportZip generates a ZIP from an in-memory scenario.
// It saves to a temp directory first, then packages as ZIP.
func ExportZip(ctx context.Context, s *scenario.Scenario, zipPath string) error {
	tmpDir, err := os.MkdirTemp("", "tamamo-export-*")
	if err != nil {
		return goerr.Wrap(err, "failed to create temp directory",
			goerr.T(errutil.TagInternal),
		)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := Save(ctx, tmpDir, s); err != nil {
		return goerr.Wrap(err, "failed to save scenario to temp directory",
			goerr.T(errutil.TagInternal),
		)
	}

	if err := SaveAsZip(tmpDir, zipPath); err != nil {
		return goerr.Wrap(err, fmt.Sprintf("failed to create zip at %s", zipPath),
			goerr.T(errutil.TagInternal),
		)
	}

	return nil
}
