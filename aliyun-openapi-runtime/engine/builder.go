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
	"github.com/aliyun/aliyun-openapi-runtime/redact"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
)

// Engine dispatches OpenAPI commands against a lazy Loader. It carries only
// immutable wiring/parser configuration; each call passes its own host and
// presentation state in Request.
//
// The loader is constructed on first use and memoised. Product ownership is then resolved on demand for only the product named by the request.
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
	// so a command that lives only in a non-default version is still
	// routed to the engine (the user can then select it via
	// --api-version).
	return ldr.CommandExists(product, command)
}

// Request carries one engine call. Shared by Dispatch and ProductHelp:
//
//	Dispatch:     Args = "<product> <command> [--flag ...]" (Host usually required)
//	ProductHelp:  Args = "<product> [--api-version ...]"    (Host unused)
type Request struct {
	Args []string
	Out  io.Writer
	// Lang selects help/description locale ("zh" / "en"). Empty => en.
	Lang string
	// Host supplies region + credentials. May be nil for help or dry-run
	// that needs neither (endpoint then resolves region-less).
	Host runtime.Host
}

func (e *Engine) HasProduct(product string) bool {
	ldr, err := e.getLoader()
	if err != nil {
		return false
	}
	return ldr.EnsureProduct(product) == nil
}

// ProductHelp renders the product's kebab-case command list from its selected metadata index.
// req.Args must start with the product code; optional --api-version selects a non-default version.
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

