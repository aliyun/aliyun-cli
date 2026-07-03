package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type AbstractCase struct {
	StateName                   string     `json:"stateName"`
	TokenRaw                    string     `json:"tokenRaw"`
	TokenClass                  TokenClass `json:"tokenClass"`
	TokenFlag                   string     `json:"tokenFlag"`
	TokenValue                  string     `json:"tokenValue"`
	TokenParseError             ErrorInfo  `json:"tokenParseError"`
	AllowedDashValueEnhancement bool       `json:"allowedDashValueEnhancement"`
	OldObservable               Observable `json:"oldObservable"`
	NewObservable               Observable `json:"newObservable"`
}

type TokenClass struct {
	SplitterClass  string `json:"splitterClass"`
	PrefixShape    string `json:"prefixShape"`
	DetectorResult string `json:"detectorResult"`
}

type ErrorInfo struct {
	Class string `json:"class"`
	Text  string `json:"text"`
}

type FlagObservable struct {
	Assigned bool     `json:"assigned"`
	Value    string   `json:"value"`
	Values   []string `json:"values"`
}

type Observable struct {
	Current     int                       `json:"current"`
	OutArg      string                    `json:"outArg"`
	OutFlag     string                    `json:"outFlag"`
	OutMore     bool                      `json:"outMore"`
	Err         ErrorInfo                 `json:"err"`
	Flags       map[string]FlagObservable `json:"flags"`
	PendingFlag string                    `json:"pendingFlag"`
	PendingFrom string                    `json:"pendingFrom"`
}

type GeneratedCase struct {
	Name                        string     `json:"name"`
	StateName                   string     `json:"stateName"`
	Argv                        []string   `json:"argv"`
	TokenRaw                    string     `json:"tokenRaw"`
	TokenClass                  TokenClass `json:"tokenClass"`
	TokenFlag                   string     `json:"tokenFlag"`
	TokenValue                  string     `json:"tokenValue"`
	TokenParseError             ErrorInfo  `json:"tokenParseError"`
	AllowedDashValueEnhancement bool       `json:"allowedDashValueEnhancement"`
	ExpectedSource              string     `json:"expectedSource"`
	Expected                    Observable `json:"expected"`
	OldObservable               Observable `json:"oldObservable"`
}

type GeneratedCases struct {
	Source       string          `json:"source"`
	Model        string          `json:"model"`
	ExpectedFrom string          `json:"expectedFrom"`
	Cases        []GeneratedCase `json:"cases"`
}

type Coverage struct {
	TotalCases              int
	AllowedEnhancementCases int
	StateCounts             map[string]int
	TokenCounts             map[string]int
	PairCounts              map[string]int
}

func main() {
	dumpPath := flag.String("dump", "formal/parser/generated/abstract_cases.dump", "TLC dump produced by make formal-parser-cases")
	outPath := flag.String("out", "cli/testdata/parser_contract_cases.json", "generated Go contract testdata JSON path")
	coveragePath := flag.String("coverage", "docs/formal/parser-case-coverage.md", "generated coverage matrix markdown path")
	flag.Parse()

	content, err := os.ReadFile(*dumpPath)
	if err != nil {
		exitf("read dump: %v", err)
	}
	abstractCases, err := ParseDump(content)
	if err != nil {
		exitf("parse dump: %v", err)
	}
	generated, err := BuildGeneratedCases(abstractCases)
	if err != nil {
		exitf("build cases: %v", err)
	}
	generated.Source = *dumpPath

	if err := writeJSON(*outPath, generated); err != nil {
		exitf("write json: %v", err)
	}
	if err := writeCoverage(*coveragePath, BuildCoverage(generated.Cases)); err != nil {
		exitf("write coverage: %v", err)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func ParseDump(content []byte) ([]AbstractCase, error) {
	exprs, err := extractSelectedCaseExpressions(string(content))
	if err != nil {
		return nil, err
	}
	cases := make([]AbstractCase, 0, len(exprs))
	for i, expr := range exprs {
		value, err := newTLAParser(expr).parseValue()
		if err != nil {
			return nil, fmt.Errorf("state %d: %w", i+1, err)
		}
		record, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("state %d: selectedCase is %T, want record", i+1, value)
		}
		c, err := decodeAbstractCase(record)
		if err != nil {
			return nil, fmt.Errorf("state %d: %w", i+1, err)
		}
		cases = append(cases, c)
	}
	if len(cases) == 0 {
		return nil, errors.New("no selectedCase records found")
	}
	return cases, nil
}

func extractSelectedCaseExpressions(text string) ([]string, error) {
	lines := strings.Split(text, "\n")
	var exprs []string
	for i := 0; i < len(lines); i++ {
		idx := strings.Index(lines[i], "/\\ selectedCase = ")
		if idx < 0 {
			continue
		}
		start := strings.TrimSpace(lines[i][idx+len("/\\ selectedCase = "):])
		var b strings.Builder
		b.WriteString(start)
		depth := bracketDelta(start)
		for depth > 0 {
			i++
			if i >= len(lines) {
				return nil, errors.New("unterminated selectedCase record")
			}
			b.WriteByte('\n')
			line := lines[i]
			b.WriteString(line)
			depth += bracketDelta(line)
		}
		exprs = append(exprs, b.String())
	}
	return exprs, nil
}

func bracketDelta(s string) int {
	delta := 0
	inString := false
	escaped := false
	for _, r := range s {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}
		switch r {
		case '"':
			inString = true
		case '[':
			delta++
		case ']':
			delta--
		}
	}
	return delta
}

