package openapi

import (
	"os"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/stretchr/testify/assert"
)

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
