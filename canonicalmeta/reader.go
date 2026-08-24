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

// ReadProducts reads the Canonical product catalog.
func (r *Reader) ReadProducts() (*ProductsIndex, error) {
	paths := []string{
		"metadatas/products.json",
		"canonical/metadatas/products.json",
	}
	var lastErr error
	for _, candidate := range paths {
		data, err := fs.ReadFile(r.fs, candidate)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				lastErr = err
				continue
			}
			return nil, fmt.Errorf("read canonical products %s failed: %w", candidate, err)
		}
		var products ProductsIndex
		if err := json.Unmarshal(data, &products); err != nil {
			return nil, fmt.Errorf("parse canonical products %s failed: %w", candidate, err)
		}
		return &products, nil
	}
	return nil, fmt.Errorf("read canonical products failed: %w", lastErr)
}

// ReadVersionIndex reads one product/version index.
func (r *Reader) ReadVersionIndex(product, version string) (*VersionIndex, error) {
	lower := strings.ToLower(product)
	paths := []string{
		fmt.Sprintf("canonical/%s/%s/version.json", lower, version),
		fmt.Sprintf("canonical/%s/canonical/%s/%s/version.json", lower, lower, version),
	}
	var lastErr error
	for _, candidate := range paths {
		data, err := fs.ReadFile(r.fs, candidate)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				lastErr = err
				continue
			}
			return nil, fmt.Errorf("read canonical version index %s failed: %w", candidate, err)
		}
		var index VersionIndex
		if err := json.Unmarshal(data, &index); err != nil {
			return nil, fmt.Errorf("parse canonical version index %s failed: %w", candidate, err)
		}
		return &index, nil
	}
	return nil, fmt.Errorf("read canonical version index for %s/%s failed: %w", lower, version, lastErr)
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
			// Do not let a case-insensitive filesystem turn a mistyped API name
			// (for example, describeinstances) into DescribeInstances. Canonical
			// API filenames and their embedded names use the exact APIName.
			if api.Name != apiName {
				return nil, fmt.Errorf("canonical API name mismatch: requested %q, found %q", apiName, api.Name)
			}
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
			if err := validateV1Parameter(p); err != nil {
				return err
			}
		}
	}
	if api.V1Parameters != nil {
		for _, p := range *api.V1Parameters {
			if err := validateV1Parameter(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateV1Parameter(p V1Parameter) error {
	if !isKnownLocation(p.Position) {
		return fmt.Errorf("unknown v1 parameter position %q for parameter %s", p.Position, p.Name)
	}
	for _, child := range p.SubParameters {
		if err := validateV1Parameter(child); err != nil {
			return err
		}
	}
	return nil
}

func isKnownLocation(location string) bool {
	switch strings.ToLower(location) {
	case "query", "body", "host", "domain", "path", "header", "form", "formdata":
		return true
	default:
		return false
	}
}
