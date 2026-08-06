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

// Package engine is the meta-driven command core: it turns a raw argv
// tail into a resolved API call and renders the result. It is
// deliberately free of any CLI-framework, config, or i18n dependency
// so the whole aliyun-openapi-runtime module can be published standalone.
//
// The embedding application (aliyun-cli) wraps Engine.Dispatch in its
// own cli.Command and supplies a runtime.Host for credentials;
// see openapi/runtimehost in the main module.
package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/jmespath/go-jmespath"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/redact"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
)

// Engine dispatches OpenAPI commands against a lazy Loader. It carries only immutable wiring/parser configuration;
// each call passes its own host and presentation state in Request.
type Engine struct {
	loaderFunc    func() (loader.Loader, error)
	executor      runtime.Executor
	externalFlags []argparser.ExternalFlagSpec

	once   sync.Once
	loader loader.Loader
	lodErr error
}

const listAPIVersionsCommand = "list-api-versions"

func NewEngine(loaderFunc func() (loader.Loader, error), executor runtime.Executor) *Engine {
	return NewEngineWithOptions(loaderFunc, executor, Options{})
}

// Options configures immutable parser extensions for an Engine.
type Options struct {
	ExternalFlags []argparser.ExternalFlagSpec
}

func NewEngineWithOptions(loaderFunc func() (loader.Loader, error), executor runtime.Executor, opts Options) *Engine {
	if executor == nil {
		executor = runtime.NewExecutor()
	}
	return &Engine{
		loaderFunc:    loaderFunc,
		executor:      executor,
		externalFlags: append([]argparser.ExternalFlagSpec(nil), opts.ExternalFlags...),
	}
}

func (e *Engine) getLoader() (loader.Loader, error) {
	e.once.Do(func() { e.loader, e.lodErr = e.loaderFunc() })
	return e.loader, e.lodErr
}

// Resolvable reports whether the engine can handle "<product> <command>" (i.e. it resolves to a known API in baseline or a user meta plugin).
// It is used by the host router to decide, for products WITHOUT an installed Go plugin, whether to route here or fall back to legacy handling.
// Resolves only the requested product on first call.
func (e *Engine) Resolvable(product, command string) bool {
	ldr, err := e.getLoader()
	if err != nil {
		return false
	}
	if err := ldr.EnsureProduct(product); err != nil {
		return false
	}
	if command == listAPIVersionsCommand {
		p := ldr.LookupProduct(product)
		return p != nil && len(p.Versions) > 1
	}
	// Check across all of the product's versions, not just the default,
	// so a command that lives only in a non-default version is still routed to the engine (the user can then select it via --api-version).
	return ldr.CommandExists(product, command)
}

// Request carries one engine call. Shared by Dispatch, ProductHelp, and APIHelp:
//
//	Dispatch:     Args = "<product> <command> [--flag ...]" (Host usually required)
//	ProductHelp:  Args = "<product> [--api-version ...]"    (Host unused)
//	APIHelp:      Args = "<product> <command> [--api-version ...]" (Host unused)
type Request struct {
	Args []string
	Out  io.Writer
	// Lang selects help/description locale ("zh" / "en"). Empty => en.
	Lang string
	// Host supplies region + credentials. May be nil for help or dry-run that needs neither (endpoint then resolves region-less).
	Host runtime.Host
}

func (e *Engine) HasProduct(product string) bool {
	ldr, err := e.getLoader()
	if err != nil {
		return false
	}
	return ldr.EnsureProduct(product) == nil
}

func (e *Engine) ProductHelp(req Request) error {
	ldr, err := e.getLoader()
	if err != nil {
		return fmt.Errorf("openapi-runtime loader: %w", err)
	}
	if len(req.Args) < 1 {
		return errors.New("product is required")
	}
	productCode := strings.ToLower(strings.TrimSpace(req.Args[0]))
	if productCode == "" {
		return errors.New("product is required")
	}
	if err := ldr.EnsureProduct(productCode); err != nil {
		return err
	}
	product := ldr.LookupProduct(productCode)
	if product == nil {
		return fmt.Errorf("unknown product %q", productCode)
	}
	requestedVersion := scanAPIVersion(req.Args[1:])
	version, err := ldr.ResolveVersion(productCode, requestedVersion)
	if err != nil {
		return err
	}
	index, err := ldr.GetAPIIndex(productCode, version)
	if err != nil {
		return fmt.Errorf("load product index %s@%s: %w", productCode, version, err)
	}
	return printProductHelp(req.Out, product, index, req.Lang, requestedVersion == "" && len(product.Versions) > 1)
}

