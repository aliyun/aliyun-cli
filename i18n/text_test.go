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

func TestGet(t *testing.T) {
	tx := T("hello", "你好")
	assert.Equal(t, "hello", tx.Get("en"))
	assert.Equal(t, "你好", tx.Get("zh"))
	assert.Equal(t, "", tx.Get("jp"))
}

func TestText(t *testing.T) {
	tx := T("hello", "你好")
	language = "en"
	assert.Equal(t, "hello", tx.Text())
	language = "zh"
	assert.Equal(t, "你好", tx.Text())
}

func TestGetData(t *testing.T) {
	tx := T("hello", "你好")
	expected := make(map[string]string)
	expected["en"] = "hello"
	expected["zh"] = "你好"
	assert.Equal(t, expected, tx.GetData())
}

func TestLibrary(t *testing.T) {
	//Test T(en string, zh string)*Text
	text := T("hello", "你好")
	assert.Equal(t, "hello", text.dic["en"])
	assert.Equal(t, "你好", text.dic["zh"])

	text = T("", "你好")
	assert.Equal(t, "", text.dic["en"])
	assert.Equal(t, "你好", text.dic["zh"])

	text = T("hello", "")
	assert.Equal(t, "hello", text.dic["en"])
	assert.Equal(t, "", text.dic["zh"])
}

// 临时保存原始的GetLanguage函数
var originalGetLanguage func() string

// 创建mock语言设置函数
func mockGetLanguage(lang string) {
	// set env LANG = "zh_CN.UTF-8"
	switch lang {
	case "zh":
		os.Setenv("LANG", "zh_CN.UTF-8")
	case "en":
		os.Setenv("LANG", "en_US.UTF-8")
	case "fr":
		os.Setenv("LANG", "fr_FR.UTF-8")
	}
	language = lang
}

func TestText_GetMessage(t *testing.T) {
	// 测试数据
	testEn := "Hello World"
	testZh := "你好，世界"

	// 创建Text实例
	text := T(testEn, testZh)

	// 测试英文环境
	mockGetLanguage("en")
	if msg := text.GetMessage(); msg != testEn {
		t.Errorf("Expected English message to be %q, got %q", testEn, msg)
	}

	// 测试中文环境
	mockGetLanguage("zh")
	if msg := text.GetMessage(); msg != testZh {
		t.Errorf("Expected Chinese message to be %q, got %q", testZh, msg)
	}

	// 测试不支持的语言
	mockGetLanguage("fr")
	if msg := text.GetMessage(); msg != "" {
		t.Errorf("Expected unsupported language message to be empty, got %q", msg)
	}
}
