//go:build aliyun_cli_packed_meta

package bundledmeta

import (
	_ "embed"
	"io/fs"
)

//go:embed .generated/canonical.pack
var canonicalPack []byte

func newMetadataFS() fs.FS {
	return newPackedMetadataFS(canonicalPack)
}
