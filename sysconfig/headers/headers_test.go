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

package headers

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/otel"
)

func clearOtelEnvs(t *testing.T) {
	t.Helper()
	t.Setenv(otel.EnvTraceparent, "")
	t.Setenv(otel.EnvBaggage, "")
	t.Setenv(otel.EnvEnabled, "")
}

func TestCollect_Empty(t *testing.T) {
	clearOtelEnvs(t)
	got := Collect()
	assert.Empty(t, got)
}

func TestCollect_OtelHeaders(t *testing.T) {
	clearOtelEnvs(t)
	t.Setenv(otel.EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	t.Setenv(otel.EnvBaggage, "sessionId=abc-123,userId=user-001")

	got := Collect()
	assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", got[otel.HeaderTraceparent])
	assert.Equal(t, "sessionId=abc-123,userId=user-001", got[otel.HeaderBaggage])
}

func TestMergeIntoPluginEnvs_NoHeaders(t *testing.T) {
	clearOtelEnvs(t)
	envs := map[string]string{"FOO": "bar"}
	MergeIntoPluginEnvs(envs)

	_, ok := envs[EnvPluginHeaders]
	assert.False(t, ok, "no headers => env var must NOT be set")
	assert.Equal(t, "bar", envs["FOO"], "must not touch unrelated entries")
}

func TestMergeIntoPluginEnvs_DisabledViaEnabledFalse(t *testing.T) {
	clearOtelEnvs(t)
	t.Setenv(otel.EnvEnabled, "false")
	t.Setenv(otel.EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	envs := map[string]string{}
	MergeIntoPluginEnvs(envs)

	_, ok := envs[EnvPluginHeaders]
	assert.False(t, ok, "OTel disabled => header env must NOT be set")
}

func TestMergeIntoPluginEnvs_WritesBase64JSON(t *testing.T) {
	clearOtelEnvs(t)
	t.Setenv(otel.EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	t.Setenv(otel.EnvBaggage, "sessionId=abc-123,userId=user-001")

	envs := map[string]string{}
	MergeIntoPluginEnvs(envs)

	raw, ok := envs[EnvPluginHeaders]
	require.True(t, ok)

	decoded, err := base64.StdEncoding.DecodeString(raw)
	require.NoError(t, err, "transmission value must be valid base64")

	var got map[string]string
	require.NoError(t, json.Unmarshal(decoded, &got))
	assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", got[otel.HeaderTraceparent])
	assert.Equal(t, "sessionId=abc-123,userId=user-001", got[otel.HeaderBaggage])
}

// 校验 base64 之后的 payload 字符集只包含 [A-Za-z0-9+/=]，
// 不会带 " \ $ 等容易被传输层误伤的字符。
func TestMergeIntoPluginEnvs_TransportSafeCharset(t *testing.T) {
	clearOtelEnvs(t)
	t.Setenv(otel.EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	t.Setenv(otel.EnvBaggage, `k="v",x=$y,z=<a&b>`) // 故意塞各种 shell / json 敏感字符

	envs := map[string]string{}
	MergeIntoPluginEnvs(envs)

	raw := envs[EnvPluginHeaders]
	require.NotEmpty(t, raw)
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		ok := (c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') ||
			c == '+' || c == '/' || c == '='
		assert.True(t, ok, "unexpected byte %q at %d", c, i)
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	require.NoError(t, err)
	var got map[string]string
	require.NoError(t, json.Unmarshal(decoded, &got))
	assert.Equal(t, `k="v",x=$y,z=<a&b>`, got[otel.HeaderBaggage], "value must round-trip losslessly")
}

func TestMergeIntoPluginEnvs_NilMapIsNoop(t *testing.T) {
	clearOtelEnvs(t)
	t.Setenv(otel.EnvTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	assert.NotPanics(t, func() {
		MergeIntoPluginEnvs(nil)
	})
}