func BuildGeneratedCases(abstractCases []AbstractCase) (GeneratedCases, error) {
	cases := make([]GeneratedCase, 0, len(abstractCases))
	seen := map[string]struct{}{}
	for _, c := range abstractCases {
		prefix, err := statePrefix(c.StateName)
		if err != nil {
			return GeneratedCases{}, err
		}
		argv := append(append([]string{}, prefix...), c.TokenRaw)
		name := caseName(c.StateName, c.TokenRaw, c.TokenClass)
		key := c.StateName + "\x00" + c.TokenRaw
		if _, ok := seen[key]; ok {
			return GeneratedCases{}, fmt.Errorf("duplicate case for state %q token %q", c.StateName, c.TokenRaw)
		}
		seen[key] = struct{}{}
		cases = append(cases, GeneratedCase{
			Name:                        name,
			StateName:                   c.StateName,
			Argv:                        argv,
			TokenRaw:                    c.TokenRaw,
			TokenClass:                  c.TokenClass,
			TokenFlag:                   c.TokenFlag,
			TokenValue:                  c.TokenValue,
			TokenParseError:             c.TokenParseError,
			AllowedDashValueEnhancement: c.AllowedDashValueEnhancement,
			ExpectedSource:              "selectedCase.newObservable",
			Expected:                    c.NewObservable,
			OldObservable:               c.OldObservable,
		})
	}
	sort.Slice(cases, func(i, j int) bool {
		if stateRank(cases[i].StateName) != stateRank(cases[j].StateName) {
			return stateRank(cases[i].StateName) < stateRank(cases[j].StateName)
		}
		return cases[i].TokenRaw < cases[j].TokenRaw
	})
	return GeneratedCases{
		Model:        "formal/parser/Parser.tla",
		ExpectedFrom: "selectedCase.newObservable",
		Cases:        cases,
	}, nil
}

func BuildCoverage(cases []GeneratedCase) Coverage {
	coverage := Coverage{
		TotalCases:  len(cases),
		StateCounts: map[string]int{},
		TokenCounts: map[string]int{},
		PairCounts:  map[string]int{},
	}
	for _, c := range cases {
		coverage.StateCounts[c.StateName]++
		coverage.TokenCounts[c.TokenRaw]++
		coverage.PairCounts[c.StateName+"\x00"+c.TokenRaw]++
		if c.AllowedDashValueEnhancement {
			coverage.AllowedEnhancementCases++
		}
	}
	return coverage
}

func statePrefix(stateName string) ([]string, error) {
	switch stateName {
	case "init":
		return nil, nil
	case "pending_PortRange":
		return []string{"--PortRange"}, nil
	case "pending_Count":
		return []string{"--Count"}, nil
	case "pending_Name":
		return []string{"--Name"}, nil
	case "pending_RegionId":
		return []string{"--RegionId"}, nil
	case "pending_short_Count":
		return []string{"-c"}, nil
	case "pending_dynamic_long_dynamic":
		return []string{"--dynamic"}, nil
	case "pending_dynamic_short_u":
		return []string{"-u"}, nil
	default:
		return nil, fmt.Errorf("unknown stateName %q", stateName)
	}
}

func stateRank(stateName string) int {
	switch stateName {
	case "init":
		return 0
	case "pending_PortRange":
		return 1
	case "pending_Count":
		return 2
	case "pending_Name":
		return 3
	case "pending_RegionId":
		return 4
	case "pending_short_Count":
		return 5
	case "pending_dynamic_long_dynamic":
		return 6
	case "pending_dynamic_short_u":
		return 7
	default:
		return 100
	}
}

