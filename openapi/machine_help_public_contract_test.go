package openapi

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMachineHelpJSONUsesPublicHelpLevelEnvelope(t *testing.T) {
	documents := []struct {
		name      string
		helpLevel string
		document  any
	}{
		{name: "root", helpLevel: "root", document: &machineHelpRootDocument{
			SchemaVersion: machineHelpSchemaVersion, Kind: "root",
			Target:  machineHelpTarget{Path: []string{"aliyun"}, RequestedStyle: "root"},
			Matches: []machineHelpRootMatch{{Kind: "product", Name: "demo"}},
		}},
		{name: "product", helpLevel: "product", document: &machineHelpProductDocument{
			SchemaVersion: machineHelpSchemaVersion, Kind: "product",
			Target: machineHelpTarget{Path: []string{"aliyun", "demo"}, RequestedStyle: "kebab"},
			Product: machineHelpProduct{
				Code: "demo", PluginDefaultVersion: "2026-01-01", Distribution: "meta",
				Plugin: "aliyun-cli-demo", PluginVersion: "1.2.3",
			},
		}},
		{name: "api request", helpLevel: "api", document: &machineHelpAPIDocument{
			SchemaVersion: machineHelpSchemaVersion, Kind: "api", Section: helpSectionRequest,
			Target:  machineHelpTarget{Path: []string{"aliyun", "demo", "create-thing"}, RequestedStyle: "kebab"},
			Product: machineHelpProduct{Code: "demo", Plugin: "aliyun-cli-demo", PluginVersion: "1.2.3"},
		}},
		{name: "api response", helpLevel: "api", document: &machineHelpAPIResponseDocument{
			SchemaVersion: machineHelpSchemaVersion, Kind: "api", Section: helpSectionResponse,
			Target:   machineHelpTarget{Path: []string{"aliyun", "demo", "create-thing"}, RequestedStyle: "kebab"},
			Provider: "aliyun-cli-demo (1.2.3)",
		}},
		{name: "parameter", helpLevel: "parameter", document: &machineHelpParameterDocument{
			SchemaVersion: machineHelpSchemaVersion, Kind: "parameter",
			Target:    machineHelpTarget{Path: []string{"aliyun", "demo", "create-thing", "--name"}, RequestedStyle: "kebab"},
			Parameter: machineHelpParameter{Name: "name", Type: "string"},
		}},
		{name: "utility", helpLevel: "utility", document: &machineHelpUtilityDocument{
			SchemaVersion: machineHelpSchemaVersion, Kind: "utility",
			Target: machineHelpTarget{Path: []string{"aliyun", "utils"}, RequestedStyle: "utility"},
			Name:   "utils",
		}},
	}

	for _, test := range documents {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			require.NoError(t, encodeMachineHelpJSON(&output, test.document, true))
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &envelope))
			assert.Equal(t, test.helpLevel, envelope["helpLevel"])
			assert.NotContains(t, envelope, "kind")
			assert.NotContains(t, envelope, "target")
			if test.name == "root" {
				matches := envelope["matches"].([]any)
				assert.Equal(t, "product", matches[0].(map[string]any)["kind"])
			}
		})
	}

	var output bytes.Buffer
	require.NoError(t, encodeMachineHelpJSON(&output, documents[1].document, true))
	var productEnvelope map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &productEnvelope))
	product := productEnvelope["product"].(map[string]any)
	assert.NotContains(t, product, "plugin")
	assert.NotContains(t, product, "pluginVersion")
	assert.Equal(t, "2026-01-01", product["pluginDefaultVersion"])
	assert.Equal(t, "meta", product["distribution"])

	output.Reset()
	require.NoError(t, encodeMachineHelpJSON(&output, documents[3].document, true))
	var responseEnvelope map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &responseEnvelope))
	assert.Equal(t, "response", responseEnvelope["section"])
	assert.NotContains(t, responseEnvelope, "provider")
}

func TestMachineHelpJSONPreservesBusinessPluginAndProviderSchemaFields(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"plugin":{"type":"string"},"nested":{"type":"object","properties":{"provider":{"type":"string"}}}}}`)
	documents := []struct {
		name     string
		document *machineHelpAPIResponseDocument
	}{
		{name: "responses and components", document: &machineHelpAPIResponseDocument{
			SchemaVersion: machineHelpSchemaVersion, Kind: "api", Section: helpSectionResponse,
			Provider:   "internal-provider",
			Responses:  json.RawMessage(`{"200":{"schema":{"type":"object","properties":{"plugin":{"type":"string"},"provider":{"type":"string"}}}}}`),
			Components: &machineHelpComponents{Schemas: map[string]json.RawMessage{"Business": schema}},
		}},
		{name: "output schema", document: &machineHelpAPIResponseDocument{
			SchemaVersion: machineHelpSchemaVersion, Kind: "api", Section: helpSectionResponse,
			Provider: "internal-provider",
			OutputSchema: &machineHelpOutputSchema{
				StatusCode: "200", Schema: schema,
				Components: &machineHelpComponents{Schemas: map[string]json.RawMessage{"Business": schema}},
			},
		}},
	}

	for _, test := range documents {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			require.NoError(t, encodeMachineHelpJSON(&output, test.document, true))
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &envelope))
			assert.NotContains(t, envelope, "provider")
			assert.Equal(t, 2, bytes.Count(output.Bytes(), []byte(`"plugin"`)))
			assert.Equal(t, 2, bytes.Count(output.Bytes(), []byte(`"provider"`)))
		})
	}
}
