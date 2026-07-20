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

package jsoncmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// printAPIHelp renders per-API help as following
//
//	Description: ...
//	API Version: ...
//	Usage:
//	  aliyun <product> <command> [parameters]
//	Parameters:
//	  --flag   type (required), help
//	           structure: {...}
//	           format: --flag ...
//	Global Parameters:
//	  --cli-dry-run   help
//	Examples:
//	  ...
func printAPIHelp(w io.Writer, product string, api *meta.API, lang string) error {
	l := helpLabels(lang)

	if d := api.Description.Localized(lang); d != "" {
		fmt.Fprintf(w, "\n%s: %s\n", l.description, d)
	}
	fmt.Fprintln(w)
	if api.Version != "" {
		fmt.Fprintf(w, "%s: %s\n\n", l.apiVersion, api.Version)
	}
	fmt.Fprintf(w, "%s:\n  aliyun %s %s [parameters]\n\n", l.usage, product, api.CmdName)

	fmt.Fprintf(w, "%s:\n", l.parameters)
	renderParameters(w, api.Parameters, lang)

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s:\n", l.globalParameters)
	renderGlobalParameters(w, lang)

	if len(api.Examples) > 0 {
		fmt.Fprintf(w, "\n%s:\n", l.examples)
		for _, e := range api.Examples {
			fmt.Fprintf(w, "  %s\n", e)
		}
	}
	return nil
}

// labels holds the localized section headers. The engine stays
// host-agnostic (no dependency on aliyun-cli's i18n package): the host
// passes a lang string and we pick zh/en here
type labels struct {
	description      string
	apiVersion       string
	usage            string
	parameters       string
	globalParameters string
	examples         string
}

func helpLabels(lang string) labels {
	if lang == "zh" {
		return labels{
			description:      "描述",
			apiVersion:       "API 版本",
			usage:            "使用",
			parameters:       "参数",
			globalParameters: "全局参数",
			examples:         "示例",
		}
	}
	return labels{
		description:      "Description",
		apiVersion:       "API Version",
		usage:            "Usage",
		parameters:       "Parameters",
		globalParameters: "Global Parameters",
		examples:         "Examples",
	}
}

const helpNameWidth = 24

// renderParameters lists the API parameters, required first (each group sorted by flag), with structure/format hints for composite types.
func renderParameters(w io.Writer, params []meta.Parameter, lang string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	var required, optional []*meta.Parameter
	for i := range params {
		p := &params[i]
		if p.Required {
			required = append(required, p)
		} else {
			optional = append(optional, p)
		}
	}
	byFlag := func(s []*meta.Parameter) {
		sort.Slice(s, func(i, j int) bool { return displayFlag(s[i]) < displayFlag(s[j]) })
	}
	byFlag(required)
	byFlag(optional)

	for _, p := range append(required, optional...) {
		renderOneParameter(tw, p, lang)
	}
}

func renderOneParameter(tw *tabwriter.Writer, p *meta.Parameter, lang string) {
	name := displayFlag(p)
	namePadded := fmt.Sprintf("%-*s", helpNameWidth, name)
	emptyPadded := fmt.Sprintf("%-*s", helpNameWidth, "")

	typ := typeName(p)
	if p.Required {
		typ += " (required)"
	}
	help := collapse(p.Description.Localized(lang))

	head := typ
	if help != "" {
		head += ", " + help
	}
	printWrappedLine(tw, namePadded, head, helpNameWidth)

	structure, format := describeHint(name, p)
	if structure != "" {
		printWrappedLine(tw, emptyPadded, "structure: "+structure, helpNameWidth)
	}
	if format != "" {
		printWrappedLine(tw, emptyPadded, format, helpNameWidth)
	}
}

