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

package engine

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

type responseExecutor struct {
	response *runtime.Response
	err      error
	calls    int
}

func (e *responseExecutor) Execute(ec *runtime.ExecContext) (*runtime.Response, error) {
	e.calls++
	if e.err != nil {
		return nil, e.err
	}
	if ec.DryRun {
		assembled, err := runtime.Assemble(ec)
		if err != nil {
			return nil, err
		}
		return &runtime.Response{Assembled: assembled}, nil
	}
	return e.response, nil
}

func newBuilderCoverageEngine(executor runtime.Executor) *Engine {
	return NewEngine(func() (loader.Loader, error) {
		return loader.New(dispatchErrorTestSource{}), nil
	}, executor)
}

type multiVersionSource struct{}

func (multiVersionSource) Kind() source.Kind { return source.KindBaseline }

func (multiVersionSource) LoadProduct(code string) (*meta.Product, *source.Provenance, error) {
	if code != "demo" {
		return nil, nil, source.ErrNotFound
	}
	return &meta.Product{Code: "demo", Versions: []string{"v1", "v2"}, DefaultVersion: "v2"}, &source.Provenance{}, nil
}

func (multiVersionSource) LoadAPIIndex(code, version string) (*meta.APIIndex, error) {
	if code != "demo" {
		return nil, source.ErrNotFound
	}
	index := &meta.APIIndex{ProductCode: code, Version: version, Entries: map[string]meta.APIIndexEntry{
		"RunThing": {APIName: "RunThing", CmdName: "run-thing"},
	}}
	if version == "v1" {
		index.Entries["LegacyThing"] = meta.APIIndexEntry{APIName: "LegacyThing", CmdName: "legacy-thing"}
	}
	index.BuildCmdIndex()
	return index, nil
}

func (multiVersionSource) LoadAPI(string, string, string) (*meta.API, error) {
	return nil, source.ErrNotFound
}

func newMultiVersionEngine() *Engine {
	return NewEngine(func() (loader.Loader, error) {
		return loader.New(multiVersionSource{}), nil
	}, &responseExecutor{})
}

