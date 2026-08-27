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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
)

type fakeExec struct {
	bodies  []string
	n       int
	queries []map[string]string
}

func (f *fakeExec) Execute(ec *ExecContext) (*Response, error) {
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
	resp, err := CallWithPager(exec, ec, &argparser.PagerConfig{
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
	resp, err := CallWithPager(exec, ec, &argparser.PagerConfig{
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
	resp, err := CallWithWaiter(exec, ec, &argparser.WaiterConfig{
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
	_, err := CallWithWaiter(&fakeExec{}, &ExecContext{}, &argparser.WaiterConfig{})
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

func TestPagerHelpersAndDefaults(t *testing.T) {
	p := NewPager(nil)
	if p.PageNumberExpr != "PageNumber" || p.PageSizeExpr != "PageSize" || p.TotalCountExpr != "TotalCount" || p.NextTokenExpr != "NextToken" {
		t.Fatalf("default pager = %#v", p)
	}
	custom := NewPager(&argparser.PagerConfig{Path: "Items[]", PageNumber: "PN", PageSize: "PS", TotalCount: "TC", NextToken: "NT"})
	if custom.CollectionPath != "Items[]" || custom.PageNumberExpr != "PN" || custom.NextTokenExpr != "NT" {
		t.Fatalf("custom pager = %#v", custom)
	}

	values := []struct {
		value any
		want  int
		ok    bool
	}{
		{json.Number("12"), 12, true}, {json.Number("1.2"), 0, false}, {float64(3.8), 3, true},
		{int(4), 4, true}, {int64(5), 5, true}, {"6", 6, true}, {"bad", 0, false}, {nil, 0, false},
	}
	for _, tc := range values {
		got, ok := asIntValue(tc.value)
		if got != tc.want || ok != tc.ok {
			t.Errorf("asIntValue(%#v) = %d, %v", tc.value, got, ok)
		}
	}
	if jmespathSearch("[", map[string]any{}) != nil || jmespathSearch("Value", map[string]any{"Value": "ok"}) != "ok" {
		t.Fatal("jmespathSearch did not handle success/error")
	}
	if detectArrayPath([]any{}) != "" || detectArrayPath(map[string]any{"plain": "value"}) != "" {
		t.Fatal("detectArrayPath accepted a non-object/non-nested collection")
	}
	if got := detectArrayPath(map[string]any{"outer": map[string]any{"items": []any{1}}}); got != "outer.items[]" {
		t.Fatalf("detectArrayPath = %q", got)
	}
}

func TestPagerFeedResponseErrors(t *testing.T) {
	tests := []struct {
		name string
		p    *Pager
		body string
		want string
	}{
		{name: "empty", p: NewPager(nil), body: "", want: "empty response body"},
		{name: "invalid json", p: NewPager(nil), body: "{", want: "unmarshal failed"},
		{name: "cannot detect", p: NewPager(nil), body: `{"Value":1}`, want: "can't auto recognize"},
		{name: "bad expression", p: NewPager(&argparser.PagerConfig{Path: "["}), body: `{"Items":[]}`, want: "jmespath search failed"},
		{name: "nil result", p: NewPager(&argparser.PagerConfig{Path: "Missing"}), body: `{"Items":[]}`, want: "jmespath result empty"},
		{name: "not array", p: NewPager(&argparser.PagerConfig{Path: "Value"}), body: `{"Value":1}`, want: "is not an array"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.p.FeedResponse([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("FeedResponse error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestPagerMovementAndCollectionShapes(t *testing.T) {
	p := NewPager(&argparser.PagerConfig{NextToken: "Data.Token"})
	p.nextTokenMode = true
	p.nextToken = "next"
	if !p.HasMore() {
		t.Fatal("token pager should have more")
	}
	ec := &ExecContext{}
	p.MoveNextPage(ec)
	if ec.ExtraQuery["Token"] != "next" {
		t.Fatalf("token query = %#v", ec.ExtraQuery)
	}
	p.nextToken = ""
	if p.HasMore() {
		t.Fatal("empty token should stop")
	}

	p = NewPager(&argparser.PagerConfig{PageNumber: "Page.Number"})
	p.PageSize, p.totalCount, p.currentPageNumber = 10, 21, 2
	if !p.HasMore() {
		t.Fatal("number pager should have a third page")
	}
	ec = &ExecContext{ExtraQuery: map[string]string{}}
	p.MoveNextPage(ec)
	if ec.ExtraQuery["Number"] != "3" {
		t.Fatalf("page query = %#v", ec.ExtraQuery)
	}
	p.currentPageNumber = 3
	if p.HasMore() {
		t.Fatal("last numbered page should stop")
	}

	p = &Pager{results: []any{1, 2}}
	if got := p.GetResponseCollection(); got != `[1,2]` {
		t.Fatalf("flat collection = %s", got)
	}
	p.CollectionPath = "Items[]"
	if got := p.GetResponseCollection(); got != `{"Items":[1,2]}` {
		t.Fatalf("single-level collection = %s", got)
	}
	p.CollectionPath = "Data.Items[]"
	if got := p.GetResponseCollection(); got != `{"Data":{"Items":[1,2]}}` {
		t.Fatalf("nested collection = %s", got)
	}
	p.CollectionPath = ""
	p.results = []any{func() {}}
	if got := p.GetResponseCollection(); got != "[]" {
		t.Fatalf("unmarshalable flat collection = %s", got)
	}
	p.CollectionPath = "Items[]"
	if got := p.GetResponseCollection(); got != "{}" {
		t.Fatalf("unmarshalable object collection = %s", got)
	}
}

type errorPagerExecutor struct{}

func (errorPagerExecutor) Execute(*ExecContext) (*Response, error) {
	return nil, errors.New("execute failed")
}

func TestCallWithPagerPropagatesFailures(t *testing.T) {
	if _, err := CallWithPager(errorPagerExecutor{}, &ExecContext{}, nil); err == nil || !strings.Contains(err.Error(), "execute failed") {
		t.Fatalf("execute error = %v", err)
	}
	exec := &fakeExec{bodies: []string{"{"}}
	if _, err := CallWithPager(exec, &ExecContext{}, nil); err == nil || !strings.Contains(err.Error(), "pager: unmarshal failed") {
		t.Fatalf("feed error = %v", err)
	}
}
