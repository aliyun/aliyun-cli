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
package i18n

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvironment(t *testing.T) {
	SetLanguage("en")
	assert.Equal(t, "en", GetLanguage())
	SetLanguage("zh")
	assert.Equal(t, "zh", GetLanguage())
}

func TestGetLanguageFromEnv(t *testing.T) {
	originLANG := os.Getenv("LANG")
	defer os.Setenv("LANG", originLANG)
	os.Setenv("LANG", "zh_CN.UTF-8")
	assert.Equal(t, "zh", getLanguageFromEnv())
	os.Setenv("LANG", "en_US.UTF-8")
	assert.Equal(t, "en", getLanguageFromEnv())
}