func TestEngineDiscoveryAndHelpEntryPoints(t *testing.T) {
	engine := newBuilderCoverageEngine(&responseExecutor{})
	if !engine.HasProduct("demo") || engine.HasProduct("missing") {
		t.Fatal("HasProduct returned unexpected result")
	}
	if !engine.Resolvable("demo", "run-thing") || engine.Resolvable("demo", "missing") || engine.Resolvable("missing", "run-thing") {
		t.Fatal("Resolvable returned unexpected result")
	}

	if err := engine.ProductHelp(Request{}); err == nil {
		t.Fatal("ProductHelp without product succeeded")
	}
	if err := engine.ProductHelp(Request{Args: []string{" "}}); err == nil {
		t.Fatal("ProductHelp with empty product succeeded")
	}
	var productHelp bytes.Buffer
	if err := engine.ProductHelp(Request{Args: []string{"demo"}, Out: &productHelp}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(productHelp.String(), "run-thing") {
		t.Fatalf("ProductHelp output = %q", productHelp.String())
	}

	if err := engine.APIHelp(Request{Args: []string{"demo"}}); err == nil {
		t.Fatal("APIHelp without command succeeded")
	}
	var apiHelp bytes.Buffer
	if err := engine.APIHelp(Request{Args: []string{"demo", "run-thing"}, Out: &apiHelp}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(apiHelp.String(), "--instance-type") {
		t.Fatalf("APIHelp output = %q", apiHelp.String())
	}
	if !strings.Contains(apiHelp.String(), "Global Parameters:") ||
		!strings.Contains(apiHelp.String(), "--cli-dry-run") {
		t.Fatalf("default Action Help omitted Runtime global parameters: %q", apiHelp.String())
	}

	var aiHelp bytes.Buffer
	if err := engine.APIHelp(Request{
		Args: []string{"demo", "run-thing"}, Out: &aiHelp, AIMode: true,
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(aiHelp.String(), `"schemaVersion":"v1"`) ||
		!strings.Contains(aiHelp.String(), `"kind":"api"`) ||
		strings.Count(aiHelp.String(), "\n") != 1 {
		t.Fatalf("AI Help must force Runtime v1 JSON: %q", aiHelp.String())
	}
}

func TestEngineStructuredResponseHelp(t *testing.T) {
	engine := newBuilderCoverageEngine(&responseExecutor{})

	var requestOutput bytes.Buffer
	err := engine.Dispatch(Request{
		Args: []string{"demo", "run-thing", "--cli-section", "request"},
		Out:  &requestOutput,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Global Parameters:", "--cli-dry-run"} {
		if !strings.Contains(requestOutput.String(), want) {
			t.Fatalf("structured request Help missing %q: %s", want, requestOutput.String())
		}
	}
	var output bytes.Buffer
	err = engine.Dispatch(Request{
		Args: []string{
			"demo", "run-thing",
			"--cli-section", "response",
			"--help-search", "request id",
			"--cli-output", "json",
		},
		Out: &output, Lang: "zh",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{`"section": "response"`, `"RequestId"`, `"请求 ID"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("response Help missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, `"Unused"`) || strings.Contains(got, `"responses"`) {
		t.Fatalf("searched response Help must contain only projected output schema: %s", got)
	}

	if err := engine.Dispatch(Request{
		Args: []string{"demo", "run-thing", "--cli-output", "json"},
		Out:  io.Discard,
	}); err == nil || !strings.Contains(err.Error(), "only valid for Help") {
		t.Fatalf("execution --cli-output error = %v", err)
	}
	if err := engine.Dispatch(Request{
		Args: []string{"demo", "run-thing", "--help", "--help-all"},
		Out:  io.Discard,
	}); err == nil || !strings.Contains(err.Error(), "conflicts with --help") {
		t.Fatalf("conflicting Help operations error = %v", err)
	}
}

func TestEngineKebabParameterHelp(t *testing.T) {
	executor := &responseExecutor{}
	engine := newBuilderCoverageEngine(executor)
	var output bytes.Buffer
	err := engine.Dispatch(Request{
		Args: []string{
			"demo", "run-thing", "--instance-type", "--help",
			"--cli-output", "json",
		},
		Out: &output,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"kind": "parameter"`,
		`"name": "instance-type"`,
		`"rawName": "InstanceType"`,
		`"enum":`,
		`"ecs.g6"`,
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("parameter Help missing %q: %s", want, output.String())
		}
	}
	if executor.calls != 0 {
		t.Fatalf("parameter Help executed API %d time(s)", executor.calls)
	}

	output.Reset()
	err = engine.Dispatch(Request{
		Args: []string{
			"demo", "run-thing", "--instance-type",
			"--help-search", "nested field", "--cli-output", "json",
		},
		Out: &output,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"kind": "parameter"`,
		`"query": "nested field"`,
		`"shown": 0`,
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("parameter search Help missing %q: %s", want, output.String())
		}
	}
	if executor.calls != 0 {
		t.Fatalf("parameter search Help executed API %d time(s)", executor.calls)
	}

	err = engine.Dispatch(Request{
		Args: []string{
			"demo", "run-thing", "--instance-type", "--help",
			"--cli-section", "response",
		},
		Out: io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "only supports the request section") {
		t.Fatalf("parameter response Help error = %v", err)
	}
}

func TestProductCommandsAndUnknownCommandCandidatesCoverAllVersions(t *testing.T) {
	engine := newMultiVersionEngine()
	want := []string{"legacy-thing", "run-thing"}
	if got := engine.ProductCommands("demo"); !reflect.DeepEqual(got, want) {
		t.Fatalf("ProductCommands() = %v, want %v", got, want)
	}
	if got := engine.ProductCommands("missing"); got != nil {
		t.Fatalf("ProductCommands(missing) = %v, want nil", got)
	}

	err := engine.APIHelp(Request{
		Args: []string{"demo", "rn-thing", "--api-version", "v2"},
		Out:  io.Discard,
	})
	var unknown *UnknownCommandError
	if !errors.As(err, &unknown) {
		t.Fatalf("APIHelp error = %T %v, want UnknownCommandError", err, err)
	}
	if unknown.Product != "demo" || unknown.Command != "rn-thing" ||
		!reflect.DeepEqual(unknown.Candidates, want) {
		t.Fatalf("UnknownCommandError = %#v, want candidates %v", unknown, want)
	}
	if got, wantText := err.Error(), "unknown command \"rn-thing\" for product \"demo\"; try `aliyun demo` to list commands"; got != wantText {
		t.Fatalf("UnknownCommandError text = %q, want %q", got, wantText)
	}
}

func TestEngineLoaderFailures(t *testing.T) {
	cause := errors.New("loader failed")
	engine := NewEngine(func() (loader.Loader, error) { return nil, cause }, &responseExecutor{})
	if engine.HasProduct("demo") || engine.Resolvable("demo", "run-thing") {
		t.Fatal("loader failure should not be resolvable")
	}
	for name, err := range map[string]error{
		"product help": engine.ProductHelp(Request{Args: []string{"demo"}, Out: io.Discard}),
		"api help":     engine.APIHelp(Request{Args: []string{"demo", "run-thing"}, Out: io.Discard}),
		"dispatch":     engine.Dispatch(Request{Args: []string{"demo", "run-thing"}, Out: io.Discard}),
	} {
		if !errors.Is(err, cause) {
			t.Fatalf("%s error = %v, want loader cause", name, err)
		}
	}
}

func TestDispatchHelpDryRunAndStandardResponse(t *testing.T) {
	executor := &responseExecutor{response: &runtime.Response{
		StatusCode: 200,
		Raw:        []byte(`{"Result":{"Name":"ok"}}`),
		Parsed:     map[string]any{"Result": map[string]any{"Name": "ok"}},
	}}
	engine := newBuilderCoverageEngine(executor)
	if err := engine.Dispatch(Request{Args: []string{"demo"}, Out: io.Discard}); err == nil {
		t.Fatal("Dispatch without command succeeded")
	}

	var help bytes.Buffer
	if err := engine.Dispatch(Request{Args: []string{"demo", "run-thing", "--help"}, Out: &help}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(help.String(), "Usage:") {
		t.Fatalf("help output = %q", help.String())
	}

	var dryRun bytes.Buffer
	if err := engine.Dispatch(Request{
		Args: []string{"demo", "run-thing", "--instance-type", "ecs.g6", "--cli-dry-run-json"}, Out: &dryRun,
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dryRun.String(), `"action":"RunThing"`) {
		t.Fatalf("dry-run output = %q", dryRun.String())
	}

	var standard bytes.Buffer
	if err := engine.Dispatch(Request{
		Args: []string{"demo", "run-thing", "--instance-type", "ecs.g6"}, Out: &standard, Host: runtime.StaticHost{},
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(standard.String(), `"Name": "ok"`) {
		t.Fatalf("standard output = %q", standard.String())
	}

	standard.Reset()
	if err := engine.Dispatch(Request{
		Args: []string{"demo", "run-thing", "--instance-type", "ecs.g6", "--quiet"}, Out: &standard, Host: runtime.StaticHost{},
	}); err != nil {
		t.Fatal(err)
	}
	if standard.Len() != 0 {
		t.Fatalf("quiet output = %q", standard.String())
	}
}

func TestDispatchListAPIVersionsBuiltin(t *testing.T) {
	engine := newMultiVersionEngine()
	if !engine.Resolvable("demo", listAPIVersionsCommand) {
		t.Fatal("list-api-versions should be resolvable for multi-version product")
	}
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"demo", listAPIVersionsCommand}, "Supported API versions"},
		{[]string{"demo", listAPIVersionsCommand, "--help"}, "Description: List supported API versions"},
	} {
		var out bytes.Buffer
		if err := engine.Dispatch(Request{Args: test.args, Out: &out}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), test.want) {
			t.Fatalf("builtin output = %q", out.String())
		}
	}
	if err := engine.Dispatch(Request{Args: []string{"demo", listAPIVersionsCommand, "--unknown"}, Out: io.Discard}); err == nil {
		t.Fatal("builtin accepted unknown flag")
	}
}

func TestBuildExecContextHeadersBodyFileAndHost(t *testing.T) {
	bodyFile := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(bodyFile, []byte(`{"from":"file"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	host := runtime.StaticHost{
		RegionID: "cn-test",
		SettingsVal: runtime.Settings{
			ReadTimeout: time.Second, ConnectTimeout: 2 * time.Second, RetryCount: 3,
			EndpointType: "vpc", SkipSecureVerify: true, CLIVersion: "3.0.0", UserAgent: "test-agent",
		},
		TransportOptionsVal: runtime.TransportOptions{Headers: map[string]string{"x-host": "value"}},
	}
	result := &argparser.Result{Reserved: argparser.Reserved{
		Headers: []string{" X-Test = value ", "Empty="}, BodyFile: bodyFile, BodyFileSet: true, DryRun: true,
	}}
	ec, err := buildExecContext(Request{Host: host}, &meta.API{}, result)
	if err != nil {
		t.Fatal(err)
	}
	if ec.Region != "cn-test" || ec.ExtraHeaders["X-Test"] != "value" || ec.RawBody != `{"from":"file"}` || !ec.UseVPC || ec.RetryCount != 3 || ec.CLIVersion != "3.0.0" {
		t.Fatalf("ExecContext = %#v", ec)
	}

	for _, header := range []string{"missing-separator", " =value"} {
		_, err := buildExecContext(Request{}, &meta.API{}, &argparser.Result{Reserved: argparser.Reserved{Headers: []string{header}, DryRun: true}})
		var usage *UsageError
		if !errors.As(err, &usage) || usage.Code != "INVALID_HEADER" {
			t.Fatalf("header %q error = %v", header, err)
		}
	}
	_, err = buildExecContext(Request{}, &meta.API{}, &argparser.Result{Reserved: argparser.Reserved{BodyFile: filepath.Join(t.TempDir(), "missing"), BodyFileSet: true, DryRun: true}})
	var usage *UsageError
	if !errors.As(err, &usage) || usage.Code != "INVALID_BODY_FILE" {
		t.Fatalf("missing body file error = %v", err)
	}
	_, err = buildExecContext(Request{}, &meta.API{}, &argparser.Result{})
	var credential *CredentialError
	if !errors.As(err, &credential) {
		t.Fatalf("missing host error = %v", err)
	}
	credentialCause := errors.New("credential failed")
	_, err = buildExecContext(Request{Host: runtime.StaticHost{CredErr: credentialCause}}, &meta.API{}, &argparser.Result{})
	if !errors.Is(err, credentialCause) {
		t.Fatalf("credential resolution error = %v", err)
	}
}

func TestBuilderOptionAndRenderingHelpers(t *testing.T) {
	conflicts := []argparser.Reserved{
		{DryRunJSON: true, Pager: &argparser.PagerConfig{}},
		{DryRunJSON: true, Waiter: &argparser.WaiterConfig{}},
		{DryRunJSON: true, Quiet: true},
	}
	for _, reserved := range conflicts {
		if err := validateDispatchOptions(&argparser.Result{Reserved: reserved}); err == nil {
			t.Fatalf("validateDispatchOptions(%#v) succeeded", reserved)
		}
	}
	if err := validateDispatchOptions(&argparser.Result{}); err != nil {
		t.Fatal(err)
	}

	applyMetadataPluginProvenance(nil, &source.Provenance{})
	applyMetadataPluginProvenance(&runtime.ExecContext{}, nil)
	if got := scanAPIVersion([]string{"--other", "x", "--api-version=v2"}); got != "v2" {
		t.Fatalf("scanAPIVersion(equals) = %q", got)
	}
	if got := scanAPIVersion([]string{"--api-version", "v1"}); got != "v1" {
		t.Fatalf("scanAPIVersion(pair) = %q", got)
	}
	if got := scanAPIVersion([]string{"--api-version"}); got != "" {
		t.Fatalf("scanAPIVersion(missing) = %q", got)
	}
	if got := commandVersionError("demo", "run", "v3", []string{"v1", "v2"}).Error(); !strings.Contains(got, "--api-version v1") {
		t.Fatalf("commandVersionError() = %q", got)
	}

	if response, err := aggregateSSEResponse(nil); err != nil || len(response.Raw) != 0 {
		t.Fatalf("aggregateSSEResponse(nil) = %#v, %v", response, err)
	}
	if response, err := aggregateSSEResponse([]runtime.SSEEvent{[]byte(`{"one":1}`)}); err != nil || !strings.Contains(string(response.Raw), "one") {
		t.Fatalf("aggregateSSEResponse(single) = %#v, %v", response, err)
	}
	if _, err := aggregateSSEResponse([]runtime.SSEEvent{[]byte("bad")}); err == nil {
		t.Fatal("aggregateSSEResponse accepted invalid JSON")
	}

	bodies := []struct {
		body    any
		format  string
		wantFmt string
	}{
		{nil, "", ""},
		{"text", "", "raw"},
		{"a=1", "formData", "form"},
		{[]byte("bytes"), "", "binary"},
		{[]byte("a=1"), "formData", "form"},
		{map[string]any{"name": "value"}, "", "json"},
		{map[string]any{"name": "value"}, "formData", "form"},
	}
	for _, test := range bodies {
		_, format, err := dryRunBody(test.body, test.format)
		if err != nil || format != test.wantFmt {
			t.Fatalf("dryRunBody(%#v, %q) format=%q err=%v", test.body, test.format, format, err)
		}
	}
	if _, _, err := dryRunBody(make(chan int), ""); err == nil {
		t.Fatal("dryRunBody accepted channel")
	}
}

func TestExecuteEstimateCostValidationPaths(t *testing.T) {
	engine := newBuilderCoverageEngine(&responseExecutor{})
	if err := engine.executeEstimateCost(io.Discard, &runtime.ExecContext{}, &argparser.Result{}); err == nil {
		t.Fatal("executeEstimateCost with nil API succeeded")
	}

	api := &meta.API{
		Name: "Quote", Version: "v1", Style: meta.StyleRPC, ProductCode: "demo",
		Parameters: []meta.Parameter{{Name: "dry_run", RawName: "DryRun", Type: meta.TypeBoolean, Position: meta.PosQuery}},
	}
	executorFailure := errors.New("precheck failed")
	failingEngine := newBuilderCoverageEngine(&responseExecutor{err: executorFailure})
	ec := &runtime.ExecContext{API: api, Args: map[string]any{"DryRun": true}}
	if err := failingEngine.executeEstimateCost(io.Discard, ec, &argparser.Result{}); !errors.Is(err, executorFailure) {
		t.Fatalf("precheck error = %v", err)
	}

	passEngine := newBuilderCoverageEngine(&responseExecutor{err: errors.New("Code: DryRunOperation")})
	result := &argparser.Result{Reserved: argparser.Reserved{EstimateCostContext: []string{"invalid"}}}
	if err := passEngine.executeEstimateCost(io.Discard, ec, result); err == nil || !strings.Contains(err.Error(), "expected Key=Value") {
		t.Fatalf("invalid estimate context error = %v", err)
	}

	ec = &runtime.ExecContext{API: api, Args: map[string]any{}}
	result = &argparser.Result{Reserved: argparser.Reserved{EstimateCostContext: []string{" Zone = value "}}}
	if err := engine.executeEstimateCost(io.Discard, ec, result); err == nil || !strings.Contains(err.Error(), "resolved credentials") {
		t.Fatalf("missing pricing credential error = %v", err)
	}
}

func TestBuilderErrorAndRenderBranches(t *testing.T) {
	engine := newBuilderCoverageEngine(&responseExecutor{})
	if err := engine.ProductHelp(Request{Args: []string{"missing"}, Out: io.Discard}); err == nil {
		t.Fatal("ProductHelp(unknown) succeeded")
	}
	if err := engine.APIHelp(Request{Args: []string{"demo", "missing"}, Out: io.Discard}); err == nil {
		t.Fatal("APIHelp(unknown command) succeeded")
	}
	if err := engine.Dispatch(Request{Args: []string{"missing", "run"}, Out: io.Discard}); err == nil {
		t.Fatal("Dispatch(unknown product) succeeded")
	}
	if err := engine.Dispatch(Request{Args: []string{"demo", "run-thing", "--instance-type"}, Out: io.Discard}); err == nil {
		t.Fatal("Dispatch(missing flag value) succeeded")
	}
	if err := engine.Dispatch(Request{
		Args: []string{"demo", "run-thing", "--instance-type", "ecs.g6", "--cli-dry-run-json", "--quiet"}, Out: io.Discard,
	}); err == nil || !strings.Contains(err.Error(), "cannot be used with --quiet") {
		t.Fatalf("Dispatch(conflicting options) error = %v", err)
	}
	if err := engine.Dispatch(Request{
		Args: []string{"demo", "run-thing", "--instance-type", "ecs.g6", "--estimate-cost"}, Out: io.Discard, Host: runtime.StaticHost{},
	}); err == nil || !strings.Contains(err.Error(), "resolved credentials") {
		t.Fatalf("Dispatch(estimate cost) error = %v", err)
	}

	multi := newMultiVersionEngine()
	var versionsHelp bytes.Buffer
	if err := multi.APIHelp(Request{Args: []string{"demo", listAPIVersionsCommand}, Out: &versionsHelp}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(versionsHelp.String(), "List supported API versions") {
		t.Fatalf("API versions help = %q", versionsHelp.String())
	}

	if err := renderDryRun(io.Discard, "demo", nil, false); err == nil {
		t.Fatal("renderDryRun(nil) succeeded")
	}
	if _, err := buildCliDryRunOutput("demo", &runtime.AssembledRequest{Body: make(chan int)}); err == nil {
		t.Fatal("buildCliDryRunOutput accepted channel body")
	}
	for _, body := range []any{[]byte(`{"password":"secret"}`), map[string]any{"name": "value"}} {
		var out bytes.Buffer
		if err := renderDryRun(&out, "demo", &runtime.AssembledRequest{Body: body}, false); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "Body:") {
			t.Fatalf("renderDryRun(%T) = %q", body, out.String())
		}
	}
	printSortedKV(io.Discard, "Empty", nil)

	var out bytes.Buffer
	if err := renderResponse(&out, &runtime.Response{Raw: []byte("not-json")}, "", false); err != nil || !strings.Contains(out.String(), "not-json") {
		t.Fatalf("renderResponse(raw) = %q, %v", out.String(), err)
	}
	if err := renderResponse(io.Discard, &runtime.Response{Parsed: map[string]any{}}, "[", false); err == nil {
		t.Fatal("renderResponse accepted invalid JMESPath")
	}
	if err := writeJSON(io.Discard, make(chan int), nil, "filtered", false); err == nil {
		t.Fatal("writeJSON accepted channel")
	}
}
