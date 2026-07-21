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

package openapi

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/throttlingretry"
)

const (
	defaultThrottlingRetryMaxAttempts = 3
	defaultThrottlingRetryMaxDelayMS  = int64(60000)
)

var throttlingRetrySleep = time.Sleep

func (a *BasicInvoker) callWithThrottlingRetry(call func() (*responses.CommonResponse, error)) (*responses.CommonResponse, error) {
	retried := false
	retryDelayMS := int64(0)
	maxAttempts := a.throttlingRetryMaxAttempts()
	for retryAttempt := 0; ; retryAttempt++ {
		if retryAttempt > 0 {
			a.applyRetryRequestHeaders(retryAttempt, retryDelayMS)
		}
		resp, err := call()
		if err == nil {
			return resp, nil
		}

		delayMS, ok := a.throttlingRetryDelay(err)
		if !ok || retryAttempt >= maxAttempts {
			if ok && retried {
				printThrottlingRetryExhausted(maxAttempts)
			}
			return resp, err
		}

		printThrottlingRetryNotice(delayMS, retryAttempt+1, maxAttempts)
		retryDelayMS = delayMS
		throttlingRetrySleep(time.Duration(delayMS) * time.Millisecond)
		retried = true
	}
}

func (a *BasicInvoker) throttlingRetryDelay(err error) (int64, bool) {
	if !a.throttlingRetryEnabled() {
		return 0, false
	}

	var serverErr *sdkerrors.ServerError
	if !errors.As(err, &serverErr) {
		return 0, false
	}

	delayMS, ok := retryAfterFromHeaders(serverErr.RespHeaders)
	if !ok {
		return 0, false
	}
	if maxDelay := a.throttlingRetryMaxDelayMS(); maxDelay > 0 && delayMS > maxDelay {
		delayMS = maxDelay
	}
	return delayMS, true
}

func retryAfterFromHeaders(headers map[string][]string) (int64, bool) {
	for key, values := range headers {
		if !strings.EqualFold(key, "x-acs-retry-after") || len(values) == 0 {
			continue
		}
		delayMS, err := strconv.ParseInt(strings.TrimSpace(values[0]), 10, 64)
		if err != nil || delayMS < 0 {
			return 0, false
		}
		return delayMS, true
	}
	return 0, false
}

func (a *BasicInvoker) throttlingRetryEnabled() bool {
	cfg := a.throttlingRetryConfigOrDefault()
	return cfg.Enabled == nil || *cfg.Enabled
}

func (a *BasicInvoker) throttlingRetryMaxAttempts() int {
	cfg := a.throttlingRetryConfigOrDefault()
	if cfg.MaxAttempts > 0 {
		return cfg.MaxAttempts
	}
	return defaultThrottlingRetryMaxAttempts
}

func (a *BasicInvoker) throttlingRetryMaxDelayMS() int64 {
	cfg := a.throttlingRetryConfigOrDefault()
	if cfg.MaxDelayMS > 0 {
		return cfg.MaxDelayMS
	}
	return defaultThrottlingRetryMaxDelayMS
}

func (a *BasicInvoker) throttlingRetryConfigOrDefault() *throttlingretry.Config {
	if a != nil && a.throttlingRetryConfig != nil {
		return a.throttlingRetryConfig
	}
	return throttlingretry.Default()
}

func (a *BasicInvoker) applyRetryRequestHeaders(retryAttempt int, delayMS int64) {
	if a == nil || a.request == nil || retryAttempt <= 0 {
		return
	}
	if a.request.Headers == nil {
		a.request.Headers = make(map[string]string)
	}
	a.request.Headers["x-acs-retry-attempts"] = fmt.Sprintf("%d", retryAttempt)
	a.request.Headers["x-acs-retry-delay"] = fmt.Sprintf("%d", delayMS)
}
