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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	jmespath "github.com/jmespath/go-jmespath"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
)

// Pager aggregates multi-page API responses. Semantics mirror
// aliyun-cli-runtime/http.Pager (path / PageNumber / PageSize /
// TotalCount / NextToken).
type Pager struct {
	PageNumberExpr string
	PageSizeExpr   string
	TotalCountExpr string
	NextTokenExpr  string
	CollectionPath string

	PageSize          int
	totalCount        int
	currentPageNumber int
	nextTokenMode     bool
	nextToken         string
	results           []any
}

// NewPager builds a Pager from the argparser config, applying the same
// field defaults as the Go plugin executor.
func NewPager(cfg *argparser.PagerConfig) *Pager {
	p := &Pager{results: make([]any, 0)}
	if cfg == nil {
		cfg = &argparser.PagerConfig{}
	}
	p.CollectionPath = cfg.Path
	p.PageNumberExpr = cfg.PageNumber
	p.PageSizeExpr = cfg.PageSize
	p.TotalCountExpr = cfg.TotalCount
	p.NextTokenExpr = cfg.NextToken

	if p.PageNumberExpr == "" {
		p.PageNumberExpr = "PageNumber"
	}
	if p.PageSizeExpr == "" {
		p.PageSizeExpr = "PageSize"
	}
	if p.TotalCountExpr == "" {
		p.TotalCountExpr = "TotalCount"
	}
	if p.NextTokenExpr == "" {
		p.NextTokenExpr = "NextToken"
	}
	return p
}

// CallWithPager repeatedly Execute-s until no more pages, then returns
// a Response whose Raw/Parsed hold the merged collection document.
func CallWithPager(ctx context.Context, exec Executor, ec *ExecContext, cfg *argparser.PagerConfig) (*Response, error) {
	p := NewPager(cfg)
	var last *Response
	for {
		resp, err := exec.Execute(ctx, ec)
		if err != nil {
			return nil, err
		}
		last = resp
		if err := p.FeedResponse(resp.Raw); err != nil {
			return nil, fmt.Errorf("pager: %w", err)
		}
		if !p.HasMore() {
			break
		}
		p.MoveNextPage(ec)
	}
	raw := []byte(p.GetResponseCollection())
	out := &Response{StatusCode: 200, Raw: raw}
	if last != nil {
		out.StatusCode = last.StatusCode
		out.Headers = last.Headers
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var parsed any
	if dec.Decode(&parsed) == nil {
		out.Parsed = parsed
	}
	return out, nil
}

// FeedResponse consumes one page body and appends its collection items.
func (p *Pager) FeedResponse(body []byte) error {
	if len(body) == 0 {
		return fmt.Errorf("empty response body")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var j any
	if err := dec.Decode(&j); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	if p.NextTokenExpr != "" {
		token, err := jmespath.Search(p.NextTokenExpr, j)
		if err == nil {
			if tokenStr, ok := token.(string); ok && tokenStr != "" {
				p.nextToken = tokenStr
				p.nextTokenMode = true
			} else {
				p.nextToken = ""
			}
		}
	}

	if p.TotalCountExpr != "" {
		if n, ok := asIntValue(jmespathSearch(p.TotalCountExpr, j)); ok {
			p.totalCount = n
		}
	}
	if p.PageSizeExpr != "" {
		if n, ok := asIntValue(jmespathSearch(p.PageSizeExpr, j)); ok {
			p.PageSize = n
		}
	}
	if p.PageNumberExpr != "" {
		if n, ok := asIntValue(jmespathSearch(p.PageNumberExpr, j)); ok {
			p.currentPageNumber = n
		}
	}

	if p.CollectionPath == "" {
		p2 := detectArrayPath(j)
		if p2 == "" {
			return fmt.Errorf("can't auto recognize collections path: you need add `--pager path=[jmespath]` to assign manually")
		}
		p.CollectionPath = p2
	}

	items, err := jmespath.Search(p.CollectionPath, j)
	if err != nil {
		return fmt.Errorf("jmespath search failed: %w", err)
	}
	if items == nil {
		return fmt.Errorf("jmespath result empty: %s", p.CollectionPath)
	}
	arr, ok := items.([]any)
	if !ok {
		return fmt.Errorf("jmespath result at %q is not an array, got %T", p.CollectionPath, items)
	}
	p.results = append(p.results, arr...)
	return nil
}

func jmespathSearch(expr string, j any) any {
	v, err := jmespath.Search(expr, j)
	if err != nil {
		return nil
	}
	return v
}

func asIntValue(v any) (int, bool) {
	switch n := v.(type) {
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case string:
		i, err := strconv.Atoi(n)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

// detectArrayPath walks one level of top-level object keys; if the
// value is an object, looks for the first nested key whose value is a
// JSON array and returns "outer.inner[]".
func detectArrayPath(d any) string {
	m, ok := d.(map[string]any)
	if !ok {
		return ""
	}
	for k, v := range m {
		m2, ok := v.(map[string]any)
		if !ok {
			continue
		}
		for k2, v2 := range m2 {
			if _, ok := v2.([]any); ok {
				return fmt.Sprintf("%s.%s[]", k, k2)
			}
		}
	}
	return ""
}

// HasMore reports whether another page should be fetched.
func (p *Pager) HasMore() bool {
	if p.nextTokenMode {
		return p.nextToken != ""
	}
	if p.PageSize > 0 && p.totalCount > 0 {
		pages := int(math.Ceil(float64(p.totalCount) / float64(p.PageSize)))
		return p.currentPageNumber < pages
	}
	return false
}

// MoveNextPage injects the next-page query param into ec.ExtraQuery.
func (p *Pager) MoveNextPage(ec *ExecContext) {
	if ec.ExtraQuery == nil {
		ec.ExtraQuery = map[string]string{}
	}
	if p.nextTokenMode {
		if p.NextTokenExpr != "" {
			parts := strings.Split(p.NextTokenExpr, ".")
			ec.ExtraQuery[parts[len(parts)-1]] = p.nextToken
		}
		return
	}
	p.currentPageNumber++
	if p.PageNumberExpr != "" {
		parts := strings.Split(p.PageNumberExpr, ".")
		ec.ExtraQuery[parts[len(parts)-1]] = strconv.Itoa(p.currentPageNumber)
	}
}

// GetResponseCollection rebuilds a JSON document with the merged
// collection under the detected/configured path.
func (p *Pager) GetResponseCollection() string {
	if p.CollectionPath == "" {
		result, err := json.Marshal(p.results)
		if err != nil {
			return "[]"
		}
		return string(result)
	}

	root := make(map[string]any)
	current := make(map[string]any)
	path := p.CollectionPath

	parts := strings.Split(path, ".")
	if len(parts) > 1 {
		root[parts[0]] = current
		key := strings.TrimSuffix(parts[len(parts)-1], "[]")
		current[key] = p.results
	} else {
		key := strings.TrimSuffix(path, "[]")
		root[key] = p.results
	}

	result, err := json.Marshal(root)
	if err != nil {
		return "{}"
	}
	return string(result)
}
