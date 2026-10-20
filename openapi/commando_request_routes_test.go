package openapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/require"
)

func TestCommandoOpenapiRPCDryRunUsesDefaultPathAndMethod(t *testing.T) {
	for _, flagName := range []string{CliDryRunFlagName, CliDryRunJsonFlagName} {
		t.Run(flagName, func(t *testing.T) {
			c, ctx, output, _ := newCanonicalHelpTestContext(t)
			config.AddFlags(ctx.Flags())
			writeOssTestConfig(t, ctx)
			ctx.SetUnknownFlags(cli.NewFlagSet())
			ctx.Flags().Get(flagName).SetAssigned(true)
			c.profile.RegionId = "cn-hangzhou"
			product := &meta.Product{Code: "sls", Version: "2026-01-01", ApiStyle: "rpc", GlobalEndpoint: "local.test"}
			api := &canonicalmeta.API{Name: "CreateReport"}
			require.NoError(t, c.processApiInvoke(ctx, product, api, "", ""))
			require.Contains(t, output.String(), "CreateReport")
			require.Contains(t, output.String(), "cn-hangzhou.log.aliyuncs.com")
			created, err := c.createHttpContext(ctx, product, api, "", "")
			require.NoError(t, err)
			oc, ok := created.(*OpenapiContext)
			require.True(t, ok)
			require.Equal(t, "POST", oc.method)
			require.Equal(t, "/", oc.path)
			product.ApiStyle = "restful"
			_, err = c.createHttpContext(ctx, product, api, "CreateReport", "")
			require.ErrorContains(t, err, "proper restful call")
		})
	}
}

func TestCommandoBothInvokePathsValidateBeforeTransport(t *testing.T) {
	for _, path := range []string{"openapi", "legacy"} {
		for _, invalid := range []string{"doc required", "constraint", "unknown parameter"} {
			t.Run(path+"/"+invalid, func(t *testing.T) {
				c, ctx, _, _ := newCanonicalHelpTestContext(t)
				config.AddFlags(ctx.Flags())
				writeOssTestConfig(t, ctx)
				c.profile = config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou"}
				ctx.SetUnknownFlags(cli.NewFlagSet())
				CliAIModeFlag(ctx.Flags()).SetAssigned(true)
				api := &canonicalmeta.API{Name: "CreateReport", Method: "POST", Protocol: "HTTPS", Parameters: []canonicalmeta.Parameter{{Name: "Name", RawName: "Name", Type: "string", Location: "query", DocRequired: true, Enum: []string{"valid"}}}}
				want := "missing required"
				if invalid == "constraint" {
					assignLegacyUnknown(t, ctx, "Name", "invalid")
					want = "not allowed"
				} else if invalid == "unknown parameter" {
					api.Parameters[0].DocRequired = false
					assignLegacyUnknown(t, ctx, "Wrong", "value")
					want = "Wrong"
				}
				product := &meta.Product{Code: "sls", Version: "2026-01-01", ApiStyle: "rpc", GlobalEndpoint: "local.test", ApiNames: []string{"CreateReport"}}
				var err error
				if path == "openapi" {
					err = c.processApiInvoke(ctx, product, api, "POST", "/")
				} else {
					product.Code = "demo"
					repo, loadErr := meta.MockLoadRepository([]meta.Product{*product})
					require.NoError(t, loadErr)
					canonical := newFakeCanonicalRepo()
					canonical.AddAPI("demo", product.Version, api)
					c.library.builtinRepo, c.library.canonicalRepo = repo, canonical
					err = c.processInvoke(ctx, "demo", "CreateReport", "")
				}
				require.ErrorContains(t, err, want)
			})
		}
	}
}

