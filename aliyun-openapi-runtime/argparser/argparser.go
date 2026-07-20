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

// Package argparser turns a raw argv tail into a structured argument
// map, driven by a command's meta.Parameter schema.
//
// It exists because the main repo's cli.Parser is a forward-only state
// machine with no global flag view: it rejects values that begin with
// '-' (e.g. "-1/-1") and cannot express repeatable composite inputs
// like "--tags key1=v1 --tags key2=v2". aliyun-openapi-runtime commands are
// registered with KeepArgs=true so the whole tail is handed to this
// package verbatim, side-stepping those limitations entirely.
//
// Supported input forms (chosen to align with the plugin-common /
// aly argument model):
//
//	scalar          --image-cache-name foo
//	array<scalar>   --images a b c   |  --images a --images b
//	object          --network-config key=v host=1.2.3.4
//	map             --labels env=prod region=cn
//	array<object>   --tags key=k1 value=v1 --tags key=k2 value=v2
//
// Values may start with '-' because THIS tokenizer owns the split:
// a token is only treated as a flag when it starts with "--" (long)
// and matches a known option; everything else feeds the current flag.
package argparser

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// Reserved carries the well-known flags that steer the runtime rather
// than becoming API parameters. They are recognised regardless of the
// command schema so every aliyun-openapi-runtime command accepts them
// uniformly.
type Reserved struct {
	Region   string // --region: profile/wire region for signing + endpoint
	Endpoint string // --endpoint: explicit endpoint override
	Version  string // --api-version: API version override
	CliQuery string // --cli-query: jmespath expression applied to the response
	LogLevel string // --log-level: DEBUG|INFO|WARN|ERROR (and plugin aliases)

	// Dry-run has two flavours, matching the plugin-common
	// convention so users get the same UX across engines:
	//   --cli-dry-run       -> DryRun, human-readable request dump
	//   --cli-dry-run-json  -> DryRun + DryRunJSON, one-line meta JSON
	// --dry-run is kept as an ergonomic alias of --cli-dry-run.
	DryRun     bool
	DryRunJSON bool

	Help  bool // --help / -h
	Quiet bool // --quiet / -q

	Secure   bool // --secure: force HTTPS
	Insecure bool // --insecure: force HTTP

	Headers  []string // --header Name=Value (repeatable)
	Body     string   // --body raw string
	BodyFile string   // --body-file path

	// OutputTable is --output / -o with plugin object form
	// cols=... [rows=...] [num=...]. Absent => default pretty JSON.
	OutputTable *OutputTableConfig

	EstimateCost        bool     // --estimate-cost
	EstimateCostContext []string // --estimate-cost-context Key=Value (repeatable)

	NoStream bool // --no-stream (SSE only; ignored otherwise)

	// Pager / Waiter mirror aliyun-cli-runtime's global ObjectArg
	// helpers. Non-nil means the flag was present (even bare --pager).
	Pager  *PagerConfig
	Waiter *WaiterConfig
}

// OutputTableConfig is the plugin-style --output cols/rows/num object.
type OutputTableConfig struct {
	Cols    []string
	Rows    string
	ShowNum bool
}

// PagerConfig is the --pager / --all-pages object. Empty fields take
// the plugin-compatible defaults at execution time.
type PagerConfig struct {
	Path       string // JMESPath to the collection (e.g. "Data.CategoryList[]")
	PageNumber string // request/response field for page number
	PageSize   string
	TotalCount string
	NextToken  string
}

// WaiterConfig is the --waiter object. Expr and To are required;
// Timeout/Interval default to 180s / 5s when zero.
type WaiterConfig struct {
	Expr     string
	To       string
	Timeout  int // seconds; 0 -> default 180
	Interval int // seconds; 0 -> default 5
}

// reservedNames maps each reserved flag's long name to a small parse
// descriptor. takesValue=false means the flag is a boolean switch.
var reservedSpec = map[string]struct {
	takesValue bool
	apply      func(r *Reserved, v string)
}{
	"region":           {true, func(r *Reserved, v string) { r.Region = v }},
	"endpoint":         {true, func(r *Reserved, v string) { r.Endpoint = v }},
	"api-version":      {true, func(r *Reserved, v string) { r.Version = v }},
	"cli-query":        {true, func(r *Reserved, v string) { r.CliQuery = v }},
	"log-level":        {true, func(r *Reserved, v string) { r.LogLevel = v }},
	"body":             {true, func(r *Reserved, v string) { r.Body = v }},
	"body-file":        {true, func(r *Reserved, v string) { r.BodyFile = v }},
	"cli-dry-run":      {false, func(r *Reserved, _ string) { r.DryRun = true }},
	"cli-dry-run-json": {false, func(r *Reserved, _ string) { r.DryRun = true; r.DryRunJSON = true }},
	"dry-run":          {false, func(r *Reserved, _ string) { r.DryRun = true }},
	"help":             {false, func(r *Reserved, _ string) { r.Help = true }},
	"quiet":            {false, func(r *Reserved, _ string) { r.Quiet = true }},
	"secure":           {false, func(r *Reserved, _ string) { r.Secure = true }},
	"insecure":         {false, func(r *Reserved, _ string) { r.Insecure = true }},
	"estimate-cost":    {false, func(r *Reserved, _ string) { r.EstimateCost = true }},
	"no-stream":        {false, func(r *Reserved, _ string) { r.NoStream = true }},
}

