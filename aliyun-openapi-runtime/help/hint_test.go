package help

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func runtimeNextJSON(t *testing.T, next *Next) map[string]any {
	t.Helper()
	if next == nil {
		t.Fatal("next is nil")
	}
	raw, err := json.Marshal(next)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func hintTestProductIndex(count int) (*meta.Product, *meta.APIIndex) {
	entries := make(map[string]meta.APIIndexEntry, count)
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("DescribeReport%02d", i)
		entries[name] = meta.APIIndexEntry{APIName: name, CmdName: fmt.Sprintf("describe-report-%02d", i)}
	}
	return &meta.Product{Code: "demo", Versions: []string{"2026-01-01"}}, &meta.APIIndex{
		ProductCode: "demo", Version: "2026-01-01", Entries: entries,
	}
}

func hintTestAPI(composite bool) *meta.API {
	filter := meta.Parameter{
		Name: "report_filter", RawName: "ReportFilter", Type: meta.TypeObject,
		Options: []string{"--report-filter"},
	}
	if composite {
		filter.Fields = []meta.Parameter{{Name: "status", RawName: "Status", Type: meta.TypeString}}
	}
	return &meta.API{
		Name: "CreateReport", CmdName: "create-report", ProductCode: "demo", Version: "2026-01-01",
		Parameters: []meta.Parameter{
			filter,
			{Name: "report_filter_extra", RawName: "ReportFilterExtra", Type: meta.TypeString, Options: []string{"--report-filter-extra"}},
		},
	}
}

func TestRuntimeHelpAllAndSearchAlwaysExposeCurrentLevelNavigation(t *testing.T) {
	product, index := hintTestProductIndex(25)

	t.Run("all offers search even when complete", func(t *testing.T) {
		document := BuildProductDocument(product, index, HelpOptions{All: true})
		if document.Result.Truncated {
			t.Fatalf("all result unexpectedly truncated: %+v", document.Result)
		}
		next := runtimeNextJSON(t, document.Next)
		if next["search"] != "aliyun demo --help-search <keyword>" {
			t.Fatalf("search = %v", next["search"])
		}
		if _, ok := next["showAll"]; ok {
			t.Fatalf("all next self-references showAll: %#v", next)
		}
	})

	tests := []struct {
		name      string
		query     string
		all       bool
		wantShown int
		wantTotal int
		truncated bool
		searchAll bool
	}{
		{name: "zero of zero", query: "absent", wantShown: 0, wantTotal: 0, searchAll: true},
		{name: "one of one", query: "describe-report-00", wantShown: 1, wantTotal: 1, searchAll: true},
		{name: "twenty of N", query: "report", wantShown: 20, wantTotal: 25, truncated: true, searchAll: true},
		{name: "search all has no self reference", query: "report", all: true, wantShown: 25, wantTotal: 25},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := BuildProductDocument(product, index, HelpOptions{Search: test.query, All: test.all})
			if document.Result != (Result{Shown: test.wantShown, Total: test.wantTotal, Truncated: test.truncated}) {
				t.Fatalf("result = %+v", document.Result)
			}
			next := runtimeNextJSON(t, document.Next)
			if next["search"] != "aliyun demo --help-search <keyword>" {
				t.Fatalf("search = %v", next["search"])
			}
			_, hasSearchAll := next["searchAll"]
			if hasSearchAll != test.searchAll {
				t.Fatalf("searchAll presence = %t, want %t: %#v", hasSearchAll, test.searchAll, next)
			}
		})
	}
}