// renderGlobalParameters lists the engine's reserved (global) flags. Host credential/profile flags are intentionally omitted.
func renderGlobalParameters(w io.Writer, lang string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()
	for _, g := range argparser.ReservedFlags() {
		name := "--" + g.Name
		namePadded := fmt.Sprintf("%-*s", helpNameWidth, name)
		typ := "bool"
		if g.TakesValue {
			typ = "string"
		}
		printWrappedLine(tw, namePadded, typ+", "+g.Desc(lang), helpNameWidth)
	}
}

// ============================================================================
// wrapping / alignment
// ============================================================================

func getMaxLineLength() int {
	const defaultMaxLineLength = 80
	envValue := os.Getenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH")
	if envValue == "" {
		return defaultMaxLineLength
	}
	if length, err := strconv.Atoi(envValue); err == nil && length > 0 {
		return length
	}
	return defaultMaxLineLength
}

// printWrappedLine writes "  <prefix>\t<text>" with text wrapped to the
// remaining width. Continuation lines keep an empty padded prefix so
// the description column stays aligned under its first-line start — same behaviour as the Go plugin help.
func printWrappedLine(w *tabwriter.Writer, prefix string, text string, prefixWidth int) {
	maxLineLength := getMaxLineLength()
	// Account for: "  " (2) + prefix (prefixWidth) + tab (~2-4).
	maxTextLength := maxLineLength - prefixWidth - 4
	if maxTextLength < 20 {
		maxTextLength = 20
	}

	runes := []rune(text)
	if len(runes) <= maxTextLength {
		fmt.Fprintf(w, "  %s\t%s\n", prefix, text)
		return
	}

	emptyPrefix := fmt.Sprintf("%-*s", prefixWidth, "")
	first := true
	start := 0
	for start < len(runes) {
		end := start + maxTextLength
		if end >= len(runes) {
			chunk := string(runes[start:])
			if first {
				fmt.Fprintf(w, "  %s\t%s\n", prefix, chunk)
			} else {
				fmt.Fprintf(w, "  %s\t%s\n", emptyPrefix, chunk)
			}
			break
		}

		breakPoint := end
		for i := end - 1; i > start+maxTextLength/2 && i < len(runes); i-- {
			ch := runes[i]
			if ch == ' ' || ch == ',' || ch == '.' || ch == ';' || ch == '，' || ch == '。' || ch == '、' {
				breakPoint = i + 1
				break
			}
		}

		chunk := strings.TrimSpace(string(runes[start:breakPoint]))
		if first {
			fmt.Fprintf(w, "  %s\t%s\n", prefix, chunk)
			first = false
		} else {
			fmt.Fprintf(w, "  %s\t%s\n", emptyPrefix, chunk)
		}

		start = breakPoint
		for start < len(runes) && runes[start] == ' ' {
			start++
		}
	}
}

// ============================================================================
// helpers
// ============================================================================

// collapse trims and squeezes internal whitespace/newlines to single
// spaces so multi-line meta help renders as one logical paragraph that
// printWrappedLine can then re-wrap at the column boundary.
func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// displayFlag returns the user-facing flag spelling: the curated
// option(s) joined, or the derived kebab of the parameter name.
func displayFlag(p *meta.Parameter) string {
	if len(p.Options) > 0 {
		return strings.Join(p.Options, ", ")
	}
	return "--" + kebab(p.Name)
}

// typeName maps a meta.DataType to the plugin engine's help vocabulary
// (string / int / bool / list / object / map / float).
func typeName(p *meta.Parameter) string {
	switch p.Type {
	case meta.TypeArray:
		return "list"
	case meta.TypeObject:
		return "object"
	case meta.TypeMap:
		return "map"
	case meta.TypeInteger, meta.TypeLong:
		return "int"
	case meta.TypeFloat:
		return "float"
	case meta.TypeBoolean:
		return "bool"
	default:
		return "string"
	}
}

func isScalar(p *meta.Parameter) bool {
	switch p.Type {
	case meta.TypeObject, meta.TypeArray, meta.TypeMap:
		return false
	default:
		return true
	}
}