func caseName(stateName, tokenRaw string, tokenClass TokenClass) string {
	raw := tokenRaw
	if raw == "" {
		raw = "empty"
	}
	return sanitizeName(strings.Join([]string{
		stateName,
		tokenClass.SplitterClass,
		tokenClass.PrefixShape,
		tokenClass.DetectorResult,
		raw,
	}, "__"))
}

func sanitizeName(s string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		ok := unicode.IsLetter(r) || unicode.IsDigit(r)
		if ok {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func decodeAbstractCase(record map[string]any) (AbstractCase, error) {
	tokenClass, err := decodeTokenClass(requiredRecord(record, "tokenClass"))
	if err != nil {
		return AbstractCase{}, err
	}
	tokenParseError, err := decodeError(requiredRecord(record, "tokenParseError"))
	if err != nil {
		return AbstractCase{}, err
	}
	oldObservable, err := decodeObservable(requiredRecord(record, "oldObservable"))
	if err != nil {
		return AbstractCase{}, fmt.Errorf("oldObservable: %w", err)
	}
	newObservable, err := decodeObservable(requiredRecord(record, "newObservable"))
	if err != nil {
		return AbstractCase{}, fmt.Errorf("newObservable: %w", err)
	}
	return AbstractCase{
		StateName:                   requiredString(record, "stateName"),
		TokenRaw:                    requiredString(record, "tokenRaw"),
		TokenClass:                  tokenClass,
		TokenFlag:                   requiredString(record, "tokenFlag"),
		TokenValue:                  requiredString(record, "tokenValue"),
		TokenParseError:             tokenParseError,
		AllowedDashValueEnhancement: requiredBool(record, "allowedDashValueEnhancement"),
		OldObservable:               oldObservable,
		NewObservable:               newObservable,
	}, nil
}

func decodeTokenClass(record map[string]any) (TokenClass, error) {
	return TokenClass{
		SplitterClass:  requiredString(record, "splitterClass"),
		PrefixShape:    requiredString(record, "prefixShape"),
		DetectorResult: requiredString(record, "detectorResult"),
	}, nil
}

func decodeObservable(record map[string]any) (Observable, error) {
	flagsRecord := requiredRecord(record, "flags")
	flags := make(map[string]FlagObservable, len(flagsRecord))
	for name, value := range flagsRecord {
		flagRecord, ok := value.(map[string]any)
		if !ok {
			return Observable{}, fmt.Errorf("flag %s is %T, want record", name, value)
		}
		values, err := stringSlice(flagRecord["values"])
		if err != nil {
			return Observable{}, fmt.Errorf("flag %s values: %w", name, err)
		}
		flags[name] = FlagObservable{
			Assigned: requiredBool(flagRecord, "assigned"),
			Value:    requiredString(flagRecord, "value"),
			Values:   values,
		}
	}
	errInfo, err := decodeError(requiredRecord(record, "err"))
	if err != nil {
		return Observable{}, err
	}
	return Observable{
		Current:     requiredInt(record, "current"),
		OutArg:      requiredString(record, "outArg"),
		OutFlag:     requiredString(record, "outFlag"),
		OutMore:     requiredBool(record, "outMore"),
		Err:         errInfo,
		Flags:       flags,
		PendingFlag: requiredString(record, "pendingFlag"),
		PendingFrom: requiredString(record, "pendingFrom"),
	}, nil
}

func decodeError(record map[string]any) (ErrorInfo, error) {
	return ErrorInfo{
		Class: requiredString(record, "class"),
		Text:  requiredString(record, "text"),
	}, nil
}

func requiredRecord(record map[string]any, key string) map[string]any {
	value, ok := record[key]
	if !ok {
		panic(fmt.Sprintf("missing record key %q", key))
	}
	nested, ok := value.(map[string]any)
	if !ok {
		panic(fmt.Sprintf("key %q is %T, want record", key, value))
	}
	return nested
}

func requiredString(record map[string]any, key string) string {
	value, ok := record[key]
	if !ok {
		panic(fmt.Sprintf("missing string key %q", key))
	}
	s, ok := value.(string)
	if !ok {
		panic(fmt.Sprintf("key %q is %T, want string", key, value))
	}
	return s
}

func requiredBool(record map[string]any, key string) bool {
	value, ok := record[key]
	if !ok {
		panic(fmt.Sprintf("missing bool key %q", key))
	}
	b, ok := value.(bool)
	if !ok {
		panic(fmt.Sprintf("key %q is %T, want bool", key, value))
	}
	return b
}

func requiredInt(record map[string]any, key string) int {
	value, ok := record[key]
	if !ok {
		panic(fmt.Sprintf("missing int key %q", key))
	}
	i, ok := value.(int)
	if !ok {
		panic(fmt.Sprintf("key %q is %T, want int", key, value))
	}
	return i
}

func stringSlice(value any) ([]string, error) {
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("is %T, want sequence", value)
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("element is %T, want string", value)
		}
		out = append(out, s)
	}
	return out, nil
}

