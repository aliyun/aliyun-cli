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

package cloudsso

import (
	"net/http"
	"time"
)

const defaultCloudSSOHTTPTimeout = 10 * time.Second

// cloudSSOHTTPClient guarantees a finite request deadline while preserving a
// caller's transport, redirect policy, cookie jar, and explicitly set timeout.
func cloudSSOHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return &http.Client{Timeout: defaultCloudSSOHTTPTimeout}
	}
	if client.Timeout > 0 {
		return client
	}

	clientWithTimeout := *client
	clientWithTimeout.Timeout = defaultCloudSSOHTTPTimeout
	return &clientWithTimeout
}