// reservedObjectFlags are object-style reserved flags that consume a
// trail of key=value tokens (or nothing, for bare --pager).
var reservedObjectFlags = map[string]func(r *Reserved, kv map[string]string) error{
	"pager":     applyPager,
	"all-pages": applyPager, // alias of --pager
	"waiter":    applyWaiter,
}

var pagerFields = map[string]bool{
	"path": true, "PageNumber": true, "PageSize": true, "TotalCount": true, "NextToken": true,
}

var waiterFields = map[string]bool{
	"expr": true, "to": true, "timeout": true, "interval": true,
}

func applyPager(r *Reserved, kv map[string]string) error {
	for k := range kv {
		if !pagerFields[k] {
			return fmt.Errorf("--pager: unknown field %q (want path/PageNumber/PageSize/TotalCount/NextToken)", k)
		}
	}
	if r.Pager == nil {
		r.Pager = &PagerConfig{}
	}
	if v, ok := kv["path"]; ok {
		r.Pager.Path = v
	}
	if v, ok := kv["PageNumber"]; ok {
		r.Pager.PageNumber = v
	}
	if v, ok := kv["PageSize"]; ok {
		r.Pager.PageSize = v
	}
	if v, ok := kv["TotalCount"]; ok {
		r.Pager.TotalCount = v
	}
	if v, ok := kv["NextToken"]; ok {
		r.Pager.NextToken = v
	}
	return nil
}

func applyWaiter(r *Reserved, kv map[string]string) error {
	for k := range kv {
		if !waiterFields[k] {
			return fmt.Errorf("--waiter: unknown field %q (want expr/to/timeout/interval)", k)
		}
	}
	if r.Waiter == nil {
		r.Waiter = &WaiterConfig{}
	}
	if v, ok := kv["expr"]; ok {
		r.Waiter.Expr = v
	}
	if v, ok := kv["to"]; ok {
		r.Waiter.To = v
	}
	if v, ok := kv["timeout"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return fmt.Errorf("--waiter timeout: invalid value %q", v)
		}
		r.Waiter.Timeout = n
	}
	if v, ok := kv["interval"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return fmt.Errorf("--waiter interval: invalid value %q", v)
		}
		r.Waiter.Interval = n
	}
	return nil
}

func applyOutputTable(r *Reserved, kv map[string]string) error {
	for k := range kv {
		if k != "cols" && k != "rows" && k != "num" {
			return fmt.Errorf("--output: unknown field %q (want cols/rows/num)", k)
		}
	}
	cols, ok := kv["cols"]
	if !ok || cols == "" {
		return fmt.Errorf("--output object form requires cols=...")
	}
	cfg := &OutputTableConfig{Rows: kv["rows"]}
	for _, c := range strings.Split(cols, ",") {
		c = strings.TrimSpace(c)
		if c != "" {
			cfg.Cols = append(cfg.Cols, c)
		}
	}
	if n := kv["num"]; n == "true" || n == "1" {
		cfg.ShowNum = true
	}
	r.OutputTable = cfg
	return nil
}

// ReservedFlag describes one engine reserved (global) flag for help
// rendering. It is the public, ordered counterpart of the internal
// reservedSpec parsing table.
type ReservedFlag struct {
	Name       string // long name without the leading "--"
	TakesValue bool   // whether it consumes a value (vs boolean switch)
	DescZH     string
	DescEN     string
}

// Desc returns the localized description ("zh" -> Chinese, otherwise
// English).
func (f ReservedFlag) Desc(lang string) string {
	if lang == "zh" && f.DescZH != "" {
		return f.DescZH
	}
	if f.DescEN != "" {
		return f.DescEN
	}
	return f.DescZH
}

