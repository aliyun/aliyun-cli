package openapi

import (
	"bytes"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductHelpUsesSelectedVersionEndpoints(t *testing.T) {
	files := fstest.MapFS{
		"metadatas/products.json":                {Data: []byte(`{"products":[{"code":"demo","version":"2026-01-01","plugin_default_version":"2025-01-01","versions":["2025-01-01","2026-01-01"],"regional_endpoints":{"cn-beijing":"old.example.com"},"regional_vpc_endpoints":{"cn-beijing":"old-vpc.example.com"}}]}`)},
		"canonical/demo/2026-01-01/version.json": {Data: []byte(`{"version":"2026-01-01","apis":{},"endpoints":[{"region_id":"cn-beijing","region_name_zh":"北京","region_name_en":"Beijing","endpoint":"new.example.com","vpc_endpoint":"new-vpc.example.com"}]}`)},
		"canonical/demo/2025-01-01/version.json": {Data: []byte(`{"version":"2025-01-01","apis":{},"endpoints":[{"region_id":"cn-beijing","region_name_zh":"北京","region_name_en":"Beijing","endpoint":"previous.example.com","vpc_endpoint":"previous-vpc.example.com"}]}`)},
	}
	service := newMachineHelpService(canonicalmeta.NewRepository(files))
	oldLanguage := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(oldLanguage) })
	for _, language := range []string{"zh", "en"} {
		i18n.SetLanguage(language)
		for _, tc := range []struct{ style, version, endpoint, vpc string }{
			{"camel", "", "new.example.com", "new-vpc.example.com"},
			{"kebab", "", "previous.example.com", "previous-vpc.example.com"},
			{"camel", "2025-01-01", "previous.example.com", "previous-vpc.example.com"},
		} {
			doc, err := service.buildProductForStyle("demo", tc.version, tc.style)
			require.NoError(t, err)
			name := map[string]string{"zh": "北京", "en": "Beijing"}[language]
			assert.Equal(t, []machineHelpEndpoint{{RegionID: "cn-beijing", RegionName: name, Endpoint: tc.endpoint, VPCEndpoint: tc.vpc}}, doc.Endpoints)
			var output bytes.Buffer
			require.NoError(t, encodeMachineHelpJSON(&output, doc, false, "endpoints"))
			assert.Contains(t, output.String(), `"regionId": "cn-beijing"`)
			assert.Contains(t, output.String(), `"regionName": "`+name+`"`)
			assert.Contains(t, output.String(), `"vpcEndpoint": "`+tc.vpc+`"`)
			assert.NotContains(t, output.String(), "region_name_")
			output.Reset()
			require.NoError(t, renderCanonicalProductText(&output, doc, ""))
			assert.Contains(t, output.String(), name)
			assert.Contains(t, output.String(), tc.vpc)
			assert.NotContains(t, output.String(), "old.example.com")
		}
	}
}

func TestProductHelpExplicitEmptyVersionEndpointsDoesNotUseProductMaps(t *testing.T) {
	files := fstest.MapFS{
		"metadatas/products.json":        {Data: []byte(`{"products":[{"code":"demo","version":"v1","regional_endpoints":{"cn-beijing":"old.example.com"}}]}`)},
		"canonical/demo/v1/version.json": {Data: []byte(`{"version":"v1","apis":{},"endpoints":[]}`)},
	}
	doc, err := newMachineHelpService(canonicalmeta.NewRepository(files)).buildProductForStyle("demo", "", "camel")
	require.NoError(t, err)
	assert.Empty(t, doc.Endpoints)
}
