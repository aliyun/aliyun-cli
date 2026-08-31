package help

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type ResponseQueryContext struct {
	Document                 ResponseDocumentation
	PaginationCollectionPath string
	Product                  string
	API                      string
	APIVersion               string
	RequiredFlags            []string
}

type arrayCandidate struct {
	names             []string
	arrays            []bool
	node              map[string]any
	paginationSibling bool
	order             int
}

// BuildResponseQueryExample picks a useful response array and emits local,
// style-preserving schema and JMESPath command examples.
func BuildResponseQueryExample(context ResponseQueryContext) (*QueryExample, error) {
	root, err := decodeJSONObject(context.Document.Schema)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, nil
	}
	components := map[string]map[string]any{}
	for name, raw := range context.Document.Components {
		component, decodeErr := decodeJSONObject(raw)
		if decodeErr != nil {
			return nil, decodeErr
		}
		components[name] = component
	}
	if schemaNodeIsArray(root, components, map[string]bool{}) {
		return nil, nil
	}
	var candidates []arrayCandidate
	collectArrays(root, nil, nil, components, map[string]bool{}, &candidates)
	if len(candidates) == 0 {
		return nil, nil
	}
	selected := selectArray(candidates, context.API, firstNonEmpty(context.PaginationCollectionPath, context.Document.PaginationCollectionPath))
	path := candidateQueryPath(selected)
	if fields := projectionFields(selected.node, components, context.API); len(fields) > 0 {
		parts := make([]string, 0, len(fields))
		for _, field := range fields {
			id := jmesIdentifier(field)
			parts = append(parts, id+":"+id)
		}
		path += "[*].{" + strings.Join(parts, ",") + "}"
	}
	product := strings.TrimSpace(context.Product)
	api := kebabCase(context.API)
	if !safeToken(product) || !safeToken(api) || context.APIVersion != "" && !safeToken(context.APIVersion) {
		return nil, nil
	}
	schema := []string{"aliyun", "help", product, api}
	query := []string{"aliyun", product, api}
	if context.APIVersion != "" {
		schema = append(schema, "--api-version", context.APIVersion)
		query = append(query, "--api-version", context.APIVersion)
	}
	schema = append(schema, "--cli-section", "response")
	for _, flag := range context.RequiredFlags {
		if safeToken(strings.TrimLeft(flag, "-")) {
			query = append(query, "--"+strings.TrimLeft(flag, "-"), "<value>")
		}
	}
	query = append(query, "--cli-query", shellQuote(path))
	return &QueryExample{
		Path: path, SchemaCommand: strings.Join(schema, " "), QueryCommand: strings.Join(query, " "),
	}, nil
}

func collectArrays(node map[string]any, names []string, arrays []bool, components map[string]map[string]any, refs map[string]bool, output *[]arrayCandidate) {
	node = resolveNode(node, components, refs)
	if node == nil {
		return
	}
	properties, _ := node["properties"].(map[string]any)
	paginated := hasPaginationProperty(properties)
	propertyNames := make([]string, 0, len(properties))
	for name := range properties {
		propertyNames = append(propertyNames, name)
	}
	sort.Strings(propertyNames)
	for _, name := range propertyNames {
		raw := properties[name]
		child, _ := raw.(map[string]any)
		isArray := schemaNodeIsArray(child, components, map[string]bool{})
		childNames := append(append([]string(nil), names...), name)
		childArrays := append(append([]bool(nil), arrays...), isArray)
		if isArray {
			*output = append(*output, arrayCandidate{
				names: childNames, arrays: childArrays, node: child,
				paginationSibling: paginated, order: len(*output),
			})
		}
		collectArrays(child, childNames, childArrays, components, refs, output)
	}
	if items, ok := node["items"].(map[string]any); ok {
		collectArrays(items, names, arrays, components, refs, output)
	}
	for _, keyword := range []string{"allOf", "oneOf", "anyOf"} {
		if branches, ok := node[keyword].([]any); ok {
			for _, raw := range branches {
				branch, _ := raw.(map[string]any)
				collectArrays(branch, names, arrays, components, refs, output)
			}
		}
	}
}

func selectArray(candidates []arrayCandidate, apiName, explicit string) arrayCandidate {
	explicit = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(explicit), "$."), "[]")
	for _, candidate := range candidates {
		if strings.Join(candidate.names, ".") == explicit || candidateQueryPath(candidate) == explicit {
			return candidate
		}
	}
	for _, candidate := range candidates {
		if candidate.paginationSibling {
			return candidate
		}
	}
	resource := resourceTokens(apiName)
	for _, candidate := range candidates {
		for _, name := range candidate.names {
			if equalTokens(singularTokens(splitTokens(name)), resource) {
				return candidate
			}
		}
	}
	for _, candidate := range candidates {
		last := strings.Join(splitTokens(candidate.names[len(candidate.names)-1]), "")
		switch last {
		case "items", "records", "results", "list":
			return candidate
		}
	}
	return candidates[0]
}

