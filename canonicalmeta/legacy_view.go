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
	SourceV1
)

// LegacyParameterView is a read-only reference to a Canonical parameter node.
// It provides old CLI compatibility methods without creating a second copy of metadata.
type LegacyParameterView struct {
	source           LegacyParameterSource
	canonical        *Parameter
	field            *Field
	body             *V1Parameter
	topLocation      string // inherited from top-level parameter for nested fields
	resolvedPosition string // pre-resolved uppercase position for top-level and body views
	isTopBody        bool   // true for top-level v1_body_parameters views
	isWildcard       bool   // inherited from the canonical parameter for V1 views
}

// Constraints contains optional schema restrictions used by AI-mode validation.
type Constraints struct {
	Enum      []string
	Minimum   string
	Maximum   string
	MinLength string
	MaxLength string
	Pattern   string
}

// Constraints returns canonical restrictions for this parameter. V1-only
// compatibility projections intentionally have no constraints.
func (v *LegacyParameterView) Constraints() Constraints {
	switch v.source {
	case SourceCanonical:
		if v.canonical.Type == "array" && isScalarTypeShape(v.canonical.Element) {
			return constraintsFromTypeShape(v.canonical.Element)
		}
		return Constraints{
			Enum:      v.canonical.Enum,
			Minimum:   v.canonical.Minimum,
			Maximum:   v.canonical.Maximum,
			MinLength: v.canonical.MinLength,
			MaxLength: v.canonical.MaxLength,
			Pattern:   v.canonical.Pattern,
		}
	case SourceField:
		if v.field.Type == "array" && isScalarTypeShape(v.field.Element) {
			return constraintsFromTypeShape(v.field.Element)
		}
		return Constraints{
			Enum:      v.field.Enum,
			Minimum:   v.field.Minimum,
			Maximum:   v.field.Maximum,
			MinLength: v.field.MinLength,
			MaxLength: v.field.MaxLength,
			Pattern:   v.field.Pattern,
		}
	default:
		return Constraints{}
	}
}

// ConstraintType returns the scalar type to which Constraints applies. Legacy
// RepeatList leaves expose the array element as one flag value at a time.
func (v *LegacyParameterView) ConstraintType() string {
	switch v.source {
	case SourceCanonical:
		if v.canonical.Type == "array" && isScalarTypeShape(v.canonical.Element) {
			return v.canonical.Element.Type
		}
	case SourceField:
		if v.field.Type == "array" && isScalarTypeShape(v.field.Element) {
			return v.field.Element.Type
		}
	}
	return v.LegacyType()
}

func isScalarTypeShape(shape *TypeShape) bool {
	if shape == nil {
		return false
	}
	switch strings.ToLower(shape.Type) {
	case "string", "int", "integer", "int32", "int64", "long", "float", "double", "number", "bool", "boolean":
		return true
	default:
		return false
	}
}

