package help

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestTextRenderersHideProvenanceButDocumentsKeepIt(t *testing.T) {
	provenance := &MetadataProvenance{Kind: "plugin", PluginName: "aliyun-cli-demo", PluginVersion: "1.2.3"}
	documents := []any{
		&ProductDocument{Provenance: provenance},
		&ActionDocument{Provenance: provenance},
		&RequestDocument{Provenance: provenance},
		&APIParameterDocument{Provenance: provenance},
		&APIResponseDocument{Provenance: provenance},
	}
	for _, document := range documents {
		var text bytes.Buffer
		if err := Render(&text, document, HelpOptions{Format: FormatText, Language: "en"}); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(text.String(), "Metadata provider") || strings.Contains(text.String(), "aliyun-cli-demo") {
			t.Fatalf("Text Help leaked provenance for %T:\n%s", document, text.String())
		}

		if provenance.PluginName != "aliyun-cli-demo" || provenance.PluginVersion != "1.2.3" {
			t.Fatalf("Text rendering mutated provenance for %T: %#v", document, provenance)
		}
	}
}

func TestActionAndRequestTextGroupTopLevelParametersStably(t *testing.T) {
	parameters := []Parameter{
		{Name: "optional-first", Type: meta.TypeString},
		{Name: "required-first", Type: meta.TypeString, Required: true, Fields: []Parameter{{Name: "nested-required", Type: meta.TypeString, Required: true}}},
		{Name: "optional-second", Type: meta.TypeString},
		{Name: "required-second", Type: meta.TypeString, Required: true},
	}
	for _, document := range []any{
		&ActionDocument{Command: "create-thing", Parameters: parameters},
		&RequestDocument{Command: "create-thing", Parameters: parameters},
	} {
		var output bytes.Buffer
		if err := Render(&output, document, HelpOptions{Format: FormatText, Language: "en"}); err != nil {
			t.Fatal(err)
		}
		text := output.String()
		for _, want := range []string{"Required Parameters:", "Optional Parameters:", "--required-first", "--required-second", "--optional-first", "--optional-second"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%T missing %q:\n%s", document, want, text)
			}
		}
		if !(strings.Index(text, "--required-first") < strings.Index(text, "--required-second") &&
			strings.Index(text, "--required-second") < strings.Index(text, "--optional-first") &&
			strings.Index(text, "--optional-first") < strings.Index(text, "--optional-second")) {
			t.Fatalf("%T did not preserve stable group order:\n%s", document, text)
		}
		if strings.Contains(text, "(required)") || strings.Contains(text, "(optional)") {
			t.Fatalf("%T repeated inline requirement labels:\n%s", document, text)
		}
	}
}

func TestParameterGroupsHideEmptySectionsAndLocalize(t *testing.T) {
	tests := []struct {
		name       string
		language   string
		parameters []Parameter
		present    string
		absent     string
	}{
		{name: "required only English", language: "en", parameters: []Parameter{{Name: "id", Type: meta.TypeString, Required: true}}, present: "Required Parameters:", absent: "Optional Parameters:"},
		{name: "optional only Chinese", language: "zh", parameters: []Parameter{{Name: "name", Type: meta.TypeString}}, present: "可选参数:", absent: "必填参数:"},
		{name: "empty", language: "en", present: "Usage:", absent: "Parameters:"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			document := &RequestDocument{Command: "list-things", Parameters: test.parameters}
			if err := Render(&output, document, HelpOptions{Format: FormatText, Language: test.language}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), test.present) || strings.Contains(output.String(), test.absent) {
				t.Fatalf("unexpected localized groups:\n%s", output.String())
			}
		})
	}
}