// reservedHelp is the ordered, documented list of engine reserved
// flags. Kept in a stable, user-facing order (most useful first). It
// mirrors reservedSpec; keep the two in sync when adding a flag.
var reservedHelp = []ReservedFlag{
	{"cli-dry-run", false, "组装请求但不发送，打印请求详情", "Assemble the request and print it without sending"},
	{"cli-dry-run-json", false, "以一行 JSON 打印调用元信息，不发送", "Print one-line invocation metadata as JSON without sending"},
	{"region", true, "指定调用的地域，用于签名与 endpoint 解析", "Region used for signing and endpoint resolution"},
	{"endpoint", true, "显式指定接入 endpoint", "Explicit endpoint override"},
	{"api-version", true, "覆盖 API 版本", "Override the API version"},
	{"cli-query", true, "对响应应用 jmespath 表达式过滤", "Apply a jmespath expression to the response"},
	{"log-level", true, "设置日志级别: DEBUG、INFO、WARN、ERROR(默认: ERROR)", "Set log level: DEBUG, INFO, WARN, ERROR (default: ERROR)"},
	{"quiet", false, "抑制正常输出", "Suppress normal output"},
	{"pager", true, "合并可分页 API 的多页结果（可用 --all-pages）；可选 path/PageNumber/PageSize/TotalCount/NextToken", "Merge pages for pageable APIs (alias --all-pages); optional path/PageNumber/PageSize/TotalCount/NextToken"},
}

// ReservedFlags returns the engine's reserved (global) flags for help
// rendering, in a stable user-facing order. The --help / -h, --dry-run
// alias, hidden --output / --waiter / --estimate-cost* are intentionally
// omitted (plugin parity).
func ReservedFlags() []ReservedFlag {
	out := make([]ReservedFlag, len(reservedHelp))
	copy(out, reservedHelp)
	return out
}

// Result bundles the parsed API arguments and the reserved flags.
type Result struct {
	// Args maps each provided parameter's wire RawName to its parsed
	// value (nested object keys are also RawName). Scalars are string /
	// json.Number; arrays are []any; objects/maps are map[string]any.
	Args map[string]any

	// Reserved holds the runtime-steering flags.
	Reserved Reserved
}

// Parse interprets args against the parameter schema. Unknown flags
// produce an error carrying a suggestion-friendly message; the caller
// (L6) decides whether to surface it or fall back to help.
func Parse(params []meta.Parameter, args []string) (*Result, error) {
	idx := newParamIndex(params)
	res := &Result{Args: map[string]any{}}

	i := 0
	for i < len(args) {
		tok := args[i]
		// Short aliases matching plugin globals.
		if tok == "-q" {
			res.Reserved.Quiet = true
			i++
			continue
		}
		if tok == "-h" {
			res.Reserved.Help = true
			i++
			continue
		}
		if tok == "-o" || strings.HasPrefix(tok, "-o=") {
			name, inlineVal, hasInline := "output", "", false
			if strings.HasPrefix(tok, "-o=") {
				inlineVal, hasInline = tok[3:], true
			}
			i++
			var occ []string
			if hasInline {
				occ = []string{inlineVal}
			} else {
				occ, i = takeValues(args, i)
			}
			if err := parseOutputFlag(&res.Reserved, occ); err != nil {
				return nil, err
			}
			_ = name
			continue
		}
		name, inlineVal, hasInline, isFlag := splitFlag(tok)
		if !isFlag {
			return nil, fmt.Errorf("unexpected positional argument %q", tok)
		}
		i++

		// Reserved flags win over schema flags: they use fixed
		// names that products are not allowed to shadow.
		if applyObj, ok := reservedObjectFlags[name]; ok {
			var occ []string
			if hasInline {
				occ = []string{inlineVal}
			} else {
				occ, i = takeValues(args, i)
			}
			kv, err := parseReservedObject(occ)
			if err != nil {
				return nil, fmt.Errorf("--%s: %w", name, err)
			}
			if err := applyObj(&res.Reserved, kv); err != nil {
				return nil, err
			}
			continue
		}
		if name == "output" {
			var occ []string
			if hasInline {
				occ = []string{inlineVal}
			} else {
				occ, i = takeValues(args, i)
			}
			if err := parseOutputFlag(&res.Reserved, occ); err != nil {
				return nil, err
			}
			continue
		}
		if name == "header" {
			var occ []string
			if hasInline {
				occ = []string{inlineVal}
			} else {
				occ, i = takeValues(args, i)
			}
			if len(occ) == 0 {
				return nil, fmt.Errorf("--header expects Name=Value")
			}
			res.Reserved.Headers = append(res.Reserved.Headers, occ...)
			continue
		}
		if name == "estimate-cost-context" {
			var occ []string
			if hasInline {
				occ = []string{inlineVal}
			} else {
				occ, i = takeValues(args, i)
			}
			if len(occ) == 0 {
				return nil, fmt.Errorf("--estimate-cost-context expects Key=Value")
			}
			res.Reserved.EstimateCostContext = append(res.Reserved.EstimateCostContext, occ...)
			continue
		}
		if spec, ok := reservedSpec[name]; ok {
			if !spec.takesValue {
				spec.apply(&res.Reserved, "")
				continue
			}
			val := inlineVal
			if !hasInline {
				val, i = takeOneValue(args, i)
			}
			spec.apply(&res.Reserved, val)
			continue
		}

		p := idx.lookup(name)
		if p == nil {
			return nil, &UnknownFlagError{Flag: name, Known: idx.optionNames()}
		}

		// Collect this occurrence's value tokens: everything up to
		// the next flag-looking token.
		var occ []string
		if hasInline {
			occ = []string{inlineVal}
		} else {
			occ, i = takeValues(args, i)
		}

		if err := assign(res.Args, p, occ); err != nil {
			return nil, err
		}
	}

	if len(res.Reserved.EstimateCostContext) > 0 && !res.Reserved.EstimateCost {
		return nil, fmt.Errorf("--estimate-cost-context requires --estimate-cost")
	}
	return res, nil
}

