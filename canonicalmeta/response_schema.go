package canonicalmeta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const localSchemaRefPrefix = "#/components/schemas/"

// ResponseSchemaDocument is the renderer-independent projection of one
// Canonical API's selected successful response body. Schema and Components
// retain JSON encoding so callers can render or inspect them without losing
// OpenAPI fields such as $ref. Components maps reachable schema names to their
// encoded definitions; callers that expose OpenAPI-shaped JSON can wrap it as
// components.schemas.
type ResponseSchemaDocument struct {
	StatusCode  string                     `json:"statusCode,omitempty"`
	ContentType string                     `json:"contentType,omitempty"`
	Schema      json.RawMessage            `json:"schema,omitempty"`
	Components  map[string]json.RawMessage `json:"components,omitempty"`
	Warnings    []string                   `json:"warnings,omitempty"`
}

// ResponseSectionDocument is the lossless response Help view. Responses keeps
// every declared response status/content object, while Components contains
// only schemas reachable from those responses. References are preserved and
// never recursively inlined.
type ResponseSectionDocument struct {
	Responses  json.RawMessage            `json:"responses,omitempty"`
	Components map[string]json.RawMessage `json:"components,omitempty"`
	Warnings   []string                   `json:"warnings,omitempty"`
}

// HasSchema reports whether the selected successful response has a body
// schema. A false result is the stable no-schema state and is not an error.
func (d ResponseSchemaDocument) HasSchema() bool {
	return hasJSONValue(d.Schema)
}

// ResponseSchemaError identifies malformed lazy Canonical response metadata.
// Field is either "responses" or "components" so callers can distinguish the
// local metadata source while retaining the underlying JSON error via Unwrap.
type ResponseSchemaError struct {
	Field string
	Err   error
}

func (e *ResponseSchemaError) Error() string {
	return fmt.Sprintf("parse canonical response %s failed: %v", e.Field, e.Err)
}

func (e *ResponseSchemaError) Unwrap() error {
	return e.Err
}

