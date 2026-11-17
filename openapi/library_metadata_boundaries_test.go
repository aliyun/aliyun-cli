package openapi

import (
	"bytes"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/require"
)

func TestProductUsageHandlesIncompleteAndAnonymousMetadata(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, style := range []string{"rpc", "restful"} {
		t.Run(style, func(t *testing.T) {
			repo, err := meta.MockLoadRepository([]meta.Product{{Code: "demo", Version: "v1", ApiStyle: style, ApiNames: []string{"Missing", "Deprecated", "Anonymous", "Undocumented"}}})
			require.NoError(t, err)
			var output bytes.Buffer
			lib := &Library{builtinRepo: repo, writer: &output, canonicalRepo: &byNameOnlyCanonicalRepo{apis: map[string]*canonicalmeta.API{
				"Deprecated":   {Name: "Deprecated", Method: "GET", PathPattern: "/old", Deprecated: true, DescriptionEn: "old endpoint", DescriptionZh: "old endpoint"},
				"Anonymous":    {Name: "Anonymous", Method: "GET", PathPattern: "/public", Security: []string{"Anonymous"}, DescriptionEn: "public endpoint", DescriptionZh: "public endpoint"},
				"Undocumented": {Name: "Undocumented", Method: "GET", PathPattern: "/unknown"},
			}}}
			require.NoError(t, lib.PrintProductUsage("demo", true))
			require.Contains(t, output.String(), "[Deprecated]")
			require.Contains(t, output.String(), "Undocumented")
			if style == "rpc" {
				require.Contains(t, output.String(), "[Anonymous]")
			}
		})
	}
}

func TestCompletionNamesDeduplicateAliasesAndFallBackToAPINames(t *testing.T) {
	lib := &Library{canonicalRepo: canonicalmeta.NewRepository(fstest.MapFS{
		"canonical/demo/v1/version.json":    {Data: []byte(`{"version":"v1","apis":{"First":{"cmd_name":"same"},"Second":{"cmd_name":"SAME"},"CreateThing":{}}}`)},
		"canonical/demo/empty/version.json": {Data: []byte(`{"version":"empty","apis":{}}`)},
	})}
	product := meta.Product{Code: "demo", Version: "v1", ApiNames: []string{"FallbackName"}}
	require.Equal(t, []string{"create-thing", "same"}, lib.GetAPINamesForCompletion(product, true))
	product.Version = "empty"
	require.Equal(t, []string{"fallback-name"}, lib.GetAPINamesForCompletion(product, true))
	lib.canonicalRepo = nil
	require.Equal(t, []string{"fallback-name"}, lib.GetAPINamesForCompletion(product, true))
}
