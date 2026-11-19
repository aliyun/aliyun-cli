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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOIDCTokenFlagHelpSafetyWording(t *testing.T) {
	flag := NewOIDCTokenFlag()
	assert.Equal(t, OIDCTokenFlagName, flag.Name)

	for _, lang := range []string{"en", "zh"} {
		text := flag.Short.Get(lang)
		assert.Contains(t, text, "--oidc-token-file", lang)
		assert.Contains(t, text, "0600", lang)
		assert.True(t, strings.Contains(text, "configure set"), lang)
	}

	en := flag.Short.Get("en")
	assert.Contains(t, en, "argv")
	assert.Contains(t, en, "shell history")
	assert.Contains(t, en, "process listings")
	assert.Contains(t, en, "logs")
	assert.Contains(t, en, "unencrypted")

	zh := flag.Short.Get("zh")
	assert.Contains(t, zh, "命令行参数")
	assert.Contains(t, zh, "Shell 历史")
	assert.Contains(t, zh, "进程列表")
	assert.Contains(t, zh, "日志")
	assert.Contains(t, zh, "未加密")
}

func TestOIDCTokenFileFlagHelpPrecedenceWording(t *testing.T) {
	flag := NewOIDCTokenFileFlag()
	assert.Equal(t, OIDCTokenFileFlagName, flag.Name)

	for _, lang := range []string{"en", "zh"} {
		text := flag.Short.Get(lang)
		assert.Contains(t, text, "--oidc-token-value", lang)
		assert.Contains(t, text, "ALIBABA_CLOUD_OIDC_TOKEN", lang)
	}

	assert.Contains(t, flag.Short.Get("en"), "takes precedence")
	assert.Contains(t, flag.Short.Get("zh"), "该环境变量优先")
}