func TestCommandoSSEBranchPropagatesStreamingFailure(t *testing.T) {
	c, ctx, output, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	writeOssTestConfig(t, ctx)
	c.profile = config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou"}
	ctx.SetUnknownFlags(cli.NewFlagSet())
	previous := hookHttpContextCall
	t.Cleanup(func() { hookHttpContextCall = previous })
	cause := errors.New("stream disconnected")
	called := false
	hookHttpContextCall = func(call func() error) func() error { return func() error { called = true; return cause } }
	err := c.processApiInvoke(ctx, &meta.Product{Code: "sls", Version: "2026-01-01", ApiStyle: "restful", GlobalEndpoint: "local.test"}, &canonicalmeta.API{Name: "Stream", Protocol: "HTTPS|SSE"}, "POST", "/stream")
	require.ErrorIs(t, err, cause)
	require.True(t, called)
	require.Empty(t, output.String())
}

func TestCommandoPluginManagerFailuresAreReportedWithoutNetwork(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(path, []byte("file"), 0600))
	t.Setenv(plugin.EnvPluginsDir, path)
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	c.pluginLoaded, c.localLoaded, c.localManifest = false, false, nil
	c.loadPlugins()
	require.Error(t, c.pluginIndexErr)
	c.loadLocalPlugins()
	require.Error(t, c.localManifestErr)
	_, err := c.findAndInstallPlugin(ctx, "demo list", "demo")
	require.Error(t, err)
	ctx.SetUnknownFlags(cli.NewFlagSet())
	require.ErrorContains(t, c.main(ctx, []string{"unknown-product", "list"}), "failed to check plugin status")
	previous := stdin
	t.Cleanup(func() { stdin = previous })
	stdin = &failedInputReader{}
	_, err = c.interactiveInstallPlugin(ctx, nil, "demo", "demo list", false)
	require.ErrorContains(t, err, "failed to read user input")
	ctx.SetRuntimeEnvs(nil)
	c.setLangEnv(ctx)
	require.Equal(t, "en_US.UTF-8", ctx.GetRuntimeEnvs()["LANG"])
}

type failedInputReader struct{}

