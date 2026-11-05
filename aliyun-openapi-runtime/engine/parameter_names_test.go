package engine

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
)

type parameterNamesSource struct{ dispatchErrorTestSource }

func (parameterNamesSource) LoadAPI(product, version, name string) (*meta.API, error) {
	api, err := (dispatchErrorTestSource{}).LoadAPI(product, version, name)
	if err != nil {
		return nil, err
	}
	api.Parameters[0].Options = []string{"--InstanceType"}
	api.Parameters = append(api.Parameters,
		meta.Parameter{Name: "names", RawName: "Names", Type: meta.TypeArray, Options: []string{"--names"}, ItemType: &meta.Parameter{Type: meta.TypeString}},
		meta.Parameter{Name: "count", RawName: "Count", Type: meta.TypeInteger, Options: []string{"--Count"}},
	)
	return api, nil
}

func newParameterNamesEngine() (*Engine, *responseExecutor) {
	executor := &responseExecutor{response: &runtime.Response{Parsed: map[string]any{"ok": true}}}
	return NewEngineWithOptions(func() (loader.Loader, error) {
		return loader.New(parameterNamesSource{}), nil
	}, executor, Options{ExternalFlags: []argparser.ExternalFlagSpec{{Name: "profile", Shorthand: 'p', Mode: argparser.ExternalFlagRequired}}}), executor
}

func TestEngineParameterNamesObserveActualParsing(t *testing.T) {
	for _, test := range []struct {
		name, operation string
		tail            []string
		want            []string
		wantError       bool
	}{
		{"scalar dash-leading value", "dispatch", []string{"--InstanceType", "--private-scalar", "--cli-dry-run-json"}, []string{"--InstanceType", "--cli-dry-run-json"}, false},
		{"composite values and repetitions", "dispatch", []string{"--names", "a", "--private-first", "--private-second", "--names", "b", "--InstanceType=ecs.g6", "--cli-dry-run-json"}, []string{"--names", "--names", "--InstanceType", "--cli-dry-run-json"}, false},
		{"host aliases and values", "dispatch", []string{"--profile", "--private-profile", "-p:private-profile", "--InstanceType", "ecs.g6", "--cli-dry-run-json"}, []string{"--profile", "-p", "--InstanceType", "--cli-dry-run-json"}, false},
		{"missing scalar value", "dispatch", []string{"--InstanceType"}, []string{"--InstanceType"}, true},
		{"type error", "dispatch", []string{"--InstanceType", "ecs.g6", "--Count", "--private-count"}, []string{"--InstanceType", "--Count"}, true},
		{"unknown flag after scalar", "dispatch", []string{"--InstanceType", "ecs.g6", "--private-unknown", "value"}, []string{"--InstanceType"}, true},
		{"unknown first flag", "dispatch", []string{"--private-unknown", "--InstanceType", "ecs.g6"}, nil, true},
		{"complete array does not consume unknown flag", "dispatch", []string{"--names", "[]", "--private-unknown"}, []string{"--names"}, true},
		{"dispatch help", "dispatch", []string{"--quiet", "--quiet", "--help", "--help"}, []string{"--quiet", "--quiet", "--help", "--help"}, false},
		{"api help", "api help", []string{"--profile=private", "--help", "--quiet"}, []string{"--profile", "--help", "--quiet"}, false},
		{"product help", "product help", []string{"--profile=private", "--help", "--quiet"}, []string{"--profile", "--help", "--quiet"}, false},
		{"product help failure", "product help", []string{"--profile", "private", "--private-unknown"}, []string{"--profile"}, true},
		{"parameter help target position", "dispatch", []string{"--quiet", "--quiet", "--InstanceType", "--help", "--cli-output", "json"}, []string{"--quiet", "--quiet", "--InstanceType", "--help", "--cli-output"}, false},
		{"parameter help repeated operations", "api help", []string{"--help", "--InstanceType", "--help", "--quiet"}, []string{"--help", "--InstanceType", "--help", "--quiet"}, false},
		{"parameter help composite values", "dispatch", []string{"--names", "--private-array", "--InstanceType", "-h"}, []string{"--names", "--InstanceType", "-h"}, false},
		{"parameter help parse failure", "dispatch", []string{"--Count", "--private-count", "--InstanceType", "--help"}, []string{"--Count", "--InstanceType"}, true},
		{"parameter help failure after target", "api help", []string{"--InstanceType", "--help", "--Count", "--private-count"}, []string{"--InstanceType", "--help", "--Count"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var baselineOutput, observedOutput bytes.Buffer
			baseline, baselineExecutor := newParameterNamesEngine()
			observed, observedExecutor := newParameterNamesEngine()
			args := append([]string{"demo", "run-thing"}, test.tail...)
			if test.operation == "product help" {
				args = append([]string{"demo"}, test.tail...)
			}
			baselineRequest := Request{Args: args, Out: &baselineOutput}
			observedRequest := Request{Args: args, Out: &observedOutput}
			calls := 0
			var names []string
			observedRequest.OnParameterNames = func(snapshot []string) {
				calls++
				names = append([]string(nil), snapshot...)
			}
			invoke := func(engine *Engine, req Request) error {
				switch test.operation {
				case "api help":
					return engine.APIHelp(req)
				case "product help":
					return engine.ProductHelp(req)
				default:
					return engine.Dispatch(req)
				}
			}
			baselineErr := invoke(baseline, baselineRequest)
			observedErr := invoke(observed, observedRequest)
			if (observedErr != nil) != test.wantError {
				t.Fatalf("error = %v, want error %v", observedErr, test.wantError)
			}
			if calls != 1 || !reflect.DeepEqual(names, test.want) {
				t.Fatalf("observer calls = %d, names = %#v; want 1, %#v", calls, names, test.want)
			}
			if reflect.TypeOf(baselineErr) != reflect.TypeOf(observedErr) || errorText(baselineErr) != errorText(observedErr) {
				t.Fatalf("observer changed error: baseline %T %v, observed %T %v", baselineErr, baselineErr, observedErr, observedErr)
			}
			if baselineOutput.String() != observedOutput.String() || baselineExecutor.calls != observedExecutor.calls {
				t.Fatal("observer changed output or execution count")
			}
		})
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestEngineParameterNamesBuiltinAndUnparsedHelp(t *testing.T) {
	engine := newMultiVersionEngine()
	calls := 0
	var names []string
	req := Request{
		Args: []string{"demo", listAPIVersionsCommand, "--quiet", "--help"}, Out: io.Discard,
		OnParameterNames: func(snapshot []string) { calls++; names = append([]string(nil), snapshot...) },
	}
	if err := engine.Dispatch(req); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !reflect.DeepEqual(names, []string{"--quiet", "--help"}) {
		t.Fatalf("builtin snapshot: calls = %d, names = %#v", calls, names)
	}
	// APIHelp for this built-in does not parse its tail; do not guess names.
	if err := engine.APIHelp(req); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("unparsed builtin Help synthesized parameter names")
	}
	apiEngine, _ := newParameterNamesEngine()
	req.Args = []string{"demo", "run-thing", "--private-unknown"}
	if !apiEngine.HasProduct("demo") {
		t.Fatal("fixture product missing")
	}
	if _, err := apiEngine.BuildAPIHelp(req, HelpOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := apiEngine.BuildParameterHelp(req, "InstanceType", HelpOptions{}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("document builders synthesized parameter names without parsing")
	}
}