func TestParameterTextUsesStructuredOrderedBlockAndKeepsNestedShapes(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "60")
	url := "https://help.aliyun.com/document_detail/12345678901234567890.html"
	document := &APIParameterDocument{
		Target:  Target{Product: "demo", API: "CreateThing", APIVersion: "2026-01-01"},
		Command: "create-thing",
		Parameter: Parameter{
			Name: "config", RawName: "Config", Type: meta.TypeObject, Location: meta.PosQuery,
			Required: true, Serialization: "json", Help: LocalizedText{EN: "First paragraph.\n\nSecond paragraph. Valid values: safe fast\n请使用" + url + "接口"},
			Example: `{"mode":"safe"}`,
			Fields: []Parameter{
				{Name: "mode", RawName: "Mode", Type: meta.TypeString, Required: true, Constraints: Constraints{Enum: []string{"safe", "fast"}}},
				{Name: "fallback-name", Type: meta.TypeString},
			},
			Element: &Parameter{Type: meta.TypeString, Constraints: Constraints{Pattern: "^res-"}},
			Value:   &Parameter{Type: meta.TypeInteger},
		},
	}
	var output bytes.Buffer
	if err := Render(&output, document, HelpOptions{Format: FormatText, Language: "en"}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	ordered := []string{
		"Parameter: --config", "API: create-thing", "API Version: 2026-01-01", "Type: object",
		"Location: query", "Required: true", "Description:", "First paragraph.\n\n  Second paragraph.",
		"Example:", `{"mode":"safe"}`, "Fields:", "Mode", "Constraints:", "Enum: safe, fast", "Element:", "Value:",
	}
	last := -1
	for _, want := range ordered {
		index := strings.Index(text, want)
		if index < 0 || index < last {
			t.Fatalf("structured parameter block missing or out of order at %q:\n%s", want, text)
		}
		last = index
	}
	if !strings.Contains(text, "  Valid values:\n  safe fast") || !strings.Contains(text, "\n  "+url+"\n") || strings.Contains(text, "help.aliyun.com/\n") {
		t.Fatalf("structured description did not preserve marker/URL wrapping:\n%s", text)
	}
}

type runtimeHelpFDWriter struct {
	bytes.Buffer
	fd uintptr
}

func (w *runtimeHelpFDWriter) Fd() uintptr { return w.fd }

func TestHelpWidthAndWrappingContracts(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "")
	originalIsTerminal, originalGetSize := helpIsTerminal, helpGetSize
	helpIsTerminal = func(fd int) bool { return fd == 42 }
	helpGetSize = func(fd int) (int, int, error) { return 140, 40, nil }
	t.Cleanup(func() { helpIsTerminal, helpGetSize = originalIsTerminal, originalGetSize })
	if got := helpMaxLineLength(&runtimeHelpFDWriter{fd: 42}); got != 120 {
		t.Fatalf("terminal width cap = %d, want 120", got)
	}
	if got := helpMaxLineLength(&bytes.Buffer{}); got != 80 {
		t.Fatalf("non-TTY width = %d, want 80", got)
	}
	for _, explicit := range []string{"60", "100", "120"} {
		t.Run(explicit, func(t *testing.T) {
			t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", explicit)
			if got := helpMaxLineLength(&runtimeHelpFDWriter{fd: 42}); got != mustAtoi(t, explicit) {
				t.Fatalf("explicit width = %d, want %s", got, explicit)
			}
		})
	}

	lines := wrapHelpText("First paragraph.\n\nSecond paragraph. Valid values: one two", 40)
	want := []string{"First paragraph.", "", "Second paragraph.", "Valid values:", "one two"}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("paragraph/marker wrapping = %#v, want %#v", lines, want)
	}
	if got := wrapHelpText("Valid values: one two", 40); strings.Join(got, "|") != "Valid values:|one two" {
		t.Fatalf("leading marker wrapping = %#v", got)
	}
	url := "https://example.com/a/very/long/path/that/must/not/break"
	lines = wrapHelpText("See "+url+" for details.", 20)
	if strings.Join(lines, "|") != "See|"+url+"|for details." {
		t.Fatalf("URL wrapping = %#v", lines)
	}
	lines = wrapHelpText("请使用"+url+"接口查看详情。", 20)
	if strings.Join(lines, "|") != "请使用|"+url+"|接口查看详情。" {
		t.Fatalf("adjacent Chinese URL wrapping = %#v", lines)
	}
}

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	result := 0
	for _, digit := range value {
		result = result*10 + int(digit-'0')
	}
	return result
}

