package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
	"github.com/stretchr/testify/require"
)

func TestInstalledPluginCredentialFailureHonorsProfileRequirement(t *testing.T) {
	for _, required := range []bool{true, false} {
		t.Run(fmt.Sprintf("profile required=%v", required), func(t *testing.T) {
			c, ctx, _, _ := newCanonicalHelpTestContext(t)
			config.AddFlags(ctx.Flags())
			writeOssTestConfig(t, ctx)
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()
			t.Setenv(config.EnvDisableExternalProcess, "")
			path, _ := config.ConfigurePathFlag(ctx.Flags()).GetValue()
			require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`{"current":"default","profiles":[{"name":"default","mode":"CredentialsURI","credentials_uri":%q,"region_id":"cn-hangzhou"}]}`, server.URL)), 0600))
			root := t.TempDir()
			t.Setenv(plugin.EnvPluginsDir, root)
			manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{"aliyun-cli-recoveryfixture": {
				Name: "aliyun-cli-recoveryfixture", Command: "recoveryfixture", Path: root, Type: plugin.PluginTypeGo, ProfileRequired: &required,
			}}}
			data, err := json.Marshal(manifest)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), data, 0600))
			err = c.main(ctx, []string{"recoveryfixture", "list"})
			require.EqualValues(t, 1, calls.Load())
			if required {
				var credentialError *credentialConfigurationError
				require.ErrorAs(t, err, &credentialError)
				require.ErrorContains(t, err, "status code 500")
				require.Empty(t, ctx.GetRuntimeEnvs())
			} else {
				var pluginError *externalPluginError
				require.ErrorAs(t, err, &pluginError)
				require.ErrorContains(t, err, "plugin binary not found")
				require.NotContains(t, ctx.GetRuntimeEnvs(), "ALIBABA_CLOUD_ACCESS_KEY_ID")
				require.NotContains(t, ctx.GetRuntimeEnvs(), "ALIBABA_CLOUD_ACCESS_KEY_SECRET")
				require.Contains(t, ctx.GetRuntimeEnvs(), safety.EnvSafetyPolicyFile)
			}
		})
	}
}

func TestPluginInstallRecoveryRespectsInteractiveAndAutomaticModes(t *testing.T) {
	for _, mode := range []string{"noninteractive", "cancel", "automatic prerelease", "interactive prerelease"} {
		t.Run(mode, func(t *testing.T) {
			c, ctx, output, stderr := newCanonicalHelpTestContext(t)
			root := t.TempDir()
			t.Setenv(plugin.EnvPluginsDir, root)
			t.Setenv(plugin.EnvNoCache, "")
			require.NoError(t, os.WriteFile(filepath.Join(root, "plugin_search_index_cache.json"), []byte(`{"recoveryfixture list":"aliyun-cli-recoveryfixture"}`), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(root, "plugin_pkg_index_cache.json"), []byte(`{"plugins":[]}`), 0600))
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "unexpected index request", 500)
			}))
			defer server.Close()
			t.Setenv(pluginsettings.EnvSourceBase, server.URL)
			previousInput, previousInteractive := stdin, isInteractiveInput
			t.Cleanup(func() { stdin, isInteractiveInput = previousInput, previousInteractive })
			stdin = strings.NewReader("no\n")
			isInteractiveInput = func() bool { return mode != "noninteractive" }
			c.profile.AutoPluginInstallEnablePre = true
			c.profile.AutoPluginInstall = mode == "automatic prerelease"
			if mode == "interactive prerelease" {
				stdin = strings.NewReader("yes\n")
			}
			name, err := c.findAndInstallPlugin(ctx, "recoveryfixture list", "recoveryfixture")
			require.Empty(t, name)
			if strings.Contains(mode, "prerelease") {
				require.ErrorContains(t, err, "not found")
				require.Contains(t, stderr.String(), "including pre-release versions")
			} else {
				require.NoError(t, err)
				if mode == "cancel" {
					require.Contains(t, stderr.String(), "Installation cancelled")
				} else {
					require.Contains(t, stderr.String(), "plugin install --names aliyun-cli-recoveryfixture")
				}
			}
			require.Zero(t, calls.Load())
			require.Empty(t, output.String())
		})
	}
}

func TestCommandOwnershipAndCorruptAIModeConfigurationPreserveErrors(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	writeOssTestConfig(t, ctx)
	path, _ := config.ConfigurePathFlag(ctx.Flags()).GetValue()
	require.NoError(t, os.Mkdir(filepath.Join(filepath.Dir(path), aimode.AiConfigFileName), 0700))
	require.False(t, c.applyEffectiveAIMode(ctx))
	require.False(t, legacyAIModeEnabled(ctx))
	require.Empty(t, aiModeSuffixForContext(ctx))
	c.SetRootHelpSpecs([]RootCommandSpec{{Path: []string{"extensionfixture"}, Group: RootGroupExtension}}, nil)
	c.InitWithCommand(ctx.Command())
	t.Setenv("NO_COLOR", "")
	t.Cleanup(func() { cli.SetNoColorOverride(false) })
	colorBefore := cli.Colorized(cli.Red, "extension output")
	require.Contains(t, colorBefore, "\x1b[")
	CliAIModeFlag(ctx.Flags()).SetAssigned(true)
	ctx.Command().BeforeExecute(ctx, []string{"extensionfixture"})
	require.Equal(t, colorBefore, cli.Colorized(cli.Red, "extension output"))
	cause := errors.New("extension request rejected")
	require.ErrorIs(t, c.finishCommandRun(ctx, []string{"extensionfixture"}, cause), cause)
	require.NoError(t, c.finishCommandRun(ctx, nil, nil))
	previous := runtimeTryHelp
	t.Cleanup(func() { runtimeTryHelp = previous })
	runtimeTryHelp = func(context *cli.Context, args []string) (bool, error) {
		require.Same(t, ctx, context)
		require.Equal(t, []string{"unregistered", "list", "--help"}, args)
		return true, cause
	}
	require.ErrorIs(t, c.help(ctx, []string{"unregistered", "list"}), cause)
}

func TestInteractiveInputTreatsRedirectedAndClosedStdinAsNoninteractive(t *testing.T) {
	previous := os.Stdin
	t.Cleanup(func() { os.Stdin = previous })
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	require.NoError(t, err)
	t.Cleanup(func() { file.Close() })
	os.Stdin = file
	require.False(t, isInteractiveInput())
	require.NoError(t, file.Close())
	require.False(t, isInteractiveInput())
}

func TestRecoverySearchDeclinesIncompleteInstalledMetadata(t *testing.T) {
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	for _, catalog := range []string{"{", `{"products":[{"code":"fixture","version":"v1","versions":["v1"]}]}`} {
		c.library.helpRepo = canonicalmeta.NewRepository(fstest.MapFS{"metadatas/products.json": {Data: []byte(catalog)}})
		c.library.baselineHelpRepo = c.library.helpRepo
		require.False(t, c.validateRecoverySearch(ctx, RecoverySearchRequest{Keyword: "absent"}))
		require.False(t, c.validateRecoverySearch(ctx, RecoverySearchRequest{Product: "fixture", Version: "v1", Style: "camel", Keyword: "report"}))
	}
	doc := &machineHelpRootDocument{GlobalFlags: []machineHelpFlagSummary{{Name: "--profile", Visibility: string(RootVisibilityDefault)}}}
	applyRootHelpOptions(doc, helpOptions{All: true}, false)
	require.Len(t, doc.GlobalFlags, 1)
	require.Equal(t, "--profile", doc.GlobalFlags[0].Name)
}
