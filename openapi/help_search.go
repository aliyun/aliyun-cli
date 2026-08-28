package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	helpSearchResultLimit = 20
	// helpListingLimit is retained for the legacy renderer until integration
	// switches its default lists to the 100-line policy in help_policy.go.
	helpListingLimit = helpSearchResultLimit
	helpListingHint  = "Use --help-search <keyword> to narrow the list, or --help-all to show everything."
)

// HelpSearchRank is the stable relevance tier assigned to a local Help match.
type HelpSearchRank uint8

const (
	HelpSearchExactName HelpSearchRank = iota + 1
	HelpSearchNameTokenPrefix
	HelpSearchNameContains
	HelpSearchTextContains
)

// HelpSearchCandidate contains the searchable identity and localized text for
// one Help entry. Value is carried through untouched so renderers can attach
// their own DTO without coupling the search algorithm to a Help document type.
type HelpSearchCandidate struct {
	Kind          string
	Name          string
	Aliases       []string
	TitleEN       string
	TitleZH       string
	DescriptionEN string
	DescriptionZH string
	Value         any
}

// HelpSearchMatch is one candidate together with its relevance tier.
type HelpSearchMatch struct {
	Candidate HelpSearchCandidate
	Rank      HelpSearchRank
}

// HelpParameterSearchInput describes the active API parameter set and the
// global CLI parameters that participate in the same result collection.
type HelpParameterSearchInput struct {
	ActiveParameterSet string
	ParameterSets      map[string][]HelpSearchCandidate
	GlobalParameters   []HelpSearchCandidate
}

// HelpSearchValidation is the local, side-effect-free result used before an
// error recovery path advertises a search command.
type HelpSearchValidation struct {
	Matched    bool
	MatchCount int
}

// HelpSearchValidationRequest is the provider-aware context accepted by an
// injected validator. A nil HelpSearchValidator means that the active Help
// provider cannot validate the proposed search command.
type HelpSearchValidationRequest struct {
	Product string
	API     string
	Version string
	Section string
	Style   string
	Keyword string
}

// HelpSearchValidator validates a proposed recovery search against the same
// local provider and inputs that real Help will use.
type HelpSearchValidator func(HelpSearchValidationRequest) HelpSearchValidation

// SearchHelpCandidates returns every local match. It deliberately has no
// listing limit; the caller may project an unsearched Root/Product list with
// ProjectHelpListing after search policy has been decided.
func SearchHelpCandidates(candidates []HelpSearchCandidate, keyword string) []HelpSearchMatch {
	query := newHelpSearchText(keyword)
	if query.compact == "" {
		return nil
	}

	matches := make([]HelpSearchMatch, 0)
	for _, candidate := range candidates {
		if rank, ok := rankHelpSearchCandidate(candidate, query); ok {
			matches = append(matches, HelpSearchMatch{Candidate: candidate, Rank: rank})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Rank != matches[j].Rank {
			return matches[i].Rank < matches[j].Rank
		}
		left := newHelpSearchText(matches[i].Candidate.Name).compact
		right := newHelpSearchText(matches[j].Candidate.Name).compact
		if left == right {
			return false
		}
		return left < right
	})
	return matches
}

// HelpSearchResults is the renderer-independent full-rank-then-limit result.
// Validation intentionally uses the unprojected matcher so MatchCount remains
// the real total even when only twenty matches are rendered.
type HelpSearchResults struct {
	Matches []HelpSearchMatch `json:"matches"`
	Result  HelpResult        `json:"result"`
}

// ProjectHelpSearchMatches caps a completely ranked match set at the single
// Help Search limit used by every target and output mode.
func ProjectHelpSearchMatches(matches []HelpSearchMatch) HelpSearchResults {
	total := len(matches)
	shown := total
	if shown > helpSearchResultLimit {
		shown = helpSearchResultLimit
	}
	return HelpSearchResults{
		Matches: append([]HelpSearchMatch(nil), matches[:shown]...),
		Result: HelpResult{
			Shown:     shown,
			Total:     total,
			Truncated: shown < total,
		},
	}
}

