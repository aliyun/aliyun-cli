package openapi

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectOriginalRequestHelpTextPreservesLayoutWhileCappingAndSearching(t *testing.T) {
	var source strings.Builder
	source.WriteString("Description: original renderer\n\nParameters:\n")
	for index := 1; index <= 23; index++ {
		fmt.Fprintf(&source, "  --parameter-%02d              string, help %02d\n", index, index)
		if index == 2 {
			source.WriteString("                              continuation stays aligned\n")
		}
	}
	source.WriteString("\nGlobal Parameters:\n  --region                      string, original global help\n\nExamples:\n  aliyun demo list-items\n")

	t.Run("AI mode caps only API parameters", func(t *testing.T) {
		projected, listing := projectOriginalRequestHelpText(source.String(), helpOptions{}, true)

		require.NotNil(t, listing)
		assert.Equal(t, 20, listing.Shown)
		assert.Equal(t, 23, listing.Total)
		assert.Contains(t, projected, "Description: original renderer")
		assert.Contains(t, projected, "--parameter-20")
		assert.NotContains(t, projected, "--parameter-21")
		assert.Contains(t, projected, "--region                      string, original global help")
		assert.Contains(t, projected, "Examples:\n  aliyun demo list-items")
		assert.Contains(t, projected, "Showing 20 of 23 parameters.")
	})

	t.Run("search keeps original matching blocks and all globals", func(t *testing.T) {
		projected, listing := projectOriginalRequestHelpText(source.String(), helpOptions{Search: "parameter-02"}, true)

		assert.Nil(t, listing)
		assert.Contains(t, projected, "--parameter-02")
		assert.Contains(t, projected, "continuation stays aligned")
		assert.NotContains(t, projected, "--parameter-01")
		assert.NotContains(t, projected, "--parameter-03")
		assert.Contains(t, projected, "--region                      string, original global help")
		assert.NotContains(t, projected, "Showing ")
	})
}

func TestValidateRecoverySearchUsesRealCanonicalHelpProvider(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)

	tests := []struct {
		name    string
		request RecoverySearchRequest
		want    bool
	}{
		{name: "root product", request: RecoverySearchRequest{Keyword: "演示"}, want: true},
		{
			name:    "Pascal API uses legacy default version",
			request: RecoverySearchRequest{Product: "demo", Style: "pascal", Keyword: "report"},
			want:    true,
		},
		{
			name:    "kebab API uses plugin default version",
			request: RecoverySearchRequest{Product: "demo", Style: "kebab", Keyword: "regions"},
			want:    true,
		},
		{
			name:    "request parameter",
			request: RecoverySearchRequest{Product: "demo", API: "create-report", Version: "2026-01-01", Section: "request", Keyword: "workspace-id"},
			want:    true,
		},
		{
			name:    "global parameter",
			request: RecoverySearchRequest{Product: "demo", API: "create-report", Version: "2026-01-01", Section: "request", Keyword: "header"},
			want:    true,
		},
		{
			name:    "response field",
			request: RecoverySearchRequest{Product: "demo", API: "CreateReport", Version: "2026-01-01", Section: "response", Keyword: "report-id"},
			want:    true,
		},
		{
			name:    "no result",
			request: RecoverySearchRequest{Product: "demo", API: "CreateReport", Version: "2026-01-01", Section: "response", Keyword: "not-present"},
			want:    false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, c.validateRecoverySearch(ctx, test.request))
		})
	}
}

func TestValidateRecoverySearchRefusesInstalledPluginTextProvider(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-demo": {Name: "aliyun-cli-demo", Type: plugin.PluginTypeMeta},
	}}

	assert.False(t, c.validateRecoverySearch(ctx, RecoverySearchRequest{
		Product: "demo", Style: "pascal", Keyword: "report",
	}))
}

func TestMachineHelpAIModeHintFollowsEffectiveMode(t *testing.T) {
	for _, test := range []struct {
		name string
		env  string
		want bool
	}{
		{name: "off", env: "0", want: true},
		{name: "on", env: "1", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
			t.Setenv(aimode.EnvAIMode, test.env)
			c.library.helpRepo = canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
			MachineHelpFormatFlag(ctx.Flags()).SetAssigned(true)
			MachineHelpFormatFlag(ctx.Flags()).SetValue("json")

			assert.NoError(t, c.help(ctx, nil))
			assert.Equal(t, test.want, containsJSONField(stdout.Bytes(), "aiModeHint"))
		})
	}
}

func containsJSONField(data []byte, field string) bool {
	return strings.Contains(string(data), `"`+field+`"`)
}