// parseOutputFlag accepts only the plugin object form:
// cols=... [rows=...] [num=...]. Default (absent --output) is pretty JSON.
func parseOutputFlag(r *Reserved, occ []string) error {
	if len(occ) == 0 {
		return fmt.Errorf("--output expects cols=... [rows=...] [num=...]")
	}
	kv, err := parseReservedObject(occ)
	if err != nil {
		return fmt.Errorf("--output: %w", err)
	}
	return applyOutputTable(r, kv)
}

// parseReservedObject turns object-flag value tokens into a flat
// key=value map. Empty tokens (bare --pager) yield an empty map.
func parseReservedObject(tokens []string) (map[string]string, error) {
	kv := map[string]string{}
	for _, t := range tokens {
		if t == "" {
			continue
		}
		k, v, ok := strings.Cut(t, "=")
		if !ok {
			return nil, fmt.Errorf("expected key=value, got %q", t)
		}
		if k == "" {
			return nil, fmt.Errorf("empty key in %q", t)
		}
		kv[k] = v
	}
	return kv, nil
}

// assign folds one flag occurrence's raw tokens into res under the
// parameter's WIRE key (RawName), honouring the parameter's composite
// shape and merging repeated occurrences. A parameter without a RawName
// in metadata is rejected: the args map is keyed strictly by RawName.
func assign(dst map[string]any, p *meta.Parameter, tokens []string) error {
	key, err := resolveWire(p)
	if err != nil {
		return err
	}
	switch p.Type {
	case meta.TypeArray:
		return assignArray(dst, key, p, tokens)
	case meta.TypeObject:
		return assignObject(dst, key, p, tokens)
	case meta.TypeMap:
		return assignMap(dst, key, p, tokens)
	default:
		return assignScalar(dst, key, p, tokens)
	}
}

func assignScalar(dst map[string]any, key string, p *meta.Parameter, tokens []string) error {
	if len(tokens) == 0 {
		return fmt.Errorf("--%s expects a value", displayName(p))
	}
	if len(tokens) > 1 {
		return fmt.Errorf("--%s expects a single value, got %d", displayName(p), len(tokens))
	}
	v, err := coerceScalar(p.Type, tokens[0])
	if err != nil {
		return fmt.Errorf("--%s: %w", displayName(p), err)
	}
	dst[key] = v
	return nil
}

func assignArray(dst map[string]any, key string, p *meta.Parameter, tokens []string) error {
	existing, _ := dst[key].([]any)

	// JSON-first for this occurrence (plugin parity): a JSON array is
	// expanded into multiple elements; a JSON object/scalar becomes a
	// single element. Field names inside are resolved to wire RawNames.
	if v, ok := tryFlagJSON(tokens); ok {
		if arr, isArr := v.([]any); isArr {
			for _, e := range arr {
				rv, err := resolveNames(p.ItemType, e)
				if err != nil {
					return fmt.Errorf("--%s: %w", displayName(p), err)
				}
				existing = append(existing, rv)
			}
		} else {
			rv, err := resolveNames(p.ItemType, v)
			if err != nil {
				return fmt.Errorf("--%s: %w", displayName(p), err)
			}
			existing = append(existing, rv)
		}
		dst[key] = existing
		return nil
	}

	elemObject := p.ItemType != nil && p.ItemType.Type == meta.TypeObject
	if elemObject {
		// Each occurrence is one object; its tokens are key=value
		// pairs (dotted keys / array indices allowed for nesting).
		// Field keys are addressed and emitted by their wire RawName.
		obj, err := parseKVPairs(tokens, p.ItemType.Fields)
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		dst[key] = append(existing, obj)
		return nil
	}

	// Array of scalars: append each token (optionally comma-split
	// for the single-token "a,b,c" convenience form).
	var elemType meta.DataType = meta.TypeString
	if p.ItemType != nil {
		elemType = p.ItemType.Type
	}
	for _, t := range tokens {
		parts := []string{t}
		if len(tokens) == 1 && strings.Contains(t, ",") {
			parts = strings.Split(t, ",")
		}
		for _, part := range parts {
			v, err := coerceScalar(elemType, part)
			if err != nil {
				return fmt.Errorf("--%s: %w", displayName(p), err)
			}
			existing = append(existing, v)
		}
	}
	dst[key] = existing
	return nil
}

