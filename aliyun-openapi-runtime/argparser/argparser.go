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
	"io"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// Result bundles the parsed API arguments and the reserved flags.
type Result struct {
	// Args maps each provided parameter's wire RawName to its parsed
	// value (nested object keys are also RawName).
	// Scalars are string / json.Number / float64 / bool;
	// arrays are []any;
	// objects/maps are map[string]any.
	Args map[string]any

	// Reserved holds the runtime-steering flags.
	Reserved Reserved
}

// Parse interprets args against the parameter schema.
// Unknown flags produce an error carrying a suggestion-friendly message;
// the caller (L6) decides whether to surface it or fall back to help.
func Parse(params []meta.Parameter, args []string) (*Result, error) {
	return ParseWithOptions(params, args, ParseOptions{})
}

// ParseWithOptions parses engine/API arguments while syntactically consuming external flags whose values are already owned by the embedding host.
func ParseWithOptions(params []meta.Parameter, args []string, opts ParseOptions) (*Result, error) {
	apiParams := newParamIndex(params)
	externalFlags, err := newExternalFlagIndex(opts.ExternalFlags)
	if err != nil {
		return nil, err
	}
	res := &Result{Args: map[string]any{}}

	i := 0
	for i < len(args) {
		tok := args[i]
		// Runtime-owned flags, long and shorthand, have highest priority.
		if _, spec, inlineVal, hasInline, ok := reservedFlags.match(tok); ok {
			i++
			i, err = consumeReservedFlag(args, i, externalFlags, apiParams, spec, inlineVal, hasInline, &res.Reserved)
			if err != nil {
				return nil, err
			}
			continue
		}

		// Host-owned external flags have priority over API metadata parameters.
		if spec, inlineVal, hasInline, ok := externalFlags.match(tok); ok {
			i++
			i, err = consumeExternalFlag(args, i, externalFlags, apiParams, spec, inlineVal, hasInline)
			if err != nil {
				return nil, err
			}
			continue
		}
		name, inlineVal, hasInline, isFlag := splitLongFlag(tok)
		if !isFlag {
			return nil, fmt.Errorf("unexpected positional argument %q", tok)
		}
		i++

		p := apiParams.lookup(name)
		if p == nil {
			return nil, &UnknownFlagError{Flag: name, Known: apiParams.optionNames()}
		}
		if hasInline {
			switch p.Type {
			case meta.TypeArray, meta.TypeObject, meta.TypeMap:
				return nil, fmt.Errorf("--%s does not support an inline value; use --%s <value>", name, name)
			}
		}

		// Scalars consume one value; composites consume through the next registered flag.
		var occ []string
		if hasInline {
			occ = []string{inlineVal}
		} else if p.IsComposite() {
			occ, i = takeValues(args, i, externalFlags, apiParams)
		} else {
			value, next := takeOneValue(args, i, externalFlags, apiParams)
			if next != i {
				occ = []string{value}
				i = next
			}
		}

		if err := assign(res.Args, p, occ); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// assign folds one flag occurrence's raw tokens into res under the parameter's WIRE key (RawName).
// Arrays accumulate repeated occurrences; maps and objects must be specified once.
// A parameter without a RawName in metadata is rejected: the args map is keyed strictly by RawName.
func assign(dst map[string]any, p *meta.Parameter, tokens []string) error {
	key, err := resolveWire(p)
	if err != nil {
		return err
	}
	if p.Type == meta.TypeObject || p.Type == meta.TypeMap {
		if _, exists := dst[key]; exists {
			return fmt.Errorf(
				"--%s may only be specified once; pass multiple key=value pairs after one flag or use a single JSON object",
				displayName(p),
			)
		}
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

func isScalarArrayItem(item *meta.Parameter) bool {
	if item == nil {
		return true
	}
	switch item.Type {
	case meta.TypeObject, meta.TypeMap, meta.TypeArray, meta.TypeAny:
		return false
	default:
		return true
	}
}

func assignArray(dst map[string]any, key string, p *meta.Parameter, tokens []string) error {
	existing, _ := dst[key].([]any)

	// JSON-first for this occurrence: a JSON array is expanded into multiple elements.
	// A JSON object may be shorthand for one composite element, but is invalid for
	// arrays whose declared element type is scalar.
	// For scalar arrays, accepting a JSON array is an intentional extension beyond the legacy Go plugin.
	// Field names inside are resolved to wire RawNames.
	if v, recognized, err := tryFlagJSON(tokens); recognized {
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		if arr, isArr := v.([]any); isArr {
			if p.ItemType != nil && p.ItemType.Type == meta.TypeArray {
				return assignNestedArrayJSON(dst, key, p, existing, arr)
			}
			// Preserve the distinction between an omitted flag and an explicitly supplied empty JSON array.
			// Appending zero elements to a nil slice would otherwise make JSON serialization produce null instead of [].
			if len(arr) == 0 && existing == nil {
				existing = []any{}
			}
			for _, e := range arr {
				rv, err := resolveNames(p.ItemType, e)
				if err != nil {
					return fmt.Errorf("--%s: %w", displayName(p), err)
				}
				existing = append(existing, rv)
			}
		} else {
			if isScalarArrayItem(p.ItemType) {
				return fmt.Errorf("--%s: expected JSON array, got %T", displayName(p), v)
			}
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
		// Each occurrence is one object;
		// its tokens are key=value pairs (dotted keys / array indices allowed for nesting).
		// Field keys are addressed and emitted by their wire RawName.
		obj, err := parseKVPairs(tokens, p.ItemType.Fields)
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		dst[key] = append(existing, obj)
		return nil
	}

	// Array of scalars: append each token as one element.
	// Commas are common string content and are intentionally not treated as separators.
	var elemType meta.DataType = meta.TypeString
	if p.ItemType != nil {
		elemType = p.ItemType.Type
	}
	for _, t := range tokens {
		v, err := coerceScalar(elemType, t)
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		existing = append(existing, v)
	}
	dst[key] = existing
	return nil
}

// assignNestedArrayJSON matches aliyun-cli-runtime's array-of-array input convention.
// A JSON value containing inner arrays is a complete outer array, while a single flat JSON array is one inner array and can be repeated:
//
//	--nodes '[["a","b"],["c"]]'
//	--nodes '["a","b"]' --nodes '["c"]'
func assignNestedArrayJSON(dst map[string]any, key string, p *meta.Parameter, existing []any, decoded []any) error {
	innerValues := decoded
	if len(decoded) == 0 {
		innerValues = []any{decoded}
	} else if _, nested := decoded[0].([]any); !nested {
		innerValues = []any{decoded}
	}

	for i, value := range innerValues {
		if _, ok := value.([]any); !ok {
			value = []any{value}
		}
		resolved, err := resolveNames(p.ItemType, value)
		if err != nil {
			return fmt.Errorf("--%s: invalid array element at index %d: %w", displayName(p), len(existing)+i, err)
		}
		existing = append(existing, resolved)
	}
	dst[key] = existing
	return nil
}

func assignObject(dst map[string]any, key string, p *meta.Parameter, tokens []string) error {
	// JSON-first (plugin parity): "--cfg '{...}'" is parsed as JSON and its field names resolved to wire RawNames;
	// otherwise fall back to the key=value form.
	if v, recognized, err := tryFlagJSON(tokens); recognized {
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		m, isMap := v.(map[string]any)
		if !isMap {
			return fmt.Errorf("--%s: expected a JSON object", displayName(p))
		}
		rv, err := resolveNames(p, m)
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		dst[key] = rv.(map[string]any)
		return nil
	}

	obj, err := parseKVPairs(tokens, p.Fields)
	if err != nil {
		return fmt.Errorf("--%s: %w", displayName(p), err)
	}
	dst[key] = obj
	return nil
}

func assignMap(dst map[string]any, key string, p *meta.Parameter, tokens []string) error {
	// JSON-first (plugin parity), then the flat key=value form.
	// Keys are free-form (no schema, no dotted nesting);
	// values are coerced to the map's declared ValueType.
	if v, recognized, err := tryFlagJSON(tokens); recognized {
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		m, isMap := v.(map[string]any)
		if !isMap {
			return fmt.Errorf("--%s: expected a JSON object", displayName(p))
		}
		rv, err := resolveNames(p, m)
		if err != nil {
			return fmt.Errorf("--%s: %w", displayName(p), err)
		}
		dst[key] = rv.(map[string]any)
		return nil
	}

	existing, _ := dst[key].(map[string]any)
	if existing == nil {
		existing = map[string]any{}
	}
	if p.ValueType != nil && p.ValueType.IsComposite() {
		return compositeMapJSONError(p)
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

func compositeMapJSONError(p *meta.Parameter) error {
	return fmt.Errorf("--%s: map values of type %s require a complete JSON object", displayName(p), p.ValueType.Type)
}

// tryFlagJSON parses a flag occurrence that looks like an object or array as
// exactly one complete JSON value. Tokens are joined with spaces and one layer
// of matching outer quotes is stripped. Numbers are preserved as json.Number.
// recognized distinguishes non-JSON key=value input from malformed JSON-looking
// input, which must be reported rather than silently falling back to key=value.
func tryFlagJSON(tokens []string) (value any, recognized bool, err error) {
	s := stripOuterQuotes(strings.TrimSpace(strings.Join(tokens, " ")))
	if !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "[") {
		return nil, false, nil
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, true, fmt.Errorf("invalid JSON: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, true, fmt.Errorf("invalid JSON: unexpected trailing content")
	}
	return v, true, nil
}

// resolveNames recursively validates the keys of a decoded JSON value against the parameter schema.
// Object fields are addressed by their wire RawName.
// An object with declared fields rejects unknown keys; an object without fields remains open and passes free-form keys through verbatim.
// Known values are resolved recursively with the declared metadata type, matching the legacy runtime's Arg.Resolve behavior for JSON object/array/map input as well as key=value input.
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
			} else if len(p.Fields) > 0 {
				return nil, fmt.Errorf("unknown field: %s", k)
			}
			rv, err := resolveNames(fp, val)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", k, err)
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
				return nil, fmt.Errorf("element %d: %w", i, err)
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
				return nil, fmt.Errorf("value for key %q: %w", k, err)
			}
			out[k] = rv
		}
		return out, nil
	default:
		return coerceDecodedScalar(p.Type, v)
	}
}

// coerceDecodedScalar applies the same declared-type conversion to a scalar
// decoded from JSON that coerceScalar applies to key=value and ordinary flag
// input. Without this pass, equivalent spellings were syntax-dependent: for
// example enabled=yes became true, while {"enabled":"yes"} stayed a string.
//
// nil follows the legacy Arg.Resolve defaults. Any intentionally remains
// schema-free and is returned unchanged.
func coerceDecodedScalar(t meta.DataType, v any) (any, error) {
	switch t {
	case meta.TypeAny:
		return v, nil
	case meta.TypeString:
		if v == nil {
			return "", nil
		}
		if s, ok := v.(string); ok {
			return s, nil
		}
		return fmt.Sprintf("%v", v), nil
	case meta.TypeBoolean:
		if v == nil {
			return false, nil
		}
		switch value := v.(type) {
		case bool:
			return value, nil
		case string:
			return parseBoolean(value)
		case json.Number:
			return parseBoolean(value.String())
		default:
			return nil, fmt.Errorf("cannot convert %T to boolean", v)
		}
	case meta.TypeInteger, meta.TypeLong:
		if v == nil {
			return json.Number("0"), nil
		}
		switch value := v.(type) {
		case json.Number:
			return coerceScalar(t, value.String())
		case string:
			return coerceScalar(t, strings.TrimSpace(value))
		default:
			return nil, fmt.Errorf("cannot convert %T to number", v)
		}
	case meta.TypeFloat:
		if v == nil {
			return float64(0), nil
		}
		switch value := v.(type) {
		case json.Number:
			return coerceScalar(t, value.String())
		case string:
			return coerceScalar(t, strings.TrimSpace(value))
		default:
			return nil, fmt.Errorf("cannot convert %T to float", v)
		}
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
// Field keys resolve to their wire RawName and leaf values are coerced to the field's declared type.
// Objects with declared fields reject unknown keys; objects without fields remain open and accept free-form keys.
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

// setSchemaValue assigns rawVal at keyPath within obj, walking the field schema to resolve wire names, coerce leaf types, descend nested objects and index arrays.
func setSchemaValue(obj map[string]any, keyPath, rawVal string, fields []meta.Parameter) error {
	firstKey, rest, isIndex := parseKeyPath(keyPath)
	if firstKey == "" {
		return fmt.Errorf("invalid key path %q", keyPath)
	}
	f := findField(fields, firstKey)
	if f == nil && len(fields) > 0 {
		return fmt.Errorf("unknown field: %s", firstKey)
	}
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

	// A fieldless object is open: free-form dotted paths become nested objects.
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
		return fmt.Errorf("field %q is a map; set the complete map field as JSON", firstKey)

	default:
		return fmt.Errorf("field %q is a scalar; cannot descend %q", firstKey, rest)
	}
}

// coerceLeaf converts a leaf value against its field schema.
// Object / map / array leaves accept a JSON literal ('{...}' / '[...]');
// scalarsgo through coerceScalar. A nil field belongs to an open object and is preserved as a verbatim string.
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

// resolveWire returns the parameter's RawName, or an error when metadata omitted it.
// Args keys (top-level and nested) are always RawName.
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

// parseKeyPath splits the leading segment from a key path, reporting whether that segment indexes an array.
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

// splitIndex parses a leading "[n]" from rest, returning the index and the remaining path (with a leading dot trimmed).
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

// decodeJSONObject parses a '{...}' literal (optionally wrapped in matching quotes) into a map, preserving numbers as json.Number.
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

// decodeJSONArray parses a '[...]' literal (optionally quote-wrapped) into a slice, preserving numbers as json.Number.
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
// layer expects. Integers become json.Number to preserve int64/large
// precision end-to-end (see the precision contract in the module
// architecture doc). Floats become float64, matching OpenAPI
// number/double and aliyun-cli-runtime's FloatArg contract.
//
// Booleans become Go bools so JSON body and param_style=json values use
// JSON boolean literals, matching the legacy Go plugin. Ordinary query
// serialization still renders them as the strings "true" / "false" on
// the wire.
func coerceScalar(t meta.DataType, raw string) (any, error) {
	switch t {
	case meta.TypeInteger, meta.TypeLong:
		// Preserve exactly as typed; json.Number marshals without
		// quotes and never routes through float64.
		if !looksNumeric(raw) {
			return nil, fmt.Errorf("invalid number %q", raw)
		}
		return json.Number(raw), nil
	case meta.TypeFloat:
		if !looksNumeric(raw) {
			return nil, fmt.Errorf("invalid float %q", raw)
		}
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float %q: %w", raw, err)
		}
		return value, nil
	case meta.TypeBoolean:
		return parseBoolean(raw)
	case meta.TypeAny:
		return parseAny(raw), nil
	default:
		return raw, nil
	}
}

// parseBoolean accepts the spellings supported by the legacy runtime's
// BooleanArg and returns a typed value for the request serializer.
func parseBoolean(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "t", "yes", "y", "1":
		return true, nil
	case "false", "f", "no", "n", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", raw)
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
	if isValidJSONNumberLiteral(t) {
		return json.Number(t)
	}
	return s
}

func isValidJSONNumberLiteral(s string) bool {
	if !json.Valid([]byte(s)) {
		return false
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return false
	}
	_, ok := value.(json.Number)
	return ok
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
	return strings.TrimPrefix(p.Options[0], "--")
}
