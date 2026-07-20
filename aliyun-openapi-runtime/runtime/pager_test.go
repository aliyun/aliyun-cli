// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
)

type fakeExec struct {
	bodies []string
	n      int
	queries []map[string]string
}

func (f *fakeExec) Execute(_ context.Context, ec *ExecContext) (*Response, error) {
	q := map[string]string{}
	for k, v := range ec.ExtraQuery {
		q[k] = v
	}
	f.queries = append(f.queries, q)
	if f.n >= len(f.bodies) {
		return nil, fmt.Errorf("no more pages")
	}
	body := f.bodies[f.n]
	f.n++
	var parsed any
	_ = json.Unmarshal([]byte(body), &parsed)
	return &Response{StatusCode: 200, Raw: []byte(body), Parsed: parsed}, nil
}

func TestPagerNextToken(t *testing.T) {
	exec := &fakeExec{bodies: []string{
		`{"Data":{"Items":[{"id":1}],"NextToken":"tok2","MaxResults":1}}`,
		`{"Data":{"Items":[{"id":2}],"NextToken":"","MaxResults":1}}`,
	}}
	ec := &ExecContext{}
	resp, err := CallWithPager(context.Background(), exec, ec, &argparser.PagerConfig{
		Path:      "Data.Items[]",
		NextToken: "Data.NextToken",
		PageSize:  "Data.MaxResults",
	})
	if err != nil {
		t.Fatal(err)
	}
	if exec.n != 2 {
		t.Fatalf("calls=%d", exec.n)
	}
	if len(exec.queries) < 2 || exec.queries[1]["NextToken"] != "tok2" {
		t.Fatalf("second call ExtraQuery = %#v", exec.queries)
	}
	var root map[string]any
	if err := json.Unmarshal(resp.Raw, &root); err != nil {
		t.Fatal(err)
	}
	data := root["Data"].(map[string]any)
	items := data["Items"].([]any)
	if len(items) != 2 {
		t.Fatalf("merged items=%d raw=%s", len(items), resp.Raw)
	}
}

func TestPagerPageNumber(t *testing.T) {
	exec := &fakeExec{bodies: []string{
		`{"Wrap":{"List":[{"id":1}],"PageNumber":1,"PageSize":1,"TotalCount":2}}`,
		`{"Wrap":{"List":[{"id":2}],"PageNumber":2,"PageSize":1,"TotalCount":2}}`,
	}}
	ec := &ExecContext{}
	resp, err := CallWithPager(context.Background(), exec, ec, &argparser.PagerConfig{
		Path:       "Wrap.List[]",
		PageNumber: "Wrap.PageNumber",
		PageSize:   "Wrap.PageSize",
		TotalCount: "Wrap.TotalCount",
	})
	if err != nil {
		t.Fatal(err)
	}
	if exec.n != 2 {
		t.Fatalf("calls=%d", exec.n)
	}
	if exec.queries[1]["PageNumber"] != "2" {
		t.Fatalf("second ExtraQuery = %#v", exec.queries[1])
	}
	if !strings.Contains(string(resp.Raw), `"id":1`) || !strings.Contains(string(resp.Raw), `"id":2`) {
		t.Fatalf("merged raw=%s", resp.Raw)
	}
}

func TestPagerAutoDetectPath(t *testing.T) {
	p := NewPager(&argparser.PagerConfig{})
	body := `{"DescribeClustersResponse":{"Clusters":[{"id":"a"}],"PageNumber":1,"PageSize":10,"TotalCount":1}}`
	if err := p.FeedResponse([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if p.CollectionPath == "" {
		t.Fatal("auto-detect should set CollectionPath")
	}
	if len(p.results) != 1 {
		t.Fatalf("results=%d", len(p.results))
	}
	if p.HasMore() {
		t.Fatal("single page should not HasMore")
	}
}

func TestWaiterMatches(t *testing.T) {
	exec := &fakeExec{bodies: []string{`{"Status":"Running"}`}}
	ec := &ExecContext{}
	resp, err := CallWithWaiter(context.Background(), exec, ec, &argparser.WaiterConfig{
		Expr: "Status", To: "Running", Timeout: 10, Interval: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(resp.Raw), "Running") {
		t.Fatalf("raw=%s", resp.Raw)
	}
	if exec.n != 1 {
		t.Fatalf("calls=%d", exec.n)
	}
}

func TestWaiterRequiresExprTo(t *testing.T) {
	_, err := CallWithWaiter(context.Background(), &fakeExec{}, &ExecContext{}, &argparser.WaiterConfig{})
	if err == nil || !strings.Contains(err.Error(), "requires") {
		t.Fatalf("err=%v", err)
	}
}

func TestAssembleExtraQuery(t *testing.T) {
	ec := &ExecContext{
		API:        rpcAPI(),
		Args:       map[string]any{},
		ExtraQuery: map[string]string{"NextToken": "abc"},
		DryRun:     true,
	}
	req, err := Assemble(ec)
	if err != nil {
		t.Fatal(err)
	}
	if req.Query["NextToken"] != "abc" {
		t.Fatalf("query=%v", req.Query)
	}
}