func assignObject(dst map[string]any, key string, p *meta.Parameter, tokens []string) error {
	// JSON-first (plugin parity): "--cfg '{...}'" is parsed as JSON and
	// its field names resolved to wire RawNames; otherwise fall back to
	// the key=value form.
	if v, ok := tryFlagJSON(tokens); ok {
		m, isMap := v.(map[string]any)
		if !isMap {
			return fmt.Errorf("--%s: expected a JSON object", displayName(p))
		}
		rv, err := resolveNames(p, m)
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		mergeObject(dst, key, rv.(map[string]any))
		return nil
	}

	obj, err := parseKVPairs(tokens, p.Fields)
	if err != nil {
		return fmt.Errorf("--%s: %w", displayName(p), err)
	}
	mergeObject(dst, key, obj)
	return nil
}

func assignMap(dst map[string]any, key string, p *meta.Parameter, tokens []string) error {
	// JSON-first (plugin parity), then the flat key=value form. Keys are
	// free-form (no schema, no dotted nesting); values are coerced to
	// the map's declared ValueType.
	if v, ok := tryFlagJSON(tokens); ok {
		m, isMap := v.(map[string]any)
		if !isMap {
			return fmt.Errorf("--%s: expected a JSON object", displayName(p))
		}
		rv, err := resolveNames(p, m)
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		mergeObject(dst, key, rv.(map[string]any))
		return nil
	}

	existing, _ := dst[key].(map[string]any)
	if existing == nil {
		existing = map[string]any{}
	}
	vt := meta.TypeString
	if p.ValueType != nil {
		vt = p.ValueType.Type
	}
	for _, t := range tokens {
		k, v, ok := strings.Cut(t, "=")
		if !ok {
			return fmt.Errorf("--%s: expected key=value, got %q", displayName(p), t)
		}
		if k == "" {
			return fmt.Errorf("--%s: empty key in %q", displayName(p), t)
		}
		cv, err := coerceScalar(vt, v)
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		existing[k] = cv
	}
	dst[key] = existing
	return nil
}

// mergeObject stores obj under key, merging into an existing map from a
// prior occurrence of the same flag.
func mergeObject(dst map[string]any, key string, obj map[string]any) {
	if existing, ok := dst[key].(map[string]any); ok {
		for k, v := range obj {
			existing[k] = v
		}
		return
	}
	dst[key] = obj
}

// tryFlagJSON attempts to parse a flag occurrence's tokens as a single
// JSON value. Tokens are joined with spaces and one layer of matching
// outer quotes is stripped; only values starting with '{' or '[' are
// considered (mirroring the plugin's tryParseJSONString). Numbers are
// preserved as json.Number.
func tryFlagJSON(tokens []string) (any, bool) {
	s := stripOuterQuotes(strings.TrimSpace(strings.Join(tokens, " ")))
	if !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "[") {
		return nil, false
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, false
	}
	return v, true
}

// resolveNames recursively validates the keys of a decoded JSON value
// against the parameter schema: a key that matches a known field must
// address it by its wire RawName (a field lacking a RawName is a
// metadata error and is rejected). Unknown keys pass through verbatim
// (no format conversion). Value types (json.Number / bool / string) are
// left intact so explicit JSON input round-trips faithfully.
func resolveNames(p *meta.Parameter, v any) (any, error) {
	if p == nil {
		return v, nil
	}
	switch p.Type {
	case meta.TypeObject:
		m, ok := v.(map[string]any)
		if !ok {
			return v, nil
		}
		out := make(map[string]any, len(m))
		for k, val := range m {
			wk := k
			var fp *meta.Parameter
			if f := findField(p.Fields, k); f != nil {
				w, err := resolveWire(f)
				if err != nil {
					return nil, err
				}
				wk = w
				fp = f
			}
			rv, err := resolveNames(fp, val)
			if err != nil {
				return nil, err
			}
			out[wk] = rv
		}
		return out, nil
	case meta.TypeArray:
		a, ok := v.([]any)
		if !ok {
			return v, nil
		}
		out := make([]any, len(a))
		for i, e := range a {
			rv, err := resolveNames(p.ItemType, e)
			if err != nil {
				return nil, err
			}
			out[i] = rv
		}
		return out, nil
	case meta.TypeMap:
		m, ok := v.(map[string]any)
		if !ok {
			return v, nil
		}
		out := make(map[string]any, len(m))
		for k, val := range m {
			rv, err := resolveNames(p.ValueType, val)
			if err != nil {
				return nil, err
			}
			out[k] = rv
		}
		return out, nil
	default:
		return v, nil
	}
}