// SearchHelpParameters searches only the selected request parameter style and
// the global CLI parameters. Inactive parameter sets never leak into results.
func SearchHelpParameters(input HelpParameterSearchInput, keyword string) []HelpSearchMatch {
	active := input.ParameterSets[input.ActiveParameterSet]
	candidates := make([]HelpSearchCandidate, 0, len(active)+len(input.GlobalParameters))
	candidates = append(candidates, active...)
	candidates = append(candidates, input.GlobalParameters...)
	return SearchHelpCandidates(candidates, keyword)
}

// ValidateHelpSearch uses the real candidate matcher and reports whether the
// proposed keyword has at least one result.
func ValidateHelpSearch(candidates []HelpSearchCandidate, keyword string) HelpSearchValidation {
	matches := SearchHelpCandidates(candidates, keyword)
	return HelpSearchValidation{Matched: len(matches) > 0, MatchCount: len(matches)}
}

func rankHelpSearchCandidate(candidate HelpSearchCandidate, query helpSearchText) (HelpSearchRank, bool) {
	best := HelpSearchRank(0)
	names := make([]string, 0, 1+len(candidate.Aliases))
	names = append(names, candidate.Name)
	names = append(names, candidate.Aliases...)
	for _, name := range names {
		nameText := newHelpSearchText(name)
		var rank HelpSearchRank
		switch {
		case nameText.compact != "" && nameText.compact == query.compact:
			rank = HelpSearchExactName
		case tokenPrefixMatch(nameText.tokens, query.tokens):
			rank = HelpSearchNameTokenPrefix
		case nameText.compact != "" && strings.Contains(nameText.compact, query.compact):
			rank = HelpSearchNameContains
		}
		if rank != 0 && (best == 0 || rank < best) {
			best = rank
		}
	}
	if best != 0 {
		return best, true
	}

	for _, text := range []string{
		candidate.TitleEN,
		candidate.TitleZH,
		candidate.DescriptionEN,
		candidate.DescriptionZH,
	} {
		if helpTextContains(text, query) {
			return HelpSearchTextContains, true
		}
	}
	return 0, false
}

type helpSearchText struct {
	raw     string
	compact string
	tokens  []string
}

func newHelpSearchText(value string) helpSearchText {
	tokens := splitHelpSearchTokens(value)
	return helpSearchText{
		raw:     strings.ToLower(strings.TrimSpace(value)),
		compact: strings.Join(tokens, ""),
		tokens:  tokens,
	}
}

func splitHelpSearchTokens(value string) []string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) == 0 {
		return nil
	}

	tokens := make([]string, 0, 4)
	current := make([]rune, 0, len(runes))
	flush := func() {
		if len(current) == 0 {
			return
		}
		tokens = append(tokens, strings.ToLower(string(current)))
		current = current[:0]
	}

	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}

		boundary := false
		if len(current) > 0 {
			previous := runes[i-1]
			switch {
			case unicode.IsDigit(r) != unicode.IsDigit(previous):
				boundary = true
			case unicode.IsUpper(r):
				nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				pluralAcronymSuffix := unicode.IsUpper(previous) && nextIsLower && runes[i+1] == 's' &&
					(i+2 == len(runes) || !unicode.IsLower(runes[i+2]))
				boundary = unicode.IsLower(previous) ||
					(unicode.IsUpper(previous) && nextIsLower && len(current) > 1 && !pluralAcronymSuffix)
			}
		}
		if boundary {
			flush()
		}
		current = append(current, r)
	}
	flush()
	return tokens
}