func constraintsFromTypeShape(shape *TypeShape) Constraints {
	return Constraints{
		Enum:      shape.Enum,
		Minimum:   shape.Minimum,
		Maximum:   shape.Maximum,
		MinLength: shape.MinLength,
		MaxLength: shape.MaxLength,
		Pattern:   shape.Pattern,
	}
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

// NewBodyView creates a view wrapping a v1_body_parameters entry.
func NewBodyView(b *V1Parameter) *LegacyParameterView {
	return &LegacyParameterView{
		source: SourceBody,
		body:   b,
	}
}

// NewV1View creates a view wrapping a complete V1 parameter.
func NewV1View(b *V1Parameter) *LegacyParameterView {
	return &LegacyParameterView{
		source: SourceV1,
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
	case SourceBody, SourceV1:
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
	case SourceV1:
		if v.resolvedPosition != "" {
			return v.resolvedPosition
		}
		return legacyPosition(v.body.Position)
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
	case SourceBody, SourceV1:
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
	case SourceBody, SourceV1:
		return v.body.Required
	}
	return false
}

func (v *LegacyParameterView) DocRequired() bool {
	switch v.source {
	case SourceCanonical:
		return v.canonical.DocRequired
	case SourceField:
		return v.field.DocRequired
	}
	return false
}

// IsWildcard reports whether this path parameter replaces the complete path template.
func (v *LegacyParameterView) IsWildcard() bool {
	if v.source == SourceCanonical {
		return v.canonical.IsWildcard
	}
	return v.isWildcard
}

// LegacyDescription returns the legacy help text for the given locale.
// Canonical parameters/fields consume help_* only; v1 legacy parameters keep
// description_* because the v1 projection does not yet provide help_* broadly.
func (v *LegacyParameterView) LegacyDescription(language string) string {
	switch v.source {
	case SourceCanonical:
		return v.canonical.Help(language)
	case SourceField:
		return v.field.Help(language)
	case SourceBody, SourceV1:
		return v.body.HelpOrDescription(language)
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
	case SourceBody, SourceV1:
		return v.body.Example
	}
	return ""
}

// LegacyHasChildren returns true if this parameter has sub-parameters
// that the old CLI would expose via --X.1.Field syntax.
func (v *LegacyParameterView) LegacyHasChildren() bool {
	if v.source == SourceBody || v.source == SourceV1 {
		return len(v.body.SubParameters) > 0
	}
	if v.source == SourceField {
		// The legacy metadata generator projected only the fields directly below
		// a top-level RepeatList. A nested RepeatList was emitted as a leaf even
		// when the canonical schema describes object fields below it. Preserve
		// that truncation so PascalCase flags keep the old --X.1.Y.1 shape.
		return false
	}
	if v.source != SourceCanonical {
		return false
	}
	p := v.canonical
	return p.Type == "array" &&
		p.ParamStyle != "flat" &&
		len(objectShapeFields(p.Element)) > 0
}

// IsLegacyRepeatList returns true if this parameter behaves as a RepeatList
// in old CLI semantics (accepts --X.1 syntax).
func (v *LegacyParameterView) IsLegacyRepeatList() bool {
	if v.source == SourceBody || v.source == SourceV1 {
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
// For Canonical parameters, returns the object fields under element.
// For V1 parameters, returns sub_parameters as-is.
func (v *LegacyParameterView) LegacyChildren() []*LegacyParameterView {
	var children []*LegacyParameterView

	switch v.source {
	case SourceCanonical:
		if !v.LegacyHasChildren() {
			return nil
		}
		topLoc := v.canonical.Location
		fields := objectShapeFields(v.canonical.Element)
		for i := range fields {
			children = append(children, NewFieldView(&fields[i], topLoc))
		}
	case SourceField:
		// Nested canonical fields were not present in generated legacy metadata.
		return nil
	case SourceBody, SourceV1:
		for i := range v.body.SubParameters {
			child := NewBodyView(&v.body.SubParameters[i])
			// Sub-parameters are NOT top-level; form -> FormData, not Body
			child.isTopBody = false
			children = append(children, child)
		}
	}

	return children
}

// objectShapeFields returns fields only when the recursive shape is an object.
func objectShapeFields(shape *TypeShape) []Field {
	if shape == nil || shape.Type != "object" {
		return nil
	}
	return shape.Fields
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
	case "domain":
		return "Domain"
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
	case "domain":
		return "Domain"
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

// isBodyRepeatList returns true if a V1 parameter behaves as a RepeatList.
// v1_body_parameters use unified lowercase types with param_style (spec 3.2).
func isBodyRepeatList(b *V1Parameter) bool {
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
// If v1_parameters exists, it is the complete V1 parameter list and no other
// parameter source is merged. Otherwise, applies v1_body_parameters three-state
// logic and deduplication.
func (api *API) LegacyTopLevelParameters() []*LegacyParameterView {
	if api.V1Parameters != nil {
		result := make([]*LegacyParameterView, 0, len(*api.V1Parameters))
		seen := make(map[string]bool)
		for i := range *api.V1Parameters {
			b := &(*api.V1Parameters)[i]
			if excludedParamNames[b.Name] {
				continue
			}
			if seen[b.Name] {
				continue
			}
			seen[b.Name] = true

			v := NewV1View(b)
			v.resolvedPosition = legacyPosition(b.Position)
			for j := range api.Parameters {
				if api.Parameters[j].RawName == b.Name {
					v.isWildcard = api.Parameters[j].IsWildcard
					break
				}
			}
			result = append(result, v)
		}

		sort.SliceStable(result, func(i, j int) bool {
			return result[i].LegacyName() < result[j].LegacyName()
		})

		return result
	}

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

// IsTopLevelBody reports whether this view is a top-level v1 body parameter.
func (v *LegacyParameterView) IsTopLevelBody() bool {
	return v.source == SourceBody && v.isTopBody
}

// LegacyBodyFields returns the canonical body/form parameters in declaration
// order, excluding protocol-level names. It is intended for help display of
// the fields carried inside a top-level --body parameter; it does not affect
// parameter lookup or execution semantics.
func (api *API) LegacyBodyFields() []*Parameter {
	if api.V1Parameters != nil {
		return nil
	}

	var fields []*Parameter
	for i := range api.Parameters {
		p := &api.Parameters[i]
		if excludedParamNames[p.RawName] {
			continue
		}
		if isBodyLocation(p.Location) {
			fields = append(fields, p)
		}
	}
	return fields
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

	if api.V1Parameters != nil {
		seen := make(map[string]bool)
		for i := range *api.V1Parameters {
			b := &(*api.V1Parameters)[i]
			if excludedParamNames[b.Name] {
				continue
			}
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

		return formatMissingRequiredParameters(missing)
	}

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

	return formatMissingRequiredParameters(missing)
}

func formatMissingRequiredParameters(missing []string) error {
	if len(missing) == 0 {
		return nil
	}

	s := ""
	for _, name := range missing {
		s += "\n  --" + name
	}
	return fmt.Errorf("required parameters not assigned: %s", s)
}