// ResponseSchema selects and projects the API's successful response body.
// Components are decoded only when a local schema reference makes them
// relevant, preserving lazy behavior for ordinary execution and inline
// response schemas.
func (a *API) ResponseSchema() (ResponseSchemaDocument, error) {
	var document ResponseSchemaDocument
	if a == nil || !hasJSONValue(a.Responses) {
		return document, nil
	}

	var responses map[string]json.RawMessage
	if err := json.Unmarshal(a.Responses, &responses); err != nil {
		return document, responseSchemaError("responses", err)
	}

	statusCode, encodedResponse := selectSuccessfulResponse(responses)
	if statusCode == "" {
		return document, nil
	}
	document.StatusCode = statusCode

	var response responseObject
	if err := json.Unmarshal(encodedResponse, &response); err != nil {
		return ResponseSchemaDocument{}, responseSchemaError("responses", err)
	}

	schema, contentType, err := selectResponseBodySchema(response)
	if err != nil {
		return ResponseSchemaDocument{}, responseSchemaError("responses", err)
	}
	if !hasJSONValue(schema) {
		return document, nil
	}
	document.ContentType = contentType
	document.Schema = cloneRawMessage(schema)

	rootRefs, rootWarnings, err := schemaReferences(schema)
	if err != nil {
		return ResponseSchemaDocument{}, responseSchemaError("responses", err)
	}
	warnings := newWarningCollector()
	warnings.addAll(rootWarnings)
	if len(rootRefs) == 0 {
		document.Warnings = warnings.values()
		return document, nil
	}

	schemas, err := decodeComponentSchemas(a.Components)
	if err != nil {
		return ResponseSchemaDocument{}, responseSchemaError("components", err)
	}

	reachable := make(map[string]json.RawMessage)
	states := make(map[string]componentVisitState)
	var visit func(string) error
	visit = func(name string) error {
		ref := localSchemaRefPrefix + escapeJSONPointerSegment(name)
		switch states[name] {
		case componentVisiting:
			warnings.add(fmt.Sprintf("cyclic schema reference %q was preserved", ref))
			return nil
		case componentVisited:
			return nil
		}

		encodedSchema, ok := schemas[name]
		if !ok {
			warnings.add(fmt.Sprintf("schema reference %q was not found in components.schemas", ref))
			return nil
		}

		states[name] = componentVisiting
		reachable[name] = cloneRawMessage(encodedSchema)
		refs, refWarnings, err := schemaReferences(encodedSchema)
		if err != nil {
			return err
		}
		warnings.addAll(refWarnings)
		for _, dependency := range refs {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		states[name] = componentVisited
		return nil
	}

	for _, name := range rootRefs {
		if err := visit(name); err != nil {
			return ResponseSchemaDocument{}, responseSchemaError("components", err)
		}
	}

	if len(reachable) > 0 {
		document.Components = reachable
	}
	document.Warnings = warnings.values()
	return document, nil
}

// ResponseSection projects all response metadata for explicit Response Help.
// It is separate from ResponseSchema, whose purpose is selecting one success
// body for query guidance.
func (a *API) ResponseSection() (ResponseSectionDocument, error) {
	var document ResponseSectionDocument
	if a == nil || !hasJSONValue(a.Responses) {
		return document, nil
	}

	var responses map[string]json.RawMessage
	if err := json.Unmarshal(a.Responses, &responses); err != nil {
		return document, responseSchemaError("responses", err)
	}
	document.Responses = cloneRawMessage(a.Responses)

	refs, refWarnings, err := schemaReferences(a.Responses)
	if err != nil {
		return ResponseSectionDocument{}, responseSchemaError("responses", err)
	}
	warnings := newWarningCollector()
	warnings.addAll(refWarnings)
	if len(refs) == 0 {
		document.Warnings = warnings.values()
		return document, nil
	}

	schemas, err := decodeComponentSchemas(a.Components)
	if err != nil {
		return ResponseSectionDocument{}, responseSchemaError("components", err)
	}
	reachable := make(map[string]json.RawMessage)
	states := make(map[string]componentVisitState)
	var visit func(string) error
	visit = func(name string) error {
		ref := localSchemaRefPrefix + escapeJSONPointerSegment(name)
		switch states[name] {
		case componentVisiting:
			warnings.add(fmt.Sprintf("cyclic schema reference %q was preserved", ref))
			return nil
		case componentVisited:
			return nil
		}

		schema, ok := schemas[name]
		if !ok {
			warnings.add(fmt.Sprintf("schema reference %q was not found in components.schemas", ref))
			return nil
		}
		states[name] = componentVisiting
		reachable[name] = cloneRawMessage(schema)
		dependencies, dependencyWarnings, dependencyErr := schemaReferences(schema)
		if dependencyErr != nil {
			return dependencyErr
		}
		warnings.addAll(dependencyWarnings)
		for _, dependency := range dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		states[name] = componentVisited
		return nil
	}

	for _, name := range refs {
		if err := visit(name); err != nil {
			return ResponseSectionDocument{}, responseSchemaError("components", err)
		}
	}
	if len(reachable) > 0 {
		document.Components = reachable
	}
	document.Warnings = warnings.values()
	return document, nil
}

type responseObject struct {
	Content map[string]json.RawMessage `json:"content"`
	Schema  json.RawMessage            `json:"schema"`
}

type mediaTypeObject struct {
	Schema json.RawMessage `json:"schema"`
}

func selectSuccessfulResponse(responses map[string]json.RawMessage) (string, json.RawMessage) {
	if response, ok := responses["200"]; ok {
		return "200", response
	}

	type statusResponse struct {
		code     int
		status   string
		response json.RawMessage
	}
	var successful []statusResponse
	for status, response := range responses {
		if len(status) != 3 || status[0] != '2' {
			continue
		}
		code, err := strconv.Atoi(status)
		if err != nil || code == 200 {
			continue
		}
		successful = append(successful, statusResponse{code: code, status: status, response: response})
	}
	if len(successful) > 0 {
		sort.Slice(successful, func(i, j int) bool {
			if successful[i].code != successful[j].code {
				return successful[i].code < successful[j].code
			}
			return successful[i].status < successful[j].status
		})
		selected := successful[0]
		return selected.status, selected.response
	}

	if response, ok := responses["default"]; ok {
		return "default", response
	}
	return "", nil
}

func selectResponseBodySchema(response responseObject) (json.RawMessage, string, error) {
	contentTypes := make([]string, 0, len(response.Content))
	for contentType := range response.Content {
		contentTypes = append(contentTypes, contentType)
	}
	sort.Slice(contentTypes, func(i, j int) bool {
		iRank := contentTypeRank(contentTypes[i])
		jRank := contentTypeRank(contentTypes[j])
		if iRank != jRank {
			return iRank < jRank
		}
		return contentTypes[i] < contentTypes[j]
	})

	for _, contentType := range contentTypes {
		var media mediaTypeObject
		if err := json.Unmarshal(response.Content[contentType], &media); err != nil {
			return nil, "", fmt.Errorf("parse media type %q failed: %w", contentType, err)
		}
		if hasJSONValue(media.Schema) {
			return media.Schema, contentType, nil
		}
	}

	if hasJSONValue(response.Schema) {
		return response.Schema, "", nil
	}
	return nil, "", nil
}

func contentTypeRank(contentType string) int {
	if contentType == "application/json" {
		return 0
	}
	if strings.HasPrefix(contentType, "application/") && strings.HasSuffix(contentType, "+json") {
		return 1
	}
	return 2
}

type rawComponents struct {
	Schemas map[string]json.RawMessage `json:"schemas"`
}

func decodeComponentSchemas(encoded json.RawMessage) (map[string]json.RawMessage, error) {
	if !hasJSONValue(encoded) {
		return nil, nil
	}
	var components rawComponents
	if err := json.Unmarshal(encoded, &components); err != nil {
		return nil, err
	}
	return components.Schemas, nil
}

func schemaReferences(encoded json.RawMessage) ([]string, []string, error) {
	var node any
	if err := json.Unmarshal(encoded, &node); err != nil {
		return nil, nil, err
	}

	refs := make([]string, 0)
	warnings := newWarningCollector()
	walkJSONNode(node, func(ref string) {
		if name, ok := localComponentName(ref); ok {
			refs = append(refs, name)
			return
		}
		warnings.add(fmt.Sprintf("non-local schema reference %q was not resolved", ref))
	})
	return refs, warnings.values(), nil
}

func walkJSONNode(node any, onRef func(string)) {
	switch value := node.(type) {
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			child := value[key]
			if key == "$ref" {
				if ref, ok := child.(string); ok {
					onRef(ref)
				}
			}
			walkJSONNode(child, onRef)
		}
	case []any:
		for _, child := range value {
			walkJSONNode(child, onRef)
		}
	}
}

