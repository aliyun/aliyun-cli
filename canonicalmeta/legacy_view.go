package canonicalmeta

import (
	"fmt"
	"sort"
	"strings"
)

// LegacyParameterSource indicates which source a LegacyParameterView wraps.
type LegacyParameterSource int

const (
	SourceCanonical LegacyParameterSource = iota
	SourceField
	SourceBody
)

// LegacyParameterView is a read-only reference to a Canonical parameter node.
// It provides old CLI compatibility methods without creating a second copy of metadata.
type LegacyParameterView struct {
	source           LegacyParameterSource
	canonical        *Parameter
	field            *Field
	body             *LegacyBodyParameter
	topLocation      string // inherited from top-level parameter for nested fields
	resolvedPosition string // pre-resolved uppercase position for top-level and body views
	isTopBody        bool   // true for top-level v1_body_parameters views
}

// NewCanonicalView creates a view wrapping a Canonical parameter.
func NewCanonicalView(p *Parameter) *LegacyParameterView {
	return &LegacyParameterView{
		source:    SourceCanonical,
		canonical: p,
	}
}

// NewFieldView creates a view wrapping a nested field, inheriting parent location.
func NewFieldView(f *Field, topLocation string) *LegacyParameterView {
	return &LegacyParameterView{
		source:      SourceField,
		field:       f,
		topLocation: topLocation,
	}
}

// NewBodyView creates a view wrapping a V1 body parameter.
func NewBodyView(b *LegacyBodyParameter) *LegacyParameterView {
	return &LegacyParameterView{
		source: SourceBody,
		body:   b,
	}
}

// LegacyName returns the old CLI parameter name (PascalCase).
func (v *LegacyParameterView) LegacyName() string {
	switch v.source {
	case SourceCanonical:
		return v.canonical.RawName
	case SourceField:
		return v.field.RawName
	case SourceBody:
		return v.body.Name
	}
	return ""
}

// LegacyPosition returns the old CLI position (uppercase).
func (v *LegacyParameterView) LegacyPosition() string {
	switch v.source {
	case SourceCanonical:
		if v.resolvedPosition != "" {
			return v.resolvedPosition
		}
		return legacyPosition(v.canonical.Location)
	case SourceField:
		// Sub-parameters use sub-position mapping: form -> FormData (spec 6.3)
		return legacySubPosition(v.topLocation)
	case SourceBody:
		if v.isTopBody {
			return "Body"
		}
		return legacySubPosition(v.body.Position)
	}
	return ""
}

// LegacyType returns the parameter type for help display.
// Uses Canonical lowercase type; RepeatList detection is separate.
func (v *LegacyParameterView) LegacyType() string {
	switch v.source {
	case SourceCanonical:
		return v.canonical.Type
	case SourceField:
		return v.field.Type
	case SourceBody:
		return v.body.Type
	}
	return ""
}

// LegacyRequired returns whether the parameter is required.
func (v *LegacyParameterView) LegacyRequired() bool {
	switch v.source {
	case SourceCanonical:
		return v.canonical.Required
	case SourceField:
		return v.field.Required
	case SourceBody:
		return v.body.Required
	}
	return false
}

// LegacyDescription returns the parameter description for the given locale.
func (v *LegacyParameterView) LegacyDescription(language string) string {
	switch v.source {
	case SourceCanonical:
		if language == "en" {
			return v.canonical.DescriptionEn
		}
		return v.canonical.DescriptionZh
	case SourceField:
		if language == "en" {
			return v.field.DescriptionEn
		}
		return v.field.DescriptionZh
	case SourceBody:
		if language == "en" {
			return v.body.DescriptionEn
		}
		return v.body.DescriptionZh
	}
	return ""
}

// LegacyExample returns the parameter example value.
func (v *LegacyParameterView) LegacyExample() string {
	switch v.source {
	case SourceCanonical:
		return v.canonical.Example
	case SourceField:
		return v.field.Example
	case SourceBody:
		return v.body.Example
	}
	return ""
}

