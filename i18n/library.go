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

var library = make(map[string]*Text)

//func En(id string, text string) (*Text) {
//	return putText(id, "en", text)
//}
//
//func Zh(id string, text string) (*Text) {
//	return putText(id, "zh", text)
//}

func T(en string, zh string) *Text {
	t := &Text{
		id:  "",
		dic: make(map[string]string),
	}
	t.dic["en"] = en
	if zh != "" {
		t.dic["zh"] = zh
	}
	return t
}

func putText(id string, lang string, text string) *Text {
	t, ok := library[id]
	if !ok {
		t = &Text{
			id:  id,
			dic: make(map[string]string),
		}
		library[id] = t
	}
	t.Put(lang, text)
	return t
}

func getText(id string, lang string) (string, bool) {
	t, ok := library[id]
	if !ok {
		return "", false
	}
	return t.Get(lang), ok
}
