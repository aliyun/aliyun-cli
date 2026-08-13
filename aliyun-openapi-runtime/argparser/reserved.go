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

package argparser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Reserved carries the well-known flags that steer the runtime rather than becoming API parameters.
// They are recognised regardless of the command schema so every aliyun-openapi-runtime command accepts them uniformly.
type Reserved struct {
	Region   string // --region: profile/wire region for signing + endpoint
	Endpoint string // --endpoint: explicit endpoint override
	Version  string // --api-version: API version override
	CliQuery string // --cli-query: jmespath expression applied to the response
	LogLevel string // --log-level: DEBUG|INFO|WARN|ERROR (and plugin aliases)

	// Dry-run has two flavours, matching the plugin-common convention so users get the same UX across engines:
	//   --cli-dry-run       -> DryRun, human-readable request dump
	//   --cli-dry-run-json  -> DryRun + DryRunJSON, one-line full request JSON
	// Do NOT reserve --dry-run: that name belongs to API params
	// (e.g. DryRun -> --dry-run true|false). CLI preflight uses
	// --cli-dry-run / --cli-dry-run-json only.
	DryRun     bool
	DryRunJSON bool

	Help  bool // --help / -h
	Quiet bool // --quiet / -q

	Secure   bool // --secure: force HTTPS
	Insecure bool // --insecure: force HTTP

	Headers     []string // --header Name=Value (repeatable)
	Body        string   // --body raw string
	BodyFile    string   // --body-file path
	BodySet     bool     // distinguishes --body '' from an absent escape hatch
	BodyFileSet bool     // distinguishes an explicitly supplied --body-file

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

// PagerConfig is the --pager / --all-pages object. Empty fields take the plugin-compatible defaults at execution time.
type PagerConfig struct {
	Path       string // JMESPath to the collection (e.g. "Data.CategoryList[]")
	PageNumber string // request/response field for page number
	PageSize   string
	TotalCount string
	NextToken  string
}

// WaiterConfig is the --waiter object.
// Expr and To are required; Timeout/Interval default to 180s / 5s when zero.
type WaiterConfig struct {
	Expr     string
	To       string
	Timeout  int // seconds; 0 -> default 180
	Interval int // seconds; 0 -> default 5
}

type reservedValueMode uint8

const (
	reservedNoValue reservedValueMode = iota
	reservedSingleValue
	reservedMultiValue
)

type reservedFlagSpec struct {
	name      string
	mode      reservedValueMode
	shorthand rune
	visible   bool
	helpOrder int
	descZH    string
	descEN    string
	apply     func(r *Reserved, values []string) error
}

var regionFlagSpec = reservedFlagSpec{
	name:      "region",
	mode:      reservedSingleValue,
	visible:   true,
	helpOrder: 3,
	descZH:    "指定调用的地域，用于签名与 endpoint 解析",
	descEN:    "Region used for signing and endpoint resolution",
	apply: func(r *Reserved, values []string) error {
		r.Region = values[0]
		return nil
	},
}

var endpointFlagSpec = reservedFlagSpec{
	name:      "endpoint",
	mode:      reservedSingleValue,
	visible:   true,
	helpOrder: 4,
	descZH:    "显式指定接入 endpoint",
	descEN:    "Explicit endpoint override",
	apply: func(r *Reserved, values []string) error {
		r.Endpoint = values[0]
		return nil
	},
}

var apiVersionFlagSpec = reservedFlagSpec{
	name:      "api-version",
	mode:      reservedSingleValue,
	visible:   true,
	helpOrder: 5,
	descZH:    "覆盖 API 版本",
	descEN:    "Override the API version",
	apply: func(r *Reserved, values []string) error {
		r.Version = values[0]
		return nil
	},
}

var cliQueryFlagSpec = reservedFlagSpec{
	name:      "cli-query",
	mode:      reservedSingleValue,
	visible:   true,
	helpOrder: 6,
	descZH:    "对响应应用 jmespath 表达式过滤",
	descEN:    "Apply a jmespath expression to the response",
	apply: func(r *Reserved, values []string) error {
		r.CliQuery = values[0]
		return nil
	},
}

var logLevelFlagSpec = reservedFlagSpec{
	name:      "log-level",
	mode:      reservedSingleValue,
	visible:   true,
	helpOrder: 7,
	descZH:    "设置日志级别: DEBUG、INFO、WARN、ERROR(默认: ERROR)",
	descEN:    "Set log level: DEBUG, INFO, WARN, ERROR (default: ERROR)",
	apply: func(r *Reserved, values []string) error {
		r.LogLevel = values[0]
		return nil
	},
}

var bodyFlagSpec = reservedFlagSpec{
	name: "body",
	mode: reservedSingleValue,
	apply: func(r *Reserved, values []string) error {
		r.Body = values[0]
		r.BodySet = true
		return nil
	},
}

var bodyFileFlagSpec = reservedFlagSpec{
	name: "body-file",
	mode: reservedSingleValue,
	apply: func(r *Reserved, values []string) error {
		r.BodyFile = values[0]
		r.BodyFileSet = true
		return nil
	},
}

var cliDryRunFlagSpec = reservedFlagSpec{
	name:      "cli-dry-run",
	mode:      reservedNoValue,
	visible:   true,
	helpOrder: 1,
	descZH:    "组装请求但不发送，打印请求详情",
	descEN:    "Assemble the request and print it without sending",
	apply: func(r *Reserved, _ []string) error {
		r.DryRun = true
		return nil
	},
}

var cliDryRunJSONFlagSpec = reservedFlagSpec{
	name: "cli-dry-run-json",
	mode: reservedNoValue,
	apply: func(r *Reserved, _ []string) error {
		r.DryRun = true
		r.DryRunJSON = true
		return nil
	},
}

var helpFlagSpec = reservedFlagSpec{
	name:      "help",
	shorthand: 'h',
	mode:      reservedNoValue,
	apply: func(r *Reserved, _ []string) error {
		r.Help = true
		return nil
	},
}

var quietFlagSpec = reservedFlagSpec{
	name:      "quiet",
	shorthand: 'q',
	mode:      reservedNoValue,
	visible:   true,
	helpOrder: 8,
	descZH:    "抑制正常输出",
	descEN:    "Suppress normal output",
	apply: func(r *Reserved, _ []string) error {
		r.Quiet = true
		return nil
	},
}

var secureFlagSpec = reservedFlagSpec{
	name: "secure",
	mode: reservedNoValue,
	apply: func(r *Reserved, _ []string) error {
		r.Secure = true
		return nil
	},
}

var insecureFlagSpec = reservedFlagSpec{
	name: "insecure",
	mode: reservedNoValue,
	apply: func(r *Reserved, _ []string) error {
		r.Insecure = true
		return nil
	},
}

var estimateCostFlagSpec = reservedFlagSpec{
	name: "estimate-cost",
	mode: reservedNoValue,
	apply: func(r *Reserved, _ []string) error {
		r.EstimateCost = true
		return nil
	},
}

var noStreamFlagSpec = reservedFlagSpec{
	name: "no-stream",
	mode: reservedNoValue,
	apply: func(r *Reserved, _ []string) error {
		r.NoStream = true
		return nil
	},
}

var pagerFlagSpec = reservedFlagSpec{
	name:      "pager",
	mode:      reservedMultiValue,
	visible:   true,
	helpOrder: 9,
	descZH:    "合并可分页 API 的多页结果（可用 --all-pages）；可选 path/PageNumber/PageSize/TotalCount/NextToken",
	descEN:    "Merge pages for pageable APIs (alias --all-pages); optional path/PageNumber/PageSize/TotalCount/NextToken",
	apply: func(r *Reserved, values []string) error {
		kv, err := parseReservedObject(values)
		if err != nil {
			return fmt.Errorf("--pager: %w", err)
		}
		return applyPager(r, kv)
	},
}

var allPagesFlagSpec = reservedFlagSpec{
	name: "all-pages",
	mode: reservedMultiValue,
	apply: func(r *Reserved, values []string) error {
		kv, err := parseReservedObject(values)
		if err != nil {
			return fmt.Errorf("--all-pages: %w", err)
		}
		return applyPager(r, kv)
	},
}

var waiterFlagSpec = reservedFlagSpec{
	name: "waiter",
	mode: reservedMultiValue,
	apply: func(r *Reserved, values []string) error {
		kv, err := parseReservedObject(values)
		if err != nil {
			return fmt.Errorf("--waiter: %w", err)
		}
		return applyWaiter(r, kv)
	},
}

var outputFlagSpec = reservedFlagSpec{
	name:      "output",
	shorthand: 'o',
	mode:      reservedMultiValue,
	apply: func(r *Reserved, values []string) error {
		return parseOutputFlag(r, values)
	},
}

var headerFlagSpec = reservedFlagSpec{
	name: "header",
	mode: reservedMultiValue,
	apply: func(r *Reserved, values []string) error {
		if len(values) == 0 {
			return fmt.Errorf("--header expects Name=Value")
		}
		r.Headers = append(r.Headers, values...)
		return nil
	},
}

var estimateCostContextFlagSpec = reservedFlagSpec{
	name: "estimate-cost-context",
	mode: reservedMultiValue,
	apply: func(r *Reserved, values []string) error {
		if len(values) == 0 {
			return fmt.Errorf("--estimate-cost-context expects Key=Value")
		}
		r.EstimateCostContext = append(r.EstimateCostContext, values...)
		return nil
	},
}

// reservedFlags is the single ordered registry for runtime-owned flags.
var reservedFlags = newReservedFlagIndex(
	regionFlagSpec,
	endpointFlagSpec,
	apiVersionFlagSpec,
	cliQueryFlagSpec,
	logLevelFlagSpec,
	bodyFlagSpec,
	bodyFileFlagSpec,
	cliDryRunFlagSpec,
	cliDryRunJSONFlagSpec,
	helpFlagSpec,
	quietFlagSpec,
	secureFlagSpec,
	insecureFlagSpec,
	estimateCostFlagSpec,
	noStreamFlagSpec,
	pagerFlagSpec,
	allPagesFlagSpec,
	waiterFlagSpec,
	outputFlagSpec,
	headerFlagSpec,
	estimateCostContextFlagSpec,
)

type reservedFlagIndex struct {
	byName      map[string]reservedFlagSpec
	byShorthand map[rune]string
}

func newReservedFlagIndex(specs ...reservedFlagSpec) *reservedFlagIndex {
	reservedFlags := &reservedFlagIndex{
		byName:      make(map[string]reservedFlagSpec, len(specs)),
		byShorthand: make(map[rune]string),
	}
	for _, spec := range specs {
		if spec.name == "" {
			panic("reserved flag name is empty")
		}
		if _, duplicate := reservedFlags.byName[spec.name]; duplicate {
			panic(fmt.Sprintf("reserved flag --%s is registered more than once", spec.name))
		}
		reservedFlags.byName[spec.name] = spec
		if spec.shorthand == 0 {
			continue
		}
		if existing, duplicate := reservedFlags.byShorthand[spec.shorthand]; duplicate {
			panic(fmt.Sprintf("reserved shorthand -%c is registered by both --%s and --%s", spec.shorthand, existing, spec.name))
		}
		reservedFlags.byShorthand[spec.shorthand] = spec.name
	}
	return reservedFlags
}

func (reservedFlags *reservedFlagIndex) match(tok string) (string, reservedFlagSpec, string, bool, bool) {
	if name, inline, hasInline, isLongFlag := splitLongFlag(tok); isLongFlag {
		spec, ok := reservedFlags.byName[name]
		return name, spec, inline, hasInline, ok
	}

	prefix, inline, hasInline := tok, "", false
	if k, v, ok := strings.Cut(tok, "="); ok {
		prefix, inline, hasInline = k, v, true
	}
	runes := []rune(prefix)
	if len(runes) != 2 || runes[0] != '-' || runes[1] == '-' {
		return "", reservedFlagSpec{}, "", false, false
	}
	name, ok := reservedFlags.byShorthand[runes[1]]
	if !ok {
		return "", reservedFlagSpec{}, "", false, false
	}
	return name, reservedFlags.byName[name], inline, hasInline, true
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

// ReservedFlag describes one engine reserved (global) flag for help rendering.
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

// ReservedFlags returns the engine's reserved (global) flags for help
// rendering, filtering hidden entries directly from the reserved flag index and sorting
// visible entries by their declared help order.
func ReservedFlags() []ReservedFlag {
	out := make([]ReservedFlag, 0, len(reservedFlags.byName))
	order := make(map[string]int, len(reservedFlags.byName))
	for name, spec := range reservedFlags.byName {
		if !spec.visible {
			continue
		}
		out = append(out, ReservedFlag{
			Name:       name,
			TakesValue: spec.mode != reservedNoValue,
			DescZH:     spec.descZH,
			DescEN:     spec.descEN,
		})
		order[name] = spec.helpOrder
	}
	sort.Slice(out, func(i, j int) bool {
		if order[out[i].Name] != order[out[j].Name] {
			return order[out[i].Name] < order[out[j].Name]
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func consumeReservedFlag(
	args []string,
	i int,
	externalFlags *externalFlagIndex,
	apiParams *paramIndex,
	spec reservedFlagSpec,
	inlineVal string,
	hasInline bool,
	reserved *Reserved,
) (int, error) {
	var values []string
	switch spec.mode {
	case reservedNoValue:
		// Preserve the existing switch behavior: an inline value is ignored.
	case reservedSingleValue:
		if hasInline {
			values = []string{inlineVal}
		} else {
			value, next := takeOneValue(args, i, externalFlags, apiParams)
			i = next
			values = []string{value}
		}
	case reservedMultiValue:
		if hasInline {
			values = []string{inlineVal}
		} else {
			values, i = takeValues(args, i, externalFlags, apiParams)
		}
	default:
		return i, fmt.Errorf("unsupported reserved flag value mode %d", spec.mode)
	}
	if err := spec.apply(reserved, values); err != nil {
		return i, err
	}
	return i, nil
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
