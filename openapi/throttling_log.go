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
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/logutil"
)

func initThrottlingLog(ctx *cli.Context) {
	level := ""
	if f := LogLevelFlag(ctx.Flags()); f != nil {
		if v, ok := f.GetValue(); ok {
			level = v
		}
	}
	logutil.InitFromContext(level)
}

func printThrottlingRetryNotice(delayMS int64, retryAttempt, maxAttempts int) {
	logutil.LogThrottlingRetry("throttling, retrying in %dms (attempt %d/%d)", delayMS, retryAttempt, maxAttempts)
}

func printThrottlingRetryExhausted(maxAttempts int) {
	logutil.LogThrottlingRetry("still throttled after %d attempts, giving up", maxAttempts)
}
