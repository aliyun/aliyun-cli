package engine

import (
	"io"
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

type commandResolutionReads struct {
	product, index, api int
}

type commandResolutionSource struct {
	source.Source
	reads commandResolutionReads
}

func (s *commandResolutionSource) LoadProduct(code string) (*meta.Product, *source.Provenance, error) {
	s.reads.product++
	return s.Source.LoadProduct(code)
}

func (s *commandResolutionSource) LoadAPIIndex(code, version string) (*meta.APIIndex, error) {
	s.reads.index++
	return s.Source.LoadAPIIndex(code, version)
}

func (s *commandResolutionSource) LoadAPI(code, version, name string) (*meta.API, error) {
	s.reads.api++
	return s.Source.LoadAPI(code, version, name)
}

type commandResolutionVersionSource struct {
	multiVersionSource
}

func (commandResolutionVersionSource) LoadAPI(code, version, name string) (*meta.API, error) {
	if code != "demo" || version != "v1" && version != "v2" {
		return nil, source.ErrNotFound
	}
	if name != "RunThing" && !(version == "v1" && name == "LegacyThing") {
		return nil, source.ErrNotFound
	}
	api, err := (dispatchErrorTestSource{}).LoadAPI("demo", "2024-01-01", "RunThing")
	if err != nil {
		return nil, err
	}
	api.Name, api.Version = name, version
	if name == "LegacyThing" {
		api.CmdName = "legacy-thing"
	}
	return api, nil
}

func newCommandResolutionEngine(src source.Source) (*Engine, *commandResolutionSource) {
	counted := &commandResolutionSource{Source: src}
	return NewEngine(func() (loader.Loader, error) {
		return loader.New(counted), nil
	}, &responseExecutor{response: &runtime.Response{Parsed: map[string]any{"ok": true}}}), counted
}

func TestCommandResolutionObserverPrecedesParameterValidation(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{"positional path", []string{"DEMO", "run-thing", "/private/customer.csv"}},
		{"extra flag value", []string{"DEMO", "run-thing", "--instance-type", "ecs.g6", "private-business-input"}},
		{"missing required parameter", []string{"DEMO", "run-thing"}},
		{"unknown parameter", []string{"DEMO", "run-thing", "--private-input", "private-value"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine, _ := newCommandResolutionEngine(dispatchErrorTestSource{})
			var observed [][2]string
			err := engine.Dispatch(Request{
				Args: test.args, Out: io.Discard,
				OnCommandResolved: func(product, command string) {
					observed = append(observed, [2]string{product, command})
				},
			})
			if err == nil {
				t.Fatal("malformed invocation unexpectedly succeeded")
			}
			if want := [][2]string{{"DEMO", "run-thing"}}; !reflect.DeepEqual(observed, want) {
				t.Fatalf("resolved command = %#v, want %#v", observed, want)
			}
		})
	}
}

func TestCommandResolutionObserverUsesSelectedAPIVersion(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want [][2]string
	}{
		{"explicit supported version", []string{"DEMO", "legacy-thing", "--api-version", "v1"}, [][2]string{{"DEMO", "legacy-thing"}}},
		{"command absent from default version", []string{"demo", "legacy-thing"}, nil},
		{"unsupported version", []string{"demo", "run-thing", "--api-version", "v3"}, nil},
		{"unknown command", []string{"demo", "private-command"}, nil},
		{"unknown product", []string{"private-product", "run-thing"}, nil},
		{"missing command", []string{"demo"}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine, _ := newCommandResolutionEngine(commandResolutionVersionSource{})
			var observed [][2]string
			_ = engine.Dispatch(Request{
				Args: test.args, Out: io.Discard,
				OnCommandResolved: func(product, command string) {
					observed = append(observed, [2]string{product, command})
				},
			})
			if !reflect.DeepEqual(observed, test.want) {
				t.Fatalf("resolved command = %#v, want %#v", observed, test.want)
			}
		})
	}
	// The index may resolve successfully while the API payload cannot be read.
	engine, _ := newCommandResolutionEngine(multiVersionSource{})
	called := false
	err := engine.Dispatch(Request{
		Args: []string{"demo", "run-thing"}, Out: io.Discard,
		OnCommandResolved: func(string, string) { called = true },
	})
	if err == nil || called {
		t.Fatalf("unreadable API: error = %v, observer called = %v", err, called)
	}
}

