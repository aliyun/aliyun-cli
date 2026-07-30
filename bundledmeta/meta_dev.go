//go:build !aliyun_cli_packed_meta

package bundledmeta

import (
	"io/fs"
	"os"
)

const defaultMetadataDir = "aliyun-openapi-meta"

// newMetadataFS returns the source metadata tree for local development. The
// default assumes the CLI is run from the repository root. Tests and tools
// launched elsewhere can override it with ALIYUN_CLI_META_DIR.
func newMetadataFS() fs.FS {
	root := os.Getenv("ALIYUN_CLI_META_DIR")
	if root == "" {
		root = defaultMetadataDir
	}
	return os.DirFS(root)
}