// parseKVPairs turns key=value tokens into a nested object, guided by
// the object's field schema. It mirrors aliyun-cli-runtime's ObjectArg
// and supports:
//
//	dotted nesting   meta.owner=alice
//	array indices    items[0]=v  |  items[0].key=v
//	JSON leaves      cfg='{"a":1}'  |  items[0]='{"k":"v"}'
//
// Field keys resolve to their wire RawName and leaf values are coerced
// to the field's declared type. Unknown keys pass through unchanged
// (string leaves) so forward-compatible payloads still work.
func parseKVPairs(tokens []string, fields []meta.Parameter) (map[string]any, error) {
	obj := map[string]any{}
	for _, t := range tokens {
		k, v, ok := strings.Cut(t, "=")
		if !ok {
			return nil, fmt.Errorf("expected key=value, got %q", t)
		}
		if k == "" {
			return nil, fmt.Errorf("empty key in %q", t)
		}
		if err := setSchemaValue(obj, k, v, fields); err != nil {
			return nil, err
		}
	}
	return obj, nil
}

// setSchemaValue assigns rawVal at keyPath within obj, walking the
// field schema to resolve wire names, coerce leaf types, descend nested
// objects and index arrays. Mirrors the plugin's ObjectArg.setNestedValue.
func setSchemaValue(obj map[string]any, keyPath, rawVal string, fields []meta.Parameter) error {
	firstKey, rest, isIndex := parseKeyPath(keyPath)
	if firstKey == "" {
		return fmt.Errorf("invalid key path %q", keyPath)
	}
	f := findField(fields, firstKey)
	wire := firstKey
	if f != nil {
		w, err := resolveWire(f)
		if err != nil {
			return err
		}
		wire = w
	}

	// Leaf assignment (no remaining path).
	if rest == "" {
		val, err := coerceLeaf(f, rawVal)
		if err != nil {
			return fmt.Errorf("%s: %w", keyPath, err)
		}
		obj[wire] = val
		return nil
	}

	// Unknown fields become free-form nested objects (string leaves).
	if f == nil {
		child, ok := obj[wire].(map[string]any)
		if !ok {
			child = map[string]any{}
			obj[wire] = child
		}
		return setSchemaValue(child, rest, rawVal, nil)
	}

	switch f.Type {
	case meta.TypeObject:
		if isIndex {
			return fmt.Errorf("field %q is an object, not an array", firstKey)
		}
		child, ok := obj[wire].(map[string]any)
		if !ok {
			child = map[string]any{}
			obj[wire] = child
		}
		return setSchemaValue(child, rest, rawVal, f.Fields)

	case meta.TypeArray:
		if !isIndex {
			return fmt.Errorf("array field %q needs an index: %s[0]=value or %s[0].key=value", firstKey, firstKey, firstKey)
		}
		idx, nextPath, err := splitIndex(rest)
		if err != nil {
			return fmt.Errorf("field %q: %w", firstKey, err)
		}
		arr, _ := obj[wire].([]any)
		if idx >= len(arr) {
			grown := make([]any, idx+1)
			copy(grown, arr)
			arr = grown
		}
		obj[wire] = arr

		elem := f.ItemType
		if elem != nil && elem.Type == meta.TypeObject {
			if nextPath == "" {
				m, err := decodeJSONObject(rawVal)
				if err != nil {
					return fmt.Errorf("%s[%d]: object element needs a field path (%s[%d].key=value) or a JSON object ('{...}'): %w", firstKey, idx, firstKey, idx, err)
				}
				rv, err := resolveNames(elem, m)
				if err != nil {
					return fmt.Errorf("%s[%d]: %w", firstKey, idx, err)
				}
				arr[idx] = rv
				return nil
			}
			child, ok := arr[idx].(map[string]any)
			if !ok {
				child = map[string]any{}
				arr[idx] = child
			}
			return setSchemaValue(child, nextPath, rawVal, elem.Fields)
		}
		if nextPath != "" {
			return fmt.Errorf("field %q: cannot descend %q into a scalar array element", firstKey, nextPath)
		}
		et := meta.TypeString
		if elem != nil {
			et = elem.Type
		}
		val, err := coerceScalar(et, rawVal)
		if err != nil {
			return fmt.Errorf("%s[%d]: %w", firstKey, idx, err)
		}
		arr[idx] = val
		return nil

	case meta.TypeMap:
		// Map keys are free-form; the remaining path segment is the key.
		child, ok := obj[wire].(map[string]any)
		if !ok {
			child = map[string]any{}
			obj[wire] = child
		}
		vt := meta.TypeString
		if f.ValueType != nil {
			vt = f.ValueType.Type
		}
		val, err := coerceScalar(vt, rawVal)
		if err != nil {
			return fmt.Errorf("%s: %w", keyPath, err)
		}
		child[rest] = val
		return nil

	default:
		return fmt.Errorf("field %q is a scalar; cannot descend %q", firstKey, rest)
	}
}

