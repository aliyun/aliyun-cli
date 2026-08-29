package openapi

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ResponseCommandStyle controls the API token used in generated Help/query
// commands. Camel is the existing machine-Help spelling for PascalCase.
type ResponseCommandStyle string

const (
	ResponseCommandStylePascal ResponseCommandStyle = "pascal"
	ResponseCommandStyleCamel  ResponseCommandStyle = "camel"
	ResponseCommandStyleKebab  ResponseCommandStyle = "kebab"
)

// ResponseQueryContext contains only local schema and command identity. An
// APIVersion is included in generated commands only when explicitly supplied.
type ResponseQueryContext struct {
	Document                 HelpResponseSchema
	PaginationCollectionPath string
	Product                  string
	API                      string
	APIVersion               string
	Style                    ResponseCommandStyle
	// RequiredFlags are the API's required parameters in the target command
	// style's flag form. The query example embeds them as placeholders so
	// copying it cannot fail with missing-parameter errors.
	RequiredFlags []string
}

// ResponseQueryExample is shared by text and JSON Help renderers. Request Help
// can show both commands; Response Help can show QueryCommand only.
type ResponseQueryExample struct {
	Path          string `json:"path"`
	SchemaCommand string `json:"schemaCommand"`
	QueryCommand  string `json:"queryCommand"`
}

// SelectResponseArrayPath chooses one representative array by explicit
// pagination path, pagination siblings, API resource name, common result name,
// then original schema declaration order.
func SelectResponseArrayPath(input HelpResponseSchema, apiName, paginationCollectionPath string) (string, error) {
	candidate, _, err := selectResponseArrayCandidate(input, apiName, paginationCollectionPath)
	if err != nil || candidate == nil {
		return "", err
	}
	return candidate.queryPath, nil
}

// selectResponseArrayCandidate returns the representative array path plus the
// parsed schema document, so callers can inspect the array's item shape.
func selectResponseArrayCandidate(input HelpResponseSchema, apiName, paginationCollectionPath string) (*responseArrayPath, *responseSchemaDocument, error) {
	document, err := parseHelpResponseSchema(input)
	if err != nil {
		return nil, nil, err
	}
	if document.root == nil || responseNodeIsArray(document, document.root, make(map[string]bool)) {
		return nil, document, nil
	}

	collector := responseArrayCollector{document: document}
	collector.walk(document.root, nil, make(map[string]bool))
	if len(collector.paths) == 0 {
		return nil, document, nil
	}

	explicitPath := normalizeResponseCollectionPath(paginationCollectionPath)
	if explicitPath != "" {
		for i := range collector.paths {
			candidate := &collector.paths[i]
			if explicitPath == candidate.rawPath || explicitPath == candidate.queryPath {
				return candidate, document, nil
			}
		}
	}
	for i := range collector.paths {
		if collector.paths[i].paginationSibling {
			return &collector.paths[i], document, nil
		}
	}
	for i := range collector.paths {
		candidate := &collector.paths[i]
		if responseArrayMatchesAPIResource(*candidate, apiName) {
			return candidate, document, nil
		}
	}
	for i := range collector.paths {
		candidate := &collector.paths[i]
		if isCommonResponseArrayName(candidate.segments[len(candidate.segments)-1].name) {
			return candidate, document, nil
		}
	}
	return &collector.paths[0], document, nil
}

func normalizeResponseCollectionPath(path string) string {
	path = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(path), "$."))
	return strings.TrimSpace(strings.TrimSuffix(path, "[]"))
}

// BuildResponseQueryExample selects an array and renders style-preserving,
// shell-safe local commands. Invalid or scalar-only contexts omit the example.
// When the array's items are objects with scalar properties, the query
// projects the first few fields instead of returning every full object.
func BuildResponseQueryExample(context ResponseQueryContext) (*ResponseQueryExample, error) {
	candidate, document, err := selectResponseArrayCandidate(
		context.Document,
		context.API,
		context.PaginationCollectionPath,
	)
	if err != nil {
		return nil, err
	}
	if candidate == nil {
		return nil, nil
	}
	path := candidate.queryPath
	if fields := responseArrayProjectionFields(document, candidate.arrayNode, context.API); len(fields) > 0 {
		path = responseQueryProjectionPath(path, fields)
	}

	product := strings.TrimSpace(context.Product)
	api, ok := responseAPIForStyle(context.API, context.Style)
	version := strings.TrimSpace(context.APIVersion)
	if !ok || !safeResponseCommandToken(product) || !safeResponseCommandToken(api) || (version != "" && !safeResponseCommandToken(version)) {
		return nil, nil
	}

	schemaParts := []string{"aliyun", "help", product, api}
	queryParts := []string{"aliyun", product, api}
	if version != "" {
		schemaParts = append(schemaParts, "--api-version", version)
		queryParts = append(queryParts, "--api-version", version)
	}
	schemaParts = append(schemaParts, "--cli-section", "response")
	for _, flag := range context.RequiredFlags {
		if safeResponseCommandToken(strings.TrimLeft(flag, "-")) {
			queryParts = append(queryParts, flag, "<value>")
		}
	}
	queryParts = append(queryParts, "--cli-query", shellSingleQuote(path))

	return &ResponseQueryExample{
		Path:          path,
		SchemaCommand: strings.Join(schemaParts, " "),
		QueryCommand:  strings.Join(queryParts, " "),
	}, nil
}

