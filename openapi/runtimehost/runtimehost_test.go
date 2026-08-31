// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtimehost

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/bundledmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/throttlingretry"
	openapiruntime "github.com/aliyun/aliyun-openapi-runtime"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
)

// captureExecutor records the ExecContext it receives instead of
// sending, so tests can assert on how dispatch populated it.
type captureExecutor struct{ last *runtime.ExecContext }

func (c *captureExecutor) Execute(ec *runtime.ExecContext) (*runtime.Response, error) {
	c.last = ec
	return &runtime.Response{StatusCode: 200, Raw: []byte(`{}`)}, nil
}

func TestBuildUserAgentSuffix(t *testing.T) {
	t.Setenv(sysconfig.EnvUserAgent, "env-\nagent/1")
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	userAgent := &cli.Flag{Name: "user-agent", AssignedMode: cli.AssignedOnce}
	ctx.Flags().Add(userAgent)
	userAgent.SetAssigned(true)
	userAgent.SetValue("flag-\ragent/2")
	forceOff := &cli.Flag{Name: "no-cli-ai-mode", AssignedMode: cli.AssignedOnce}
	ctx.Flags().Add(forceOff)
	forceOff.SetAssigned(true)

	if got, want := buildUserAgentSuffix(ctx), "env-agent/1 flag-agent/2"; got != want {
		t.Fatalf("suffix = %q, want %q", got, want)
	}
}

func TestBuildUserAgentSuffixForDetectedAgentUsesMarkerOnly(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.Flags().Add(config.NewConfigurePathFlag())
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(t.TempDir(), "config.json"))
	ctx.Flags().Add(&cli.Flag{Name: "cli-ai-mode"})
	ctx.Flags().Add(&cli.Flag{Name: "no-cli-ai-mode"})
	ctx.SetAgentName("codex")

	if got := buildUserAgentSuffix(ctx); got != aimode.UserAgentEnabledMarker {
		t.Fatalf("agent suffix = %q, want marker only", got)
	}
}

func TestDispatchPropagatesEffectiveAIMode(t *testing.T) {
	originalDispatch := engineDispatch
	t.Cleanup(func() { engineDispatch = originalDispatch })

	var captured engine.Request
	engineDispatch = func(request engine.Request) error {
		captured = request
		return nil
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.Flags().Add(config.NewConfigurePathFlag())
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(t.TempDir(), "config.json"))
	forceOn := &cli.Flag{Name: "cli-ai-mode"}
	forceOff := &cli.Flag{Name: "no-cli-ai-mode"}
	ctx.Flags().Add(forceOn)
	ctx.Flags().Add(forceOff)

	forceOn.SetAssigned(true)
	if err := Dispatch(ctx, []string{"ecs", "describe-regions"}); err != nil {
		t.Fatal(err)
	}
	if !captured.AIMode {
		t.Fatal("engine request AIMode = false, want true")
	}

	forceOff.SetAssigned(true)
	if err := Dispatch(ctx, []string{"ecs", "describe-regions"}); err != nil {
		t.Fatal(err)
	}
	if captured.AIMode {
		t.Fatal("engine request AIMode = true when force-off is assigned")
	}

	forceOn.SetAssigned(false)
	forceOff.SetAssigned(false)
	ctx.SetAgentName("codex")
	if err := Dispatch(ctx, []string{"ecs", "describe-regions"}); err != nil {
		t.Fatal(err)
	}
	if !captured.AIMode {
		t.Fatal("engine request AIMode = false for detected agent")
	}

	forceOff.SetAssigned(true)
	if err := Dispatch(ctx, []string{"ecs", "describe-regions"}); err != nil {
		t.Fatal(err)
	}
	if captured.AIMode {
		t.Fatal("engine request AIMode = true for detected agent with force-off")
	}
}

