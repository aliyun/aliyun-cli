package openapi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	runtimemeta "github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/stretchr/testify/require"
)

func TestLegacyMetadataHelpSortsRequiredFieldsAndHidesTransportParameters(t *testing.T) {
	previous := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(previous); cli.SetNoColorOverride(false) })
	for _, language := range []string{"en", "zh"} {
		for _, color := range []bool{false, true} {
			t.Run(language+map[bool]string{false: "/plain", true: "/color"}[color], func(t *testing.T) {
				t.Setenv("NO_COLOR", "")
				i18n.SetLanguage(language)
				cli.SetNoColorOverride(!color)
				repo, err := meta.MockLoadRepository([]meta.Product{{Code: "zulu"}, {Code: "alpha", Name: map[string]string{"fr": "Service Français"}}})
				require.NoError(t, err)
				var output bytes.Buffer
				library := &Library{writer: &output, builtinRepo: repo, canonicalRepo: newFakeCanonicalRepo()}
				library.PrintProducts()
				text := stripHelpANSI(output.String())
				require.Contains(t, text, "alpha")
				require.Contains(t, text, "zulu")
				require.Less(t, strings.Index(text, "alpha"), strings.Index(text, "zulu"))
				require.Contains(t, text, "Service Français")
				require.ErrorContains(t, library.PrintApiUsage("alpha", "Missing"), "not a valid api")
				api := &canonicalmeta.API{Parameters: []canonicalmeta.Parameter{
					{Name: "Zulu", RawName: "Zulu", Type: "string", Location: "query", Required: true},
					{Name: "Alpha", RawName: "Alpha", Type: "string", Location: "query", Required: true},
					{Name: "SecretHeader", RawName: "SecretHeader", Type: "string", Location: "header"},
					{Name: "SecretDomain", RawName: "SecretDomain", Type: "string", Location: "domain"},
				}}
				output.Reset()
				printCanonicalAPI(&output, api, "")
				text = stripHelpANSI(output.String())
				require.Contains(t, text, "--Alpha")
				require.Contains(t, text, "--Zulu")
				require.Less(t, strings.Index(text, "--Alpha"), strings.Index(text, "--Zulu"))
				require.NotContains(t, text, "SecretHeader")
				require.NotContains(t, text, "SecretDomain")
				output.Reset()
				printBodyFields(&output, []*canonicalmeta.Parameter{{RawName: "Z", Type: "string"}, {RawName: "A", Type: "string"}}, language)
				text = stripHelpANSI(output.String())
				require.Contains(t, text, "|- A")
				require.Contains(t, text, "|- Z")
				require.Less(t, strings.Index(text, "|- A"), strings.Index(text, "|- Z"))
			})
		}
	}
}

func TestAPISuggestionsDeduplicateAndPreferSpecificResourceKeywords(t *testing.T) {
	candidates := []string{"", "DescribeInstanceNetwork", "DescribeInstance", "ListInstanceNetwork", "ListInstances", "DescribeNetwork", "CreateInstanceNetwork", "DescribeInstanceNetwork"}
	suggestions := apiTokenSuggestions("DescribeInstanceNetwork", candidates, 0)
	require.NotEmpty(t, suggestions)
	require.Equal(t, "DescribeInstanceNetwork", suggestions[0])
	require.LessOrEqual(t, len(suggestions), cli.DefaultSuggestLimit)
	seen := map[string]bool{}
	for _, suggestion := range suggestions {
		require.False(t, seen[suggestion])
		seen[suggestion] = true
	}
	require.Equal(t, "Network", apiRecoverySearchKeyword("DescribeMissingInstanceNetworkNetwork", []string{"DescribeInstanceNetwork", "ListInstanceNetwork", "DescribeInstance", "ListInstance"}, "pascal"))
}

func TestInstalledMetadataCatalogIgnoresGoAndIncompleteRecords(t *testing.T) {
	root := t.TempDir()
	t.Setenv(plugin.EnvPluginsDir, root)
	manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-go":     {ProductCode: "go", Type: plugin.PluginTypeGo},
		"aliyun-cli-empty":  {Type: plugin.PluginTypeMeta},
		"aliyun-cli-ecs":    {ProductCode: "ECS", Type: plugin.PluginTypeMeta},
		"aliyun-cli-absent": {ProductCode: "unknown-coverage-product", Type: plugin.PluginTypeMeta},
	}}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), data, 0600))
	entries := installedMetaPluginEntries()
	require.Len(t, entries, 1)
	require.Equal(t, "ecs", entries[0].Code)
	require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), []byte("{"), 0600))
	require.Empty(t, installedMetaPluginEntries())
	blocked := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocked, nil, 0600))
	t.Setenv(plugin.EnvPluginsDir, blocked)
	require.Empty(t, installedMetaPluginEntries())
	fields := engineFieldsToCanonical([]runtimemeta.Parameter{{Name: "parent", Type: runtimemeta.TypeObject, Fields: []runtimemeta.Parameter{{Name: "child", Type: runtimemeta.TypeString}}}})
	require.Len(t, fields[0].Fields, 1)
	require.Equal(t, "child", fields[0].Fields[0].Name)
	shape := engineShapeToCanonical(&runtimemeta.Parameter{Type: runtimemeta.TypeMap, ValueType: &runtimemeta.Parameter{Type: runtimemeta.TypeString}})
	require.Equal(t, "string", shape.Value.Type)
}