// coerceLeaf converts a leaf value against its field schema. Object /
// map / array leaves accept a JSON literal ('{...}' / '[...]'); scalars
// go through coerceScalar. A nil field (unknown key) is verbatim string.
func coerceLeaf(f *meta.Parameter, rawVal string) (any, error) {
	if f == nil {
		return rawVal, nil
	}
	switch f.Type {
	case meta.TypeObject, meta.TypeMap:
		m, err := decodeJSONObject(rawVal)
		if err != nil {
			return nil, err
		}
		return resolveNames(f, m)
	case meta.TypeArray:
		a, err := decodeJSONArray(rawVal)
		if err != nil {
			return nil, err
		}
		return resolveNames(f, a)
	default:
		return coerceScalar(f.Type, rawVal)
	}
}

// findField matches a sub-field key against its wire RawName exactly.
// No case folding and no kebab/snake conversion: nested keys are
// addressed and emitted by RawName only. Fields without a RawName are
// unreachable (and produce an error if resolveWire is called on them).
func findField(fields []meta.Parameter, key string) *meta.Parameter {
	for i := range fields {
		f := &fields[i]
		if f.RawName != "" && f.RawName == key {
			return f
		}
	}
	return nil
}

// resolveWire returns the parameter's RawName, or an error when metadata
// omitted it. Args keys (top-level and nested) are always RawName.
func resolveWire(p *meta.Parameter) (string, error) {
	if p == nil {
		return "", fmt.Errorf("nil parameter")
	}
	if p.RawName == "" {
		name := p.Name
		if name == "" {
			name = "<unnamed>"
		}
		return "", fmt.Errorf("parameter %q is missing raw_name in metadata", name)
	}
	return p.RawName, nil
}

// parseKeyPath splits the leading segment from a key path, reporting
// whether that segment indexes an array.
//
//	"key"          -> ("key", "", false)
//	"a.b"          -> ("a", "b", false)
//	"items[0]"     -> ("items", "[0]", true)
//	"items[0].key" -> ("items", "[0].key", true)
//	"a.items[0]"   -> ("a", "items[0]", false)
func parseKeyPath(keyPath string) (firstKey, rest string, isIndex bool) {
	dot := strings.Index(keyPath, ".")
	br := strings.Index(keyPath, "[")
	if dot != -1 && (br == -1 || dot < br) {
		return keyPath[:dot], keyPath[dot+1:], false
	}
	if br != -1 && (dot == -1 || br < dot) {
		return keyPath[:br], keyPath[br:], true
	}
	return keyPath, "", false
}

// splitIndex parses a leading "[n]" from rest, returning the index and
// the remaining path (with a leading dot trimmed).
func splitIndex(rest string) (idx int, nextPath string, err error) {
	if !strings.HasPrefix(rest, "[") {
		return 0, "", fmt.Errorf("expected [index], got %q", rest)
	}
	end := strings.Index(rest, "]")
	if end < 0 {
		return 0, "", fmt.Errorf("unterminated array index in %q", rest)
	}
	n, e := strconv.Atoi(rest[1:end])
	if e != nil || n < 0 {
		return 0, "", fmt.Errorf("invalid array index %q", rest[1:end])
	}
	return n, strings.TrimPrefix(rest[end+1:], "."), nil
}

// decodeJSONObject parses a '{...}' literal (optionally wrapped in
// matching quotes) into a map, preserving numbers as json.Number.
func decodeJSONObject(raw string) (map[string]any, error) {
	s := stripOuterQuotes(strings.TrimSpace(raw))
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return nil, fmt.Errorf("expected a JSON object, got %q", raw)
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("invalid JSON object: %w", err)
	}
	return m, nil
}

// decodeJSONArray parses a '[...]' literal (optionally quote-wrapped)
// into a slice, preserving numbers as json.Number.
func decodeJSONArray(raw string) ([]any, error) {
	s := stripOuterQuotes(strings.TrimSpace(raw))
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		return nil, fmt.Errorf("expected a JSON array, got %q", raw)
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var a []any
	if err := dec.Decode(&a); err != nil {
		return nil, fmt.Errorf("invalid JSON array: %w", err)
	}
	return a, nil
}