// LegacyHasChildren returns true if this parameter has sub-parameters
// that the old CLI would expose via --X.1.Field syntax.
func (v *LegacyParameterView) LegacyHasChildren() bool {
	if v.source == SourceBody {
		return len(v.body.SubParameters) > 0
	}
	if v.source == SourceField {
		return v.field.Type == "array" && len(v.field.ElementFields) > 0
	}
	if v.source != SourceCanonical {
		return false
	}
	p := v.canonical
	return p.Type == "array" &&
		p.ParamStyle != "flat" &&
		len(p.ElementFields) > 0
}

// IsLegacyRepeatList returns true if this parameter behaves as a RepeatList
// in old CLI semantics (accepts --X.1 syntax).
func (v *LegacyParameterView) IsLegacyRepeatList() bool {
	if v.source == SourceBody {
		return isBodyRepeatList(v.body)
	}
	if v.source == SourceField {
		return v.field.Type == "array"
	}
	if v.source != SourceCanonical {
		return false
	}
	p := v.canonical
	return p.Type == "array" &&
		p.ParamStyle != "flat" &&
		p.ParamStyle != "json"
}

// LegacyChildren returns the sub-parameter views for this parameter.
// For Canonical parameters, only returns element_fields (one level).
// For V1 body parameters, returns sub_parameters as-is.
func (v *LegacyParameterView) LegacyChildren() []*LegacyParameterView {
	var children []*LegacyParameterView

	switch v.source {
	case SourceCanonical:
		if !v.LegacyHasChildren() {
			return nil
		}
		topLoc := v.canonical.Location
		for i := range v.canonical.ElementFields {
			children = append(children, NewFieldView(&v.canonical.ElementFields[i], topLoc))
		}
	case SourceField:
		if !v.LegacyHasChildren() {
			return nil
		}
		for i := range v.field.ElementFields {
			children = append(children, NewFieldView(&v.field.ElementFields[i], v.topLocation))
		}
	case SourceBody:
		for i := range v.body.SubParameters {
			child := NewBodyView(&v.body.SubParameters[i])
			// Sub-parameters are NOT top-level; form -> FormData, not Body
			child.isTopBody = false
			children = append(children, child)
		}
	}

	return children
}

// legacyPosition converts Canonical lowercase location to old CLI uppercase position.
// Top-level mapping: form/formdata -> Body.
func legacyPosition(location string) string {
	switch strings.ToLower(location) {
	case "query":
		return "Query"
	case "body":
		return "Body"
	case "host":
		return "Host"
	case "path":
		return "Path"
	case "header":
		return "Header"
	case "form", "formdata":
		return "Body"
	}
	return ""
}

// legacySubPosition converts Canonical lowercase location to old CLI uppercase position
// for sub-parameters. form -> FormData (spec 6.3), unlike top-level form -> Body.
func legacySubPosition(location string) string {
	switch strings.ToLower(location) {
	case "query":
		return "Query"
	case "body":
		return "Body"
	case "host":
		return "Host"
	case "path":
		return "Path"
	case "header":
		return "Header"
	case "form":
		return "FormData"
	case "formdata":
		return "FormData"
	}
	return ""
}

// isCanonicalRepeatList returns true if a Canonical parameter behaves as a
// RepeatList in old CLI semantics (spec 6.4: array && !flat && !json).
func isCanonicalRepeatList(p *Parameter) bool {
	return p.Type == "array" &&
		p.ParamStyle != "flat" &&
		p.ParamStyle != "json"
}

// isBodyRepeatList returns true if a V1 body parameter behaves as a RepeatList.
// v1_body_parameters use unified lowercase types with param_style (spec 3.2).
func isBodyRepeatList(b *LegacyBodyParameter) bool {
	return b.Type == "array" &&
		b.ParamStyle != "flat" &&
		b.ParamStyle != "json"
}

// ── API-level legacy methods ──

