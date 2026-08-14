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
	"fmt"
	"time"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

var throttlingRetrySleep = time.Sleep

func callWithThrottlingRetry(
	options ThrottlingRetryOptions,
	call *preparedCall,
) (map[string]any, error) {
	retried := false
	retryDelayMS := int64(0)
	maxAttempts := options.EffectiveMaxAttempts()
	for retryAttempt := 0; ; retryAttempt++ {
		if retryAttempt > 0 {
			applyThrottlingRetryHeaders(call, retryAttempt, retryDelayMS)
		}
		response, err := call.client.CallApi(call.params, call.request, call.runtime)
		if err == nil {
			return response, nil
		}

		delayMS, ok := throttlingRetryDelay(err, options)
		if !ok || retryAttempt >= maxAttempts {
			if ok && retried {
				Info("still throttled after %d attempts, giving up", maxAttempts)
			}
			return response, err
		}

		Info("throttling, retrying in %dms (attempt %d/%d)", delayMS, retryAttempt+1, maxAttempts)
		retryDelayMS = delayMS
		throttlingRetrySleep(time.Duration(delayMS) * time.Millisecond)
		retried = true
	}
}

func throttlingRetryDelay(err error, options ThrottlingRetryOptions) (int64, bool) {
	if !options.IsEnabled() {
		return 0, false
	}
	var throttlingErr *openapiClient.ThrottlingError
	if !errors.As(err, &throttlingErr) {
		return 0, false
	}
	retryAfter := throttlingErr.GetRetryAfter()
	if retryAfter == nil || *retryAfter < 0 {
		return 0, false
	}
	delayMS := *retryAfter
	if maxDelay := options.EffectiveMaxDelayMS(); delayMS > maxDelay {
		delayMS = maxDelay
	}
	return delayMS, true
}

func applyThrottlingRetryHeaders(call *preparedCall, retryAttempt int, delayMS int64) {
	if call == nil || call.request == nil || retryAttempt <= 0 {
		return
	}
	if call.request.Headers == nil {
		call.request.Headers = map[string]*string{}
	}
	call.request.Headers["x-acs-retry-attempts"] = tea.String(fmt.Sprintf("%d", retryAttempt))
	call.request.Headers["x-acs-retry-delay"] = tea.String(fmt.Sprintf("%d", delayMS))
}