// stripOuterQuotes removes one layer of matching single/double quotes.
func stripOuterQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return strings.TrimSpace(s[1 : len(s)-1])
		}
	}
	return s
}

// coerceScalar converts a raw string into the typed value the wire
// layer expects. Numbers become json.Number to preserve int64/large
// precision end-to-end (see the precision contract in the module
// architecture doc).
//
// Booleans are deliberately kept as their raw string, NOT converted to
// a Go bool: aliyun-cli-runtime registers API-level boolean parameters
// as string args, so the wire form is the verbatim token ("true"). A
// Go bool would JSON-encode as an unquoted `true` in ROA/body payloads
// and diverge from the plugin (many APIs expect the quoted string).
// Only global/Reserved flags are true bools; those never pass here.
func coerceScalar(t meta.DataType, raw string) (any, error) {
	switch t {
	case meta.TypeInteger, meta.TypeLong, meta.TypeFloat:
		// Preserve exactly as typed; json.Number marshals without
		// quotes and never routes through float64.
		if !looksNumeric(raw) {
			return nil, fmt.Errorf("invalid number %q", raw)
		}
		return json.Number(raw), nil
	case meta.TypeAny:
		return parseAny(raw), nil
	default:
		return raw, nil
	}
}

// parseAny smart-parses an `any`-typed value, mirroring the plugin's
// AnyArg: JSON first (object / array / quoted string), then bool/null
// literals, then number (json.Number for precision), else raw string.
func parseAny(s string) any {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if isLikelyJSON(t) {
		dec := json.NewDecoder(strings.NewReader(t))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err == nil {
			return v
		}
	}
	switch strings.ToLower(t) {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if looksNumeric(t) {
		return json.Number(t)
	}
	return s
}

// isLikelyJSON reports whether s looks like a JSON object, array, or
// quoted string (a cheap gate before attempting a full decode).
func isLikelyJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	switch {
	case s[0] == '{' && s[len(s)-1] == '}':
		return true
	case s[0] == '[' && s[len(s)-1] == ']':
		return true
	case s[0] == '"' && s[len(s)-1] == '"':
		return true
	default:
		return false
	}
}

// ============================================================================
// tokenizer helpers
// ============================================================================

// splitFlag inspects one token. A flag is any token starting with
// "--". It may carry an inline value via "=". Returns isFlag=false for
// value tokens (including "-1", "-1/-1", bare words).
func splitFlag(tok string) (name, value string, hasInline, isFlag bool) {
	if !strings.HasPrefix(tok, "--") || tok == "--" {
		return "", "", false, false
	}
	body := tok[2:]
	if k, v, ok := strings.Cut(body, "="); ok {
		return k, v, true, true
	}
	return body, "", false, true
}

// isKnownShorthand reports whether tok is one of the recognised
// single-dash aliases (-h / -q / -o[=...]). These must stop value
// collection like a real flag (mirroring pflag), while other
// single-dash tokens (e.g. "-1", "-1/-1") remain values.
func isKnownShorthand(tok string) bool {
	return tok == "-h" || tok == "-q" || tok == "-o" || strings.HasPrefix(tok, "-o=")
}

// isFlagToken reports whether tok terminates a value run: a long "--"
// flag or a known shorthand.
func isFlagToken(tok string) bool {
	if _, _, _, isFlag := splitFlag(tok); isFlag {
		return true
	}
	return isKnownShorthand(tok)
}

// takeValues consumes consecutive non-flag tokens starting at i and
// returns them plus the new index. Used for flags that may take
// multiple value tokens in one occurrence.
func takeValues(args []string, i int) ([]string, int) {
	var out []string
	for i < len(args) {
		if isFlagToken(args[i]) {
			break
		}
		out = append(out, args[i])
		i++
	}
	return out, i
}

// takeOneValue consumes exactly one value token (for reserved
// value-flags). Returns "" if the next token is a flag.
func takeOneValue(args []string, i int) (string, int) {
	if i < len(args) {
		if !isFlagToken(args[i]) {
			return args[i], i + 1
		}
	}
	return "", i
}

func looksNumeric(s string) bool {
	if s == "" {
		return false
	}
	dot, e := false, false
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r == '-' || r == '+':
			if i != 0 && s[i-1] != 'e' && s[i-1] != 'E' {
				return false
			}
		case r == '.':
			if dot {
				return false
			}
			dot = true
		case r == 'e' || r == 'E':
			if e {
				return false
			}
			e = true
		default:
			return false
		}
	}
	return true
}

func displayName(p *meta.Parameter) string {
	if len(p.Options) > 0 {
		return strings.TrimPrefix(p.Options[0], "--")
	}
	return kebab(p.Name)
}
