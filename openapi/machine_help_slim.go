package openapi

// slimMachineHelpJSON rewrites the decoded Machine Help document so every
// encoding is token-lean without losing information:
//
//   - localized {en, zh} objects collapse to one language-selected string,
//     mirroring the text renderers' localizedMachineHelpText choice and the
//     response section's localizeHelpJSON precedent;
//   - annotation booleans whose false value is the unambiguous default
//     (deprecated, multiVersion, canonicalHelp) are omitted; decision-relevant
//     booleans such as required stay explicit;
//   - options is omitted when it repeats exactly ["--"+name].
//
// The response-schema subtrees (responses, outputSchema, components) are
// skipped: they are already localized by localizeHelpJSON and may legitimately
// contain user-shaped {en, zh} values such as schema defaults or examples.
func slimMachineHelpJSON(value any, lang string) any {
	switch typed := value.(type) {
	case map[string]any:
		if text, ok := machineHelpLocalizedJSONText(typed, lang); ok {
			return text
		}
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if machineHelpSchemaOwnedJSONKey(key) {
				result[key] = item
				continue
			}
			if omitMachineHelpJSONField(key, item, typed) {
				continue
			}
			result[key] = slimMachineHelpJSON(item, lang)
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, slimMachineHelpJSON(item, lang))
		}
		return result
	default:
		return value
	}
}

// machineHelpSchemaOwnedJSONKey marks the response-schema carrying keys whose
// subtrees localizeHelpJSON already owns.
func machineHelpSchemaOwnedJSONKey(key string) bool {
	return key == "responses" || key == "outputSchema" || key == "components"
}

// machineHelpLocalizedJSONText reports whether value is a localized-text map
// (keys drawn from {en, zh}, string values only) and returns the selected text.
func machineHelpLocalizedJSONText(value map[string]any, lang string) (string, bool) {
	if len(value) == 0 {
		return "", false
	}
	var en, zh string
	for key, item := range value {
		text, ok := item.(string)
		if !ok {
			return "", false
		}
		switch key {
		case "en":
			en = text
		case "zh":
			zh = text
		default:
			return "", false
		}
	}
	if lang == "zh" && zh != "" {
		return zh, true
	}
	if en != "" {
		return en, true
	}
	return zh, true
}

// machineHelpFalseDefaultBools are annotation flags where absence means false.
var machineHelpFalseDefaultBools = map[string]bool{
	"deprecated":    true,
	"multiVersion":  true,
	"canonicalHelp": true,
}

func omitMachineHelpJSONField(key string, item any, parent map[string]any) bool {
	if machineHelpFalseDefaultBools[key] {
		if flag, ok := item.(bool); ok && !flag {
			return true
		}
		return false
	}
	if key == "options" {
		return machineHelpOptionsRepeatName(item, parent)
	}
	return false
}

// machineHelpOptionsRepeatName reports whether options holds exactly the
// single flag "--"+name, so the entry is derivable. Spelling that genuinely
// differs (kebab snake names such as report_id, nested paths such as
// --Tags.#.Key) keeps options.
func machineHelpOptionsRepeatName(item any, parent map[string]any) bool {
	options, ok := item.([]any)
	if !ok || len(options) != 1 {
		return false
	}
	flag, ok := options[0].(string)
	if !ok {
		return false
	}
	name, ok := parent["name"].(string)
	return ok && flag == "--"+name
}