type responsePathSegment struct {
	name  string
	array bool
}

type responseArrayPath struct {
	segments          []responsePathSegment
	rawPath           string
	queryPath         string
	paginationSibling bool
	// arrayNode is the schema node of the array property itself; its items
	// shape drives the field projection.
	arrayNode *responseSchemaNode
}

type responseArrayCollector struct {
	document *responseSchemaDocument
	paths    []responseArrayPath
}

func (c *responseArrayCollector) walk(node *responseSchemaNode, path []responsePathSegment, activeRefs map[string]bool) {
	if node == nil {
		return
	}
	for _, field := range node.fields {
		switch field.name {
		case "$ref":
			name, target := c.document.resolveRef(node.ref)
			if target == nil || activeRefs[name] {
				continue
			}
			activeRefs[name] = true
			c.walk(target, path, activeRefs)
			delete(activeRefs, name)
		case "properties":
			paginationSibling := responseNodeHasPaginationField(node)
			for _, property := range node.properties {
				isArray := responseNodeIsArray(c.document, property.node, make(map[string]bool))
				propertyPath := appendResponsePath(path, responsePathSegment{name: property.name, array: isArray})
				if isArray {
					c.paths = append(c.paths, responseArrayPath{
						segments:          propertyPath,
						rawPath:           rawResponsePath(propertyPath),
						queryPath:         queryResponsePath(propertyPath),
						paginationSibling: paginationSibling,
						arrayNode:         property.node,
					})
				}
				c.walk(property.node, propertyPath, activeRefs)
			}
		case "items":
			c.walk(node.items, path, activeRefs)
		case "allOf", "oneOf", "anyOf":
			for _, branch := range node.compositions[field.name] {
				c.walk(branch, path, activeRefs)
			}
		}
	}
}

func appendResponsePath(path []responsePathSegment, segment responsePathSegment) []responsePathSegment {
	result := make([]responsePathSegment, len(path)+1)
	copy(result, path)
	result[len(path)] = segment
	return result
}

func responseNodeIsArray(document *responseSchemaDocument, node *responseSchemaNode, visited map[string]bool) bool {
	if node == nil {
		return false
	}
	if responseSchemaNodeIsArray(node) {
		return true
	}
	for _, keyword := range []string{"allOf", "oneOf", "anyOf"} {
		for _, branch := range node.compositions[keyword] {
			if responseNodeIsArray(document, branch, visited) {
				return true
			}
		}
	}
	name, target := document.resolveRef(node.ref)
	if target == nil || visited[name] {
		return false
	}
	visited[name] = true
	if responseNodeIsArray(document, target, visited) {
		return true
	}
	return false
}

func responseNodeHasPaginationField(node *responseSchemaNode) bool {
	for _, property := range node.properties {
		name := strings.ToLower(strings.Join(splitHelpSearchTokens(property.name), ""))
		switch name {
		case "nexttoken", "totalcount", "pagenumber", "pagesize", "pagecount", "currentpage", "nextpagetoken", "totalnumber":
			return true
		}
	}
	return false
}

func rawResponsePath(path []responsePathSegment) string {
	parts := make([]string, 0, len(path))
	for _, segment := range path {
		parts = append(parts, segment.name)
	}
	return strings.Join(parts, ".")
}