func (e *Engine) APIHelp(req Request) error {
	ldr, err := e.getLoader()
	if err != nil {
		return fmt.Errorf("openapi-runtime loader: %w", err)
	}
	if len(req.Args) < 2 {
		return errors.New("expected <product> <command>")
	}
	product, command := req.Args[0], req.Args[1]
	if err := ldr.EnsureProduct(product); err != nil {
		return err
	}
	if command == listAPIVersionsCommand {
		productMeta := ldr.LookupProduct(product)
		if productMeta != nil && len(productMeta.Versions) > 1 {
			return printAPIVersionsHelp(req.Out, productMeta, req.Lang)
		}
	}
	_, api, err := resolveDispatchAPI(ldr, product, command, req.Args[2:])
	if err != nil {
		return err
	}
	return printAPIHelp(req.Out, product, api, req.Lang)
}

func (e *Engine) Dispatch(req Request) error {
	ldr, err := e.getLoader()
	if err != nil {
		return fmt.Errorf("openapi-runtime loader: %w", err)
	}

	args := req.Args
	if len(args) < 2 {
		return errors.New("expected <product> <command>")
	}
	product := args[0]
	if err := ldr.EnsureProduct(product); err != nil {
		return err
	}
	cmdName := args[1]
	// deal with internal commands
	if handled, err := e.dispatchBuiltin(req, ldr, product, cmdName); handled {
		return err
	}

	ref, api, err := resolveDispatchAPI(ldr, product, cmdName, args[2:])
	if err != nil {
		return err
	}

	res, err := argparser.ParseWithOptions(api.Parameters, args[2:], argparser.ParseOptions{
		ExternalFlags: e.externalFlags,
	})
	if err != nil {
		var ufe *argparser.UnknownFlagError
		if errors.As(err, &ufe) {
			return fmt.Errorf("%v (run `aliyun %s %s --help` for accepted flags)", err, product, cmdName)
		}
		return err
	}

	if res.Reserved.Help {
		return printAPIHelp(req.Out, product, api, req.Lang)
	}

	if err := runtime.ValidateRequired(api, res.Args, res.Reserved.BodySet || res.Reserved.BodyFileSet); err != nil {
		return fmt.Errorf("%w\nrun `aliyun %s %s --help` to see all parameters", err, product, cmdName)
	}

	ec, err := buildExecContext(req, api, res)
	if err != nil {
		return err
	}

	if err := validateDispatchOptions(res); err != nil {
		return err
	}

	runtime.InitLogger(res.Reserved.LogLevel, res.Reserved.DryRun)
	runtime.LogArgs(res.Args)

	if runtime.PriceModeEnabled(res.Reserved.EstimateCost) {
		return e.executeEstimateCost(req.Out, ec, res)
	}

	if api.IsSSE && !ec.DryRun {
		return e.executeSSE(req.Out, ec, res)
	}
	return e.executeStandard(req.Out, ref.Product, ec, res)
}

func (e *Engine) dispatchBuiltin(req Request, ldr loader.Loader, product, command string) (bool, error) {
	productMeta := ldr.LookupProduct(product)
	if command != listAPIVersionsCommand || productMeta == nil || len(productMeta.Versions) <= 1 {
		return false, nil
	}
	res, err := argparser.ParseWithOptions(nil, req.Args[2:], argparser.ParseOptions{
		ExternalFlags: e.externalFlags,
	})
	if err != nil {
		return true, err
	}
	if res.Reserved.Help {
		return true, printAPIVersionsHelp(req.Out, productMeta, req.Lang)
	}
	return true, printAPIVersions(req.Out, productMeta, req.Lang)
}