// GetMethod selects the method from the candidate string with old CLI priority.
// POST > GET, matching the original API method selection behavior.
func (api *API) GetMethod() string {
	method := strings.ToUpper(api.Method)
	if strings.Contains(method, "POST") {
		return "POST"
	}
	if strings.Contains(method, "GET") {
		return "GET"
	}
	return "GET"
}

// GetProtocol selects the protocol from the candidate string with HTTPS priority.
// Matches the original API protocol selection behavior.
func (api *API) GetProtocol() string {
	lowered := strings.ToLower(api.Protocol)

	if strings.HasPrefix(lowered, "https") {
		return "https"
	}

	parts := strings.Split(lowered, "|")
	for _, v := range parts {
		if v == "https" {
			return "https"
		}
	}

	return "http"
}

// ── Excluded parameter names (protocol-level, not user-facing) ──

var excludedParamNames = map[string]bool{
	"Action":               true,
	"OwnerId":              true,
	"OwnerAccount":         true,
	"ResourceOwnerId":      true,
	"ResourceOwnerAccount": true,
}

// LegacyTopLevelParameters returns the top-level parameter views for old CLI consumption.
// Applies v1_body_parameters three-state logic and deduplication.
func (api *API) LegacyTopLevelParameters() []*LegacyParameterView {
	var result []*LegacyParameterView
	seen := make(map[string]bool)

	// If v1_body_parameters exists, skip all body/form public parameters
	skipBody := api.V1BodyParameters != nil

	// 1. Add public parameters
	for i := range api.Parameters {
		p := &api.Parameters[i]

		// Skip excluded protocol-level names
		if excludedParamNames[p.RawName] {
			continue
		}

		// Skip body/form parameters if v1_body_parameters is present
		if skipBody && isBodyLocation(p.Location) {
			continue
		}

		// First-wins dedup by raw_name
		if seen[p.RawName] {
			continue
		}
		seen[p.RawName] = true

		result = append(result, NewCanonicalView(p))
	}

	// 2. Add v1_body_parameters if present
	if api.V1BodyParameters != nil {
		for i := range *api.V1BodyParameters {
			b := &(*api.V1BodyParameters)[i]

			// v1_body_parameters nodes take priority over same-name public params
			if seen[b.Name] {
				// Remove the existing public param and replace with body param
				for j, v := range result {
					if v.LegacyName() == b.Name {
						result[j] = NewBodyView(b)
						break
					}
				}
			} else {
				seen[b.Name] = true
				result = append(result, NewBodyView(b))
			}
		}
	}

	// 3. Set resolvedPosition for canonical views and isTopBody for body views
	for _, v := range result {
		if v.source == SourceCanonical {
			v.resolvedPosition = legacyPosition(v.canonical.Location)
		}
		if v.source == SourceBody {
			v.isTopBody = true
		}
	}

	// 4. Sort by legacy name only (spec 6.2: base view sorts by raw name for
	// completion and traversal; help applies required-first sort separately).
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].LegacyName() < result[j].LegacyName()
	})

	return result
}

func isBodyLocation(location string) bool {
	loc := strings.ToLower(location)
	return loc == "body" || loc == "form" || loc == "formdata"
}

// FindLegacyParameter finds a parameter by name with old CLI's loose index parsing.
// Matches the original loose parameter lookup behavior:
//   - Case-sensitive
//   - Exact parent name match first
//   - For RepeatList without children: any non-empty suffix matches
//   - For array with children: intermediate index not validated as number
func (api *API) FindLegacyParameter(name string) *LegacyParameterView {
	return findLegacyParameter(api.LegacyTopLevelParameters(), name)
}

