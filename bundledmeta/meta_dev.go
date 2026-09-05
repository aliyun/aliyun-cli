//go:build !aliyun_cli_packed_meta

package bundledmeta

import (
	"io/fs"
	"os"
	"path/filepath"
)

const defaultMetadataDir = "aliyun-openapi-meta"

// newMetadataFS returns the source metadata tree for local development.
// ALIYUN_CLI_META_DIR wins when explicitly set. Otherwise, the bounded lookup
// covers the repository root and package test working directories
// (bundledmeta and openapi/runtimehost) without scanning arbitrary ancestors.
func newMetadataFS() fs.FS {
	root := os.Getenv("ALIYUN_CLI_META_DIR")
	if root != "" {
		return os.DirFS(root)
	}
	cwd, err := os.Getwd()
	if err == nil {
		if root, ok := findMetadataDir(cwd); ok {
			return os.DirFS(root)
		}
	}
	return os.DirFS(defaultMetadataDir)
}

func findMetadataDir(start string) (string, bool) {
	dir := start
	for depth := 0; depth <= 2; depth++ {
		candidate := filepath.Join(dir, defaultMetadataDir)
		if isMetadataDir(candidate) {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func isMetadataDir(dir string) bool {
	products, err := os.Stat(filepath.Join(dir, "metadatas", "products.json"))
	if err != nil || products.IsDir() {
		return false
	}
	canonical, err := os.Stat(filepath.Join(dir, "canonical"))
	return err == nil && canonical.IsDir()
}