func resolveDispatchAPI(ldr loader.Loader, product, command string, tail []string) (meta.APIRef, *meta.API, error) {
	requestedVersion := scanAPIVersion(tail)
	ref, err := ldr.ResolveCommandVersion(product, command, requestedVersion)
	if err != nil {
		if errors.Is(err, loader.ErrCommandNotFound) {
			if versions := ldr.FindCommandVersions(product, command); len(versions) > 0 {
				currentVersion, resolveErr := ldr.ResolveVersion(product, requestedVersion)
				if resolveErr != nil {
					return meta.APIRef{}, nil, resolveErr
				}
				return meta.APIRef{}, nil, commandVersionError(product, command, currentVersion, versions)
			}
			return meta.APIRef{}, nil, fmt.Errorf(
				"unknown command %q for product %q; try `aliyun %s` to list commands",
				command, product, product,
			)
		}
		return meta.APIRef{}, nil, err
	}
	api, err := ldr.GetAPI(ref.Product, ref.Version, ref.Name)
	if err != nil {
		return meta.APIRef{}, nil, fmt.Errorf("load api %s: %w", ref, err)
	}
	return ref, api, nil
}

func buildExecContext(req Request, api *meta.API, res *argparser.Result) (*runtime.ExecContext, error) {
	ec := &runtime.ExecContext{
		API:        api,
		Args:       res.Args,
		Region:     res.Reserved.Region,
		Endpoint:   res.Reserved.Endpoint,
		Version:    res.Reserved.Version,
		DryRun:     res.Reserved.DryRun,
		ForceHTTPS: res.Reserved.Secure,
		ForceHTTP:  res.Reserved.Insecure,
	}
	if len(res.Reserved.Headers) > 0 {
		ec.ExtraHeaders = map[string]string{}
		for _, header := range res.Reserved.Headers {
			name, value, ok := strings.Cut(header, "=")
			if !ok || strings.TrimSpace(name) == "" {
				return nil, fmt.Errorf("invalid header format %q, expected Name=Value", header)
			}
			ec.ExtraHeaders[strings.TrimSpace(name)] = strings.TrimSpace(value)
		}
	}
	if res.Reserved.BodySet {
		ec.RawBody = res.Reserved.Body
	} else if res.Reserved.BodyFileSet {
		var body []byte
		var err error
		if res.Reserved.BodyFile == "-" {
			body, err = io.ReadAll(os.Stdin)
		} else {
			body, err = os.ReadFile(res.Reserved.BodyFile)
		}
		if err != nil {
			return nil, fmt.Errorf("--body-file: %w", err)
		}
		ec.RawBody = string(body)
	}

	if req.Host != nil {
		if ec.Region == "" {
			ec.Region = req.Host.Region()
		}
		settings := req.Host.Settings()
		ec.ReadTimeout = settings.ReadTimeout
		ec.ConnectTimeout = settings.ConnectTimeout
		ec.RetryCount = settings.RetryCount
		ec.UseVPC = settings.UseVPC()
		ec.SkipSecureVerify = settings.SkipSecureVerify
		ec.CLIVersion = settings.CLIVersion
		ec.UserAgent = settings.UserAgent
	}
	if !ec.DryRun || res.Reserved.EstimateCost {
		if req.Host == nil {
			return nil, errors.New("no credential source configured; run `aliyun configure` or pass --cli-dry-run")
		}
		credential, err := req.Host.Credential()
		if err != nil {
			return nil, fmt.Errorf("resolve credential: %w", err)
		}
		ec.Credential = credential
	}
	return ec, nil
}

func validateDispatchOptions(res *argparser.Result) error {
	if !res.Reserved.DryRunJSON {
		return nil
	}
	if res.Reserved.Pager != nil {
		return fmt.Errorf("--cli-dry-run-json cannot be used with --pager")
	}
	if res.Reserved.Waiter != nil {
		return fmt.Errorf("--cli-dry-run-json cannot be used with --waiter")
	}
	if res.Reserved.Quiet {
		return fmt.Errorf("--cli-dry-run-json cannot be used with --quiet")
	}
	return nil
}

