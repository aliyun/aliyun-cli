package engine

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestStructuredHelpRejectsMissingTargetsAndLoaderFailures(t *testing.T) {
	cause := errors.New("loader failed")
	broken := NewEngine(func() (loader.Loader, error) { return nil, cause }, nil)
	for _, call := range []func(*Engine) error{
		func(e *Engine) error { _, err := e.BuildAPIHelp(Request{}, HelpOptions{}); return err },
		func(e *Engine) error { _, err := e.BuildParameterHelp(Request{}, "id", HelpOptions{}); return err },
	} {
		if err := call(broken); !errors.Is(err, cause) {
			t.Fatalf("loader error = %v", err)
		}
		if err := call(newDispatchErrorTestEngine()); err == nil || !strings.Contains(err.Error(), "product and command") {
			t.Fatalf("missing target error = %v", err)
		}
	}
	for _, args := range [][]string{nil, {" "}} {
		if _, err := newDispatchErrorTestEngine().BuildProductHelp(Request{Args: args}, HelpOptions{}); err == nil || !strings.Contains(err.Error(), "product is required") {
			t.Fatalf("missing product error = %v", err)
		}
	}
	req := Request{Args: []string{"demo", "unknown"}}
	if _, err := newDispatchErrorTestEngine().BuildAPIHelp(req, HelpOptions{}); err == nil {
		t.Fatal("unknown API accepted")
	}
	if _, err := newDispatchErrorTestEngine().BuildParameterHelp(req, "id", HelpOptions{}); err == nil {
		t.Fatal("unknown API parameter accepted")
	}
	if err := RenderHelp(nil, nil, HelpOptions{Query: "target"}); err == nil || !strings.Contains(err.Error(), "output is nil") {
		t.Fatalf("nil output error = %v", err)
	}
	if err := RenderHelp(io.Discard, make(chan int), HelpOptions{Query: "target"}); err == nil {
		t.Fatal("unserializable document accepted")
	}
}

func TestParameterHelpRejectsAmbiguousTargetsAndInvalidArguments(t *testing.T) {
	parameters := []meta.Parameter{{RawName: "InstanceType", Type: meta.TypeString}, {Name: "change_reason", Type: meta.TypeString}}
	target, cleaned, found, err := detectParameterHelp(parameters, []string{"--instance-type", "--help"})
	if err != nil || !found || target != "instance-type" || len(cleaned) != 1 {
		t.Fatalf("RawName fallback = %q %v %v %v", target, cleaned, found, err)
	}
	if _, _, found, err := detectParameterHelp(parameters, []string{"--instance-type", "--help", "--change-reason", "--help"}); !found || err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous target error = %v, found=%v", err, found)
	}
	for _, tc := range []struct {
		tail []string
		code string
	}{
		{[]string{"--instance-type", "--help", "--unknown", "value"}, "INVALID_ARGUMENT"},
		{[]string{"--instance-type", "--help", "--help-search", "region"}, "INVALID_OPTION_COMBINATION"},
	} {
		found, err := newDispatchErrorTestEngine().tryParameterHelp(Request{Args: []string{"demo", "run-thing"}, Out: io.Discard}, parameters, tc.tail)
		var usage *UsageError
		if !found || !errors.As(err, &usage) || usage.Code != tc.code {
			t.Fatalf("parameter help = %v %v", found, err)
		}
	}
	if err := validateReservedHelp(argparser.Reserved{HelpSection: "response"}, true); err == nil {
		t.Fatal("product response section accepted")
	}
	found, err = newDispatchErrorTestEngine().tryParameterHelp(Request{Args: []string{"demo", "unknown"}, Out: io.Discard}, parameters, []string{"--instance-type", "--help"})
	if !found || err == nil {
		t.Fatalf("unknown target parameter help = %v %v", found, err)
	}
	if got := kebabName("AccessKeyID"); got != "access-key-i-d" {
		t.Fatalf("kebab name = %q", got)
	}
}

type helpResponseErrorSource struct {
	dispatchErrorTestSource
	responses, components []byte
}

func (s helpResponseErrorSource) LoadAPI(product, version, name string) (*meta.API, error) {
	api, err := s.dispatchErrorTestSource.LoadAPI(product, version, name)
	if err == nil {
		api.Responses, api.Components = s.responses, s.components
	}
	return api, err
}

func TestResponseHelpPreservesMetadataErrors(t *testing.T) {
	if _, err := (runtimeResponseSource{loader: loader.New(dispatchErrorTestSource{})}).GetResponseDocumentation("demo", "2024-01-01", "Missing"); err == nil {
		t.Fatal("missing response API accepted")
	}
	for _, tc := range []struct{ responses, components string }{
		{"{", ""},
		{`{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Result"}}}}}`, `[]`},
	} {
		s := runtimeResponseSource{loader: loader.New(helpResponseErrorSource{responses: []byte(tc.responses), components: []byte(tc.components)})}
		if _, err := s.GetResponseDocumentation("demo", "2024-01-01", "RunThing"); err == nil {
			t.Fatal("invalid response metadata accepted")
		}
	}
}
