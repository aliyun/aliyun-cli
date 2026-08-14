package runtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alibabacloud-go/tea/dara"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestMarshalSSEEvent(t *testing.T) {
	id, name, data, retry := "evt-1", "chunk", `{"message":"hello"}`, 1000
	got, ok := MarshalSSEEvent(&dara.SSEEvent{
		Id: &id, Event: &name, Data: &data, Retry: &retry,
	})
	if !ok {
		t.Fatal("event was skipped")
	}
	var object map[string]any
	if err := json.Unmarshal(got, &object); err != nil {
		t.Fatal(err)
	}
	if object["id"] != id || object["event"] != name || object["retry"] != float64(retry) {
		t.Fatalf("event metadata = %#v", object)
	}
	payload, ok := object["data"].(map[string]any)
	if !ok || payload["message"] != "hello" {
		t.Fatalf("event data = %#v", object["data"])
	}

	text := "plain text"
	got, ok = MarshalSSEEvent(&dara.SSEEvent{Data: &text})
	if !ok || string(got) != `{"data":"plain text"}` {
		t.Fatalf("text event = %s, ok=%v", got, ok)
	}
	if _, ok := MarshalSSEEvent(&dara.SSEEvent{}); ok {
		t.Fatal("empty event must be skipped")
	}
}

func TestExecuteSSEAgainstMockServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "id: evt-1\nevent: chunk\nretry: 1000\ndata: {\"message\":\"hello\"}\n\n")
		fmt.Fprint(w, "data: plain text\n\n")
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	ec := &ExecContext{
		API: &meta.API{
			Name: "StreamThing", Version: "2024-01-01", Method: "POST",
			Style: meta.StyleRPC, Protocol: "HTTP", ProductCode: "demo", IsSSE: true,
		},
		Endpoint:   host,
		Credential: staticAKCredential(t),
		Args:       map[string]any{},
	}
	var events []SSEEvent
	err := NewExecutor().ExecuteSSE(ec, func(event SSEEvent) {
		events = append(events, append(SSEEvent(nil), event...))
	})
	if err != nil {
		t.Fatalf("ExecuteSSE: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d: %q", len(events), events)
	}
	if !strings.Contains(string(events[0]), `"message":"hello"`) {
		t.Fatalf("first event = %s", events[0])
	}
	if string(events[1]) != `{"data":"plain text"}` {
		t.Fatalf("second event = %s", events[1])
	}
}

func TestExecuteSSEReturnsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"Code":"InvalidRequest","Message":"bad stream","RequestId":"req-1"}`)
	}))
	defer srv.Close()

	err := NewExecutor().ExecuteSSE(&ExecContext{
		API: &meta.API{
			Name: "StreamThing", Version: "2024-01-01", Method: "POST",
			Style: meta.StyleRPC, Protocol: "HTTP", ProductCode: "demo", IsSSE: true,
		},
		Endpoint:   strings.TrimPrefix(srv.URL, "http://"),
		Credential: staticAKCredential(t),
		Args:       map[string]any{},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "InvalidRequest") {
		t.Fatalf("unexpected SSE error: %v", err)
	}
}

func TestSSEUsesProfileSDKRetryWhenThrottlingRetryDisabled(t *testing.T) {
	disabled := false
	ec := &ExecContext{
		API: &meta.API{
			Name: "StreamThing", Version: "2024-01-01", Method: "POST",
			Style: meta.StyleRPC, Protocol: "HTTP", ProductCode: "demo", IsSSE: true,
		},
		Endpoint:   "example.com",
		Credential: staticAKCredential(t),
		RetryCount: 5,
		Transport: TransportOptions{ThrottlingRetry: ThrottlingRetryOptions{
			Enabled: &disabled,
		}},
	}
	req, err := Assemble(ec)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	call, err := prepareCall(ec, req)
	if err != nil {
		t.Fatalf("prepareCall: %v", err)
	}
	if call.client.RetryOptions == nil {
		t.Fatal("SSE should configure SDK retry options from profile RetryCount")
	}
}