func TestParameterSummaryFormattingCoversCompositeShapes(t *testing.T) {
	primitive := &Parameter{Type: meta.TypeString}
	flatObject := &Parameter{Type: meta.TypeObject, Fields: []Parameter{
		{Name: "zeta", Type: meta.TypeString},
		{Name: "alpha", Type: meta.TypeInteger},
	}}
	nestedObject := &Parameter{Type: meta.TypeObject, Fields: []Parameter{
		{Name: "nested", Type: meta.TypeObject, Fields: []Parameter{{Name: "value", Type: meta.TypeString}}},
	}}
	tests := []struct {
		name          string
		parameter     Parameter
		wantStructure string
		wantFormat    string
	}{
		{name: "primitive array", parameter: Parameter{Name: "ids", Type: meta.TypeArray, Element: primitive}, wantFormat: "--ids value1 value2 value3"},
		{name: "flat object array", parameter: Parameter{Name: "items", Type: meta.TypeArray, Element: flatObject}, wantStructure: "{zeta: string, alpha: int}", wantFormat: "--items alpha=a zeta=b"},
		{name: "nested object array", parameter: Parameter{Name: "items", Type: meta.TypeArray, Element: nestedObject}, wantStructure: "[{nested: {value: string}}, ...]", wantFormat: "--items 'value'"},
		{name: "empty object", parameter: Parameter{Name: "config", Type: meta.TypeObject}, wantStructure: "{<key>=<value>, ...}", wantFormat: "--config 'value'"},
		{name: "flat object", parameter: Parameter{Name: "config", Type: meta.TypeObject, Fields: flatObject.Fields}, wantStructure: "{zeta: string, alpha: int}", wantFormat: "--config alpha=xxx zeta=xxx"},
		{name: "nested object", parameter: Parameter{Name: "config", Type: meta.TypeObject, Fields: nestedObject.Fields}, wantStructure: "{nested: {value: string}}", wantFormat: "--config 'value'"},
		{name: "primitive map", parameter: Parameter{Name: "labels", Type: meta.TypeMap, Value: primitive}, wantStructure: "{<key>: string, ...}", wantFormat: "--labels key1=value1"},
		{name: "untyped map", parameter: Parameter{Name: "labels", Type: meta.TypeMap}, wantStructure: "{<key>: <value>, ...}", wantFormat: "--labels key1=value1"},
		{name: "object map", parameter: Parameter{Name: "labels", Type: meta.TypeMap, Value: nestedObject}, wantStructure: "{<key>: {nested: {value: string}}, ...}", wantFormat: "--labels 'value'"},
		{name: "scalar", parameter: Parameter{Name: "count", Type: meta.TypeLong}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			structure, format := parameterFormatHints(test.parameter)
			if test.wantStructure != "" && structure != test.wantStructure {
				t.Fatalf("structure = %q, want %q", structure, test.wantStructure)
			}
			if test.wantFormat != "" && !strings.Contains(format, test.wantFormat) {
				t.Fatalf("format = %q, want substring %q", format, test.wantFormat)
			}
			if test.wantStructure == "" && structure != "" || test.wantFormat == "" && format != "" {
				t.Fatalf("unexpected scalar hints: %q %q", structure, format)
			}
		})
	}

	for dataType, want := range map[meta.DataType]string{
		meta.TypeMap: "string", meta.TypeArray: "list", meta.TypeInteger: "int", meta.TypeLong: "int",
		meta.TypeBoolean: "bool", meta.TypeFloat: "float",
	} {
		if got := displayParameterType(dataType); got != want {
			t.Fatalf("display type %q = %q, want %q", dataType, got, want)
		}
	}

	parameter := Parameter{
		Name: "root", Type: meta.TypeObject,
		Constraints: Constraints{Enum: []string{"one", "two"}},
		Fields:      []Parameter{{Name: "child", Type: meta.TypeString, Help: LocalizedText{EN: "child help"}}},
		Element:     &Parameter{Type: meta.TypeArray, Element: primitive},
		Value:       &Parameter{Type: meta.TypeMap, Value: primitive},
	}
	var output bytes.Buffer
	renderParameterText(&output, parameter, "en", "  ", true)
	for _, want := range []string{"--root", "--child", "Element:", "Value:"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expanded summary missing %q:\n%s", want, output.String())
		}
	}

	output.Reset()
	renderParameterShapeText(&output, parameter, "en", "  ")
	if !strings.Contains(output.String(), "--child") || !strings.Contains(output.String(), "Element:") || !strings.Contains(output.String(), "Value:") {
		t.Fatalf("shape renderer lost recursive branches:\n%s", output.String())
	}
}

