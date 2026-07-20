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
	"strconv"
	"time"

	jmespath "github.com/jmespath/go-jmespath"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
)

// Waiter polls an API until a JMESPath expression equals the expected
// value. Defaults (timeout 180s, interval 5s) match the Go plugin.
type Waiter struct {
	Expr     string
	To       string
	Timeout  time.Duration
	Interval time.Duration
}

// NewWaiter builds a Waiter from the argparser config.
func NewWaiter(cfg *argparser.WaiterConfig) *Waiter {
	if cfg == nil {
		cfg = &argparser.WaiterConfig{}
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 180
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = 5
	}
	return &Waiter{
		Expr:     cfg.Expr,
		To:       cfg.To,
		Timeout:  time.Duration(timeout) * time.Second,
		Interval: time.Duration(interval) * time.Second,
	}
}

// CallWithWaiter repeatedly Execute-s until the expression matches or
// the timeout elapses. Returns the last matching response body.
func CallWithWaiter(ctx context.Context, exec Executor, ec *ExecContext, cfg *argparser.WaiterConfig) (*Response, error) {
	w := NewWaiter(cfg)
	if w.Expr == "" || w.To == "" {
		return nil, fmt.Errorf("--waiter requires expr=... and to=...")
	}
	begin := time.Now()
	for {
		resp, err := exec.Execute(ctx, ec)
		if err != nil {
			return nil, err
		}
		v, err := w.evaluateExpr(resp.Raw)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expression: %w", err)
		}
		if v == w.To {
			return resp, nil
		}
		if time.Since(begin) > w.Timeout {
			return nil, fmt.Errorf("wait '%s' to '%s' timeout (%d seconds), last='%s'",
				w.Expr, w.To, int(w.Timeout.Seconds()), v)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(w.Interval):
		}
	}
}

func (w *Waiter) evaluateExpr(body []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	result, err := jmespath.Search(w.Expr, v)
	if err != nil {
		return "", fmt.Errorf("jmespath search failed: %w", err)
	}
	switch val := result.(type) {
	case string:
		return val, nil
	case json.Number:
		return val.String(), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(val), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case bool:
		return strconv.FormatBool(val), nil
	case nil:
		return "", nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}
