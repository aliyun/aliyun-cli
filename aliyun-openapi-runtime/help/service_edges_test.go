package help

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

type sourceWithoutProvenance struct{ DataSource }
type sourceWithNilProvenance struct{ DataSource }

func (sourceWithNilProvenance) Provenance(string) *source.Provenance { return nil }

func TestServiceFailureAtEachMetadataStage(t *testing.T) {
	cause := errors.New("metadata unavailable")
	calls := map[string]func(Service) error{
		"product": func(s Service) error { _, err := s.BuildProduct("demo", "v2", HelpOptions{}); return err },
		"action":  func(s Service) error { _, err := s.BuildAction("demo", "v2", "ListThings", HelpOptions{}); return err },
		"parameter": func(s Service) error {
			_, err := s.BuildParameter("demo", "v2", "ListThings", "resource-id", HelpOptions{})
			return err
		},
		"response": func(s Service) error {
			_, err := s.BuildResponse("demo", "v2", "ListThings", HelpOptions{})
			return err
		},
	}
	for name, call := range calls {
		for _, stage := range []string{"ensure", "resolve", "load"} {
			t.Run(name+"/"+stage, func(t *testing.T) {
				data, responses := newServiceTestFixtures()
				switch stage {
				case "ensure":
					data.ensureErr = cause
				case "resolve":
					data.resolveErr = cause
				case "load":
					data.apiErr, data.indexErr = cause, cause
				}
				if err := call(Service{Data: data, Responses: responses}); !errors.Is(err, cause) {
					t.Fatalf("error = %v, want original metadata error", err)
				}
			})
		}
	}
	if _, err := (Service{}).BuildAction("demo", "v2", "ListThings", HelpOptions{}); err == nil {
		t.Fatal("missing source accepted")
	}
	data, responses := newServiceTestFixtures()
	responses.err = cause
	if _, err := (Service{Data: data, Responses: responses}).BuildRequest("demo", "v2", "ListThings", HelpOptions{}); !errors.Is(err, cause) {
		t.Fatalf("response source error = %v", err)
	}
	for _, provider := range []DataSource{sourceWithoutProvenance{data}, sourceWithNilProvenance{data}} {
		doc, err := (Service{Data: provider}).BuildProduct("demo", "v2", HelpOptions{})
		if err != nil || doc.Provenance != nil {
			t.Fatalf("optional provenance = %#v, error = %v", doc, err)
		}
	}
}

func TestHelpDocumentFallbacksAndSearchBudget(t *testing.T) {
	data, _ := newServiceTestFixtures()
	data.index.Entries = map[string]meta.APIIndexEntry{"ListThings": {APIName: "ListThings"}}
	product := BuildProductDocument(data.product, data.index, HelpOptions{})
	if len(product.APIs) != 1 || product.APIs[0].Command != "list-things" {
		t.Fatalf("fallback command = %#v", product.APIs)
	}
	for _, missing := range []func() error{
		func() error { _, err := BuildAPIParameterDocument(nil, nil, "id", HelpOptions{}); return err },
		func() error { _, err := BuildAPIResponseDocument(nil, nil, HelpOptions{}); return err },
	} {
		if err := missing(); err == nil {
			t.Fatal("missing API accepted")
		}
	}
	api := &meta.API{Name: "RunThing", ProductCode: "demo"}
	for i := 0; i < 25; i++ {
		api.Parameters = append(api.Parameters, meta.Parameter{Name: fmt.Sprintf("cli_parameter_%02d", i), Type: meta.TypeString})
	}
	doc := BuildRequestDocument(nil, api, nil, HelpOptions{Search: "cli"})
	if len(doc.Parameters) != searchResultLimit || len(doc.GlobalParameters) != 0 || !doc.Result.Truncated || doc.Result.Total <= searchResultLimit {
		t.Fatalf("search budget = %#v, parameters=%d globals=%d", doc.Result, len(doc.Parameters), len(doc.GlobalParameters))
	}
	parameter := Parameter{Fields: []Parameter{{}}, Element: &Parameter{Fields: []Parameter{{}}}, Value: &Parameter{}}
	if got := parameterLines(parameter); got != 5 {
		t.Fatalf("nested logical lines = %d", got)
	}
	var required []meta.Parameter
	for i := 0; i < 6; i++ {
		required = append(required, meta.Parameter{Name: fmt.Sprintf("Field%d", i), Required: true})
	}
	flags := requiredFlags(required)
	if len(flags) != 4 || flags[0] != "--field-0" {
		t.Fatalf("required flags = %#v", flags)
	}
}

func TestParameterSuggestionsAndDescendantRanking(t *testing.T) {
	candidates := []parameterCandidate{
		{path: "zulu", aliases: []string{"nothing"}}, {path: "bravo", aliases: []string{"bravo"}},
		{path: "alpha", aliases: []string{"alpha"}}, {path: "region", aliases: []string{"region"}},
		{path: "charlie", aliases: []string{"charlie"}}, {path: "delta", aliases: []string{"delta"}},
	}
	want := []string{"--region", "--alpha", "--bravo", "--charlie", "--delta"}
	if got := candidateParameters(candidates, "region"); !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %v", got)
	}
	root := Parameter{Name: "config", Fields: []Parameter{
		{Name: "retry", Help: LocalizedText{EN: "Choose a regional endpoint"}},
		{Name: "region"},
	}}
	matches := searchParameterDescendants(root, "region")
	if len(matches) != 2 || matches[0].Path != "config.region" || !strings.Contains(matches[1].Parameter.Help.EN, "regional") {
		t.Fatalf("ranked matches = %#v", matches)
	}
}
