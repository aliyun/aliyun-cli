package help

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type ResponseSchema struct {
	Schema     json.RawMessage
	Components map[string]json.RawMessage
}

type ResponseSchemaSearchResult struct {
	Paths      []string
	Schema     json.RawMessage
	Components map[string]json.RawMessage
	Result     Result
}

type rankedPath struct {
	path string
	rank int
}

// SearchResponseSchema searches names and bilingual title/description fields,
// then returns a minimal root tree and reachable local component projection.
func SearchResponseSchema(input ResponseSchema, keyword string, unlimited bool) (ResponseSchemaSearchResult, error) {
	query := searchText(keyword)
	empty := ResponseSchemaSearchResult{Components: map[string]json.RawMessage{}}
	if query.compact == "" {
		return empty, nil
	}
	root, err := decodeJSONObject(input.Schema)
	if err != nil {
		return empty, fmt.Errorf("parse response schema: %w", err)
	}
	components := make(map[string]map[string]any, len(input.Components))
	for name, raw := range input.Components {
		value, decodeErr := decodeJSONObject(raw)
		if decodeErr != nil {
			return empty, fmt.Errorf("parse response component %q: %w", name, decodeErr)
		}
		components[name] = value
	}
	var matches []rankedPath
	pathRanks := map[string]int{}
	walkResponseProperties(root, nil, components, query, map[string]bool{}, func(path string, rank int) {
		if old, ok := pathRanks[path]; !ok || rank < old {
			pathRanks[path] = rank
		}
	})
	for path, rank := range pathRanks {
		matches = append(matches, rankedPath{path: path, rank: rank})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].rank != matches[j].rank {
			return matches[i].rank < matches[j].rank
		}
		return strings.ToLower(matches[i].path) < strings.ToLower(matches[j].path)
	})
	total := len(matches)
	if !unlimited && len(matches) > searchResultLimit {
		matches = matches[:searchResultLimit]
	}
	if len(matches) == 0 {
		return empty, nil
	}
	trie := &pathTrie{children: map[string]*pathTrie{}}
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		paths = append(paths, match.path)
		trie.add(strings.Split(match.path, "."))
	}
	projectedComponents := map[string]any{}
	projected := projectSchemaNode(root, trie, components, projectedComponents, map[string]bool{})
	schemaRaw, err := json.Marshal(projected)
	if err != nil {
		return empty, err
	}
	componentRaw := make(map[string]json.RawMessage, len(projectedComponents))
	for name, value := range projectedComponents {
		raw, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return empty, marshalErr
		}
		componentRaw[name] = raw
	}
	return ResponseSchemaSearchResult{
		Paths: paths, Schema: schemaRaw, Components: componentRaw,
		Result: Result{Shown: len(paths), Total: total, Truncated: len(paths) < total},
	}, nil
}

func searchAPIs(apis []APISummary, keyword string) []APISummary {
	query := searchText(keyword)
	type match struct {
		value APISummary
		rank  int
	}
	var matches []match
	for _, api := range apis {
		rank := bestSearchRank(query, api.Name, api.Command)
		if rank == 0 && textContains(api.Title.EN+" "+api.Title.ZH+" "+api.Description.EN+" "+api.Description.ZH, query) {
			rank = 4
		}
		if rank > 0 {
			matches = append(matches, match{value: api, rank: rank})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].rank != matches[j].rank {
			return matches[i].rank < matches[j].rank
		}
		return matches[i].value.Name < matches[j].value.Name
	})
	result := make([]APISummary, len(matches))
	for i := range matches {
		result[i] = matches[i].value
	}
	return result
}

func searchParameters(parameters []Parameter, keyword string) []Parameter {
	query := searchText(keyword)
	type match struct {
		value Parameter
		rank  int
	}
	var matches []match
	for _, parameter := range parameters {
		rank := parameterSearchRank(parameter, query)
		if rank > 0 {
			matches = append(matches, match{value: parameter, rank: rank})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].rank != matches[j].rank {
			return matches[i].rank < matches[j].rank
		}
		return matches[i].value.Name < matches[j].value.Name
	})
	result := make([]Parameter, len(matches))
	for i := range matches {
		result[i] = matches[i].value
	}
	return result
}