func TestWidthFallbackAndWrapperEdgeCases(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "invalid")
	if got := helpMaxLineLength(&bytes.Buffer{}); got != 80 {
		t.Fatalf("invalid explicit width = %d", got)
	}
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "")
	originalIsTerminal, originalGetSize := helpIsTerminal, helpGetSize
	helpIsTerminal = func(int) bool { return true }
	helpGetSize = func(int) (int, int, error) { return 60, 20, nil }
	if got := helpMaxLineLength(&runtimeHelpFDWriter{fd: 7}); got != 60 {
		t.Fatalf("terminal width = %d", got)
	}
	helpIsTerminal = func(int) bool { return false }
	if got := helpMaxLineLength(&runtimeHelpFDWriter{fd: 7}); got != 80 {
		t.Fatalf("non-terminal FD width = %d", got)
	}
	helpIsTerminal, helpGetSize = originalIsTerminal, originalGetSize

	if got := wrapHelpText(" ", 1); len(got) != 1 || got[0] != "" {
		t.Fatalf("blank wrapper result = %#v", got)
	}
	if got := wrapHelpURLLine("plain text", 20); strings.Join(got, " ") != "plain text" {
		t.Fatalf("URL fallback result = %#v", got)
	}
	var output bytes.Buffer
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "20")
	writeHelpWrappedWithWidth(&output, "prefix", "", 2, 20)
	writeHelpWrappedWithWidth(&output, "a very long prefix that cannot fit", "wrapped\n\ntext", 2, 1)
	if !strings.Contains(output.String(), "prefix\n") || !strings.Contains(output.String(), "wrapped\n\n  text") {
		t.Fatalf("wrapper edge output = %q", output.String())
	}
}

type runtimeFailWriter struct{}

func (runtimeFailWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestTextRendererOptionalBranchesAndErrors(t *testing.T) {
	product := &ProductDocument{
		Product: Product{Code: "demo"},
		APIs:    []APISummary{{Command: "list-things", Description: LocalizedText{EN: "List things"}}},
		Result:  Result{Shown: 1, Total: 3, Truncated: true, OmittedDeprecated: 2},
		Next:    &Next{ShowAll: "show-all", Search: "search", SearchAll: "search-all"},
	}
	var output bytes.Buffer
	if err := Render(&output, product, HelpOptions{Format: FormatText}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"List things", "Omitting 2 deprecated APIs", "Show all: show-all", "Search: search"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("product optional branch missing %q:\n%s", want, output.String())
		}
	}

	queryOptions := []QueryOption{{Name: "--query", Type: "string", HasDefault: true, Default: "request", Help: LocalizedText{EN: "Query"}}}
	queryExample := &QueryExample{Path: "Items.Item", SchemaCommand: "schema", QueryCommand: "query"}
	for _, document := range []any{
		&ActionDocument{Command: "list-things", Description: LocalizedText{EN: "Fallback description"}, QueryOptions: queryOptions, ResponseQuery: queryExample, Examples: Examples{Kebab: "action-example"}, Result: Result{Shown: 1, Total: 2, Truncated: true}},
		&RequestDocument{Command: "list-things", QueryOptions: queryOptions, ResponseQuery: queryExample, Examples: Examples{Kebab: "request-example"}},
	} {
		output.Reset()
		if err := Render(&output, document, HelpOptions{Format: FormatText}); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"default: request", "Response aggregation example", "Example:"} {
			if !strings.Contains(output.String(), want) {
				t.Fatalf("%T optional branch missing %q:\n%s", document, want, output.String())
			}
		}
	}

	output.Reset()
	parameter := &APIParameterDocument{Query: "absent", Result: Result{Shown: 0, Total: 0}}
	if err := Render(&output, parameter, HelpOptions{Format: FormatText, Language: "zh"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "没有参数条目匹配") {
		t.Fatalf("parameter zero-match branch missing:\n%s", output.String())
	}

	if err := renderResult(runtimeFailWriter{}, Result{Shown: 1, Total: 2, Truncated: true}, "en"); err == nil {
		t.Fatal("renderResult did not propagate writer failure")
	}
	if err := writePrettyJSON(runtimeFailWriter{}, []byte(`{"ok":true}`)); err == nil {
		t.Fatal("writePrettyJSON did not propagate writer failure")
	}
	if err := writePrettyJSON(&bytes.Buffer{}, []byte(`{`)); err == nil {
		t.Fatal("writePrettyJSON accepted malformed JSON")
	}
}
