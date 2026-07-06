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

// Tests for BuildBaselineEnv, the minimal env map used when a plugin opts out
// of host-side profile enforcement (`profileRequired: false` in its manifest)
// and no usable Profile is available. Kept in its own file so the function's
// contract — "carry only what the plugin can't get from inherited OS env, and
// never leak host credentials" — is easy to find and easy to extend.
package config

import (
	"bytes"
	"os"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"

	"github.com/stretchr/testify/assert"
)

func TestBuildBaselineEnv(t *testing.T) {
	// Snapshot and restore envs we touch. We deliberately also clear the
	// legacy ALIBABACLOUD_/ALICLOUD_ aliases so the "canonical-only" guarantee
	// can be asserted below without false positives from CI host env.
	keys := []string{"ALIBABA_CLOUD_REGION_ID", "ALIBABACLOUD_REGION_ID", "ALICLOUD_REGION_ID", "ALIBABA_CLOUD_ENDPOINT", "ALIBABA_CLOUD_LANGUAGE"}
	saved := make(map[string]string, len(keys))
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, k := range keys {
			if v := saved[k]; v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	})

	newEmptyCtx := func() *cli.Context {
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		AddFlags(ctx.Flags())
		return ctx
	}

	t.Run("baseline always carries CLI version and is region/endpoint-free when nothing is configured", func(t *testing.T) {
		envs := BuildBaselineEnv(newEmptyCtx())
		assert.NotEmpty(t, envs["ALIBABA_CLOUD_CLI_VERSION"], "CLI version is the only must-have")
		_, hasRegion := envs["ALIBABA_CLOUD_REGION_ID"]
		_, hasEndpoint := envs["ALIBABA_CLOUD_ENDPOINT"]
		_, hasLang := envs["ALIBABA_CLOUD_LANGUAGE"]
		assert.False(t, hasRegion)
		assert.False(t, hasEndpoint)
		assert.False(t, hasLang)
	})

	t.Run("flag values populate the env map", func(t *testing.T) {
		ctx := newEmptyCtx()
		RegionFlag(ctx.Flags()).SetAssigned(true)
		RegionFlag(ctx.Flags()).SetValue("cn-hangzhou")
		EndpointFlag(ctx.Flags()).SetAssigned(true)
		EndpointFlag(ctx.Flags()).SetValue("https://devops.cn-hangzhou.aliyuncs.com")
		LanguageFlag(ctx.Flags()).SetAssigned(true)
		LanguageFlag(ctx.Flags()).SetValue("zh")

		envs := BuildBaselineEnv(ctx)
		assert.Equal(t, "cn-hangzhou", envs["ALIBABA_CLOUD_REGION_ID"])
		assert.Equal(t, "https://devops.cn-hangzhou.aliyuncs.com", envs["ALIBABA_CLOUD_ENDPOINT"])
		assert.Equal(t, "zh", envs["ALIBABA_CLOUD_LANGUAGE"])
	})

	t.Run("env values are used when no flag is provided", func(t *testing.T) {
		os.Setenv("ALIBABA_CLOUD_REGION_ID", "cn-shanghai")
		os.Setenv("ALIBABA_CLOUD_ENDPOINT", "https://devops.cn-shanghai.aliyuncs.com")
		os.Setenv("ALIBABA_CLOUD_LANGUAGE", "en")
		t.Cleanup(func() {
			os.Unsetenv("ALIBABA_CLOUD_REGION_ID")
			os.Unsetenv("ALIBABA_CLOUD_ENDPOINT")
			os.Unsetenv("ALIBABA_CLOUD_LANGUAGE")
		})

		envs := BuildBaselineEnv(newEmptyCtx())
		assert.Equal(t, "cn-shanghai", envs["ALIBABA_CLOUD_REGION_ID"])
		assert.Equal(t, "https://devops.cn-shanghai.aliyuncs.com", envs["ALIBABA_CLOUD_ENDPOINT"])
		assert.Equal(t, "en", envs["ALIBABA_CLOUD_LANGUAGE"])
	})

	t.Run("flag wins over env", func(t *testing.T) {
		os.Setenv("ALIBABA_CLOUD_REGION_ID", "cn-shanghai")
		t.Cleanup(func() { os.Unsetenv("ALIBABA_CLOUD_REGION_ID") })

		ctx := newEmptyCtx()
		RegionFlag(ctx.Flags()).SetAssigned(true)
		RegionFlag(ctx.Flags()).SetValue("cn-beijing")

		envs := BuildBaselineEnv(ctx)
		assert.Equal(t, "cn-beijing", envs["ALIBABA_CLOUD_REGION_ID"])
	})

	t.Run("legacy region env aliases are intentionally ignored", func(t *testing.T) {
		// Only the canonical ALIBABA_CLOUD_REGION_ID is honored here; the
		// historical ALIBABACLOUD_REGION_ID / ALICLOUD_REGION_ID aliases must
		// not leak into the plugin baseline env. New entry points converge
		// on one name on purpose; existing call sites that still accept the
		// aliases stay untouched.
		os.Setenv("ALIBABACLOUD_REGION_ID", "cn-shanghai")
		os.Setenv("ALICLOUD_REGION_ID", "cn-beijing")
		t.Cleanup(func() {
			os.Unsetenv("ALIBABACLOUD_REGION_ID")
			os.Unsetenv("ALICLOUD_REGION_ID")
		})

		envs := BuildBaselineEnv(newEmptyCtx())
		_, has := envs["ALIBABA_CLOUD_REGION_ID"]
		assert.False(t, has, "legacy region aliases must not populate baseline env")
	})

	t.Run("never leaks credentials", func(t *testing.T) {
		// Even if AK/bearer-token envs are set on the host, BuildBaselineEnv
		// must not forward them — the plugin must fetch them from its own OS
		// environment so we do not implicitly authenticate the wrong identity.
		os.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", "ak-from-host")
		os.Setenv("ALIBABA_CLOUD_BEARER_TOKEN", "leaky-token")
		t.Cleanup(func() {
			os.Unsetenv("ALIBABA_CLOUD_ACCESS_KEY_ID")
			os.Unsetenv("ALIBABA_CLOUD_BEARER_TOKEN")
		})

		envs := BuildBaselineEnv(newEmptyCtx())
		_, hasAK := envs["ALIBABA_CLOUD_ACCESS_KEY_ID"]
		_, hasToken := envs["ALIBABA_CLOUD_BEARER_TOKEN"]
		assert.False(t, hasAK, "AK must not leak into baseline env")
		assert.False(t, hasToken, "bearer token must not leak into baseline env")
	})
}