func TestDispatchPluginHelpPreservesRawArgsAndMapsHelpContext(t *testing.T) {
	stdout := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, new(bytes.Buffer))
	language := config.NewLanguageFlag()
	ctx.Flags().Add(language)
	language.SetAssigned(true)
	language.SetValue("zh")
	ctx.SetAgentName("codex")

	args := []string{"ecs", "describe-regions", "--help"}
	want := append([]string(nil), args...)
	if err := DispatchPluginHelp(ctx, args); err != nil {
		t.Fatal(err)
	}
	args[1] = "mutated"

	if !reflect.DeepEqual(want, []string{"ecs", "describe-regions", "--help"}) {
		t.Fatalf("plugin Help mutated caller args: %#v", want)
	}
	var document map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil ||
		document["schemaVersion"] != "v1" || document["kind"] != "api" ||
		len(strings.Split(strings.TrimSpace(stdout.String()), "\n")) != 1 {
		t.Fatalf("AI mode must force Runtime JSON Help, got:\n%s", stdout.String())
	}
}

func TestRuntimeHelpArgsPreservesTargetsAndModifiers(t *testing.T) {
	got, product, command := runtimeHelpArgs([]string{
		"help", "demo", "--api-version", "v1", "get-thing",
		"--report-id", "--help", "--cli-section=response",
		"--help-search", "request id", "--help-all", "--cli-output", "json",
		"--language=zh",
	})
	want := []string{
		"demo", "get-thing", "--api-version", "v1",
		"--report-id", "--help", "--cli-section=response",
		"--help-search", "request id",
		"--help-all",
		"--cli-output", "json",
		"--language=zh",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runtime Help args = %#v, want %#v", got, want)
	}
	if product != "demo" || command != "get-thing" {
		t.Fatalf("runtime Help target = %q %q", product, command)
	}

	got, product, command = runtimeHelpArgs([]string{
		"demo", "--help-search", "instance", "--help-all", "--language", "zh",
	})
	want = []string{"demo", "--help-search", "instance", "--help-all", "--language", "zh"}
	if !reflect.DeepEqual(got, want) || product != "demo" || command != "" {
		t.Fatalf("product Runtime Help = %#v, %q %q", got, product, command)
	}

	got, product, command = runtimeHelpArgs([]string{
		"--profile", "work", "--language", "zh", "demo", "--api-version=v1", "get-thing", "--help",
	})
	want = []string{"demo", "get-thing", "--language", "zh", "--api-version=v1", "--help"}
	if !reflect.DeepEqual(got, want) || product != "demo" || command != "get-thing" {
		t.Fatalf("root-option Runtime Help = %#v, %q %q", got, product, command)
	}
}

func TestTryHelpRoutesRuntimeHelpLevelsWithoutHostCredentials(t *testing.T) {
	t.Setenv(aimode.EnvAIMode, "0")
	stdout := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, new(bytes.Buffer))

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "product", args: []string{"ecs", "--help"}, want: "Available APIs:"},
		{name: "action", args: []string{"ecs", "describe-regions", "--help"}, want: "Usage:"},
		{name: "request", args: []string{"ecs", "describe-regions", "--help", "--cli-section", "request"}, want: "Parameters:"},
		{name: "response", args: []string{"help", "ecs", "describe-regions", "--cli-section", "response"}, want: "Responses:"},
		{name: "parameter", args: []string{"ecs", "describe-regions", "--accept-language", "--help"}, want: "Parameter:"},
		{name: "search", args: []string{"ecs", "describe-regions", "--help-search", "language", "--help-all"}, want: "accept-language"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout.Reset()
			handled, err := TryHelp(ctx, test.args)
			if err != nil {
				t.Fatalf("TryHelp(%v): %v", test.args, err)
			}
			if !handled {
				t.Fatalf("TryHelp(%v) was not handled", test.args)
			}
			if !strings.Contains(stdout.String(), test.want) {
				t.Fatalf("TryHelp(%v) missing %q:\n%s", test.args, test.want, stdout.String())
			}
		})
	}

	for _, args := range [][]string{
		{"ecs", "DescribeRegions", "--help"},
		{"help"},
		{"not-a-runtime-product", "--help"},
	} {
		handled, err := TryHelp(ctx, args)
		if err != nil || handled {
			t.Fatalf("TryHelp(%v) = %v, %v; want Host fallback", args, handled, err)
		}
	}
}

