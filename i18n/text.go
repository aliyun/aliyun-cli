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

func T(en string, zh string) *Text {
	t := &Text{
		dic: make(map[string]string),
	}
	t.dic["en"] = en
	if zh != "" {
		t.dic["zh"] = zh
	}
	return t
}

type Text struct {
	dic map[string]string
}

func (a *Text) Text() string {
	lang := GetLanguage()
	return a.Get(lang)
}

func (a *Text) Get(lang string) string {
	s, ok := a.dic[lang]
	if !ok {
		return ""
	}
	return s
}

func (a *Text) GetData() map[string]string {
	return a.dic
}

func (a *Text) GetMessage() string {
	// determine language
	lang := GetLanguage()
	return a.Get(lang)
}