func tokenPrefixMatch(nameTokens, queryTokens []string) bool {
	if len(queryTokens) == 0 || len(queryTokens) > len(nameTokens) {
		return false
	}
	for start := 0; start+len(queryTokens) <= len(nameTokens); start++ {
		matched := true
		for i, queryToken := range queryTokens {
			if queryToken == "" || !strings.HasPrefix(nameTokens[start+i], queryToken) {
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

func helpTextContains(value string, query helpSearchText) bool {
	if strings.Contains(strings.ToLower(value), query.raw) {
		return true
	}
	text := newHelpSearchText(value)
	return text.compact != "" && strings.Contains(text.compact, query.compact)
}

// HelpListingTarget identifies lists that AI mode may cap.
type HelpListingTarget string

const (
	HelpListingRootProducts HelpListingTarget = "root-products"
	HelpListingProductAPIs  HelpListingTarget = "product-apis"
)

// HelpListingOptions contains the orthogonal mode/search/all policy inputs.
type HelpListingOptions struct {
	Target   HelpListingTarget
	AIMode   bool
	Searched bool
	All      bool
}

// HelpListingMetadata is emitted only when an AI Root/Product list is actually
// truncated. Root callers pass only products so built-in commands are excluded
// from Shown and Total.
type HelpListingMetadata struct {
	Shown int    `json:"shown"`
	Total int    `json:"total"`
	Hint  string `json:"hint"`
}

// ProjectHelpListing applies the AI-mode limit without mutating the input.
func ProjectHelpListing[T any](items []T, options HelpListingOptions) ([]T, *HelpListingMetadata) {
	mayCap := options.Target == HelpListingRootProducts || options.Target == HelpListingProductAPIs
	if !mayCap || !options.AIMode || options.Searched || options.All || len(items) <= helpListingLimit {
		return append([]T(nil), items...), nil
	}
	return append([]T(nil), items[:helpListingLimit]...), &HelpListingMetadata{
		Shown: helpListingLimit,
		Total: len(items),
		Hint:  helpListingHint,
	}
}

// HelpResponseSchema is an independent adapter DTO matching Task 1's raw
// schema/components shape without importing canonicalmeta.
type HelpResponseSchema struct {
	Schema     json.RawMessage
	Components map[string]json.RawMessage
}

// HelpResponseSchemaSearchResult is the minimal legal schema projection for
// all matched response field paths.
type HelpResponseSchemaSearchResult struct {
	Paths      []string
	Schema     json.RawMessage
	Components map[string]json.RawMessage
	Result     HelpResult
}

// SearchResponseSchema returns every matching field path, the merged minimum
// root tree, and the reachable pruned component closure.
func SearchResponseSchema(input HelpResponseSchema, keyword string) (HelpResponseSchemaSearchResult, error) {
	query := newHelpSearchText(keyword)
	if query.compact == "" {
		return emptyHelpResponseSchemaSearchResult(), nil
	}

	document, err := parseHelpResponseSchema(input)
	if err != nil {
		return HelpResponseSchemaSearchResult{}, err
	}
	if document.root == nil {
		return emptyHelpResponseSchemaSearchResult(), nil
	}

	state := newResponseSchemaSearchState(document, query)
	state.walk(document.root, nil, make(map[string]bool))
	allMatches := state.sortedPathMatches()
	if len(allMatches) == 0 {
		return emptyHelpResponseSchemaSearchResult(), nil
	}
	shown := len(allMatches)
	if shown > helpSearchResultLimit {
		shown = helpSearchResultLimit
		allowed := make(map[string]bool, shown)
		for _, match := range allMatches[:shown] {
			allowed[match.path] = true
		}
		state = newResponseSchemaSearchState(document, query)
		state.allowedPaths = allowed
		state.walk(document.root, nil, make(map[string]bool))
	}

	root, err := state.marshalNode(document.root)
	if err != nil {
		return HelpResponseSchemaSearchResult{}, err
	}
	components, err := state.reachableComponents(root)
	if err != nil {
		return HelpResponseSchemaSearchResult{}, err
	}
	return HelpResponseSchemaSearchResult{
		Paths:      responseSchemaMatchPaths(allMatches[:shown]),
		Schema:     root,
		Components: components,
		Result: HelpResult{
			Shown:     shown,
			Total:     len(allMatches),
			Truncated: shown < len(allMatches),
		},
	}, nil
}

// ValidateResponseSchemaSearch runs the exact response matcher used by real
// Help search, without executing a CLI command or touching the network.
func ValidateResponseSchemaSearch(input HelpResponseSchema, keyword string) (HelpSearchValidation, error) {
	result, err := SearchResponseSchema(input, keyword)
	if err != nil {
		return HelpSearchValidation{}, err
	}
	return HelpSearchValidation{Matched: result.Result.Total > 0, MatchCount: result.Result.Total}, nil
}

func emptyHelpResponseSchemaSearchResult() HelpResponseSchemaSearchResult {
	return HelpResponseSchemaSearchResult{Components: make(map[string]json.RawMessage)}
}

type responseSchemaDocument struct {
	root       *responseSchemaNode
	components map[string]*responseSchemaNode
}

type responseSchemaNode struct {
	raw          json.RawMessage
	fields       []responseSchemaRawField
	properties   []responseSchemaProperty
	items        *responseSchemaNode
	compositions map[string][]*responseSchemaNode
	ref          string
	typ          string
	titles       []string
	describes    []string
}

type responseSchemaRawField struct {
	name string
	raw  json.RawMessage
}

type responseSchemaProperty struct {
	name string
	node *responseSchemaNode
}

func parseHelpResponseSchema(input HelpResponseSchema) (*responseSchemaDocument, error) {
	document := &responseSchemaDocument{
		components: make(map[string]*responseSchemaNode, len(input.Components)),
	}

	if raw := bytes.TrimSpace(input.Schema); len(raw) > 0 && !bytes.Equal(raw, []byte("null")) {
		root, err := parseResponseSchemaNode(raw)
		if err != nil {
			return nil, fmt.Errorf("parse response schema: %w", err)
		}
		document.root = root
	}

	for name, raw := range input.Components {
		node, err := parseResponseSchemaNode(raw)
		if err != nil {
			return nil, fmt.Errorf("parse response component %q: %w", name, err)
		}
		document.components[name] = node
	}
	return document, nil
}

func parseResponseSchemaNode(raw json.RawMessage) (*responseSchemaNode, error) {
	raw = append(json.RawMessage(nil), bytes.TrimSpace(raw)...)
	if !json.Valid(raw) {
		return nil, fmt.Errorf("invalid JSON")
	}
	node := &responseSchemaNode{raw: raw, compositions: make(map[string][]*responseSchemaNode)}
	fields, object, err := decodeOrderedRawObject(raw)
	if err != nil {
		return nil, err
	}
	if !object {
		return node, nil
	}
	node.fields = fields

	for _, field := range fields {
		switch field.name {
		case "$ref":
			_ = json.Unmarshal(field.raw, &node.ref)
		case "type":
			_ = json.Unmarshal(field.raw, &node.typ)
		case "title", "title_en", "title_zh", "x-title-en", "x-title-zh":
			var value string
			if json.Unmarshal(field.raw, &value) == nil && value != "" {
				node.titles = append(node.titles, value)
			}
		case "description", "description_en", "description_zh", "x-description-en", "x-description-zh":
			var value string
			if json.Unmarshal(field.raw, &value) == nil && value != "" {
				node.describes = append(node.describes, value)
			}
		case "properties":
			properties, isObject, propertyErr := decodeOrderedRawObject(field.raw)
			if propertyErr != nil {
				return nil, fmt.Errorf("parse properties: %w", propertyErr)
			}
			if !isObject {
				continue
			}
			for _, property := range properties {
				propertyNode, propertyErr := parseResponseSchemaNode(property.raw)
				if propertyErr != nil {
					return nil, fmt.Errorf("parse property %q: %w", property.name, propertyErr)
				}
				node.properties = append(node.properties, responseSchemaProperty{name: property.name, node: propertyNode})
			}
		case "items":
			trimmed := bytes.TrimSpace(field.raw)
			if len(trimmed) > 0 && (trimmed[0] == '{' || bytes.Equal(trimmed, []byte("true")) || bytes.Equal(trimmed, []byte("false"))) {
				items, itemsErr := parseResponseSchemaNode(field.raw)
				if itemsErr != nil {
					return nil, fmt.Errorf("parse array items: %w", itemsErr)
				}
				node.items = items
			}
		case "allOf", "oneOf", "anyOf":
			var branches []json.RawMessage
			if json.Unmarshal(field.raw, &branches) != nil {
				continue
			}
			for index, branchRaw := range branches {
				branch, branchErr := parseResponseSchemaNode(branchRaw)
				if branchErr != nil {
					return nil, fmt.Errorf("parse %s branch %d: %w", field.name, index, branchErr)
				}
				node.compositions[field.name] = append(node.compositions[field.name], branch)
			}
		}
	}
	return node, nil
}

func decodeOrderedRawObject(raw json.RawMessage) ([]responseSchemaRawField, bool, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, false, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	token, err := decoder.Token()
	if err != nil {
		return nil, false, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, false, nil
	}

	fields := make([]responseSchemaRawField, 0)
	for decoder.More() {
		nameToken, tokenErr := decoder.Token()
		if tokenErr != nil {
			return nil, false, tokenErr
		}
		name, ok := nameToken.(string)
		if !ok {
			return nil, false, fmt.Errorf("object key is not a string")
		}
		var value json.RawMessage
		if decodeErr := decoder.Decode(&value); decodeErr != nil {
			return nil, false, decodeErr
		}
		fields = append(fields, responseSchemaRawField{name: name, raw: append(json.RawMessage(nil), value...)})
	}
	if _, err = decoder.Token(); err != nil {
		return nil, false, err
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, false, fmt.Errorf("multiple JSON values")
		}
		return nil, false, err
	}
	return fields, true, nil
}

func (d *responseSchemaDocument) resolveRef(ref string) (string, *responseSchemaNode) {
	name, ok := localResponseComponentName(ref)
	if !ok {
		return "", nil
	}
	return name, d.components[name]
}

func localResponseComponentName(ref string) (string, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	name = strings.ReplaceAll(name, "~1", "/")
	name = strings.ReplaceAll(name, "~0", "~")
	return name, true
}

type responseSchemaKeep struct {
	properties   map[int]bool
	items        bool
	compositions map[string]map[int]bool
}

type responseSchemaPathMatch struct {
	path  string
	rank  HelpSearchRank
	order int
}

type responseSchemaSearchState struct {
	document       *responseSchemaDocument
	query          helpSearchText
	keep           map[*responseSchemaNode]*responseSchemaKeep
	fullNodes      map[*responseSchemaNode]bool
	fullComponents map[string]bool
	paths          []responseSchemaPathMatch
	pathIndex      map[string]int
	allowedPaths   map[string]bool
}

func newResponseSchemaSearchState(document *responseSchemaDocument, query helpSearchText) *responseSchemaSearchState {
	return &responseSchemaSearchState{
		document:       document,
		query:          query,
		keep:           make(map[*responseSchemaNode]*responseSchemaKeep),
		fullNodes:      make(map[*responseSchemaNode]bool),
		fullComponents: make(map[string]bool),
		pathIndex:      make(map[string]int),
	}
}

func (s *responseSchemaSearchState) keepFor(node *responseSchemaNode) *responseSchemaKeep {
	if node == nil {
		return nil
	}
	result := s.keep[node]
	if result == nil {
		result = &responseSchemaKeep{
			properties:   make(map[int]bool),
			compositions: make(map[string]map[int]bool),
		}
		s.keep[node] = result
	}
	return result
}

func (s *responseSchemaSearchState) walk(node *responseSchemaNode, path []string, activeRefs map[string]bool) bool {
	if node == nil {
		return false
	}
	matched := false
	for _, field := range node.fields {
		switch field.name {
		case "$ref":
			name, target := s.document.resolveRef(node.ref)
			if target == nil || activeRefs[name] {
				continue
			}
			activeRefs[name] = true
			if s.walk(target, path, activeRefs) {
				s.keepFor(node)
				matched = true
			}
			delete(activeRefs, name)
		case "properties":
			for index, property := range node.properties {
				propertyPath := appendPath(path, property.name)
				rank, selfMatch := s.propertyMatchRank(property, propertyPath)
				pathName := strings.Join(propertyPath, ".")
				if selfMatch && (s.allowedPaths == nil || s.allowedPaths[pathName]) {
					s.keepFor(node).properties[index] = true
					s.markNodeSelf(property.node, activeRefs)
					s.addPath(pathName, rank)
					matched = true
				}
				if s.walk(property.node, propertyPath, activeRefs) {
					s.keepFor(node).properties[index] = true
					matched = true
				}
			}
		case "items":
			if node.items != nil && s.walk(node.items, path, activeRefs) {
				s.keepFor(node).items = true
				matched = true
			}
		case "allOf", "oneOf", "anyOf":
			for index, branch := range node.compositions[field.name] {
				if s.walk(branch, path, activeRefs) {
					selection := s.keepFor(node)
					if selection.compositions[field.name] == nil {
						selection.compositions[field.name] = make(map[int]bool)
					}
					selection.compositions[field.name][index] = true
					matched = true
				}
			}
		}
	}
	return matched
}

func (s *responseSchemaSearchState) propertyMatchRank(property responseSchemaProperty, path []string) (HelpSearchRank, bool) {
	titles, descriptions := s.nodeSearchText(property.node, make(map[string]bool))
	candidate := HelpSearchCandidate{
		Name:          property.name,
		TitleEN:       strings.Join(titles, " "),
		DescriptionEN: strings.Join(descriptions, " "),
	}
	if rank, matched := rankHelpSearchCandidate(candidate, s.query); matched {
		return rank, true
	}

	pathText := newHelpSearchText(strings.Join(path, "."))
	propertyText := newHelpSearchText(property.name)
	pathRank, pathMatched := rankHelpSearchCandidate(HelpSearchCandidate{Name: strings.Join(path, ".")}, s.query)
	if pathMatched && (pathText.compact == s.query.compact || len(s.query.tokens) > len(propertyText.tokens)) {
		return pathRank, true
	}
	return 0, false
}

func (s *responseSchemaSearchState) nodeSearchText(node *responseSchemaNode, visited map[string]bool) ([]string, []string) {
	if node == nil {
		return nil, nil
	}
	titles := append([]string(nil), node.titles...)
	descriptions := append([]string(nil), node.describes...)
	name, target := s.document.resolveRef(node.ref)
	if target != nil && !visited[name] {
		visited[name] = true
		referencedTitles, referencedDescriptions := s.nodeSearchText(target, visited)
		titles = append(titles, referencedTitles...)
		descriptions = append(descriptions, referencedDescriptions...)
	}
	for _, keyword := range []string{"allOf", "oneOf", "anyOf"} {
		for _, branch := range node.compositions[keyword] {
			branchTitles, branchDescriptions := s.nodeSearchText(branch, visited)
			titles = append(titles, branchTitles...)
			descriptions = append(descriptions, branchDescriptions...)
		}
	}
	return titles, descriptions
}

func appendPath(path []string, name string) []string {
	result := make([]string, len(path)+1)
	copy(result, path)
	result[len(path)] = name
	return result
}

func (s *responseSchemaSearchState) addPath(path string, rank HelpSearchRank) {
	if path == "" {
		return
	}
	if index, exists := s.pathIndex[path]; exists {
		if rank < s.paths[index].rank {
			s.paths[index].rank = rank
		}
		return
	}
	s.pathIndex[path] = len(s.paths)
	s.paths = append(s.paths, responseSchemaPathMatch{path: path, rank: rank, order: len(s.paths)})
}

func (s *responseSchemaSearchState) sortedPaths() []string {
	return responseSchemaMatchPaths(s.sortedPathMatches())
}

func (s *responseSchemaSearchState) sortedPathMatches() []responseSchemaPathMatch {
	matches := append([]responseSchemaPathMatch(nil), s.paths...)
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].rank != matches[j].rank {
			return matches[i].rank < matches[j].rank
		}
		left := newHelpSearchText(matches[i].path).compact
		right := newHelpSearchText(matches[j].path).compact
		if left != right {
			return left < right
		}
		return matches[i].order < matches[j].order
	})
	return matches
}

