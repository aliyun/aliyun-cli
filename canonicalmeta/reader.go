package canonicalmeta

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

// Reader reads Canonical API JSON files from a filesystem.
type Reader struct {
	fs fs.FS
}

// NewReader creates a Reader from a filesystem (e.g., embed.FS or os.DirFS).
func NewReader(fsys fs.FS) *Reader {
	return &Reader{fs: fsys}
}

// ReadAPI reads a single Canonical API JSON file.
// Product directory names are always lowercase.
func (r *Reader) ReadAPI(product, version, apiName string) (*API, error) {
	lower := strings.ToLower(product)
	paths := []string{
		fmt.Sprintf("canonical/%s/%s/%s.json", lower, version, apiName),
		fmt.Sprintf("canonical/%s/canonical/%s/%s/%s.json", lower, lower, version, apiName),
	}
	var lastErr error
	for _, path := range paths {
		api, err := r.readAPIFromPath(path)
		if err == nil {
			return api, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

// ReadAPIFromPath reads a Canonical API JSON from a specific path.
func (r *Reader) ReadAPIFromPath(path string) (*API, error) {
	return r.readAPIFromPath(path)
}

func (r *Reader) readAPIFromPath(path string) (*API, error) {
	data, err := fs.ReadFile(r.fs, path)
	if err != nil {
		return nil, fmt.Errorf("read canonical API %s failed: %w", path, err)
	}

	api := &API{}
	if err := json.Unmarshal(data, api); err != nil {
		return nil, fmt.Errorf("parse canonical API %s failed: %w", path, err)
	}
	if err := validateAPI(api); err != nil {
		return nil, fmt.Errorf("validate canonical API %s failed: %w", path, err)
	}

	return api, nil
}

func validateAPI(api *API) error {
	for _, p := range api.Parameters {
		if !isKnownLocation(p.Location) {
			return fmt.Errorf("unknown canonical location %q for parameter %s", p.Location, p.RawName)
		}
	}
	if api.V1BodyParameters != nil {
		for _, p := range *api.V1BodyParameters {
			if err := validateLegacyBodyParameter(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateLegacyBodyParameter(p LegacyBodyParameter) error {
	if !isKnownLocation(p.Position) {
		return fmt.Errorf("unknown v1 body position %q for parameter %s", p.Position, p.Name)
	}
	for _, child := range p.SubParameters {
		if err := validateLegacyBodyParameter(child); err != nil {
			return err
		}
	}
	return nil
}

func isKnownLocation(location string) bool {
	switch strings.ToLower(location) {
	case "query", "body", "host", "path", "header", "form", "formdata":
		return true
	default:
		return false
	}
}
