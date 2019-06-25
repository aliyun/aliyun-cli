// Copyright 1999-2019 Alibaba Group Holding Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package i18n

type Text struct {
	id  string
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

func (a *Text) Put(lang string, txt string) {
	a.dic[lang] = txt
}