func TestRuntimeProgressiveChildSearchExactFuzzyAmbiguousAndComposite(t *testing.T) {
	t.Run("product exact keeps fuzzy sibling and descends", func(t *testing.T) {
		product := &meta.Product{Code: "demo", Versions: []string{"2026-01-01"}}
		index := &meta.APIIndex{ProductCode: "demo", Version: "2026-01-01", Entries: map[string]meta.APIIndexEntry{
			"CreateReport":       {APIName: "CreateReport", CmdName: "create-report"},
			"CreateReportDetail": {APIName: "CreateReportDetail", CmdName: "create-report-detail"},
		}}
		document := BuildProductDocument(product, index, HelpOptions{
			Search: "CREATE_REPORT", RequestedVersion: "2026-01-01", Format: FormatJSON,
		})
		if document.Result.Total != 2 {
			t.Fatalf("fuzzy sibling missing: %+v", document.Result)
		}
		next := runtimeNextJSON(t, document.Next)
		want := "aliyun demo create-report --api-version 2026-01-01 --help-search <keyword> --cli-output json"
		if next["childSearch"] != want {
			t.Fatalf("childSearch = %v, want %q", next["childSearch"], want)
		}
	})

	t.Run("ambiguous exact does not descend", func(t *testing.T) {
		product := &meta.Product{Code: "demo"}
		index := &meta.APIIndex{ProductCode: "demo", Version: "v1", Entries: map[string]meta.APIIndexEntry{
			"CreateReport": {APIName: "CreateReport", CmdName: "create-report"},
			"Other":        {APIName: "Other", CmdName: "create_report"},
		}}
		document := BuildProductDocument(product, index, HelpOptions{Search: "create-report"})
		if _, ok := runtimeNextJSON(t, document.Next)["childSearch"]; ok {
			t.Fatalf("ambiguous exact unexpectedly descended: %#v", document.Next)
		}
	})

	t.Run("composite parameter descends from API and Request section", func(t *testing.T) {
		for _, explicit := range []bool{false, true} {
			options := HelpOptions{Search: "REPORT_FILTER"}
			var document any
			if explicit {
				document = BuildRequestDocument(&meta.Product{Code: "demo"}, hintTestAPI(true), nil, options)
			} else {
				document = BuildActionDocument(&meta.Product{Code: "demo"}, hintTestAPI(true), nil, options)
			}
			var next *Next
			switch typed := document.(type) {
			case *ActionDocument:
				next = typed.Next
			case *RequestDocument:
				next = typed.Next
			}
			encoded := runtimeNextJSON(t, next)
			want := "aliyun demo create-report --report-filter --help-search <keyword>"
			if encoded["childSearch"] != want {
				t.Fatalf("explicit=%t childSearch=%v", explicit, encoded["childSearch"])
			}
			if explicit && !strings.Contains(encoded["search"].(string), "aliyun help demo create-report --cli-section request") {
				t.Fatalf("request search is not section-scoped: %#v", encoded)
			}
		}
	})

	t.Run("scalar and fuzzy parameter do not descend", func(t *testing.T) {
		for _, test := range []struct {
			name      string
			query     string
			composite bool
		}{
			{name: "scalar", query: "report_filter", composite: false},
			{name: "fuzzy", query: "report", composite: true},
		} {
			t.Run(test.name, func(t *testing.T) {
				document := BuildActionDocument(&meta.Product{Code: "demo"}, hintTestAPI(test.composite), nil, HelpOptions{Search: test.query})
				if _, ok := runtimeNextJSON(t, document.Next)["childSearch"]; ok {
					t.Fatalf("unexpected childSearch: %#v", document.Next)
				}
			})
		}
	})

	t.Run("parameter and response are terminals", func(t *testing.T) {
		parameter, err := BuildAPIParameterDocument(&meta.Product{Code: "demo"}, hintTestAPI(true), "report-filter", HelpOptions{Search: "status"})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := runtimeNextJSON(t, parameter.Next)["childSearch"]; ok {
			t.Fatalf("parameter Help descended: %#v", parameter.Next)
		}

		response, err := BuildAPIResponseDocument(hintTestAPI(true), &ResponseDocumentation{
			Schema: []byte(`{"type":"object","properties":{"ReportId":{"type":"string"}}}`),
		}, HelpOptions{Search: "report-id"})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := runtimeNextJSON(t, response.Next)["childSearch"]; ok {
			t.Fatalf("response Help descended: %#v", response.Next)
		}
	})
}

