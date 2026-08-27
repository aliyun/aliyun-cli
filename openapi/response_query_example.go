package openapi

import (
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
	document, err := parseHelpResponseSchema(input)
	if err != nil {
		return "", err
	}
	if document.root == nil || responseNodeIsArray(document, document.root, make(map[string]bool)) {
		return "", nil
	}

	collector := responseArrayCollector{document: document}
	collector.walk(document.root, nil, make(map[string]bool))
	if len(collector.paths) == 0 {
		return "", nil
	}

	explicitPath := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(paginationCollectionPath), "$."))
	if explicitPath != "" {
		for _, candidate := range collector.paths {
			if explicitPath == candidate.rawPath || explicitPath == candidate.queryPath {
				return candidate.queryPath, nil
			}
		}
	}
	for _, candidate := range collector.paths {
		if candidate.paginationSibling {
			return candidate.queryPath, nil
		}
	}
	for _, candidate := range collector.paths {
		if responseArrayMatchesAPIResource(candidate, apiName) {
			return candidate.queryPath, nil
		}
	}
	for _, candidate := range collector.paths {
		if isCommonResponseArrayName(candidate.segments[len(candidate.segments)-1].name) {
			return candidate.queryPath, nil
		}
	}
	return collector.paths[0].queryPath, nil
}

// BuildResponseQueryExample selects an array and renders style-preserving,
// shell-safe local commands. Invalid or scalar-only contexts omit the example.
func BuildResponseQueryExample(context ResponseQueryContext) (*ResponseQueryExample, error) {
	path, err := SelectResponseArrayPath(
		context.Document,
		context.API,
		context.PaginationCollectionPath,
	)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
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
					})
				}
				c.walk(property.node, propertyPath, activeRefs)
			}
		case "items":
			c.walk(node.items, path, activeRefs)
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
	name, target := document.resolveRef(node.ref)
	if target == nil || visited[name] {
		return false
	}
	visited[name] = true
	return responseNodeIsArray(document, target, visited)
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

	pathTokens := make([]string, 0, len(candidate.segments))
	for _, segment := range candidate.segments {
		pathTokens = append(pathTokens, singularResponseTokens(splitHelpSearchTokens(segment.name))...)
	}
	if len(resourceTokens) > len(pathTokens) {
		return false
	}
	for start := 0; start+len(resourceTokens) <= len(pathTokens); start++ {
		matched := true
		for index := range resourceTokens {
			if resourceTokens[index] != pathTokens[start+index] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
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