func parameterSearchRank(parameter Parameter, query normalizedSearchText) int {
	names := append([]string{parameter.Name, parameter.RawName}, parameter.Options...)
	if rank := bestSearchRank(query, names...); rank > 0 {
		return rank
	}
	text := parameter.Help.EN + " " + parameter.Help.ZH + " " + parameter.Example +
		" " + strings.Join(parameter.Constraints.Enum, " ")
	if textContains(text, query) {
		return 4
	}
	for _, child := range parameter.Fields {
		if rank := parameterSearchRank(child, query); rank > 0 {
			return rank
		}
	}
	for _, child := range []*Parameter{parameter.Element, parameter.Value} {
		if child != nil {
			if rank := parameterSearchRank(*child, query); rank > 0 {
				return rank
			}
		}
	}
	return 0
}

func walkResponseProperties(node map[string]any, path []string, components map[string]map[string]any, query normalizedSearchText, refs map[string]bool, add func(string, int)) {
	if node == nil {
		return
	}
	if ref, _ := node["$ref"].(string); ref != "" {
		if name, ok := componentName(ref); ok && !refs[name] {
			refs[name] = true
			walkResponseProperties(components[name], path, components, query, refs, add)
			delete(refs, name)
		}
	}
	if properties, ok := node["properties"].(map[string]any); ok {
		for name, raw := range properties {
			child, _ := raw.(map[string]any)
			childPath := append(append([]string(nil), path...), name)
			rank := bestSearchRank(query, name, strings.Join(childPath, "."))
			if rank == 0 && child != nil && textContains(schemaSearchText(child), query) {
				rank = 4
			}
			if rank > 0 {
				add(strings.Join(childPath, "."), rank)
			}
			walkResponseProperties(child, childPath, components, query, refs, add)
		}
	}
	if items, ok := node["items"].(map[string]any); ok {
		walkResponseProperties(items, path, components, query, refs, add)
	}
	for _, keyword := range []string{"allOf", "oneOf", "anyOf"} {
		if branches, ok := node[keyword].([]any); ok {
			for _, raw := range branches {
				branch, _ := raw.(map[string]any)
				walkResponseProperties(branch, path, components, query, refs, add)
			}
		}
	}
}

func schemaSearchText(node map[string]any) string {
	var values []string
	for _, field := range []string{
		"title", "title_en", "title_zh", "x-title-en", "x-title-zh",
		"description", "description_en", "description_zh", "x-description-en", "x-description-zh",
	} {
		if value, ok := node[field].(string); ok {
			values = append(values, value)
		}
	}
	return strings.Join(values, " ")
}

type pathTrie struct {
	terminal bool
	children map[string]*pathTrie
}

func (t *pathTrie) add(parts []string) {
	if len(parts) == 0 {
		t.terminal = true
		return
	}
	child := t.children[parts[0]]
	if child == nil {
		child = &pathTrie{children: map[string]*pathTrie{}}
		t.children[parts[0]] = child
	}
	child.add(parts[1:])
}

func projectSchemaNode(node map[string]any, trie *pathTrie, components map[string]map[string]any, output map[string]any, refs map[string]bool) map[string]any {
	if node == nil {
		return nil
	}
	if trie == nil || trie.terminal {
		copy := cloneJSONMap(node)
		collectFullComponents(copy, components, output, refs)
		return copy
	}
	result := cloneJSONMap(node)
	if properties, ok := node["properties"].(map[string]any); ok {
		selected := map[string]any{}
		for name, childTrie := range trie.children {
			if raw, exists := properties[name]; exists {
				child, _ := raw.(map[string]any)
				selected[name] = projectSchemaNode(child, childTrie, components, output, refs)
			}
		}
		if len(selected) == 0 {
			delete(result, "properties")
		} else {
			result["properties"] = selected
			if required, ok := node["required"].([]any); ok {
				var filtered []any
				for _, value := range required {
					if name, ok := value.(string); ok {
						if _, keep := selected[name]; keep {
							filtered = append(filtered, name)
						}
					}
				}
				if len(filtered) > 0 {
					result["required"] = filtered
				} else {
					delete(result, "required")
				}
			}
		}
	}
	if items, ok := node["items"].(map[string]any); ok {
		result["items"] = projectSchemaNode(items, trie, components, output, refs)
	}
	if ref, _ := node["$ref"].(string); ref != "" {
		if name, ok := componentName(ref); ok && !refs[name] {
			refs[name] = true
			output[name] = projectSchemaNode(components[name], trie, components, output, refs)
			delete(refs, name)
		}
	}
	for _, keyword := range []string{"allOf", "oneOf", "anyOf"} {
		if branches, ok := node[keyword].([]any); ok {
			projected := make([]any, 0, len(branches))
			for _, raw := range branches {
				branch, _ := raw.(map[string]any)
				value := projectSchemaNode(branch, trie, components, output, refs)
				if value != nil {
					projected = append(projected, value)
				}
			}
			result[keyword] = projected
		}
	}
	return result
}