func localComponentName(ref string) (string, bool) {
	if !strings.HasPrefix(ref, localSchemaRefPrefix) {
		return "", false
	}
	remainder := strings.TrimPrefix(ref, localSchemaRefPrefix)
	if slash := strings.IndexByte(remainder, '/'); slash >= 0 {
		remainder = remainder[:slash]
	}
	if remainder == "" {
		return "", false
	}
	return unescapeJSONPointerSegment(remainder), true
}

func unescapeJSONPointerSegment(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~1", "/"), "~0", "~")
}

func escapeJSONPointerSegment(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

func hasJSONValue(value json.RawMessage) bool {
	trimmed := bytes.TrimSpace(value)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null"))
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

func responseSchemaError(field string, err error) *ResponseSchemaError {
	return &ResponseSchemaError{Field: field, Err: err}
}

type componentVisitState uint8

const (
	componentVisiting componentVisitState = iota + 1
	componentVisited
)

type warningCollector struct {
	seen    map[string]struct{}
	entries []string
}

func newWarningCollector() *warningCollector {
	return &warningCollector{seen: make(map[string]struct{})}
}

func (w *warningCollector) add(value string) {
	if _, ok := w.seen[value]; ok {
		return
	}
	w.seen[value] = struct{}{}
	w.entries = append(w.entries, value)
}

func (w *warningCollector) addAll(values []string) {
	for _, value := range values {
		w.add(value)
	}
}

func (w *warningCollector) values() []string {
	if len(w.entries) == 0 {
		return nil
	}
	return append([]string(nil), w.entries...)
}
