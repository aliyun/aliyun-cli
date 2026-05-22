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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestDoHello(t *testing.T) {
	os.Setenv("ALIBABA_CLOUD_VENDOR", "cli_test_VendorTest")

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	ctx.Flags().AddByName("skip-secure-verify")
	profile := NewProfile("default")
	profile.Mode = AK

	exw := "-----------------------------------------------\n" +
		"!!! Configure Failed please configure again !!!\n" +
		"-----------------------------------------------\n" +
		"AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first\n" +
		"-----------------------------------------------\n" +
		"!!! Configure Failed please configure again !!!\n" +
		"-----------------------------------------------\n"
	DoHello(ctx, &profile)
	assert.Equal(t, exw, w.String())

	w.Reset()
	os.Setenv("DEBUG", "sdk")
	profile.AccessKeyId = "AccessKeyId"
	profile.AccessKeySecret = "AccessKeySecret"
	profile.RegionId = "cn-hangzhou"
	DoHello(ctx, &profile)
	assert.True(t, strings.Contains(w.String(), "-----------------------------------------------\n"+
		"!!! Configure Failed please configure again !!!\n"+
		"-----------------------------------------------"))
}

func TestShouldSkipConfigureVerify(t *testing.T) {
	ctx := cli.NewCommandContext(bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	p := &Profile{Mode: BearerToken}
	assert.True(t, p.ShouldSkipConfigureVerify(ctx))

	p = &Profile{Mode: AK, SkipConfigureVerify: true}
	assert.True(t, p.ShouldSkipConfigureVerify(ctx))

	p = &Profile{Mode: AK}
	assert.False(t, p.ShouldSkipConfigureVerify(ctx))

	os.Setenv("ALIBABA_CLOUD_SKIP_CONFIGURE_VERIFY", "true")
	defer os.Unsetenv("ALIBABA_CLOUD_SKIP_CONFIGURE_VERIFY")
	assert.True(t, p.ShouldSkipConfigureVerify(ctx))
}

func TestDoHello_SkipsForBearerToken(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, bytes.NewBuffer(nil))
	profile := &Profile{
		Mode:             BearerToken,
		BearerTokenValue: "token",
		RegionId:         "cn-hangzhou",
	}

	DoHello(ctx, profile)
	assert.Contains(t, w.String(), "Configure Done!!!")
	assert.NotContains(t, w.String(), "Configure Failed")
}
