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
	"bytes"
	"os"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unsetSignatureVersionEnv(t *testing.T) {
	t.Helper()
	orig, had := os.LookupEnv(EnvSignatureVersion)
	require.NoError(t, os.Unsetenv(EnvSignatureVersion))
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(EnvSignatureVersion, orig)
		} else {
			_ = os.Unsetenv(EnvSignatureVersion)
		}
	})
}

func TestNormalizeSignatureVersion(t *testing.T) {
	got, err := NormalizeSignatureVersion("")
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = NormalizeSignatureVersion(" V4 ")
	require.NoError(t, err)
	assert.Equal(t, SignatureVersionV4, got)

	_, err = NormalizeSignatureVersion("v2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")

	_, err = NormalizeSignatureVersion("v3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")
}

func TestEffectiveSignatureVersionPrecedence(t *testing.T) {
	unsetSignatureVersionEnv(t)

	p := &Profile{SignatureVersion: SignatureVersionV4}
	assert.Equal(t, SignatureVersionV4, p.EffectiveSignatureVersion("", false))

	t.Setenv(EnvSignatureVersion, "v4")
	p.SignatureVersion = ""
	assert.Equal(t, "v4", p.EffectiveSignatureVersion("", false))

	t.Setenv(EnvSignatureVersion, "v4")
	p.SignatureVersion = "ignored"
	assert.Equal(t, "v4", p.EffectiveSignatureVersion("", false))

	// Assigned --cli-sign-version wins over env and profile.
	t.Setenv(EnvSignatureVersion, "ignored-env")
	p.SignatureVersion = "ignored-profile"
	assert.Equal(t, SignatureVersionV4, p.EffectiveSignatureVersion("v4", true))

	// Assigned empty clears env/profile v4 for this invocation.
	assert.Empty(t, p.EffectiveSignatureVersion("", true))

	// Present empty env clears profile v4.
	t.Setenv(EnvSignatureVersion, "")
	p.SignatureVersion = SignatureVersionV4
	assert.Empty(t, p.EffectiveSignatureVersion("", false))
}

func TestUseSignatureV4(t *testing.T) {
	unsetSignatureVersionEnv(t)

	ok, err := (&Profile{}).UseSignatureV4("", false)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = (&Profile{SignatureVersion: "v4"}).UseSignatureV4("", false)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = (&Profile{SignatureVersion: "V4"}).UseSignatureV4("", false)
	require.NoError(t, err)
	assert.True(t, ok)

	_, err = (&Profile{SignatureVersion: "v3"}).UseSignatureV4("", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")

	_, err = (&Profile{SignatureVersion: "v2"}).UseSignatureV4("", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")

	t.Setenv(EnvSignatureVersion, "v4")
	ok, err = (&Profile{SignatureVersion: "ignored"}).UseSignatureV4("", false)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = (&Profile{}).UseSignatureV4("v4", true)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = (&Profile{SignatureVersion: SignatureVersionV4}).UseSignatureV4("", true)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCliSignVersionFromFlags(t *testing.T) {
	v, assigned := CliSignVersionFromFlags(nil)
	assert.Empty(t, v)
	assert.False(t, assigned)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	v, assigned = CliSignVersionFromFlags(ctx.Flags())
	assert.Empty(t, v)
	assert.False(t, assigned)

	AddFlags(ctx.Flags())
	v, assigned = CliSignVersionFromFlags(ctx.Flags())
	assert.Empty(t, v)
	assert.False(t, assigned)

	f := CliSignVersionFlag(ctx.Flags())
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue("  v4  ")
	v, assigned = CliSignVersionFromFlags(ctx.Flags())
	assert.Equal(t, "v4", v)
	assert.True(t, assigned)

	f.SetValue("")
	v, assigned = CliSignVersionFromFlags(ctx.Flags())
	assert.Empty(t, v)
	assert.True(t, assigned)
}

func TestEffectiveSignatureVersionNilProfile(t *testing.T) {
	unsetSignatureVersionEnv(t)
	var p *Profile
	assert.Empty(t, p.EffectiveSignatureVersion("", false))
	assert.Equal(t, "v4", p.EffectiveSignatureVersion("v4", true))

	t.Setenv(EnvSignatureVersion, "v4")
	assert.Equal(t, "v4", p.EffectiveSignatureVersion("", false))
}

func TestUseSignatureV4UnsupportedEnv(t *testing.T) {
	t.Setenv(EnvSignatureVersion, "v1")
	_, err := (&Profile{}).UseSignatureV4("", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")
}
