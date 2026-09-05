package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

type sseTestSource struct{}

func (sseTestSource) Kind() source.Kind { return source.KindBaseline }

func (sseTestSource) LoadProduct(code string) (*meta.Product, *source.Provenance, error) {
	if code != "demo" {
		return nil, nil, source.ErrNotFound
	}
	return &meta.Product{
		Code: "demo", Versions: []string{"2024-01-01"}, DefaultVersion: "2024-01-01",
	}, &source.Provenance{Kind: source.KindBaseline}, nil
}

func (sseTestSource) LoadAPIIndex(code, version string) (*meta.APIIndex, error) {
	if code != "demo" || version != "2024-01-01" {
		return nil, source.ErrNotFound
	}
	index := &meta.APIIndex{
		ProductCode: code,
		Version:     version,
		Entries: map[string]meta.APIIndexEntry{
			"StreamThing": {APIName: "StreamThing", CmdName: "stream-thing"},
		},
	}
	index.BuildCmdIndex()
	return index, nil
}

func (sseTestSource) LoadAPI(code, version, name string) (*meta.API, error) {
	if code != "demo" || version != "2024-01-01" || name != "StreamThing" {
		return nil, source.ErrNotFound
	}
	return &meta.API{
		Name: "StreamThing", CmdName: "stream-thing", ProductCode: "demo",
		Version: "2024-01-01", Method: "POST", Protocol: "HTTPS",
		Style: meta.StyleRPC, IsSSE: true,
		Endpoints: meta.Endpoints{Global: "demo.example.com"},
	}, nil
}

type fakeSSEExecutor struct {
	events      []runtime.SSEEvent
	normalCalls int
	sseCalls    int
}

func (e *fakeSSEExecutor) Execute(ec *runtime.ExecContext) (*runtime.Response, error) {
	e.normalCalls++
	return runtime.NewExecutor().Execute(ec)
}

func (e *fakeSSEExecutor) ExecuteSSE(_ *runtime.ExecContext, yield func(runtime.SSEEvent)) error {
	e.sseCalls++
	for _, event := range e.events {
		if yield != nil {
			yield(event)
		}
	}
	return nil
}

type nonSSEExecutor struct{}

func (nonSSEExecutor) Execute(*runtime.ExecContext) (*runtime.Response, error) {
	return nil, errors.New("unexpected ordinary execution")
}

func newSSETestEngine(executor runtime.Executor) *Engine {
	return NewEngine(func() (loader.Loader, error) {
		return loader.New(sseTestSource{}), nil
	}, executor)
}

func dispatchSSETest(t *testing.T, engine *Engine, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := engine.Dispatch(Request{
		Args: args,
		Out:  &out,
		Host: runtime.StaticHost{},
	})
	return out.String(), err
}

func TestDispatchSSEStreamingAndQuiet(t *testing.T) {
	events := []runtime.SSEEvent{
		[]byte(`{"data":{"message":"one"}}`),
		[]byte(`{"data":"two"}`),
	}

	executor := &fakeSSEExecutor{events: events}
	out, err := dispatchSSETest(t, newSSETestEngine(executor), "demo", "stream-thing")
	if err != nil {
		t.Fatal(err)
	}
	if out != "{\"data\":{\"message\":\"one\"}}\n{\"data\":\"two\"}\n" {
		t.Fatalf("stream output = %q", out)
	}

	executor = &fakeSSEExecutor{events: events}
	out, err = dispatchSSETest(t, newSSETestEngine(executor), "demo", "stream-thing", "--quiet")
	if err != nil {
		t.Fatal(err)
	}
	if out != "" || executor.sseCalls != 1 {
		t.Fatalf("quiet output=%q calls=%d", out, executor.sseCalls)
	}
}

func TestDispatchSSEAggregateAndQuery(t *testing.T) {
	events := []runtime.SSEEvent{
		[]byte(`{"data":{"message":"one"}}`),
		[]byte(`{"data":{"message":"two"}}`),
	}

	out, err := dispatchSSETest(t, newSSETestEngine(&fakeSSEExecutor{events: events}),
		"demo", "stream-thing", "--no-stream")
	if err != nil {
		t.Fatal(err)
	}
	var aggregate []any
	if err := json.Unmarshal([]byte(out), &aggregate); err != nil || len(aggregate) != 2 {
		t.Fatalf("aggregate output=%q err=%v", out, err)
	}

	out, err = dispatchSSETest(t, newSSETestEngine(&fakeSSEExecutor{events: events}),
		"demo", "stream-thing", "--cli-query", "[0].data.message")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != `"one"` {
		t.Fatalf("query output = %q", out)
	}
}

func TestDispatchSSEOutputTable(t *testing.T) {
	events := []runtime.SSEEvent{
		[]byte(`{"data":{"items":[{"name":"one"},{"name":"two"}]}}`),
	}
	out, err := dispatchSSETest(t, newSSETestEngine(&fakeSSEExecutor{events: events}),
		"demo", "stream-thing", "--output", "cols=name", "rows=data.items")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "one") || !strings.Contains(out, "two") {
		t.Fatalf("table output = %q", out)
	}
}

func TestDispatchSSEDryRunUsesOrdinaryExecutor(t *testing.T) {
	executor := &fakeSSEExecutor{}
	out, err := dispatchSSETest(t, newSSETestEngine(executor),
		"demo", "stream-thing", "--cli-dry-run-json")
	if err != nil {
		t.Fatal(err)
	}
	if executor.normalCalls != 1 || executor.sseCalls != 0 {
		t.Fatalf("normal=%d sse=%d", executor.normalCalls, executor.sseCalls)
	}
	if !strings.Contains(out, `"action":"StreamThing"`) {
		t.Fatalf("dry-run output = %q", out)
	}
}

func TestDispatchSSERejectsUnsupportedExecutorAndHelpers(t *testing.T) {
	_, err := dispatchSSETest(t, newSSETestEngine(nonSSEExecutor{}), "demo", "stream-thing")
	if err == nil || !strings.Contains(err.Error(), "does not support SSE") {
		t.Fatalf("unsupported executor error = %v", err)
	}

	for _, flag := range []string{"--pager", "--waiter"} {
		_, err := dispatchSSETest(t, newSSETestEngine(&fakeSSEExecutor{}),
			"demo", "stream-thing", flag)
		if err == nil || !strings.Contains(err.Error(), "not supported for SSE") {
			t.Fatalf("%s error = %v", flag, err)
		}
	}
}