func TestRuntimeHelpHintTextAndJSONContinuity(t *testing.T) {
	product, index := hintTestProductIndex(1)

	t.Run("default text machine command precedes AI hint", func(t *testing.T) {
		document := BuildProductDocument(product, index, HelpOptions{})
		var output bytes.Buffer
		if err := Render(&output, document, HelpOptions{}); err != nil {
			t.Fatal(err)
		}
		machine := "For machine-readable Help, run:\n  aliyun demo --help --cli-output json"
		ai := "For AI agents, run:"
		if !strings.Contains(output.String(), machine) || !strings.Contains(output.String(), ai) ||
			strings.Index(output.String(), machine) > strings.Index(output.String(), ai) {
			t.Fatalf("footer ordering/output:\n%s", output.String())
		}
	})

	t.Run("all and search labels are localized and commands are optionally JSON", func(t *testing.T) {
		all := BuildProductDocument(product, index, HelpOptions{All: true, Language: "zh"})
		var allOutput bytes.Buffer
		if err := Render(&allOutput, all, HelpOptions{All: true, Language: "zh"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(allOutput.String(), "搜索当前 Help：\n  aliyun demo --help-search <keyword> [--cli-output json]") {
			t.Fatalf("Chinese all hint missing:\n%s", allOutput.String())
		}

		searched := BuildProductDocument(product, index, HelpOptions{Search: "missing", Language: "en"})
		var searchOutput bytes.Buffer
		if err := Render(&searchOutput, searched, HelpOptions{Search: "missing", Language: "en"}); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{
			"Try another keyword:\n  aliyun demo --help-search <keyword> [--cli-output json]",
			"Show all matches:\n  aliyun demo --help-search missing --help-all [--cli-output json]",
		} {
			if !strings.Contains(searchOutput.String(), want) {
				t.Fatalf("missing %q:\n%s", want, searchOutput.String())
			}
		}
	})

	t.Run("non AI JSON retains aiModeHint and executable next", func(t *testing.T) {
		document := BuildProductDocument(product, index, HelpOptions{Search: "report", Format: FormatJSON})
		var output bytes.Buffer
		if err := Render(&output, document, HelpOptions{Search: "report", Format: FormatJSON}); err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(output.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if value["aiModeHint"] == nil {
			t.Fatalf("aiModeHint missing: %s", output.String())
		}
		next := value["next"].(map[string]any)
		if !strings.HasSuffix(next["search"].(string), "--cli-output json") ||
			!strings.HasSuffix(next["searchAll"].(string), "--cli-output json") {
			t.Fatalf("JSON next commands are not continuous: %#v", next)
		}
	})

	t.Run("AI JSON omits redundant output flag and aiModeHint", func(t *testing.T) {
		document := BuildProductDocument(product, index, HelpOptions{Search: "report", AIMode: true})
		var output bytes.Buffer
		if err := Render(&output, document, HelpOptions{Search: "report", AIMode: true, Format: FormatJSON}); err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(output.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if _, ok := value["aiModeHint"]; ok {
			t.Fatalf("AI JSON contains redundant aiModeHint: %s", output.String())
		}
		next := value["next"].(map[string]any)
		if strings.Contains(next["search"].(string), "--cli-output") || strings.Contains(next["searchAll"].(string), "--cli-output") {
			t.Fatalf("AI JSON next contains redundant output flag: %#v", next)
		}
	})

	t.Run("renderer normalizes next output continuity", func(t *testing.T) {
		document := BuildProductDocument(product, index, HelpOptions{Search: "report"})
		var output bytes.Buffer
		if err := Render(&output, document, HelpOptions{Search: "report", Format: FormatJSON}); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(document.Next.Search, "--cli-output json") {
			t.Fatalf("non-AI JSON did not add output flag: %#v", document.Next)
		}
		output.Reset()
		if err := Render(&output, document, HelpOptions{Search: "report", Format: FormatJSON, AIMode: true}); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(document.Next.Search, "--cli-output") {
			t.Fatalf("AI JSON retained output flag: %#v", document.Next)
		}
	})
}

func TestRuntimeHintCommandsFallBackToKebabAPIName(t *testing.T) {
	api := hintTestAPI(true)
	api.CmdName = ""

	parameter, err := BuildAPIParameterDocument(&meta.Product{Code: "demo"}, api, "report-filter", HelpOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := parameter.Next.Search, "aliyun demo create-report --report-filter --help-search <keyword>"; got != want {
		t.Fatalf("parameter fallback search = %q, want %q", got, want)
	}

	response, err := BuildAPIResponseDocument(api, &ResponseDocumentation{
		Schema: []byte(`{"type":"object"}`),
	}, HelpOptions{Search: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := response.Next.Search, "aliyun help demo create-report --cli-section response --help-search <keyword>"; got != want {
		t.Fatalf("response fallback search = %q, want %q", got, want)
	}
}