// Dispatch resolves and runs one command described by req.Args.
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
	productMeta := ldr.LookupProduct(product)
	if cmdName == listAPIVersionsCommand && productMeta != nil && len(productMeta.Versions) > 1 {
		res, err := argparser.ParseWithOptions(nil, args[2:], argparser.ParseOptions{
			ExternalFlags: e.externalFlags,
		})
		if err != nil {
			return err
		}
		if res.Reserved.Help {
			return printAPIVersionsHelp(req.Out, productMeta, req.Lang)
		}
		return printAPIVersions(req.Out, productMeta, req.Lang)
	}

	// Pre-scan the raw tail for --api-version so command resolution and metadata loading target the requested version (not just the product default).
	// Without this, an older version requested via --api-version would still resolve/execute against the default.
	reqVersion := scanAPIVersion(args[2:])

	ref, err := ldr.ResolveCommandVersion(product, cmdName, reqVersion)
	if err != nil {
		if errors.Is(err, loader.ErrCommandNotFound) {
			if versions := ldr.FindCommandVersions(product, cmdName); len(versions) > 0 {
				currentVersion, resolveErr := ldr.ResolveVersion(product, reqVersion)
				if resolveErr != nil {
					return resolveErr
				}
				return commandVersionError(product, cmdName, currentVersion, versions)
			}
			return fmt.Errorf("unknown command %q for product %q; try `aliyun %s` to list commands",
				cmdName, product, product)
		}
		return err
	}

	api, err := ldr.GetAPI(ref.Product, ref.Version, ref.Name)
	if err != nil {
		return fmt.Errorf("load api %s: %w", ref, err)
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

	// Client-side required-parameter check: fail fast with an
	// actionable message rather than an opaque server 400.
	if err := runtime.ValidateRequired(api, res.Args); err != nil {
		return fmt.Errorf("%w\nrun `aliyun %s %s --help` to see all parameters", err, product, cmdName)
	}

	ec := &runtime.ExecContext{
		API:      api,
		Args:     res.Args,
		Region:   res.Reserved.Region,
		Endpoint: res.Reserved.Endpoint,
		Version:  res.Reserved.Version,
		DryRun:   res.Reserved.DryRun,
	}
	if len(res.Reserved.Headers) > 0 {
		ec.ExtraHeaders = map[string]string{}
		for _, h := range res.Reserved.Headers {
			k, v, ok := strings.Cut(h, "=")
			if !ok || strings.TrimSpace(k) == "" {
				return fmt.Errorf("invalid header format %q, expected Name=Value", h)
			}
			ec.ExtraHeaders[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	if res.Reserved.BodyFile != "" {
		b, err := os.ReadFile(res.Reserved.BodyFile)
		if err != nil {
			return fmt.Errorf("--body-file: %w", err)
		}
		ec.RawBody = string(b)
	} else if res.Reserved.Body != "" {
		ec.RawBody = res.Reserved.Body
	}
	ec.ForceHTTPS = res.Reserved.Secure
	ec.ForceHTTP = res.Reserved.Insecure
	// --no-stream is accepted for plugin parity; with no SSE path yet it is a no-op.
	_ = res.Reserved.NoStream

	// Apply profile-derived wire settings from the host (timeouts,
	// retry, endpoint type, TLS/UA), mirroring what the Go plugin path
	// exports as env to a plugin process.
	if req.Host != nil {
		if ec.Region == "" {
			ec.Region = req.Host.Region()
		}
		s := req.Host.Settings()
		ec.ReadTimeout = s.ReadTimeout
		ec.ConnectTimeout = s.ConnectTimeout
		ec.RetryCount = s.RetryCount
		ec.UseVPC = s.UseVPC()
		ec.SkipSecureVerify = s.SkipSecureVerify
		ec.CLIVersion = s.CLIVersion
		ec.UserAgent = s.UserAgent
	}
	needsCred := !ec.DryRun || res.Reserved.EstimateCost
	if needsCred {
		if req.Host == nil {
			return errors.New("no credential source configured; run `aliyun configure` or pass --cli-dry-run")
		}
		cred, cerr := req.Host.Credential()
		if cerr != nil {
			return fmt.Errorf("resolve credential: %w", cerr)
		}
		ec.Credential = cred
	}

	// --cli-dry-run-json cannot combine with helpers.
	if res.Reserved.DryRunJSON {
		if res.Reserved.Pager != nil {
			return fmt.Errorf("--cli-dry-run-json cannot be used with --pager")
		}
		if res.Reserved.Waiter != nil {
			return fmt.Errorf("--cli-dry-run-json cannot be used with --waiter")
		}
		if res.Reserved.Quiet {
			return fmt.Errorf("--cli-dry-run-json cannot be used with --quiet")
		}
	}

	runtime.InitLogger(res.Reserved.LogLevel, res.Reserved.DryRun)
	runtime.LogArgs(res.Args)

	if runtime.PriceModeEnabled(res.Reserved.EstimateCost) {
		assembled, aerr := runtime.Assemble(ec)
		if aerr != nil {
			return aerr
		}
		// Plugin parity: --estimate-cost + API DryRun=true runs the
		// upstream dry-run first; DryRunOperation is success, then quote
		// without DryRun. --cli-dry-run (ec.DryRun) is unrelated and must
		// not suppress that network precheck.
		if runtime.IsAPIDryRunRequested(assembled) {
			callEC := *ec
			callEC.DryRun = false
			if _, callErr := e.executor.Execute(&callEC); callErr != nil && !runtime.IsDryRunPassError(callErr) {
				return callErr
			}
			runtime.StripAPIDryRun(assembled)
		}
		pc := map[string]string{}
		for _, item := range res.Reserved.EstimateCostContext {
			k, v, ok := strings.Cut(item, "=")
			if !ok || strings.TrimSpace(k) == "" {
				return fmt.Errorf("invalid --estimate-cost-context %q, expected Key=Value", item)
			}
			pc[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
		out, perr := runtime.EstimateCost(ec, assembled, pc)
		if perr != nil {
			return perr
		}
		// Ignore --quiet: the quote IS the output (plugin parity).
		fmt.Fprintln(req.Out, out)
		return nil
	}

	var resp *runtime.Response
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
		return renderDryRun(req.Out, ref.Product, resp.Assembled, res.Reserved.DryRunJSON)
	}
	if res.Reserved.Quiet {
		return nil
	}
	if res.Reserved.OutputTable != nil {
		return renderOutputTable(req.Out, resp.Parsed, resp.Raw, res.Reserved.OutputTable)
	}
	return renderResponse(req.Out, resp, res.Reserved.CliQuery)
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
		b, _ := json.Marshal(redact.MaskAny(req.Body))
		fmt.Fprintf(w, "Body:\n  %s\n", string(b))
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

func kebab(s string) string {
	out := make([]rune, 0, len(s)+4)
	for _, r := range s {
		if r == '_' {
			out = append(out, '-')
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