func (*failedInputReader) Read([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestCommandoUnknownLowercaseCommandsRetainInstallRecovery(t *testing.T) {
	for _, outcome := range []string{"install failure", "baseline", "available plugin", "catalog unavailable"} {
		t.Run(outcome, func(t *testing.T) {
			c, ctx, _, stderr := newCanonicalHelpTestContext(t)
			config.AddFlags(ctx.Flags())
			writeOssTestConfig(t, ctx)
			t.Setenv(plugin.EnvPluginsDir, t.TempDir())
			c.library.helpRepo = nil
			c.pluginLoaded = true
			product := "routingfixture"
			if outcome == "baseline" {
				product = "ecs"
			}
			if outcome == "available plugin" {
				c.pluginIndex = &plugin.Index{Plugins: []plugin.PluginInfo{{Name: "aliyun-cli-routingfixture", ProductCode: product}}}
			}
			cause := errors.New("fixture installation unavailable")
			if outcome == "catalog unavailable" {
				c.pluginIndexErr = cause
			}
			previousDispatch, previousInstall := runtimeTryDispatch, installPluginForCommand
			t.Cleanup(func() { runtimeTryDispatch, installPluginForCommand = previousDispatch, previousInstall })
			runtimeTryDispatch = func(*cli.Context, []string) (bool, error) { return false, nil }
			installCalls := 0
			installPluginForCommand = func(command *Commando, context *cli.Context, name, code string) (string, error) {
				installCalls++
				require.Same(t, c, command)
				require.Same(t, ctx, context)
				require.Equal(t, product+" unknown-command", name)
				require.Equal(t, product, code)
				if outcome == "install failure" {
					return "", cause
				}
				return "", nil
			}
			err := c.main(ctx, []string{product, "unknown-command"})
			require.Equal(t, 1, installCalls)
			switch outcome {
			case "install failure":
				require.ErrorIs(t, err, cause)
			case "baseline":
				var invalid *InvalidBaselineCommandError
				require.ErrorAs(t, err, &invalid)
				require.Equal(t, "unknown-command", invalid.Command)
				require.NotEmpty(t, invalid.Candidates)
			case "available plugin":
				require.ErrorContains(t, err, "plugin install --name aliyun-cli-routingfixture")
			case "catalog unavailable":
				var invalid *InvalidProductOrPluginError
				require.ErrorAs(t, err, &invalid)
				require.Contains(t, stderr.String(), cause.Error())
			}
		})
	}
}

func TestCommandoMainDispatchesRPCAndMethodPathThroughOpenapi(t *testing.T) {
	for _, form := range []string{"RPC API", "restful path", "missing API"} {
		t.Run(form, func(t *testing.T) {
			c, ctx, output, _ := newCanonicalHelpTestContext(t)
			config.AddFlags(ctx.Flags())
			writeOssTestConfig(t, ctx)
			ctx.SetUnknownFlags(cli.NewFlagSet())
			CliDryRunFlag(ctx.Flags()).SetAssigned(true)
			ForceFlag(ctx.Flags()).SetAssigned(true)
			product := meta.Product{Code: "sls", Version: "v1", ApiStyle: "rpc", ApiNames: []string{"GetReport"}, GlobalEndpoint: "local.test"}
			api := &canonicalmeta.API{Name: "GetReport", Method: "GET", Protocol: "HTTPS", PathPattern: "/reports"}
			args := []string{"sls", "GetReport"}
			if form == "restful path" {
				product.ApiStyle, args = "restful", []string{"sls", "GET", "/reports"}
			}
			repo, err := meta.MockLoadRepository([]meta.Product{product})
			require.NoError(t, err)
			canonical := newFakeCanonicalRepo()
			if form != "missing API" {
				canonical.AddAPI(product.Code, product.Version, api)
			}
			c.library.builtinRepo, c.library.canonicalRepo = repo, canonical
			err = c.main(ctx, args)
			if form == "missing API" {
				require.ErrorContains(t, err, "not a valid api")
				require.Empty(t, output.String())
				return
			}
			require.NoError(t, err)
			require.Contains(t, output.String(), "GetReport")
			require.Contains(t, output.String(), "GET")
		})
	}
}

func TestCommandoMainQuotesMissingMetadataWithoutInvokingTarget(t *testing.T) {
	for _, explicitVersion := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing API", true: "unknown version"}[explicitVersion], func(t *testing.T) {
			c, ctx, output, _ := newCanonicalHelpTestContext(t)
			config.AddFlags(ctx.Flags())
			writeOssTestConfig(t, ctx)
			ctx.SetUnknownFlags(cli.NewFlagSet())
			EstimateCostFlag(ctx.Flags()).SetAssigned(true)
			config.SkipSecureVerify(ctx.Flags()).SetAssigned(true)
			version := "v1"
			if explicitVersion {
				VersionFlag(ctx.Flags()).SetAssigned(true)
				VersionFlag(ctx.Flags()).SetValue("v2")
				version = "v2"
			}
			repo, err := meta.MockLoadRepository([]meta.Product{{Code: "demo", Version: "v1", ApiStyle: "restful"}})
			require.NoError(t, err)
			c.library.builtinRepo, c.library.canonicalRepo = repo, newFakeCanonicalRepo()
			received := make(chan estimateCostRequest, 1)
			servePricingFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request estimateCostRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					http.Error(w, "invalid quote", 400)
					return
				}
				received <- request
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"price":{"success":true,"amount":42}}`)
			}))
			require.NoError(t, c.main(ctx, []string{"demo", "CreateReport"}))
			select {
			case request := <-received:
				require.Equal(t, version, request.PopVersion)
				require.Equal(t, "CreateReport", request.ApiName)
			default:
				t.Fatal("quote request was not sent")
			}
			require.Contains(t, output.String(), "42")
		})
	}
}