func queryResponsePath(path []responsePathSegment) string {
	parts := make([]string, 0, len(path))
	for index, segment := range path {
		part := responseJMESPathIdentifier(segment.name)
		if segment.array && index < len(path)-1 {
			part += "[]"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ".")
}

// responseQueryProjectionFieldLimit caps the projected fields so the example
// stays a demonstration of the pattern rather than a full field list.
const responseQueryProjectionFieldLimit = 3

// responseArrayProjectionFields picks the most useful scalar properties of the
// array's item object (resolving $ref and allOf shapes) by semantic relevance
// rather than declaration order, so the projected query surfaces identity and
// status fields instead of arbitrary leading fields. Nil means the items carry
// no projectable scalar fields and the array path stays as-is.
func responseArrayProjectionFields(document *responseSchemaDocument, arrayNode *responseSchemaNode, apiName string) []string {
	if document == nil || arrayNode == nil {
		return nil
	}
	resolvedArray := resolveResponseSchemaNode(document, arrayNode, make(map[string]bool))
	if resolvedArray == nil {
		return nil
	}
	items := resolveResponseSchemaNode(document, resolvedArray.items, make(map[string]bool))
	if items == nil {
		return nil
	}
	resourceTokens := responseProjectionResourceTokens(apiName)

	type scoredField struct {
		name  string
		score int
	}
	var candidates []scoredField
	for _, property := range responseArrayItemProperties(document, items) {
		if !responseSchemaPropertyIsScalar(document, property.node) {
			continue
		}
		score := responseProjectionFieldScore(property.name, resourceTokens)
		if score > 0 {
			candidates = append(candidates, scoredField{name: property.name, score: score})
		}
	}
	// Descending by score; SliceStable keeps declaration order among ties so
	// output is deterministic.
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	limit := responseQueryProjectionFieldLimit
	if len(candidates) < limit {
		limit = len(candidates)
	}
	fields := make([]string, 0, limit)
	for _, candidate := range candidates[:limit] {
		fields = append(fields, candidate.name)
	}
	return fields
}

// responseProjectionResourceTokens extracts the resource noun from an API name
// (DescribeInstances -> [instance], ListUsers -> [user]) by dropping leading
// action verbs and singularizing, mirroring responseArrayMatchesAPIResource.
func responseProjectionResourceTokens(apiName string) []string {
	tokens := splitHelpSearchTokens(apiName)
	for len(tokens) > 0 && isResponseAPIActionToken(tokens[0]) {
		tokens = tokens[1:]
	}
	return singularResponseTokens(tokens)
}

// responseProjectionFieldScore ranks a scalar field for projection usefulness:
// identity fields that echo the API resource rank highest, then status, then
// generic id/name/type; timestamps and long-text fields are penalized because
// they are rarely the fields an agent projects for.
func responseProjectionFieldScore(name string, resourceTokens []string) int {
	tokens := splitHelpSearchTokens(name)
	if len(tokens) == 0 {
		return 0
	}
	last := tokens[len(tokens)-1]
	matchesResource := len(resourceTokens) > 0 && responseTokensContainAll(tokens, resourceTokens)

	score := 0
	switch last {
	case "id":
		if matchesResource {
			score = 100
		} else {
			score = 50
		}
	case "name":
		if matchesResource {
			score = 90
		} else {
			score = 45
		}
	case "status", "state", "phase":
		score = 60
	case "type", "category", "kind":
		if matchesResource {
			score = 55
		} else {
			score = 20
		}
	default:
		score = 10
	}

	for _, token := range tokens {
		switch token {
		case "time", "date", "timestamp", "at":
			score -= 60
		case "description", "comment", "comments", "reason", "message", "detail", "details", "remark", "remarks":
			score -= 40
		}
	}
	return score
}

// responseTokensContainAll reports whether tokens include every wanted token.
func responseTokensContainAll(tokens, wanted []string) bool {
	set := make(map[string]bool, len(tokens))
	for _, token := range tokens {
		set[token] = true
	}
	for _, want := range wanted {
		if !set[want] {
			return false
		}
	}
	return true
}

// responseQueryProjectionPath turns an array path into a JMESPath projection
// such as `Zones.Zone[*].{ZoneId:ZoneId,LocalName:LocalName}`; keys keep the
// original field names so the mapping is self-documenting.
func responseQueryProjectionPath(path string, fields []string) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		identifier := responseJMESPathIdentifier(field)
		parts = append(parts, identifier+":"+identifier)
	}
	return path + "[*].{" + strings.Join(parts, ",") + "}"
}

// resolveResponseSchemaNode unwraps $ref chains to the concrete schema node.
func resolveResponseSchemaNode(document *responseSchemaDocument, node *responseSchemaNode, visited map[string]bool) *responseSchemaNode {
	if node == nil {
		return nil
	}
	if node.ref == "" {
		return node
	}
	name, target := document.resolveRef(node.ref)
	if target == nil || visited[name] {
		return node
	}
	visited[name] = true
	return resolveResponseSchemaNode(document, target, visited)
}