func TestCommandResolutionObserverHelpNotifiesOnce(t *testing.T) {
	for _, operation := range []string{"dispatch help", "api help", "parameter help", "build api help", "build request help", "build response help", "build parameter help", "product help"} {
		t.Run(operation, func(t *testing.T) {
			engine, _ := newCommandResolutionEngine(dispatchErrorTestSource{})
			if !engine.HasProduct("demo") {
				t.Fatal("fixture product was not available")
			}
			var observed [][2]string
			req := Request{
				Args: []string{"DEMO", "run-thing", "--help"}, Out: io.Discard,
				OnCommandResolved: func(product, command string) {
					observed = append(observed, [2]string{product, command})
				},
			}
			var err error
			switch operation {
			case "dispatch help":
				err = engine.Dispatch(req)
			case "api help":
				err = engine.APIHelp(req)
			case "parameter help":
				req.Args = []string{"DEMO", "run-thing", "--instance-type", "--help"}
				err = engine.APIHelp(req)
			case "build api help":
				_, err = engine.BuildAPIHelp(req, HelpOptions{})
			case "build request help":
				_, err = engine.BuildAPIHelp(req, HelpOptions{ExplicitSection: true})
			case "build response help":
				_, err = engine.BuildAPIHelp(req, HelpOptions{Section: "response"})
			case "build parameter help":
				_, err = engine.BuildParameterHelp(req, "instance-type", HelpOptions{})
			case "product help":
				req.Args = []string{"DEMO", "--help"}
				err = engine.ProductHelp(req)
			}
			if err != nil {
				t.Fatal(err)
			}
			var want [][2]string
			if operation != "product help" {
				want = [][2]string{{"DEMO", "run-thing"}}
			}
			if !reflect.DeepEqual(observed, want) {
				t.Fatalf("resolved command = %#v, want %#v", observed, want)
			}
			if req.OnCommandResolved == nil {
				t.Fatal("engine modified the caller's request observer")
			}
		})
	}
}

func TestCommandResolutionObserverDirectHelpRequiresReadableAPI(t *testing.T) {
	for _, operation := range []string{"action", "request", "response", "parameter"} {
		t.Run(operation, func(t *testing.T) {
			engine, _ := newCommandResolutionEngine(multiVersionSource{})
			if !engine.HasProduct("demo") {
				t.Fatal("fixture product was not available")
			}
			called := false
			req := Request{
				Args:              []string{"demo", "run-thing"},
				OnCommandResolved: func(string, string) { called = true },
			}
			var err error
			switch operation {
			case "action":
				_, err = engine.BuildAPIHelp(req, HelpOptions{})
			case "request":
				_, err = engine.BuildAPIHelp(req, HelpOptions{ExplicitSection: true})
			case "response":
				_, err = engine.BuildAPIHelp(req, HelpOptions{Section: "response"})
			case "parameter":
				_, err = engine.BuildParameterHelp(req, "instance-type", HelpOptions{})
			}
			if err == nil || called {
				t.Fatalf("unreadable API: error = %v, observer called = %v", err, called)
			}
		})
	}
}

func TestCommandResolutionObserverSuccessfulInvocation(t *testing.T) {
	engine, _ := newCommandResolutionEngine(dispatchErrorTestSource{})
	var observed [][2]string
	err := engine.Dispatch(Request{
		Args: []string{"DEMO", "run-thing", "--instance-type", "ecs.g6", "--cli-dry-run-json"}, Out: io.Discard,
		OnCommandResolved: func(product, command string) {
			observed = append(observed, [2]string{product, command})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := [][2]string{{"DEMO", "run-thing"}}; !reflect.DeepEqual(observed, want) {
		t.Fatalf("resolved command = %#v, want %#v", observed, want)
	}
}

func TestCommandResolutionObserverRecognizesBuiltinVersions(t *testing.T) {
	for _, entry := range []string{"dispatch", "api help"} {
		t.Run(entry, func(t *testing.T) {
			for _, src := range []source.Source{dispatchErrorTestSource{}, commandResolutionVersionSource{}} {
				engine, _ := newCommandResolutionEngine(src)
				calls := 0
				req := Request{
					Args: []string{"DEMO", listAPIVersionsCommand}, Out: io.Discard,
					OnCommandResolved: func(product, command string) {
						calls++
						if product != "DEMO" || command != listAPIVersionsCommand {
							t.Fatalf("unexpected resolved pair: %q %q", product, command)
						}
					},
				}
				var err error
				if entry == "dispatch" {
					err = engine.Dispatch(req)
				} else {
					err = engine.APIHelp(req)
				}
				_, multi := src.(commandResolutionVersionSource)
				if multi && (err != nil || calls != 1) || !multi && (err == nil || calls != 0) {
					t.Fatalf("multiversion = %v: error = %v, observer calls = %d", multi, err, calls)
				}
			}
		})
	}
}

func TestCommandResolutionObserverAddsNoMetadataReads(t *testing.T) {
	for _, entry := range []string{"dispatch", "api help", "build api help", "build parameter help"} {
		t.Run(entry, func(t *testing.T) {
			var baseline commandResolutionReads
			for _, observe := range []bool{false, true} {
				engine, counted := newCommandResolutionEngine(dispatchErrorTestSource{})
				req := Request{Args: []string{"demo", "run-thing", "--help"}, Out: io.Discard}
				calls := 0
				if observe {
					req.OnCommandResolved = func(string, string) { calls++ }
				}
				var err error
				switch entry {
				case "dispatch":
					err = engine.Dispatch(req)
				case "api help":
					err = engine.APIHelp(req)
				case "build api help", "build parameter help":
					if !engine.HasProduct("demo") {
						t.Fatal("fixture product was not available")
					}
					if entry == "build api help" {
						_, err = engine.BuildAPIHelp(req, HelpOptions{})
					} else {
						_, err = engine.BuildParameterHelp(req, "instance-type", HelpOptions{})
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				if !observe {
					baseline = counted.reads
					continue
				}
				if counted.reads != baseline || calls != 1 {
					t.Fatalf("observer calls = %d, metadata reads = %#v, baseline = %#v", calls, counted.reads, baseline)
				}
			}
		})
	}
}
