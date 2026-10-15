package export

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-cli/v3/bundledmeta"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/stretchr/testify/require"
)

func TestLegacyExportMetadataFixtureRoundTrip(t *testing.T) {
	previous := bundledmeta.Metadatas
	bundledmeta.Metadatas = os.DirFS("../canonicalmeta/testdata")
	t.Cleanup(func() { bundledmeta.Metadatas = previous })
	output := t.TempDir()
	require.NoError(t, LegacyExportMetadata(output))
	for _, name := range []string{"metadatas/products.json", "en-US/products.json", "zh-CN/products.json", "metadatas/demo/CreateReport.json", "en-US/demo/version.json", "zh-CN/demo/version.json"} {
		data, err := os.ReadFile(filepath.Join(output, name))
		require.NoError(t, err)
		require.NotEmpty(t, data)
	}
	var product localeProductSet
	readJSONFile(t, filepath.Join(output, "en-US/products.json"), &product)
	require.Equal(t, "Demo Service", product.Products[0].Name)
}

func TestProductExportPropagatesEveryOutputFailure(t *testing.T) {
	repo := canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	for _, name := range []string{"metadatas", "metadatas/demo/CreateReport.json", "en-US", "en-US/demo/CreateReport.json", "zh-CN", "zh-CN/demo/CreateReport.json", "en-US/demo/version.json", "zh-CN/demo/version.json"} {
		t.Run(name, func(t *testing.T) {
			output := t.TempDir()
			blocked := filepath.Join(output, name)
			require.NoError(t, os.MkdirAll(filepath.Dir(blocked), 0700))
			if filepath.Ext(name) == ".json" {
				require.NoError(t, os.Mkdir(blocked, 0700))
			} else {
				require.NoError(t, os.WriteFile(blocked, []byte("blocked"), 0600))
			}
			require.Error(t, exportProduct(repo, testExportProduct(), output))
		})
	}
}

func TestProductLoadingFallbackAndMalformedMetadata(t *testing.T) {
	previous := bundledmeta.Metadatas
	t.Cleanup(func() { bundledmeta.Metadatas = previous })
	for _, data := range []fs.FS{fstest.MapFS{}, fstest.MapFS{"metadatas/products.json": {Data: []byte("invalid")}}} {
		bundledmeta.Metadatas = data
		_, err := loadProductsExt()
		require.Error(t, err)
		require.Error(t, LegacyExportMetadata(t.TempDir()))
	}
	bundledmeta.Metadatas = fstest.MapFS{"canonical/metadatas/products.json": {Data: []byte(`{"products":[{"code":"fallback"}]}`)}}
	products, err := loadProductsExt()
	require.NoError(t, err)
	require.Equal(t, "fallback", products.Products[0].Code)
}
