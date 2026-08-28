package openapi

import (
	"bytes"
	"encoding/json"
)

// MarshalJSON keeps the lossless OpenAPI responses/components view as the
// public response Help contract. OutputSchema is an internal convenience view
// of the same successful response and is emitted only for search projections,
// where the complete responses document has intentionally been removed.
func (document machineHelpAPIResponseDocument) MarshalJSON() ([]byte, error) {
	type responseDocumentJSON machineHelpAPIResponseDocument

	responses, err := localizeMachineHelpRawJSON(document.Responses)
	if err != nil {
		return nil, err
	}
	components, err := localizeMachineHelpComponents(document.Components)
	if err != nil {
		return nil, err
	}

	var outputSchema *machineHelpOutputSchema
	if len(bytes.TrimSpace(document.Responses)) == 0 {
		outputSchema, err = localizeMachineHelpOutputSchema(document.OutputSchema)
		if err != nil {
			return nil, err
		}
	}

	return json.Marshal(struct {
		responseDocumentJSON
		Responses    json.RawMessage          `json:"responses"`
		Components   *machineHelpComponents   `json:"components"`
		OutputSchema *machineHelpOutputSchema `json:"outputSchema"`
	}{
		responseDocumentJSON: responseDocumentJSON(document),
		Responses:            responses,
		Components:           components,
		OutputSchema:         outputSchema,
	})
}

func localizeMachineHelpOutputSchema(schema *machineHelpOutputSchema) (*machineHelpOutputSchema, error) {
	if schema == nil {
		return nil, nil
	}
	localizedSchema, err := localizeMachineHelpRawJSON(schema.Schema)
	if err != nil {
		return nil, err
	}
	components, err := localizeMachineHelpComponents(schema.Components)
	if err != nil {
		return nil, err
	}
	return &machineHelpOutputSchema{
		StatusCode:  schema.StatusCode,
		ContentType: schema.ContentType,
		Schema:      localizedSchema,
		Components:  components,
	}, nil
}

func localizeMachineHelpComponents(components *machineHelpComponents) (*machineHelpComponents, error) {
	if components == nil {
		return nil, nil
	}
	localized := &machineHelpComponents{Schemas: make(map[string]json.RawMessage, len(components.Schemas))}
	for name, schema := range components.Schemas {
		value, err := localizeMachineHelpRawJSON(schema)
		if err != nil {
			return nil, err
		}
		localized.Schemas[name] = value
	}
	return localized, nil
}

func localizeMachineHelpRawJSON(raw json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	localized := localizeMachineHelpJSONValue(value, localizedMachineHelpLanguage())
	return json.Marshal(localized)
}

func localizeMachineHelpJSONValue(value any, language string) any {
	switch typed := value.(type) {
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = localizeMachineHelpJSONValue(item, language)
		}
		return result
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = localizeMachineHelpJSONValue(item, language)
		}
		projectMachineHelpLocalizedJSONText(result, "description", language)
		projectMachineHelpLocalizedJSONText(result, "title", language)
		return result
	default:
		return value
	}
}

func projectMachineHelpLocalizedJSONText(node map[string]any, field, language string) {
	enValue, enExists := node[field+"_en"]
	zhValue, zhExists := node[field+"_zh"]
	if !enExists && !zhExists {
		return
	}
	en, enIsText := enValue.(string)
	zh, zhIsText := zhValue.(string)
	if (enExists && !enIsText) || (zhExists && !zhIsText) {
		return
	}
	baseValue, baseExists := node[field]
	base, baseIsText := baseValue.(string)
	if baseExists && !baseIsText {
		return
	}

	delete(node, field+"_en")
	delete(node, field+"_zh")
	var localized string
	if language == "zh" {
		localized = firstNonEmptyMachineHelpString(zh, base, en)
	} else {
		localized = firstNonEmptyMachineHelpString(en, base, zh)
	}
	if localized == "" {
		delete(node, field)
		return
	}
	node[field] = localized
}