func writeJSON(path string, generated GeneratedCases) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(generated, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeCoverage(path string, coverage Coverage) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(renderCoverage(coverage)), 0o644)
}

func renderCoverage(coverage Coverage) string {
	var b bytes.Buffer
	fmt.Fprintln(&b, "# Parser Contract Case Coverage")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Generated from `formal/parser/generated/abstract_cases.dump`.")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Total cases: `%d`\n", coverage.TotalCases)
	fmt.Fprintf(&b, "- Allowed dash-leading enhancement cases: `%d`\n", coverage.AllowedEnhancementCases)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## State Coverage")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| stateName | cases |")
	fmt.Fprintln(&b, "|---|---:|")
	for _, state := range sortedKeys(coverage.StateCounts) {
		fmt.Fprintf(&b, "| `%s` | %d |\n", state, coverage.StateCounts[state])
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Token Coverage")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| tokenRaw | cases |")
	fmt.Fprintln(&b, "|---|---:|")
	for _, token := range sortedKeys(coverage.TokenCounts) {
		fmt.Fprintf(&b, "| `%s` | %d |\n", token, coverage.TokenCounts[token])
	}
	return b.String()
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type tlaParser struct {
	input string
	pos   int
}

func newTLAParser(input string) *tlaParser {
	return &tlaParser{input: input}
}

func (p *tlaParser) parseValue() (any, error) {
	p.skipSpace()
	switch {
	case p.consume("["):
		return p.parseRecord()
	case p.consume("<<"):
		return p.parseSeq()
	case p.peek() == '"':
		return p.parseString()
	case p.consume("TRUE"):
		return true, nil
	case p.consume("FALSE"):
		return false, nil
	case p.pos < len(p.input) && (unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '-'):
		return p.parseInt()
	default:
		return nil, fmt.Errorf("unexpected token near %q", p.remainingPreview())
	}
}

func (p *tlaParser) parseRecord() (map[string]any, error) {
	record := map[string]any{}
	for {
		p.skipSpace()
		if p.consume("]") {
			return record, nil
		}
		key, err := p.parseIdent()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if !p.consume("|->") {
			return nil, fmt.Errorf("expected |-> after %q near %q", key, p.remainingPreview())
		}
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		record[key] = value
		p.skipSpace()
		p.consume(",")
	}
}

func (p *tlaParser) parseSeq() ([]any, error) {
	var seq []any
	for {
		p.skipSpace()
		if p.consume(">>") {
			return seq, nil
		}
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		seq = append(seq, value)
		p.skipSpace()
		p.consume(",")
	}
}

func (p *tlaParser) parseString() (string, error) {
	if !p.consume("\"") {
		return "", fmt.Errorf("expected string near %q", p.remainingPreview())
	}
	var b strings.Builder
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		p.pos++
		if ch == '"' {
			return b.String(), nil
		}
		if ch == '\\' && p.pos < len(p.input) {
			next := p.input[p.pos]
			p.pos++
			b.WriteByte(next)
			continue
		}
		b.WriteByte(ch)
	}
	return "", errors.New("unterminated string")
}

func (p *tlaParser) parseInt() (int, error) {
	start := p.pos
	if p.input[p.pos] == '-' {
		p.pos++
	}
	for p.pos < len(p.input) && unicode.IsDigit(rune(p.input[p.pos])) {
		p.pos++
	}
	return strconv.Atoi(p.input[start:p.pos])
}

func (p *tlaParser) parseIdent() (string, error) {
	p.skipSpace()
	start := p.pos
	for p.pos < len(p.input) {
		r := rune(p.input[p.pos])
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			break
		}
		p.pos++
	}
	if start == p.pos {
		return "", fmt.Errorf("expected identifier near %q", p.remainingPreview())
	}
	return p.input[start:p.pos], nil
}

func (p *tlaParser) skipSpace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *tlaParser) consume(s string) bool {
	if strings.HasPrefix(p.input[p.pos:], s) {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *tlaParser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *tlaParser) remainingPreview() string {
	end := p.pos + 40
	if end > len(p.input) {
		end = len(p.input)
	}
	return p.input[p.pos:end]
}
