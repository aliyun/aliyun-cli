package canonicalmeta

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderFallbackLayoutsAndDirectAPIPath(t *testing.T) {
	fsys := fstest.MapFS{
		"canonical/metadatas/products.json":             {Data: []byte(`{"products":[{"code":"demo"}]}`)},
		"canonical/demo/canonical/demo/v1/version.json": {Data: []byte(`{"version":"v1","apis":{}}`)},
		"custom/Create.json": {Data: []byte(`{
			"name":"Create","parameters":[{"name":"id","raw_name":"Id","location":"query","type":"string"}]
		}`)},
	}
	reader := NewReader(fsys)

	products, err := reader.ReadProducts()
	require.NoError(t, err)
	require.Len(t, products.Products, 1)
	assert.Equal(t, "demo", products.Products[0].Code)

	index, err := reader.ReadVersionIndex("DEMO", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1", index.Version)

	api, err := reader.ReadAPIFromPath("custom/Create.json")
	require.NoError(t, err)
	assert.Equal(t, "Create", api.Name)
}

func TestReaderReportsMissingMalformedAndValidationErrors(t *testing.T) {
	t.Run("missing indexes", func(t *testing.T) {
		reader := NewReader(fstest.MapFS{})
		_, err := reader.ReadProducts()
		assert.ErrorContains(t, err, "read canonical products failed")
		_, err = reader.ReadVersionIndex("Demo", "v1")
		assert.ErrorContains(t, err, "demo/v1")
		_, err = reader.ReadAPI("Demo", "v1", "Missing")
		assert.Error(t, err)
	})

	t.Run("malformed indexes", func(t *testing.T) {
		reader := NewReader(fstest.MapFS{
			"metadatas/products.json":        {Data: []byte(`{broken`)},
			"canonical/demo/v1/version.json": {Data: []byte(`{broken`)},
		})
		_, err := reader.ReadProducts()
		assert.ErrorContains(t, err, "parse canonical products")
		_, err = reader.ReadVersionIndex("demo", "v1")
		assert.ErrorContains(t, err, "parse canonical version index")
	})

	t.Run("api parse and canonical parameter validation", func(t *testing.T) {
		reader := NewReader(fstest.MapFS{
			"bad.json": {Data: []byte(`{broken`)},
			"location.json": {Data: []byte(`{
				"name":"BadLocation","parameters":[{"name":"id","raw_name":"Id","location":"cookie"}]
			}`)},
			"v1.json": {Data: []byte(`{
				"name":"BadV1","parameters":[],"v1_parameters":[{
					"name":"Parent","position":"query","sub_parameters":[{"name":"Child","position":"cookie"}]
				}]
			}`)},
		})
		_, err := reader.ReadAPIFromPath("bad.json")
		assert.ErrorContains(t, err, "parse canonical API")
		_, err = reader.ReadAPIFromPath("location.json")
		assert.ErrorContains(t, err, "unknown canonical location")
		_, err = reader.ReadAPIFromPath("v1.json")
		assert.ErrorContains(t, err, "unknown v1 parameter position")
	})

	t.Run("api name mismatch", func(t *testing.T) {
		reader := NewReader(fstest.MapFS{
			"canonical/demo/v1/Wrong.json": {Data: []byte(`{"name":"Actual","parameters":[]}`)},
		})
		_, err := reader.ReadAPI("demo", "v1", "Wrong")
		assert.ErrorContains(t, err, "name mismatch")
	})
}

func TestAPIAndVersionEntrySmallContracts(t *testing.T) {
	assert.False(t, (&API{}).IsAnonymous())
	assert.True(t, (&API{Security: []string{"AK", "Anonymous"}}).IsAnonymous())
	assert.Empty(t, (*API)(nil).Title("en"))
	assert.Empty(t, (*API)(nil).TitleOrDescription("zh"))

	entry := VersionAPIEntry{TitleZh: "标题", DescriptionZh: "描述", DescriptionEn: "description"}
	assert.Equal(t, "标题", entry.Title("zh"))
	assert.Empty(t, entry.Title("en"))
	assert.Equal(t, "标题", entry.TitleOrDescription("zh"))
	assert.Equal(t, "description", entry.TitleOrDescription("en"))
}