func candidateQueryPath(candidate arrayCandidate) string {
	parts := make([]string, len(candidate.names))
	for i, name := range candidate.names {
		parts[i] = jmesIdentifier(name)
		if candidate.arrays[i] && i < len(candidate.names)-1 {
			parts[i] += "[]"
		}
	}
	return strings.Join(parts, ".")
}

func projectionFields(array map[string]any, components map[string]map[string]any, apiName string) []string {
	array = resolveNode(array, components, map[string]bool{})
	items, _ := array["items"].(map[string]any)
	items = resolveNode(items, components, map[string]bool{})
	if items == nil {
		return nil
	}
	properties := mergedProperties(items, components)
	resource := resourceTokens(apiName)
	type scored struct {
		name  string
		score int
	}
	var fields []scored
	for name, raw := range properties {
		node, _ := raw.(map[string]any)
		node = resolveNode(node, components, map[string]bool{})
		typ, _ := node["type"].(string)
		if typ != "string" && typ != "integer" && typ != "number" && typ != "boolean" {
			continue
		}
		tokens := splitTokens(name)
		score := 10
		if len(tokens) > 0 {
			switch tokens[len(tokens)-1] {
			case "id":
				score = 50
				if resourceField(tokens, resource) {
					score = 100
				}
			case "name":
				score = 45
				if resourceField(tokens, resource) {
					score = 90
				}
			case "status", "state", "phase":
				score = 60
			}
		}
		fields = append(fields, scored{name: name, score: score})
	}
	sort.SliceStable(fields, func(i, j int) bool {
		if fields[i].score != fields[j].score {
			return fields[i].score > fields[j].score
		}
		return fields[i].name < fields[j].name
	})
	if len(fields) > 3 {
		fields = fields[:3]
	}
	result := make([]string, len(fields))
	for i := range fields {
		result[i] = fields[i].name
	}
	return result
}

func mergedProperties(node map[string]any, components map[string]map[string]any) map[string]any {
	result := map[string]any{}
	if properties, ok := node["properties"].(map[string]any); ok {
		for name, raw := range properties {
			result[name] = raw
		}
	}
	if branches, ok := node["allOf"].([]any); ok {
		for _, raw := range branches {
			branch, _ := raw.(map[string]any)
			branch = resolveNode(branch, components, map[string]bool{})
			for name, child := range mergedProperties(branch, components) {
				result[name] = child
			}
		}
	}
	return result
}

func resolveNode(node map[string]any, components map[string]map[string]any, visited map[string]bool) map[string]any {
	if node == nil {
		return nil
	}
	ref, _ := node["$ref"].(string)
	name, ok := componentName(ref)
	if !ok || visited[name] || components[name] == nil {
		return node
	}
	visited[name] = true
	return resolveNode(components[name], components, visited)
}

func schemaNodeIsArray(node map[string]any, components map[string]map[string]any, visited map[string]bool) bool {
	node = resolveNode(node, components, visited)
	if node == nil {
		return false
	}
	if typ, _ := node["type"].(string); strings.EqualFold(typ, "array") {
		return true
	}
	_, hasItems := node["items"]
	return hasItems
}

func hasPaginationProperty(properties map[string]any) bool {
	for name := range properties {
		switch strings.Join(splitTokens(name), "") {
		case "nexttoken", "totalcount", "pagenumber", "pagesize", "pagecount", "currentpage", "nextpagetoken", "totalnumber":
			return true
		}
	}
	return false
}

func resourceTokens(apiName string) []string {
	tokens := splitTokens(apiName)
	for len(tokens) > 0 {
		switch tokens[0] {
		case "get", "list", "describe", "query", "search":
			tokens = tokens[1:]
		default:
			return singularTokens(tokens)
		}
	}
	return nil
}

func singularTokens(tokens []string) []string {
	result := make([]string, len(tokens))
	for i, token := range tokens {
		switch {
		case len(token) > 3 && strings.HasSuffix(token, "ies"):
			token = strings.TrimSuffix(token, "ies") + "y"
		case len(token) > 1 && strings.HasSuffix(token, "s") && !strings.HasSuffix(token, "ss"):
			token = strings.TrimSuffix(token, "s")
		}
		result[i] = token
	}
	return result
}

func equalTokens(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func resourceField(field, resource []string) bool {
	if len(field) != len(resource)+1 {
		return false
	}
	return equalTokens(field[:len(resource)], resource)
}

func jmesIdentifier(value string) string {
	for i, r := range value {
		valid := r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9'
		if !valid {
			return strconv.Quote(value)
		}
	}
	return value
}

func safeToken(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-_.", r)) {
			return false
		}
	}
	return true
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