func (e *Engine) executeEstimateCost(out io.Writer, ec *runtime.ExecContext, res *argparser.Result) error {
	assembled, err := runtime.Assemble(ec)
	if err != nil {
		return err
	}
	// API DryRun=true is an upstream validation call; CLI dry-run is unrelated.
	if runtime.IsAPIDryRunRequested(assembled) {
		callEC := *ec
		callEC.DryRun = false
		if _, callErr := e.executor.Execute(&callEC); callErr != nil && !runtime.IsDryRunPassError(callErr) {
			return callErr
		}
		runtime.StripAPIDryRun(assembled)
	}
	context := map[string]string{}
	for _, item := range res.Reserved.EstimateCostContext {
		name, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(name) == "" {
			return fmt.Errorf("invalid --estimate-cost-context %q, expected Key=Value", item)
		}
		context[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}
	quote, err := runtime.EstimateCost(ec, assembled, context)
	if err != nil {
		return err
	}
	// The quote is the result, so --quiet intentionally does not suppress it.
	_, err = fmt.Fprintln(out, quote)
	return err
}

func (e *Engine) executeSSE(out io.Writer, ec *runtime.ExecContext, res *argparser.Result) error {
	if res.Reserved.Pager != nil {
		return errors.New("--pager is not supported for SSE APIs")
	}
	if res.Reserved.Waiter != nil {
		return errors.New("--waiter is not supported for SSE APIs")
	}
	sseExecutor, ok := e.executor.(runtime.SSEExecutor)
	if !ok {
		return errors.New("configured executor does not support SSE APIs")
	}

	aggregate := res.Reserved.NoStream || res.Reserved.CliQuery != "" || res.Reserved.OutputTable != nil
	events := make([]runtime.SSEEvent, 0)
	var writeErr error
	err := sseExecutor.ExecuteSSE(ec, func(event runtime.SSEEvent) {
		if aggregate {
			events = append(events, append(runtime.SSEEvent(nil), event...))
			return
		}
		if !res.Reserved.Quiet && writeErr == nil {
			_, writeErr = fmt.Fprintln(out, string(event))
		}
	})
	if err != nil {
		return err
	}
	if writeErr != nil {
		return writeErr
	}
	if !aggregate || res.Reserved.Quiet || len(events) == 0 {
		return nil
	}
	resp, err := aggregateSSEResponse(events)
	if err != nil {
		return err
	}
	if res.Reserved.OutputTable != nil {
		return renderOutputTable(out, resp.Parsed, resp.Raw, res.Reserved.OutputTable)
	}
	return renderResponse(out, resp, res.Reserved.CliQuery)
}

func (e *Engine) executeStandard(out io.Writer, product string, ec *runtime.ExecContext, res *argparser.Result) error {
	var resp *runtime.Response
	var err error
	switch {
	case res.Reserved.Pager != nil:
		resp, err = runtime.CallWithPager(e.executor, ec, res.Reserved.Pager)
	case res.Reserved.Waiter != nil:
		resp, err = runtime.CallWithWaiter(e.executor, ec, res.Reserved.Waiter)
	default:
		resp, err = e.executor.Execute(ec)
	}
	if err != nil {
		return err
	}
	if res.Reserved.DryRun {
		return renderDryRun(out, product, resp.Assembled, res.Reserved.DryRunJSON)
	}
	if res.Reserved.Quiet {
		return nil
	}
	if res.Reserved.OutputTable != nil {
		return renderOutputTable(out, resp.Parsed, resp.Raw, res.Reserved.OutputTable)
	}
	return renderResponse(out, resp, res.Reserved.CliQuery)
}

func aggregateSSEResponse(events []runtime.SSEEvent) (*runtime.Response, error) {
	if len(events) == 0 {
		return &runtime.Response{}, nil
	}
	var raw []byte
	if len(events) == 1 {
		raw = append([]byte(nil), events[0]...)
	} else {
		values := make([]json.RawMessage, len(events))
		for i := range events {
			values[i] = json.RawMessage(events[i])
		}
		var err error
		raw, err = json.Marshal(values)
		if err != nil {
			return nil, err
		}
	}
	var parsed any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return nil, err
	}
	return &runtime.Response{Raw: raw, Parsed: parsed}, nil
}

func commandVersionError(product, command, currentVersion string, versions []string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "command %q is not available in API version %q for product %q", command, currentVersion, product)
	fmt.Fprintf(&b, "\navailable versions for this command: %s", strings.Join(versions, ", "))
	b.WriteString("\nuse an API version explicitly:")
	for _, version := range versions {
		fmt.Fprintf(&b, "\n  aliyun %s %s --api-version %s", product, command, version)
	}
	return errors.New(b.String())
}

// scanAPIVersion extracts the value of --api-version from a raw argv tail, supporting both "--api-version X" and "--api-version=X" forms.
func scanAPIVersion(args []string) string {
	const flag = "--api-version"
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == flag {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if strings.HasPrefix(a, flag+"=") {
			return a[len(flag)+1:]
		}
	}
	return ""
}