func TestProfileHostTransportOptions(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_OTEL_TRACEPARENT", "00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01")
	t.Setenv("ALIBABA_CLOUD_OTEL_BAGGAGE", "tenant=test")
	t.Setenv(sysconfig.EnvSourceIP, " 192.0.2.1 ")
	t.Setenv(sysconfig.EnvSecureTransport, " true ")
	t.Setenv(sysconfig.EnvCallContextSkipProducts, " custom, DEMO ")
	t.Setenv(throttlingretry.EnvEnabled, "false")
	t.Setenv(throttlingretry.EnvMaxAttempts, "7")
	t.Setenv(throttlingretry.EnvMaxDelayMS, "4321")

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	options := (&cliHost{ctx: ctx}).TransportOptions()
	if options.Headers["traceparent"] == "" || options.Headers["baggage"] != "tenant=test" {
		t.Fatalf("OTEL headers = %#v", options.Headers)
	}
	if options.CallContext.SourceIP != "192.0.2.1" || options.CallContext.SecureTransport != "true" {
		t.Fatalf("call context = %#v", options.CallContext)
	}
	if !reflect.DeepEqual(options.CallContext.SkipProducts, []string{"custom", "DEMO"}) {
		t.Fatalf("skip products = %#v", options.CallContext.SkipProducts)
	}
	if options.ThrottlingRetry.Enabled == nil || *options.ThrottlingRetry.Enabled {
		t.Fatalf("throttling enabled = %#v", options.ThrottlingRetry.Enabled)
	}
	if options.ThrottlingRetry.MaxAttempts != 7 || options.ThrottlingRetry.MaxDelayMS != 4321 {
		t.Fatalf("throttling = %#v", options.ThrottlingRetry)
	}
}

func TestHelpLanguagePrefersCommandFlag(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	flag := config.NewLanguageFlag()
	ctx.Flags().Add(flag)
	flag.SetAssigned(true)
	flag.SetValue("zh")

	if got := helpLanguage(ctx); got != "zh" {
		t.Fatalf("help language = %q, want zh", got)
	}
}

