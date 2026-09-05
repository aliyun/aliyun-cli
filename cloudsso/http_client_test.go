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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudSSOHTTPClient(t *testing.T) {
	t.Run("nil client gets finite timeout", func(t *testing.T) {
		client := cloudSSOHTTPClient(nil)

		require.NotNil(t, client)
		assert.Equal(t, defaultCloudSSOHTTPTimeout, client.Timeout)
	})

	t.Run("zero timeout client is cloned", func(t *testing.T) {
		transport := &http.Transport{}
		original := &http.Client{Transport: transport}

		client := cloudSSOHTTPClient(original)

		assert.NotSame(t, original, client)
		assert.Zero(t, original.Timeout)
		assert.Equal(t, defaultCloudSSOHTTPTimeout, client.Timeout)
		assert.Same(t, transport, client.Transport)
	})

	t.Run("explicit timeout is preserved", func(t *testing.T) {
		original := &http.Client{Timeout: 30 * time.Second}

		client := cloudSSOHTTPClient(original)

		assert.Same(t, original, client)
		assert.Equal(t, 30*time.Second, client.Timeout)
	})
}
