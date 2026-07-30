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
	openapiruntime "github.com/aliyun/aliyun-openapi-runtime"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/aliyun/aliyun-openapi-runtime/format/indexed"
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
		"Available Commands:",
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

// dryRunMeta mirrors the engine's one-line --cli-dry-run-json shape.
type dryRunMeta struct {
	Product  string `json:"product"`
	Version  string `json:"version"`
	API      string `json:"api"`
	Region   string `json:"region,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
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
	if !strings.Contains(defaultHelp.String(), "list-api-versions") {
		t.Fatalf("default multi-version help should advertise list-api-versions:\n%s", defaultHelp.String())
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
	for _, want := range []string{"aliyun ecs describe-regions", "Global Parameters:"} {
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
	var m dryRunMeta
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("not dryRunMeta JSON: %v\n%s", err, out)
	}
	if m.Product != "ecs" || m.API != "RunInstances" || m.Version != "2014-05-26" {
		t.Errorf("meta identity wrong: %+v", m)
	}
	if m.Region != "cn-beijing" || m.Endpoint != "ecs.cn-beijing.aliyuncs.com" {
		t.Errorf("meta region/endpoint wrong: %+v", m)
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
	index := indexed.Index{
		Schema: schema.SchemaName, SchemaVersion: schema.SchemaVersion, LayoutVersion: schema.LayoutVersion,
		DataFile: schema.MetadataDataFile, DataSize: int64(len(data)), DataSHA256: "sha256:" + hex.EncodeToString(digest[:]),
		APIs: []indexed.Record{{
			APIVersion: "2024-01-01", APIName: def.Name, CommandName: def.CmdName,
			DescriptionEN: "Describes a thing", Offset: 0, Length: int64(len(raw)),
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
	var m dryRunMeta
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("bad output: %v\n%s", err, out)
	}
	if m.API != "DescribeThing" || m.Product != "demo" {
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