func findLegacyParameter(params []*LegacyParameterView, name string) *LegacyParameterView {
	for _, v := range params {
		pName := v.LegacyName()

		// Exact match
		if pName == name {
			return v
		}

		// Check for sub-parameter access: Name.suffix
		if strings.HasPrefix(name, pName+".") {
			suffix := name[len(pName)+1:]

			children := v.LegacyChildren()
			if len(children) > 0 {
				// Has children: preserve old loose index parsing. The middle
				// segment is not validated, but a child lookup needs two dots:
				// Tag.foo.Key and Tag..Key match Key; Tag.Key does not.
				remainder := name[len(pName):]
				if len(remainder) >= 4 && remainder[0] == '.' && strings.Count(remainder, ".") >= 2 {
					dotIdx := strings.Index(suffix, ".")
					if dotIdx >= 0 {
						return findLegacyParameter(children, suffix[dotIdx+1:])
					}
				}
				return nil
			}

			// No children (RepeatList leaf): any non-empty suffix matches parent
			if v.IsLegacyRepeatList() {
				return v
			}
		}

		// RepeatList without dot: X.1 style where X has no children
		if v.IsLegacyRepeatList() && strings.HasPrefix(name, pName) {
			s := name[len(pName):]
			if len(s) >= 2 && s[0] == '.' {
				return v
			}
		}
	}

	return nil
}

// ForeachLegacyParameter iterates over all parameters with recursion for sub-parameters.
// Generates canonical .1 / .1.Field syntax (strict generation, loose find).
// Matches the original parameter traversal behavior.
func (api *API) ForeachLegacyParameter(f func(name string, v *LegacyParameterView)) {
	foreachLegacyParameter(api.LegacyTopLevelParameters(), "", f)
}

func foreachLegacyParameter(params []*LegacyParameterView, prefix string, f func(string, *LegacyParameterView)) {
	for _, v := range params {
		name := v.LegacyName()

		children := v.LegacyChildren()
		if len(children) > 0 {
			// Has children: recurse with .1. prefix
			foreachLegacyParameter(children, prefix+name+".1.", f)
		} else if v.IsLegacyRepeatList() {
			// RepeatList leaf: report as X.1
			f(prefix+name+".1", v)
		} else {
			// Regular parameter
			f(prefix+name, v)
		}
	}
}

// CheckLegacyRequiredParameters checks that all required top-level parameters are assigned.
// Iterates in declaration order (not sorted) to match the original
// required-parameter behavior, which traverses api.Parameters
// in JSON declaration order. RepeatList parameters (spec 6.4) are skipped for
// both public and v1_body_parameters, matching the old p.Type != "RepeatList".
func (api *API) CheckLegacyRequiredParameters(checker func(string) bool) error {
	var missing []string
	seen := make(map[string]bool)
	skipBody := api.V1BodyParameters != nil
	bodyNames := make(map[string]bool)
	if api.V1BodyParameters != nil {
		for i := range *api.V1BodyParameters {
			bodyNames[(*api.V1BodyParameters)[i].Name] = true
		}
	}

	// 1. Check public parameters in declaration order
	for i := range api.Parameters {
		p := &api.Parameters[i]
		if excludedParamNames[p.RawName] {
			continue
		}
		if skipBody && isBodyLocation(p.Location) {
			continue
		}
		if bodyNames[p.RawName] {
			continue
		}
		if seen[p.RawName] {
			continue
		}
		seen[p.RawName] = true

		if p.Required && !isCanonicalRepeatList(p) {
			if !checker(p.RawName) {
				missing = append(missing, p.RawName)
			}
		}
	}

	// 2. Check v1_body_parameters in declaration order
	if api.V1BodyParameters != nil {
		for i := range *api.V1BodyParameters {
			b := &(*api.V1BodyParameters)[i]
			if seen[b.Name] {
				continue
			}
			seen[b.Name] = true

			if b.Required && !isBodyRepeatList(b) {
				if !checker(b.Name) {
					missing = append(missing, b.Name)
				}
			}
		}
	}

	if len(missing) > 0 {
		s := ""
		for _, name := range missing {
			s += "\n  --" + name
		}
		return fmt.Errorf("required parameters not assigned: %s", s)
	}

	return nil
}
