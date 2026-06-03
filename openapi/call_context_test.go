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
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
)

func TestApplyCallContextRPC(t *testing.T) {
	t.Run("both env set", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "1.2.3.4")
		t.Setenv(EnvSecureTransport, "true")
		q := map[string]string{}
		applyCallContextRPC(q)
		assert.Equal(t, "1.2.3.4", q[queryKeySourceIP])
		assert.Equal(t, "true", q[queryKeySecureTransport])
	})

	t.Run("only source ip", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "10.0.0.1")
		t.Setenv(EnvSecureTransport, "")
		q := map[string]string{}
		applyCallContextRPC(q)
		assert.Equal(t, "10.0.0.1", q[queryKeySourceIP])
		_, ok := q[queryKeySecureTransport]
		assert.False(t, ok)
	})

	t.Run("only secure transport", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "")
		t.Setenv(EnvSecureTransport, "false")
		q := map[string]string{}
		applyCallContextRPC(q)
		_, ok := q[queryKeySourceIP]
		assert.False(t, ok)
		assert.Equal(t, "false", q[queryKeySecureTransport])
	})

	t.Run("none set", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "")
		t.Setenv(EnvSecureTransport, "")
		q := map[string]string{}
		applyCallContextRPC(q)
		assert.Empty(t, q)
	})

	t.Run("existing query keys are preserved", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "1.2.3.4")
		t.Setenv(EnvSecureTransport, "true")
		q := map[string]string{
			queryKeySourceIP:        "user-set",
			queryKeySecureTransport: "user-set",
		}
		applyCallContextRPC(q)
		assert.Equal(t, "user-set", q[queryKeySourceIP])
		assert.Equal(t, "user-set", q[queryKeySecureTransport])
	})

	t.Run("nil map no panic", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "1.2.3.4")
		assert.NotPanics(t, func() {
			applyCallContextRPC(nil)
		})
	})

	t.Run("trim whitespace", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "  1.2.3.4  ")
		t.Setenv(EnvSecureTransport, "  true  ")
		q := map[string]string{}
		applyCallContextRPC(q)
		assert.Equal(t, "1.2.3.4", q[queryKeySourceIP])
		assert.Equal(t, "true", q[queryKeySecureTransport])
	})
}

func TestApplyCallContextROA(t *testing.T) {
	t.Run("both env set", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "1.2.3.4")
		t.Setenv(EnvSecureTransport, "true")
		h := map[string]string{}
		applyCallContextROA(h)
		assert.Equal(t, "1.2.3.4", h[headerSourceIP])
		assert.Equal(t, "true", h[headerSecureTransport])
	})

	t.Run("existing headers are preserved", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "env-ip")
		t.Setenv(EnvSecureTransport, "true")
		h := map[string]string{
			headerSourceIP:        "user-set",
			headerSecureTransport: "user-set",
		}
		applyCallContextROA(h)
		assert.Equal(t, "user-set", h[headerSourceIP])
		assert.Equal(t, "user-set", h[headerSecureTransport])
	})

	t.Run("none set", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "")
		t.Setenv(EnvSecureTransport, "")
		h := map[string]string{}
		applyCallContextROA(h)
		assert.Empty(t, h)
	})

	t.Run("nil map no panic", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "1.2.3.4")
		assert.NotPanics(t, func() {
			applyCallContextROA(nil)
		})
	})
}

func TestApplyCallContextTeaHeaders(t *testing.T) {
	t.Run("both env set", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "1.2.3.4")
		t.Setenv(EnvSecureTransport, "true")
		h := map[string]*string{}
		applyCallContextTeaHeaders(h)
		assert.Equal(t, "1.2.3.4", tea.StringValue(h[headerSourceIP]))
		assert.Equal(t, "true", tea.StringValue(h[headerSecureTransport]))
	})

	t.Run("existing headers preserved", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "env-ip")
		t.Setenv(EnvSecureTransport, "true")
		h := map[string]*string{
			headerSourceIP: tea.String("user-set"),
		}
		applyCallContextTeaHeaders(h)
		assert.Equal(t, "user-set", tea.StringValue(h[headerSourceIP]))
		assert.Equal(t, "true", tea.StringValue(h[headerSecureTransport]))
	})

	t.Run("nil map no panic", func(t *testing.T) {
		t.Setenv(EnvSourceIP, "1.2.3.4")
		assert.NotPanics(t, func() {
			applyCallContextTeaHeaders(nil)
		})
	})
}
