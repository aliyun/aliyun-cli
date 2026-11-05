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
	assert.NoError(t, err)
	assert.Empty(t, got)

	got, err = NormalizeSignatureVersion(" V4 ")
	assert.NoError(t, err)
	assert.Equal(t, SignatureVersionV4, got)

	_, err = NormalizeSignatureVersion("v2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")

	_, err = NormalizeSignatureVersion("v3")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")
}

func TestResolveSignatureVersionPrecedence(t *testing.T) {
	unsetSignatureVersionEnv(t)

	// Profile only.
	got, err := ResolveSignatureVersion("", false, SignatureVersionV4)
	assert.NoError(t, err)
	assert.Equal(t, SignatureVersionV4, got)

	// Env fills when profile empty.
	t.Setenv(EnvSignatureVersion, "v4")
	got, err = ResolveSignatureVersion("", false, "")
	assert.NoError(t, err)
	assert.Equal(t, SignatureVersionV4, got)

	// Profile beats env.
	t.Setenv(EnvSignatureVersion, "ignored-env")
	got, err = ResolveSignatureVersion("", false, SignatureVersionV4)
	assert.NoError(t, err)
	assert.Equal(t, SignatureVersionV4, got)

	// Flag beats profile and env.
	t.Setenv(EnvSignatureVersion, "ignored-env")
	got, err = ResolveSignatureVersion("v4", true, "ignored-profile")
	assert.NoError(t, err)
	assert.Equal(t, SignatureVersionV4, got)

	// Assigned empty flag clears; env must not backfill.
	t.Setenv(EnvSignatureVersion, "v4")
	got, err = ResolveSignatureVersion("", true, SignatureVersionV4)
	assert.NoError(t, err)
	assert.Empty(t, got)

	// Empty env does not clear a non-empty profile (profile > env).
	t.Setenv(EnvSignatureVersion, "")
	got, err = ResolveSignatureVersion("", false, SignatureVersionV4)
	assert.NoError(t, err)
	assert.Equal(t, SignatureVersionV4, got)
}

func TestOverwriteWithFlagsSignatureVersion(t *testing.T) {
	unsetSignatureVersionEnv(t)

	newCtx := func() *cli.Context {
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		AddFlags(ctx.Flags())
		return ctx
	}

	// Profile only.
	p := &Profile{SignatureVersion: SignatureVersionV4}
	p.OverwriteWithFlags(newCtx())
	assert.Equal(t, SignatureVersionV4, p.SignatureVersion)

	// Env fills empty profile.
	t.Setenv(EnvSignatureVersion, "v4")
	p = &Profile{}
	p.OverwriteWithFlags(newCtx())
	assert.Equal(t, SignatureVersionV4, p.SignatureVersion)

	// Profile beats env.
	t.Setenv(EnvSignatureVersion, "v3")
	p = &Profile{SignatureVersion: SignatureVersionV4}
	p.OverwriteWithFlags(newCtx())
	assert.Equal(t, SignatureVersionV4, p.SignatureVersion)

	// Flag beats profile/env.
	ctx := newCtx()
	f := CliSignVersionFlag(ctx.Flags())
	f.SetAssigned(true)
	f.SetValue("v4")
	t.Setenv(EnvSignatureVersion, "ignored")
	p = &Profile{SignatureVersion: "ignored"}
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, SignatureVersionV4, p.SignatureVersion)

	// Assigned empty flag clears; env must not backfill.
	ctx = newCtx()
	f = CliSignVersionFlag(ctx.Flags())
	f.SetAssigned(true)
	f.SetValue("")
	t.Setenv(EnvSignatureVersion, "v4")
	p = &Profile{SignatureVersion: SignatureVersionV4}
	p.OverwriteWithFlags(ctx)
	assert.Empty(t, p.SignatureVersion)
}

func TestUseSignatureV4(t *testing.T) {
	unsetSignatureVersionEnv(t)

	ok, err := (&Profile{}).UseSignatureV4()
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = (&Profile{SignatureVersion: "v4"}).UseSignatureV4()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = (&Profile{SignatureVersion: "V4"}).UseSignatureV4()
	assert.NoError(t, err)
	assert.True(t, ok)

	_, err = (&Profile{SignatureVersion: "v3"}).UseSignatureV4()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")

	_, err = (&Profile{SignatureVersion: "v2"}).UseSignatureV4()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")
}

func TestCliSignVersionFromFlags(t *testing.T) {
	v, assigned := CliSignVersionFromFlags(nil)
	assert.Empty(t, v)
	assert.False(t, assigned)

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	AddFlags(ctx.Flags())
	v, assigned = CliSignVersionFromFlags(ctx.Flags())
	assert.Empty(t, v)
	assert.False(t, assigned)

	f := CliSignVersionFlag(ctx.Flags())
	f.SetAssigned(true)
	f.SetValue("v4")
	v, assigned = CliSignVersionFromFlags(ctx.Flags())
	assert.Equal(t, "v4", v)
	assert.True(t, assigned)

	f.SetValue("")
	v, assigned = CliSignVersionFromFlags(ctx.Flags())
	assert.Empty(t, v)
	assert.True(t, assigned)
}

func TestUseSignatureV4NilProfile(t *testing.T) {
	var p *Profile
	ok, err := p.UseSignatureV4()
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestResolveSignatureVersionUnsupported(t *testing.T) {
	unsetSignatureVersionEnv(t)
	_, err := ResolveSignatureVersion("v1", true, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")

	t.Setenv(EnvSignatureVersion, "v1")
	_, err = ResolveSignatureVersion("", false, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")
}