// responseArrayItemProperties returns an object node's own properties plus
// those contributed by its allOf branches, in declaration order.
func responseArrayItemProperties(document *responseSchemaDocument, node *responseSchemaNode) []responseSchemaProperty {
	if node == nil {
		return nil
	}
	properties := append([]responseSchemaProperty(nil), node.properties...)
	for _, branch := range node.compositions["allOf"] {
		resolved := resolveResponseSchemaNode(document, branch, make(map[string]bool))
		if resolved != nil {
			properties = append(properties, resolved.properties...)
		}
	}
	return properties
}

func responseSchemaPropertyIsScalar(document *responseSchemaDocument, node *responseSchemaNode) bool {
	resolved := resolveResponseSchemaNode(document, node, make(map[string]bool))
	if resolved == nil {
		return false
	}
	switch resolved.typ {
	case "string", "integer", "number", "boolean":
		return true
	}
	return false
}

func responseJMESPathIdentifier(value string) string {
	if value != "" {
		valid := true
		for index, r := range value {
			if index == 0 {
				valid = r == '_' || isASCIIResponseLetter(r)
			} else {
				valid = r == '_' || isASCIIResponseLetter(r) || (r >= '0' && r <= '9')
			}
			if !valid {
				break
			}
		}
		if valid {
			return value
		}
	}
	return strconv.Quote(value)
}

func isASCIIResponseLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func responseArrayMatchesAPIResource(candidate responseArrayPath, apiName string) bool {
	resourceTokens := splitHelpSearchTokens(apiName)
	for len(resourceTokens) > 0 && isResponseAPIActionToken(resourceTokens[0]) {
		resourceTokens = resourceTokens[1:]
	}
	resourceTokens = singularResponseTokens(resourceTokens)
	if len(resourceTokens) == 0 {
		return false
	}

	for _, segment := range candidate.segments {
		segmentTokens := singularResponseTokens(splitHelpSearchTokens(segment.name))
		if equalResponseTokens(resourceTokens, segmentTokens) {
			return true
		}
	}
	return false
}

func equalResponseTokens(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func isResponseAPIActionToken(token string) bool {
	switch strings.ToLower(token) {
	case "get", "list", "describe", "query", "search":
		return true
	default:
		return false
	}
}

func singularResponseTokens(tokens []string) []string {
	result := make([]string, len(tokens))
	for index, token := range tokens {
		result[index] = singularResponseToken(token)
	}
	return result
}

func singularResponseToken(token string) string {
	token = strings.ToLower(token)
	switch {
	case len(token) > 3 && strings.HasSuffix(token, "ies"):
		return strings.TrimSuffix(token, "ies") + "y"
	case len(token) > 4 && (strings.HasSuffix(token, "ches") || strings.HasSuffix(token, "shes")):
		return strings.TrimSuffix(token, "es")
	case len(token) > 3 && (strings.HasSuffix(token, "xes") || strings.HasSuffix(token, "zes") || strings.HasSuffix(token, "sses")):
		return strings.TrimSuffix(token, "es")
	case len(token) > 1 && strings.HasSuffix(token, "s") && !strings.HasSuffix(token, "ss"):
		return strings.TrimSuffix(token, "s")
	default:
		return token
	}
}

func isCommonResponseArrayName(name string) bool {
	compact := strings.ToLower(strings.Join(splitHelpSearchTokens(name), ""))
	switch compact {
	case "items", "records", "results", "list":
		return true
	default:
		return false
	}
}

func responseAPIForStyle(api string, style ResponseCommandStyle) (string, bool) {
	api = strings.TrimSpace(api)
	if api == "" {
		return "", false
	}
	switch style {
	case ResponseCommandStylePascal, ResponseCommandStyleCamel:
		if strings.ContainsAny(api, "-_ ") {
			tokens := splitHelpSearchTokens(api)
			if len(tokens) == 0 {
				return "", false
			}
			var builder strings.Builder
			for _, token := range tokens {
				builder.WriteString(firstRuneUpper(token))
			}
			return builder.String(), true
		}
		return firstRuneUpper(api), true
	case ResponseCommandStyleKebab:
		tokens := splitHelpSearchTokens(api)
		if len(tokens) == 0 {
			return "", false
		}
		return strings.Join(tokens, "-"), true
	default:
		return "", false
	}
}

func safeResponseCommandToken(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

// firstRuneUpper preserves existing acronym/casing while making a lower-camel
// command usable as the Pascal-style command token.
func firstRuneUpper(value string) string {
	if value == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(value)
	return string(unicode.ToUpper(r)) + value[size:]
}
