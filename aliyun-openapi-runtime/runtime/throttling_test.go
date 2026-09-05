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
	"errors"
	"testing"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/tea"
)

func TestThrottlingRetryOptionsDefaultsAndOverrides(t *testing.T) {
	defaults := ThrottlingRetryOptions{}
	if !defaults.IsEnabled() {
		t.Fatal("zero-value throttling retry should be enabled")
	}
	if defaults.EffectiveMaxAttempts() != 3 || defaults.EffectiveMaxDelayMS() != 60000 {
		t.Fatalf("unexpected defaults: attempts=%d delay=%d", defaults.EffectiveMaxAttempts(), defaults.EffectiveMaxDelayMS())
	}
	disabled := false
	custom := ThrottlingRetryOptions{Enabled: &disabled, MaxAttempts: 5, MaxDelayMS: 2500}
	if custom.IsEnabled() {
		t.Fatal("explicit false should disable throttling retry")
	}
	if custom.EffectiveMaxAttempts() != 5 || custom.EffectiveMaxDelayMS() != 2500 {
		t.Fatalf("unexpected custom values: attempts=%d delay=%d", custom.EffectiveMaxAttempts(), custom.EffectiveMaxDelayMS())
	}
}

func TestThrottlingRetryDelay(t *testing.T) {
	retryAfter := int64(5000)
	err := &openapiClient.ThrottlingError{RetryAfter: &retryAfter}
	delay, ok := throttlingRetryDelay(err, ThrottlingRetryOptions{MaxDelayMS: 1200})
	if !ok || delay != 1200 {
		t.Fatalf("delay=%d ok=%v, want capped 1200", delay, ok)
	}

	disabled := false
	if _, ok := throttlingRetryDelay(err, ThrottlingRetryOptions{Enabled: &disabled}); ok {
		t.Fatal("disabled retry accepted throttling error")
	}
	if _, ok := throttlingRetryDelay(errors.New("other"), ThrottlingRetryOptions{}); ok {
		t.Fatal("ordinary error accepted as throttling")
	}
}

func TestApplyThrottlingRetryHeaders(t *testing.T) {
	call := &preparedCall{request: &openapiutil.OpenApiRequest{}}
	applyThrottlingRetryHeaders(call, 2, 345)
	if got := tea.StringValue(call.request.Headers["x-acs-retry-attempts"]); got != "2" {
		t.Fatalf("retry attempts = %q", got)
	}
	if got := tea.StringValue(call.request.Headers["x-acs-retry-delay"]); got != "345" {
		t.Fatalf("retry delay = %q", got)
	}
}