func responseSchemaMatchPaths(matches []responseSchemaPathMatch) []string {
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, match.path)
	}
	return result
}

func (s *responseSchemaSearchState) markNodeSelf(node *responseSchemaNode, activeRefs map[string]bool) {
	if node == nil {
		return
	}
	s.keepFor(node)
	if len(node.compositions) > 0 {
		s.markFullNode(node)
		return
	}
	if responseSchemaNodeIsArray(node) && node.items != nil {
		s.keepFor(node).items = true
		s.markFullNode(node.items)
	}
	if name, target := s.document.resolveRef(node.ref); target != nil {
		s.keepFor(target)
		if !activeRefs[name] {
			activeRefs[name] = true
			s.markNodeSelf(target, activeRefs)
			delete(activeRefs, name)
		}
	}
}

func responseSchemaNodeIsArray(node *responseSchemaNode) bool {
	return node != nil && (strings.EqualFold(node.typ, "array") || node.items != nil)
}

func (s *responseSchemaSearchState) markFullNode(node *responseSchemaNode) {
	if node == nil || s.fullNodes[node] {
		return
	}
	s.fullNodes[node] = true
	for _, name := range collectLocalResponseRefs(node.raw) {
		s.markFullComponent(name)
	}
}

func (s *responseSchemaSearchState) markFullComponent(name string) {
	if name == "" || s.fullComponents[name] {
		return
	}
	node := s.document.components[name]
	if node == nil {
		return
	}
	s.fullComponents[name] = true
	s.fullNodes[node] = true
	for _, referenced := range collectLocalResponseRefs(node.raw) {
		s.markFullComponent(referenced)
	}
}