func isFlatObject(p *meta.Parameter) bool {
	if p == nil || p.Type != meta.TypeObject || len(p.Fields) == 0 {
		return false
	}
	for i := range p.Fields {
		if !isScalar(&p.Fields[i]) {
			return false
		}
	}
	return true
}

// describeHint returns the "structure:" description and "format:" usage
// example for composite parameters, matching the plugin engine's hints.
// Scalars produce no hint.
func describeHint(flag string, p *meta.Parameter) (structure, format string) {
	bare := strings.TrimLeft(primaryOption(flag), "-")
	switch p.Type {
	case meta.TypeArray:
		elem := p.ItemType
		if elem == nil || isScalar(elem) {
			format = fmt.Sprintf("format: --%s value1 value2 value3", bare)
			return
		}
		if isFlatObject(elem) {
			structure = describeObject(elem)
			keys := objectFieldNames(elem)
			format = fmt.Sprintf("format: --%s %s --%s %s",
				bare, kvExample(keys, []string{"a", "b", "c", "d"}),
				bare, kvExample(keys, []string{"e", "f", "g", "h"}))
			return
		}
		structure = describeStructure(p)
		format = fmt.Sprintf("format: --%s 'value'", bare)
	case meta.TypeObject:
		structure = describeObject(p)
		if isFlatObject(p) {
			format = fmt.Sprintf("format: --%s %s ...", bare, kvExample(objectFieldNames(p), []string{"xxx"}))
		} else {
			format = fmt.Sprintf("format: --%s 'value'", bare)
		}
	case meta.TypeMap:
		structure = describeStructure(p)
		if p.ValueType == nil || isScalar(p.ValueType) {
			format = fmt.Sprintf("format: --%s key1=value1 key2=value2 ...", bare)
		} else {
			format = fmt.Sprintf("format: --%s 'value'", bare)
		}
	}
	return
}

// describeStructure renders a nested type sketch, e.g.
// {Key: string, Value: string} / [string, ...] / map[string]object.
func describeStructure(p *meta.Parameter) string {
	if p == nil {
		return ""
	}
	switch p.Type {
	case meta.TypeObject:
		return describeObject(p)
	case meta.TypeArray:
		if p.ItemType == nil {
			return "[]"
		}
		return "[" + describeStructure(p.ItemType) + ", ...]"
	case meta.TypeMap:
		v := "string"
		if p.ValueType != nil {
			v = describeStructure(p.ValueType)
		}
		return "map[string]" + v
	default:
		return typeName(p)
	}
}

func describeObject(p *meta.Parameter) string {
	names := objectFieldNames(p)
	parts := make([]string, 0, len(names))
	byName := map[string]*meta.Parameter{}
	for i := range p.Fields {
		byName[objectFieldName(&p.Fields[i])] = &p.Fields[i]
	}
	for _, n := range names {
		parts = append(parts, n+": "+describeStructure(byName[n]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func objectFieldNames(p *meta.Parameter) []string {
	if p == nil {
		return nil
	}
	names := make([]string, 0, len(p.Fields))
	for i := range p.Fields {
		names = append(names, objectFieldName(&p.Fields[i]))
	}
	sort.Strings(names)
	return names
}

func objectFieldName(p *meta.Parameter) string {
	if p.RawName != "" {
		return p.RawName
	}
	return p.Name
}

func kvExample(keys []string, values []string) string {
	parts := make([]string, 0, len(keys))
	for i, k := range keys {
		parts = append(parts, k+"="+values[i%len(values)])
	}
	return strings.Join(parts, " ")
}

// primaryOption returns the first option out of a possibly-joined
// "--a, --b" display string.
func primaryOption(flag string) string {
	if i := strings.IndexByte(flag, ','); i >= 0 {
		return strings.TrimSpace(flag[:i])
	}
	return flag
}
