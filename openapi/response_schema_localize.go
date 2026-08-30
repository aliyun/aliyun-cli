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

package openapi

import (
	"encoding/json"

	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// localizeHelpJSON rewrites canonical response/schema JSON for Help output:
// description_en/description_zh (and title_en/title_zh) collapse into one
// localized field, and empty strings are dropped. The saved bytes scale with
// the schema size because Canonical ships every description bilingually.
func localizeHelpJSON(raw json.RawMessage, lang string) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	localized := localizeHelpValue(value, lang)
	encoded, err := json.Marshal(localized)
	if err != nil {
		return raw
	}
	return json.RawMessage(encoded)
}

func localizeHelpValue(value any, lang string) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			switch key {
			case "description_en", "description_zh":
				if (key == "description_zh") == (lang == "zh") {
					if text, ok := item.(string); ok && text != "" {
						result["description"] = text
					}
				}
			case "title_en", "title_zh":
				if (key == "title_zh") == (lang == "zh") {
					if text, ok := item.(string); ok && text != "" {
						result["title"] = text
					}
				}
			default:
				localized := localizeHelpValue(item, lang)
				if text, ok := localized.(string); ok && key != "example" && key != "pattern" && text == "" {
					continue
				}
				result[key] = localized
			}
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, localizeHelpValue(item, lang))
		}
		return result
	default:
		return value
	}
}

// localizeHelpComponents localizes every named component schema.
func localizeHelpComponents(components map[string]json.RawMessage, lang string) map[string]json.RawMessage {
	if components == nil {
		return nil
	}
	result := make(map[string]json.RawMessage, len(components))
	for name, schema := range components {
		result[name] = localizeHelpJSON(schema, lang)
	}
	return result
}

func helpResponseLanguage() string {
	if language := i18n.GetLanguage(); language == "zh" {
		return "zh"
	}
	return "en"
}