func collectLocalResponseRefs(raw json.RawMessage) []string {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	set := make(map[string]bool)
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			if ref, ok := typed["$ref"].(string); ok {
				if name, local := localResponseComponentName(ref); local {
					set[name] = true
				}
			}
			for _, child := range typed {
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(value)
	result := make([]string, 0, len(set))
	for name := range set {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (s *responseSchemaSearchState) marshalNode(node *responseSchemaNode) (json.RawMessage, error) {
	if node == nil {
		return nil, nil
	}
	if s.fullNodes[node] || len(node.fields) == 0 {
		return append(json.RawMessage(nil), node.raw...), nil
	}
	selection := s.keep[node]
	selectedProperties := map[int]bool(nil)
	keepItems := false
	selectedCompositions := map[string]map[int]bool(nil)
	if selection != nil {
		selectedProperties = selection.properties
		keepItems = selection.items
		selectedCompositions = selection.compositions
	}

	fields := make([]responseSchemaRawField, 0, len(node.fields))
	for _, field := range node.fields {
		switch field.name {
		case "properties":
			if len(selectedProperties) == 0 {
				continue
			}
			properties, err := s.marshalSelectedProperties(node, selectedProperties)
			if err != nil {
				return nil, err
			}
			fields = append(fields, responseSchemaRawField{name: field.name, raw: properties})
		case "required":
			if len(node.properties) == 0 {
				fields = append(fields, field)
				continue
			}
			required := filterResponseRequired(field.raw, node.properties, selectedProperties)
			if len(required) > 0 {
				fields = append(fields, responseSchemaRawField{name: field.name, raw: required})
			}
		case "items":
			if !keepItems || node.items == nil {
				continue
			}
			items, err := s.marshalNode(node.items)
			if err != nil {
				return nil, err
			}
			fields = append(fields, responseSchemaRawField{name: field.name, raw: items})
		case "allOf", "oneOf", "anyOf":
			selected := selectedCompositions[field.name]
			if len(selected) == 0 {
				continue
			}
			branches, err := s.marshalSelectedComposition(node.compositions[field.name], selected)
			if err != nil {
				return nil, err
			}
			fields = append(fields, responseSchemaRawField{name: field.name, raw: branches})
		default:
			fields = append(fields, field)
		}
	}
	return marshalOrderedRawObject(fields), nil
}

func (s *responseSchemaSearchState) marshalSelectedComposition(branches []*responseSchemaNode, selected map[int]bool) (json.RawMessage, error) {
	values := make([]json.RawMessage, 0, len(selected))
	for index, branch := range branches {
		if !selected[index] {
			continue
		}
		raw, err := s.marshalNode(branch)
		if err != nil {
			return nil, err
		}
		values = append(values, raw)
	}
	return json.Marshal(values)
}

func (s *responseSchemaSearchState) marshalSelectedProperties(node *responseSchemaNode, selected map[int]bool) (json.RawMessage, error) {
	fields := make([]responseSchemaRawField, 0, len(selected))
	for index, property := range node.properties {
		if !selected[index] {
			continue
		}
		raw, err := s.marshalNode(property.node)
		if err != nil {
			return nil, err
		}
		fields = append(fields, responseSchemaRawField{name: property.name, raw: raw})
	}
	return marshalOrderedRawObject(fields), nil
}

func filterResponseRequired(raw json.RawMessage, properties []responseSchemaProperty, selected map[int]bool) json.RawMessage {
	if len(selected) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(selected))
	for index := range selected {
		if index >= 0 && index < len(properties) {
			allowed[properties[index].name] = true
		}
	}
	var required []string
	if json.Unmarshal(raw, &required) != nil {
		return append(json.RawMessage(nil), raw...)
	}
	filtered := make([]string, 0, len(required))
	for _, name := range required {
		if allowed[name] {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	result, _ := json.Marshal(filtered)
	return result
}

func marshalOrderedRawObject(fields []responseSchemaRawField) json.RawMessage {
	var buffer bytes.Buffer
	buffer.WriteByte('{')
	for index, field := range fields {
		if index > 0 {
			buffer.WriteByte(',')
		}
		buffer.WriteString(strconv.Quote(field.name))
		buffer.WriteByte(':')
		buffer.Write(field.raw)
	}
	buffer.WriteByte('}')
	return buffer.Bytes()
}

func (s *responseSchemaSearchState) reachableComponents(root json.RawMessage) (map[string]json.RawMessage, error) {
	result := make(map[string]json.RawMessage)
	queue := collectLocalResponseRefs(root)
	queued := make(map[string]bool, len(queue))
	for _, name := range queue {
		queued[name] = true
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if _, done := result[name]; done {
			continue
		}
		node := s.document.components[name]
		if node == nil {
			continue
		}
		raw, err := s.marshalNode(node)
		if err != nil {
			return nil, err
		}
		result[name] = raw
		for _, referenced := range collectLocalResponseRefs(raw) {
			if !queued[referenced] {
				queued[referenced] = true
				queue = append(queue, referenced)
			}
		}
	}
	return result, nil
}
