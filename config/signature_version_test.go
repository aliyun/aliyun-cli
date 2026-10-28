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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectiveSignatureVersionEnvWinsOverProfile(t *testing.T) {
	t.Setenv(EnvSignatureVersion, "")
	p := &Profile{SignatureVersion: SignatureVersionV4}
	assert.Equal(t, SignatureVersionV4, p.EffectiveSignatureVersion())

	t.Setenv(EnvSignatureVersion, "v4")
	p.SignatureVersion = ""
	assert.Equal(t, "v4", p.EffectiveSignatureVersion())

	t.Setenv(EnvSignatureVersion, "v4")
	p.SignatureVersion = "ignored"
	assert.Equal(t, "v4", p.EffectiveSignatureVersion())
}

func TestUseSignatureV4(t *testing.T) {
	t.Setenv(EnvSignatureVersion, "")
	ok, err := (&Profile{}).UseSignatureV4()
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = (&Profile{SignatureVersion: "v4"}).UseSignatureV4()
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = (&Profile{SignatureVersion: "V4"}).UseSignatureV4()
	require.NoError(t, err)
	assert.True(t, ok)

	_, err = (&Profile{SignatureVersion: "v3"}).UseSignatureV4()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")

	t.Setenv(EnvSignatureVersion, "v4")
	ok, err = (&Profile{SignatureVersion: "v3"}).UseSignatureV4()
	require.NoError(t, err)
	assert.True(t, ok)
}