// ============================================================================
// dry-run rendering
// ============================================================================
type dryRunMeta struct {
	Product  string `json:"product"`
	Version  string `json:"version"`
	API      string `json:"api"`
	Region   string `json:"region,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

// renderDryRun prints the assembled request. jsonMeta selects the
// terse one-line metadata form (--cli-dry-run-json); otherwise a
// human-readable multi-line dump (--cli-dry-run) matching the plugin
// engine's layout.
func renderDryRun(w io.Writer, product string, req *runtime.AssembledRequest, jsonMeta bool) error {
	if req == nil {
		return fmt.Errorf("dry-run produced no request")
	}
	if jsonMeta {
		b, err := json.Marshal(dryRunMeta{
			Product:  product,
			Version:  req.Version,
			API:      req.Action,
			Region:   req.Region,
			Endpoint: stripScheme(req.Endpoint),
		})
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(b))
		return nil
	}

	const bar = "============================================================"
	fmt.Fprintf(w, "%s\nDRY-RUN MODE: Request Details (No actual API call)\n%s\n", bar, bar)
	fmt.Fprintf(w, "Method: %s\n", req.Method)
	fmt.Fprintf(w, "URL: %s\n", req.Pathname)
	if req.Endpoint != "" {
		fmt.Fprintf(w, "Endpoint: %s\n", req.Endpoint)
	}
	if req.Version != "" {
		fmt.Fprintf(w, "API Version: %s\n", req.Version)
	}
	if req.Action != "" {
		fmt.Fprintf(w, "API Action: %s\n", req.Action)
	}
	if req.Protocol != "" {
		fmt.Fprintf(w, "Protocol: %s\n", req.Protocol)
	}
	if req.Style != "" {
		fmt.Fprintf(w, "Style: %s\n", req.Style)
	}
	printSortedKV(w, "Headers", req.Headers)
	printSortedKV(w, "Query Parameters", req.Query)
	if req.Body != nil {
		switch body := req.Body.(type) {
		case string:
			// Raw --body/--body-file strings are sent as-is by the SDK.
			fmt.Fprintf(w, "Body:\n  %s\n", redact.MaskBody(body))
		case []byte:
			fmt.Fprintf(w, "Body:\n  %s\n", redact.MaskBody(string(body)))
		default:
			b, _ := json.Marshal(redact.MaskAny(req.Body))
			fmt.Fprintf(w, "Body:\n  %s\n", string(b))
		}
	}
	return nil
}

func printSortedKV(w io.Writer, title string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Fprintf(w, "%s:\n", title)
	for _, k := range keys {
		fmt.Fprintf(w, "  %s: %s\n", k, redact.MaskKV(k, m[k]))
	}
}

func stripScheme(ep string) string {
	ep = strings.TrimPrefix(strings.TrimPrefix(ep, "https://"), "http://")
	return strings.TrimSuffix(ep, "/")
}

// ============================================================================
// output helpers
// ============================================================================

func renderResponse(w io.Writer, resp *runtime.Response, filter string) error {
	// Start from the precision-preserving parsed value when available;
	// fall back to the raw bytes otherwise.
	var data any = resp.Parsed
	if data == nil && len(resp.Raw) > 0 {
		if err := json.Unmarshal(resp.Raw, &data); err != nil {
			// Not JSON: emit raw and stop.
			fmt.Fprintln(w, string(resp.Raw))
			return nil
		}
	}

	if filter != "" {
		out, err := jmespath.Search(filter, data)
		if err != nil {
			return fmt.Errorf("cli-query %q: %w", filter, err)
		}
		data = out
	}

	return writeJSON(w, data, resp.Raw, filter)
}

func writeJSON(w io.Writer, data any, raw []byte, filtered string) error {
	// Prefer re-indenting the original raw bytes when no filter was
	// applied, so key order and numeric precision survive. Fall back to
	// MarshalIndent on the parsed value (filtered results, or missing raw).
	if filtered == "" && len(raw) > 0 {
		var buf bytes.Buffer
		if err := json.Indent(&buf, raw, "", "\t"); err == nil {
			fmt.Fprintln(w, buf.String())
			return nil
		}
		// Not valid JSON: emit raw and stop.
		fmt.Fprintln(w, string(raw))
		return nil
	}
	b, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(b))
	return nil
}