// TestHostSettingsAppliedToExecContext verifies the engine copies the
// host's profile-derived wire settings (timeouts / retry / endpoint
// type) into the ExecContext, mirroring the Go plugin env behaviour.
func TestHostSettingsAppliedToExecContext(t *testing.T) {
	cap := &captureExecutor{}
	eng := openapiruntime.NewEngine(openapiruntime.Options{BaselineFS: bundledmeta.Metadatas, BundledBy: "test"}, cap)

	host := runtime.StaticHost{
		RegionID: "cn-hangzhou",
		SettingsVal: runtime.Settings{
			ReadTimeout:      30 * time.Second,
			ConnectTimeout:   10 * time.Second,
			RetryCount:       3,
			EndpointType:     "vpc",
			Language:         "en",
			SkipSecureVerify: true,
			CLIVersion:       "3.0.234",
			UserAgent:        "tool/1",
		},
		TransportOptionsVal: runtime.TransportOptions{
			Headers: map[string]string{"traceparent": "00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01"},
			CallContext: runtime.CallContextOptions{
				SourceIP: "192.0.2.1",
			},
			ThrottlingRetry: runtime.ThrottlingRetryOptions{
				MaxAttempts: 5,
				MaxDelayMS:  2500,
			},
		},
	}
	var buf bytes.Buffer
	// describe-regions has no required params; not a dry-run so the
	// (capturing) executor is invoked with the fully-populated ec.
	err := eng.Dispatch(engine.Request{
		Args: []string{"ecs", "describe-regions"},
		Out:  &buf,
		Lang: "en",
		Host: host,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if cap.last == nil {
		t.Fatal("executor was not invoked")
	}
	if cap.last.ReadTimeout != 30*time.Second || cap.last.ConnectTimeout != 10*time.Second {
		t.Errorf("timeouts not applied: read=%v connect=%v", cap.last.ReadTimeout, cap.last.ConnectTimeout)
	}
	if cap.last.RetryCount != 3 {
		t.Errorf("retry not applied: %d", cap.last.RetryCount)
	}
	if !cap.last.UseVPC {
		t.Errorf("endpoint-type=vpc should set UseVPC")
	}
	if cap.last.Region != "cn-hangzhou" {
		t.Errorf("region = %q", cap.last.Region)
	}
	if !cap.last.SkipSecureVerify {
		t.Error("SkipSecureVerify not applied")
	}
	if cap.last.CLIVersion != "3.0.234" {
		t.Errorf("CLIVersion = %q", cap.last.CLIVersion)
	}
	if cap.last.UserAgent != "tool/1" {
		t.Errorf("UserAgent = %q", cap.last.UserAgent)
	}
	if cap.last.Transport.Headers["traceparent"] == "" {
		t.Error("transport headers not applied")
	}
	if cap.last.Transport.CallContext.SourceIP != "192.0.2.1" {
		t.Errorf("SourceIP = %q", cap.last.Transport.CallContext.SourceIP)
	}
	if cap.last.Transport.ThrottlingRetry.MaxAttempts != 5 {
		t.Errorf("throttling max attempts = %d", cap.last.Transport.ThrottlingRetry.MaxAttempts)
	}
}

// baselineEngine boots an engine over the embedded baseline metadata,
// exactly as the production wiring does (minus user/override layers).
func baselineEngine(t *testing.T) *engine.Engine {
	t.Helper()
	return openapiruntime.NewEngine(openapiruntime.Options{
		BaselineFS: bundledmeta.Metadatas,
		BundledBy:  "aliyun-cli test",
	}, nil)
}

func TestEngineProductHelpReadsBaselineIndex(t *testing.T) {
	eng := baselineEngine(t)
	var buf bytes.Buffer
	err := eng.ProductHelp(engine.Request{
		Args: []string{"ecs"},
		Out:  &buf,
		Lang: "en",
	})
	if err != nil {
		t.Fatalf("ProductHelp: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Product: ecs",
		"API Version: 2014-05-26",
		"Available APIs:",
		"describe-instances",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("product help missing %q in output", want)
		}
	}
	if strings.Contains(out, "  DescribeInstances") {
		t.Fatalf("common-runtime product help must use kebab commands:\n%s", out)
	}
}

// runOapi drives one dispatch and captures stdout. A StaticHost with a
// fixed region keeps the test hermetic (dry-run never touches creds).
func runOapi(t *testing.T, eng *engine.Engine, region string, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	err := eng.Dispatch(engine.Request{
		Args: args,
		Out:  &buf,
		Lang: "en",
		Host: runtime.StaticHost{RegionID: region},
	})
	return buf.String(), err
}

// dryRunOutput mirrors aliyun-cli's full --cli-dry-run-json request shape.
type dryRunOutput struct {
	Product     string            `json:"product"`
	API         string            `json:"api"`
	Region      string            `json:"region,omitempty"`
	Style       string            `json:"style"`
	Endpoint    string            `json:"endpoint"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	Query       map[string]string `json:"query,omitempty"`
	Body        string            `json:"body,omitempty"`
	BodyFormat  string            `json:"bodyFormat,omitempty"`
	PathPattern string            `json:"pathPattern,omitempty"`
	Pathname    string            `json:"pathname,omitempty"`
	PathParams  map[string]string `json:"pathParams,omitempty"`
	Action      string            `json:"action,omitempty"`
	Version     string            `json:"version,omitempty"`
}

// ---------------------------------------------------------------------------
// Loader / routing (was aliyun-openapi-runtime/register_test.go)
// ---------------------------------------------------------------------------

func baselineLoaderFor(t *testing.T, product string) loader.Loader {
	t.Helper()
	l := openapiruntime.NewLoader(openapiruntime.Options{
		BaselineFS: bundledmeta.Metadatas,
		BundledBy:  "aliyun-cli test",
	})
	if err := l.EnsureProduct(product); err != nil {
		t.Fatalf("ensure product %s: %v", product, err)
	}
	return l
}

func TestEnsureProductReadsEcsFromBaseline(t *testing.T) {
	l := baselineLoaderFor(t, "ecs")
	ecs := l.LookupProduct("ecs")
	if ecs == nil || len(ecs.Versions) == 0 || ecs.DefaultVersion == "" {
		t.Fatalf("ecs product missing/incomplete: %+v", ecs)
	}
	api, err := l.GetAPI("ecs", "2014-05-26", "DescribeInstances")
	if err != nil {
		t.Fatalf("GetAPI: %v", err)
	}
	if api.Name != "DescribeInstances" || api.Style != meta.StyleRPC {
		t.Fatalf("unexpected api: %+v", api)
	}
	if len(api.Parameters) == 0 || len(api.Endpoints.Public) == 0 {
		t.Fatal("api missing parameters/endpoints")
	}
}

func TestNestedParameterMappingPreserved(t *testing.T) {
	l := baselineLoaderFor(t, "ecs")
	api, err := l.GetAPI("ecs", "2014-05-26", "AllocateDedicatedHosts")
	if err != nil {
		t.Fatalf("GetAPI: %v", err)
	}
	tag := api.FindParameter("tag")
	if tag == nil || tag.Type != meta.TypeArray || tag.ItemType == nil || tag.ItemType.Type != meta.TypeObject {
		t.Fatalf("nested tag param not preserved: %+v", tag)
	}
	if len(tag.ItemType.Fields) == 0 {
		t.Fatal("tag object has no fields")
	}
}

func TestResolveCommandRoundTrip(t *testing.T) {
	l := baselineLoaderFor(t, "ecs")
	ref, err := l.ResolveCommand("ecs", "describe-instances")
	if err != nil {
		t.Fatalf("ResolveCommand: %v", err)
	}
	if ref.Name != "DescribeInstances" || ref.Product != "ecs" || ref.Version == "" {
		t.Fatalf("unexpected ref: %+v", ref)
	}
	if _, err := l.ResolveCommand("ecs", "definitely-not-real"); !errors.Is(err, loader.ErrCommandNotFound) {
		t.Fatalf("expected ErrCommandNotFound, got %v", err)
	}
}

// TestMultiVersionResolutionFromBaseline exercises the unified
// version-resolution path against real embedded data: bailian ships two
// API versions (default 2023-12-29, older 2023-06-01) whose command
// sets do not overlap. create-token exists only in the older version;
// add-category only in the default.
func TestMultiVersionResolutionFromBaseline(t *testing.T) {
	l := baselineLoaderFor(t, "bailian")

	// Default command, no --api-version -> default-version fast path.
	def, err := l.ResolveCommandVersion("bailian", "add-category", "")
	if err != nil {
		t.Fatalf("default resolve: %v", err)
	}
	if def.Version != "2023-12-29" || def.Name != "AddCategory" {
		t.Fatalf("default resolution wrong: %+v", def)
	}

	// Explicit default version resolves identically (still fast path).
	if got, err := l.ResolveCommandVersion("bailian", "add-category", "2023-12-29"); err != nil || got != def {
		t.Fatalf("explicit default mismatch: %+v err=%v", got, err)
	}

	// Command that only exists in the older version requires the flag.
	older, err := l.ResolveCommandVersion("bailian", "create-token", "2023-06-01")
	if err != nil {
		t.Fatalf("older resolve: %v", err)
	}
	if older.Version != "2023-06-01" || older.Name != "CreateToken" {
		t.Fatalf("older resolution wrong: %+v", older)
	}

	// Without the flag it must NOT resolve against the default version.
	if _, err := l.ResolveCommandVersion("bailian", "create-token", ""); !errors.Is(err, loader.ErrCommandNotFound) {
		t.Fatalf("create-token should be unresolvable on the default version, got %v", err)
	}

	// An undeclared version is rejected via the `versions` list.
	if _, err := l.ResolveCommandVersion("bailian", "add-category", "1999-01-01"); err == nil {
		t.Fatal("undeclared version should error")
	}

	versions := l.FindCommandVersions("bailian", "create-document-tag")
	if !reflect.DeepEqual(versions, []string{"2023-06-01"}) {
		t.Fatalf("create-document-tag versions = %v", versions)
	}
}

func TestBaselineSSEMetadata(t *testing.T) {
	l := baselineLoaderFor(t, "rdsai")
	ref, err := l.ResolveCommand("rdsai", "chat-messages")
	if err != nil {
		t.Fatalf("resolve SSE command: %v", err)
	}
	api, err := l.GetAPI(ref.Product, ref.Version, ref.Name)
	if err != nil {
		t.Fatalf("load SSE API: %v", err)
	}
	if !api.IsSSE {
		t.Fatalf("rdsai chat-messages was not marked SSE: %+v", api)
	}
}

func TestListAPIVersionsFromBaseline(t *testing.T) {
	eng := baselineEngine(t)
	if !eng.Resolvable("bailian", "list-api-versions") {
		t.Fatal("multi-version product should expose list-api-versions")
	}

	out, err := runOapi(t, eng, "", "bailian", "list-api-versions")
	if err != nil {
		t.Fatalf("list-api-versions: %v", err)
	}
	for _, want := range []string{
		"Product: bailian",
		"2023-06-01",
		"* 2023-12-29 (default)",
		"ALIBABA_CLOUD_BAILIAN_API_VERSION=<version>",
		"aliyun bailian --api-version <version>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("version list missing %q:\n%s", want, out)
		}
	}

	help, err := runOapi(t, eng, "", "bailian", "list-api-versions", "--help")
	if err != nil {
		t.Fatalf("list-api-versions help: %v", err)
	}
	if !strings.Contains(help, "Description: List supported API versions") ||
		!strings.Contains(help, "aliyun bailian list-api-versions") {
		t.Fatalf("unexpected list-api-versions help:\n%s", help)
	}

	if eng.Resolvable("ecs", "list-api-versions") {
		t.Fatal("single-version product should not expose list-api-versions")
	}

	var defaultHelp bytes.Buffer
	if err := eng.ProductHelp(engine.Request{Args: []string{"bailian"}, Out: &defaultHelp, Lang: "en"}); err != nil {
		t.Fatalf("default product help: %v", err)
	}
	if strings.Contains(defaultHelp.String(), "list-api-versions") {
		t.Fatalf("Runtime Help v1 product API list must contain metadata APIs only:\n%s", defaultHelp.String())
	}

	var versionHelp bytes.Buffer
	if err := eng.ProductHelp(engine.Request{
		Args: []string{"bailian", "--api-version", "2023-06-01"},
		Out:  &versionHelp,
		Lang: "en",
	}); err != nil {
		t.Fatalf("version product help: %v", err)
	}
	if strings.Contains(versionHelp.String(), "list-api-versions") {
		t.Fatalf("version-specific product help should hide list-api-versions:\n%s", versionHelp.String())
	}
}

func TestAPIHelpFromBaseline(t *testing.T) {
	eng := baselineEngine(t)

	var help bytes.Buffer
	err := eng.APIHelp(engine.Request{
		Args: []string{"ecs", "describe-regions"},
		Out:  &help,
		Lang: "en",
	})
	if err != nil {
		t.Fatalf("APIHelp: %v", err)
	}
	for _, want := range []string{"aliyun ecs describe-regions", "Query Options:"} {
		if !strings.Contains(help.String(), want) {
			t.Errorf("API help missing %q:\n%s", want, help.String())
		}
	}

	var versionsHelp bytes.Buffer
	err = eng.APIHelp(engine.Request{
		Args: []string{"bailian", "list-api-versions"},
		Out:  &versionsHelp,
		Lang: "en",
	})
	if err != nil {
		t.Fatalf("list-api-versions APIHelp: %v", err)
	}
	if !strings.Contains(versionsHelp.String(), "Description: List supported API versions") {
		t.Fatalf("unexpected list-api-versions API help:\n%s", versionsHelp.String())
	}
}

func TestCrossVersionCommandErrorSuggestsAPIVersion(t *testing.T) {
	eng := baselineEngine(t)
	_, err := runOapi(t, eng, "", "bailian", "create-document-tag", "--help")
	if err == nil {
		t.Fatal("expected command version error")
	}
	for _, want := range []string{
		`command "create-document-tag" is not available in API version "2023-12-29"`,
		"available versions for this command: 2023-06-01",
		"aliyun bailian create-document-tag --api-version 2023-06-01",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("version error missing %q:\n%s", want, err)
		}
	}
}

// ---------------------------------------------------------------------------
// End-to-end dispatch (was aliyun-openapi-runtime/engine/oapi_e2e_test.go)
// ---------------------------------------------------------------------------

func TestOapiDryRunJSONEndToEnd(t *testing.T) {
	eng := baselineEngine(t)
	out, err := runOapi(t, eng, "cn-beijing",
		"ecs", "run-instances",
		"--biz-region-id", "cn-beijing",
		"--image-id", "img1",
		"--instance-type", "ecs.g6.large",
		"--cli-dry-run-json",
	)
	if err != nil {
		t.Fatalf("dry-run-json: %v\n%s", err, out)
	}
	var m dryRunOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("not dryRunOutput JSON: %v\n%s", err, out)
	}
	if m.Style != "RPC" || m.Action != "RunInstances" || m.Version != "2014-05-26" || m.Method != "POST" {
		t.Errorf("request identity wrong: %+v", m)
	}
	if m.Product != "ecs" || m.API != "RunInstances" || m.Region != "cn-beijing" {
		t.Errorf("legacy metadata identity wrong: %+v", m)
	}
	if m.Endpoint != "ecs.cn-beijing.aliyuncs.com" || m.Query["ImageId"] != "img1" || m.Query["InstanceType"] != "ecs.g6.large" {
		t.Errorf("request details wrong: %+v", m)
	}
	if strings.Count(strings.TrimSpace(out), "\n") != 0 {
		t.Errorf("dry-run-json should be one line:\n%s", out)
	}
}

func TestOapiDryRunHumanEndToEnd(t *testing.T) {
	eng := baselineEngine(t)
	out, err := runOapi(t, eng, "cn-hangzhou",
		"ecs", "run-instances",
		"--biz-region-id", "cn-hangzhou",
		"--instance-type", "ecs.g6.large",
		"--tag", "Key=env", "Value=prod",
		"--cli-dry-run",
	)
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, out)
	}
	for _, want := range []string{"DRY-RUN MODE", "API Action: RunInstances", "Tag.1.Key: env", "Tag.1.Value: prod"} {
		if !strings.Contains(out, want) {
			t.Errorf("human dry-run missing %q in:\n%s", want, out)
		}
	}
}

func TestOapiCloudControlWildcardPathDryRunEndToEnd(t *testing.T) {
	eng := baselineEngine(t)
	out, err := runOapi(t, eng, "cn-hangzhou",
		"cloudcontrol", "get-resources",
		"--request-path", "/api/v1/providers/qqq/products/dd/resources/dddd:4",
		"--cli-dry-run",
	)
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, out)
	}
	if want := "URL: /api/v1/providers/qqq/products/dd/resources/dddd%3A4"; !strings.Contains(out, want) {
		t.Fatalf("wildcard URL missing %q in:\n%s", want, out)
	}
	for _, unwanted := range []string{"%7Bprovider%7D", "%7Bproduct%7D", "%2A"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("unresolved wildcard template %q in:\n%s", unwanted, out)
		}
	}
}

func TestOapiDashPrefixedValueEndToEnd(t *testing.T) {
	eng := baselineEngine(t)
	out, err := runOapi(t, eng, "cn-hangzhou",
		"ecs", "run-instances",
		"--biz-region-id", "cn-hangzhou",
		"--instance-name", "-1/-1",
		"--cli-dry-run",
	)
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "InstanceName: -1/-1") {
		t.Fatalf("dash value lost in:\n%s", out)
	}
}

func TestOapiMissingRequiredParam(t *testing.T) {
	eng := baselineEngine(t)
	_, err := runOapi(t, eng, "cn-hangzhou", "ecs", "run-instances", "--cli-dry-run")
	if err == nil || !strings.Contains(err.Error(), "biz-region-id") {
		t.Fatalf("expected missing biz-region-id error, got %v", err)
	}
}

func TestOapiUnknownCommand(t *testing.T) {
	eng := baselineEngine(t)
	_, err := runOapi(t, eng, "cn-hangzhou", "ecs", "definitely-not-a-real-command")
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

// TestUserMetaPluginOwnsProduct proves product-level exclusivity: a
// JSON meta plugin for a product that baseline does not ship is served
// entirely from the user plugins dir. There is no cross-layer merge —
// a product is either all-plugin or all-baseline.
func TestUserMetaPluginOwnsProduct(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "aliyun-cli-demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const manifestJSON = `{
  "name":"aliyun-cli-demo",
  "type":"meta",
  "productCode":"demo",
  "command":"demo",
  "apiVersions":{"default":"2024-01-01","supported":["2024-01-01"]},
  "metadata":{"format":"json","schema":"aliyun-openapi-meta","schemaVersion":1,"layout":"jsonl","layoutVersion":1,"index":"metadata.index.json","data":"metadata.jsonl"}
}`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	def := schema.CommandDefinition{
		Name: "DescribeThing", CmdName: "describe-thing",
		Operation: &schema.OperationConfig{
			Action: "DescribeThing", APIVersion: "2024-01-01", Method: "POST", APIStyle: "RPC", Protocol: "HTTPS",
		},
		Parameters: []schema.ArgumentDefinition{
			{Name: "region_id", RawName: "RegionId", Type: "string", Options: []string{"--region-id"}, Required: true, Location: "query"},
		},
	}
	raw, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	data := append(raw, '\n')
	if err := os.WriteFile(filepath.Join(pluginDir, schema.MetadataDataFile), data, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	index := map[string]any{
		"schema":        schema.SchemaName,
		"schemaVersion": schema.SchemaVersion,
		"layoutVersion": schema.LayoutVersion,
		"dataFile":      schema.MetadataDataFile,
		"dataSize":      int64(len(data)),
		"dataSha256":    "sha256:" + hex.EncodeToString(digest[:]),
		"apis": []map[string]any{{
			"apiVersion":    "2024-01-01",
			"apiName":       def.Name,
			"commandName":   def.CmdName,
			"descriptionEn": "Describes a thing",
			"offset":        int64(0),
			"length":        int64(len(raw)),
		}},
	}
	indexRaw, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, schema.MetadataIndexFile), indexRaw, 0o644); err != nil {
		t.Fatal(err)
	}

	eng := openapiruntime.NewEngine(openapiruntime.Options{
		BaselineFS:     bundledmeta.Metadatas,
		BundledBy:      "aliyun-cli test",
		UserPluginsDir: dir,
	}, nil)
	var help bytes.Buffer
	if err := eng.ProductHelp(engine.Request{Args: []string{"demo"}, Out: &help, Lang: "en"}); err != nil {
		t.Fatalf("user JSONL product help failed: %v", err)
	}
	if out := help.String(); !strings.Contains(out, "describe-thing") || !strings.Contains(out, "Describes a thing") {
		t.Fatalf("user JSONL product help did not use plugin index:\n%s", out)
	}

	out, err := runOapi(t, eng, "cn-hangzhou",
		"demo", "describe-thing", "--region-id", "cn-hangzhou",
		"--endpoint", "demo.cn-hangzhou.aliyuncs.com", "--cli-dry-run-json")
	if err != nil {
		t.Fatalf("user meta plugin command failed: %v\n%s", err, out)
	}
	var m dryRunOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("bad output: %v\n%s", err, out)
	}
	if m.Product != "demo" || m.API != "DescribeThing" || m.Action != "DescribeThing" || m.Version != "2024-01-01" {
		t.Fatalf("user plugin not routed: %+v", m)
	}

	// Baseline products remain reachable (different product code).
	out, err = runOapi(t, eng, "cn-hangzhou",
		"ecs", "run-instances",
		"--biz-region-id", "cn-hangzhou",
		"--instance-type", "ecs.g6.large",
		"--cli-dry-run-json")
	if err != nil {
		t.Fatalf("baseline product should still work: %v\n%s", err, out)
	}
}