func collectFullComponents(value any, components map[string]map[string]any, output map[string]any, refs map[string]bool) {
	switch typed := value.(type) {
	case map[string]any:
		if ref, _ := typed["$ref"].(string); ref != "" {
			if name, ok := componentName(ref); ok && !refs[name] {
				if component := components[name]; component != nil {
					refs[name] = true
					copy := cloneJSONMap(component)
					output[name] = copy
					collectFullComponents(copy, components, output, refs)
					delete(refs, name)
				}
			}
		}
		for _, child := range typed {
			collectFullComponents(child, components, output, refs)
		}
	case []any:
		for _, child := range typed {
			collectFullComponents(child, components, output, refs)
		}
	}
}

func decodeJSONObject(raw json.RawMessage) (map[string]any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func cloneJSONMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	raw, _ := json.Marshal(value)
	var result map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	_ = decoder.Decode(&result)
	return result
}

func componentName(ref string) (string, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~"), true
}

type normalizedSearchText struct {
	raw     string
	compact string
	tokens  []string
}

func searchText(value string) normalizedSearchText {
	tokens := splitTokens(value)
	return normalizedSearchText{
		raw: strings.ToLower(strings.TrimSpace(value)), compact: strings.Join(tokens, ""), tokens: tokens,
	}
}

func bestSearchRank(query normalizedSearchText, names ...string) int {
	best := 0
	for _, name := range names {
		text := searchText(name)
		rank := 0
		switch {
		case text.compact != "" && text.compact == query.compact:
			rank = 1
		case tokenPrefix(text.tokens, query.tokens):
			rank = 2
		case text.compact != "" && strings.Contains(text.compact, query.compact):
			rank = 3
		}
		if rank > 0 && (best == 0 || rank < best) {
			best = rank
		}
	}
	return best
}

func textContains(value string, query normalizedSearchText) bool {
	if query.raw != "" && strings.Contains(strings.ToLower(value), query.raw) {
		return true
	}
	return query.compact != "" && strings.Contains(searchText(value).compact, query.compact)
}

func tokenPrefix(name, query []string) bool {
	if len(query) == 0 || len(query) > len(name) {
		return false
	}
	for start := 0; start+len(query) <= len(name); start++ {
		matched := true
		for i := range query {
			if query[i] == "" || !strings.HasPrefix(name[start+i], query[i]) {
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

func splitTokens(value string) []string {
	runes := []rune(strings.TrimSpace(value))
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, strings.ToLower(string(current)))
			current = nil
		}
	}
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if len(current) > 0 {
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsDigit(r) != unicode.IsDigit(prev) ||
				(unicode.IsUpper(r) && (unicode.IsLower(prev) || unicode.IsUpper(prev) && nextLower && len(current) > 1)) {
				flush()
			}
		}
		current = append(current, r)
	}
	flush()
	return tokens
}

func kebabCase(value string) string {
	return strings.Join(splitTokens(value), "-")
}

func cloneRaw(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

func cloneRawMap(values map[string]json.RawMessage) map[string]json.RawMessage {
	if values == nil {
		return nil
	}
	result := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		result[key] = cloneRaw(value)
	}
	return result
}

func reachableComponentRaw(schema json.RawMessage, components map[string]json.RawMessage) map[string]json.RawMessage {
	var root any
	if json.Unmarshal(schema, &root) != nil {
		return nil
	}
	result := map[string]json.RawMessage{}
	active := map[string]bool{}
	var collect func(any)
	collect = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			if ref, _ := typed["$ref"].(string); ref != "" {
				if name, ok := componentName(ref); ok && !active[name] {
					if raw, exists := components[name]; exists {
						active[name] = true
						result[name] = cloneRaw(raw)
						var component any
						if json.Unmarshal(raw, &component) == nil {
							collect(component)
						}
						delete(active, name)
					}
				}
			}
			for _, child := range typed {
				collect(child)
			}
		case []any:
			for _, child := range typed {
				collect(child)
			}
		}
	}
	collect(root)
	if len(result) == 0 {
		return nil
	}
	return result
}
