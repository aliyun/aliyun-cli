package canonicalmeta

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

// Repository provides query access to Canonical API metadata.
type Repository struct {
	reader *Reader
}

// NewRepository creates a Repository from a filesystem.
func NewRepository(fsys fs.FS) *Repository {
	return &Repository{
		reader: NewReader(fsys),
	}
}

// GetAPI retrieves a Canonical API by product, version, and API name.
func (r *Repository) GetAPI(product, version, apiName string) (*API, error) {
	return r.reader.ReadAPI(product, version, apiName)
}

// GetProducts returns the Canonical product catalog.
func (r *Repository) GetProducts() (*ProductsIndex, error) {
	return r.reader.ReadProducts()
}

// GetVersionIndex returns the lightweight API index for one product version.
func (r *Repository) GetVersionIndex(product, version string) (*VersionIndex, error) {
	return r.reader.ReadVersionIndex(product, version)
}

// GetAPIByPath finds a Canonical API by matching method and path pattern.
// Used for ROA-style APIs where the API is identified by HTTP method + URL path.
func (r *Repository) GetAPIByPath(product, version, method, path string, apiNames []string) (*API, error) {
	for _, apiName := range apiNames {
		api, err := r.reader.ReadAPI(product, version, apiName)
		if err != nil {
			continue
		}

		if api.MatchLegacyPath(method, path) {
			return api, nil
		}
	}

	return nil, fmt.Errorf("no canonical API found for %s %s", method, path)
}

func (api *API) MatchLegacyPath(method, path string) bool {
	apiMethods := strings.ToUpper(api.Method)
	if !strings.Contains(apiMethods, strings.ToUpper(method)) {
		return false
	}

	pattern := replacePathPattern(api.PathPattern)
	re := regexp.MustCompile("^" + pattern + "$")
	return re.MatchString(path)
}

// replacePathPattern converts [param] placeholders to regex patterns.
func replacePathPattern(pattern string) string {
	re := regexp.MustCompile(`\[[^\]]+\]`)
	return re.ReplaceAllString(pattern, "[0-9a-zA-Z_\\-\\.{}]+")
}
