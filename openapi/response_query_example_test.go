package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseQuerySelectArrayPathHandlesSingleInlineArray(t *testing.T) {
	document := responseQueryDocument(`{
		"type":"object",
		"properties":{
			"RequestId":{"type":"string"},
			"Items":{"type":"array","items":{"type":"string"}}
		}
	}`, nil)

	path, err := SelectResponseArrayPath(document, "ListThings", "")
	require.NoError(t, err)
	assert.Equal(t, "Items", path)
}

func TestResponseQuerySelectArrayPathUsesExplicitPaginationCollection(t *testing.T) {
	document := responseQueryDocument(`{
		"type":"object",
		"properties":{
			"Items":{"type":"array","items":{"type":"string"}},
			"Payload":{"type":"object","properties":{
				"Entries":{"type":"array","items":{"type":"object"}}
			}}
		}
	}`, nil)

	path, err := SelectResponseArrayPath(document, "ListThings", "Payload.Entries")
	require.NoError(t, err)
	assert.Equal(t, "Payload.Entries", path)

	path, err = SelectResponseArrayPath(document, "ListThings", "Payload.DoesNotExist")
	require.NoError(t, err)
	assert.Equal(t, "Items", path)
}

func TestResponseQuerySelectArrayPathPrefersPaginationSibling(t *testing.T) {
	document := responseQueryDocument(`{
		"type":"object",
		"properties":{
			"Items":{"type":"array","items":{"type":"string"}},
			"Data":{"type":"object","properties":{
				"NextToken":{"type":"string"},
				"Records":{"type":"array","items":{"type":"object"}}
			}}
		}
	}`, nil)

	path, err := SelectResponseArrayPath(document, "ListItems", "")
	require.NoError(t, err)
	assert.Equal(t, "Data.Records", path)
}

func TestResponseQuerySelectArrayPathMatchesAPIResourceName(t *testing.T) {
	document := responseQueryDocument(`{
		"type":"object",
		"properties":{
			"Warnings":{"type":"array","items":{"type":"string"}},
			"Instances":{"type":"object","properties":{
				"Instance":{"type":"array","items":{"type":"object"}}
			}},
			"Items":{"type":"array","items":{"type":"object"}}
		}
	}`, nil)

	path, err := SelectResponseArrayPath(document, "DescribeInstances", "")
	require.NoError(t, err)
	assert.Equal(t, "Instances.Instance", path)
}

func TestResponseQuerySelectArrayPathPrefersCommonNameThenStableFirst(t *testing.T) {
	t.Run("common result name", func(t *testing.T) {
		document := responseQueryDocument(`{
			"type":"object",
			"properties":{
				"Warnings":{"type":"array","items":{"type":"string"}},
				"Items":{"type":"array","items":{"type":"string"}}
			}
		}`, nil)

		path, err := SelectResponseArrayPath(document, "DoWork", "")
		require.NoError(t, err)
		assert.Equal(t, "Items", path)
	})

	t.Run("schema declaration order", func(t *testing.T) {
		document := responseQueryDocument(`{
			"type":"object",
			"properties":{
				"Zeta":{"type":"array","items":{"type":"string"}},
				"Alpha":{"type":"array","items":{"type":"string"}}
			}
		}`, nil)

		path, err := SelectResponseArrayPath(document, "DoWork", "")
		require.NoError(t, err)
		assert.Equal(t, "Zeta", path)
	})
}

func TestResponseQuerySelectArrayPathTraversesRefsAndProtectsCycles(t *testing.T) {
	document := responseQueryDocument(
		`{"type":"object","properties":{"Root":{"$ref":"#/components/schemas/Node"}}}`,
		map[string]string{
			"Node": `{
				"type":"object",
				"properties":{
					"Child":{"$ref":"#/components/schemas/Node"},
					"Records":{"type":"array","items":{"$ref":"#/components/schemas/Record"}}
				}
			}`,
			"Record": `{"type":"object","properties":{"Name":{"type":"string"}}}`,
		},
	)

	path, err := SelectResponseArrayPath(document, "ListRecords", "")
	require.NoError(t, err)
	assert.Equal(t, "Root.Records", path)
}

func TestResponseQuerySelectArrayPathIgnoresMissingRefsAndMapValues(t *testing.T) {
	document := responseQueryDocument(
		`{
			"type":"object",
			"properties":{
				"Missing":{"$ref":"#/components/schemas/Missing"},
				"ById":{"type":"object","additionalProperties":{
					"type":"array","items":{"type":"string"}
				}}
			}
		}`,
		nil,
	)

	path, err := SelectResponseArrayPath(document, "ListValues", "")
	require.NoError(t, err)
	assert.Empty(t, path)
}

func TestResponseQuerySelectArrayPathReturnsEmptyForScalarOrRootArray(t *testing.T) {
	for _, schema := range []string{
		`{"type":"object","properties":{"RequestId":{"type":"string"}}}`,
		`{"type":"array","items":{"type":"string"}}`,
	} {
		path, err := SelectResponseArrayPath(responseQueryDocument(schema, nil), "ListThings", "")
		require.NoError(t, err)
		assert.Empty(t, path)
	}
}

func TestBuildResponseQueryExamplePreservesCommandStyleAndVersion(t *testing.T) {
	document := responseQueryDocument(`{
		"type":"object",
		"properties":{
			"Instances":{"type":"object","properties":{
				"Instance":{"type":"array","items":{"type":"object"}}
			}}
		}
	}`, nil)

	tests := []struct {
		name          string
		context       ResponseQueryContext
		wantSchemaCmd string
		wantQueryCmd  string
	}{
		{
			name: "pascal without explicit version",
			context: ResponseQueryContext{
				Document: document,
				Product:  "ecs",
				API:      "DescribeInstances",
				Style:    ResponseCommandStylePascal,
			},
			wantSchemaCmd: "aliyun help ecs DescribeInstances --cli-section response",
			wantQueryCmd:  "aliyun ecs DescribeInstances --cli-query 'Instances.Instance'",
		},
		{
			name: "kebab with explicit version and canonical API name",
			context: ResponseQueryContext{
				Document:   document,
				Product:    "ecs",
				API:        "DescribeInstances",
				APIVersion: "2014-05-26",
				Style:      ResponseCommandStyleKebab,
			},
			wantSchemaCmd: "aliyun help ecs describe-instances --api-version 2014-05-26 --cli-section response",
			wantQueryCmd:  "aliyun ecs describe-instances --api-version 2014-05-26 --cli-query 'Instances.Instance'",
		},
		{
			name: "machine camel style is pascal",
			context: ResponseQueryContext{
				Document: document,
				Product:  "ecs",
				API:      "describe-instances",
				Style:    ResponseCommandStyleCamel,
			},
			wantSchemaCmd: "aliyun help ecs DescribeInstances --cli-section response",
			wantQueryCmd:  "aliyun ecs DescribeInstances --cli-query 'Instances.Instance'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			example, err := BuildResponseQueryExample(tt.context)
			require.NoError(t, err)
			require.NotNil(t, example)
			assert.Equal(t, "Instances.Instance", example.Path)
			assert.Equal(t, tt.wantSchemaCmd, example.SchemaCommand)
			assert.Equal(t, tt.wantQueryCmd, example.QueryCommand)
		})
	}
}

func TestBuildResponseQueryExamplePreservesPluralAcronymInKebabStyle(t *testing.T) {
	document := responseQueryDocument(`{
		"type":"object",
		"properties":{"VPCs":{"type":"array","items":{"type":"object"}}}
	}`, nil)

	example, err := BuildResponseQueryExample(ResponseQueryContext{
		Document: document,
		Product:  "vpc",
		API:      "DescribeVPCs",
		Style:    ResponseCommandStyleKebab,
	})
	require.NoError(t, err)
	require.NotNil(t, example)
	assert.Equal(t, "aliyun help vpc describe-vpcs --cli-section response", example.SchemaCommand)
	assert.Equal(t, "aliyun vpc describe-vpcs --cli-query 'VPCs'", example.QueryCommand)
}

func TestBuildResponseQueryExamplePreservesAlibabaVResourceToken(t *testing.T) {
	document := responseQueryDocument(`{
		"type":"object",
		"properties":{"VRouters":{"type":"array","items":{"type":"object"}}}
	}`, nil)

	example, err := BuildResponseQueryExample(ResponseQueryContext{
		Document: document,
		Product:  "vpc",
		API:      "DescribeVRouters",
		Style:    ResponseCommandStyleKebab,
	})
	require.NoError(t, err)
	require.NotNil(t, example)
	assert.Equal(t, "aliyun help vpc describe-vrouters --cli-section response", example.SchemaCommand)
	assert.Equal(t, "aliyun vpc describe-vrouters --cli-query 'VRouters'", example.QueryCommand)
}

func TestBuildResponseQueryExampleQuotesUnicodeJMESPathIdentifier(t *testing.T) {
	document := responseQueryDocument(`{
		"type":"object",
		"properties":{"实例列表":{"type":"array","items":{"type":"object"}}}
	}`, nil)

	example, err := BuildResponseQueryExample(ResponseQueryContext{
		Document: document,
		Product:  "ecs",
		API:      "ListInstances",
		Style:    ResponseCommandStylePascal,
	})
	require.NoError(t, err)
	require.NotNil(t, example)
	assert.Equal(t, `"实例列表"`, example.Path)
	assert.Equal(t, `aliyun ecs ListInstances --cli-query '"实例列表"'`, example.QueryCommand)
}

func TestResponseQueryUsesOnlyFilteredResponseSearchSchema(t *testing.T) {
	document := responseQueryDocument(`{
		"type":"object",
		"properties":{
			"InstanceId":{"type":"string"},
			"Tags":{"type":"array","items":{"type":"object","properties":{"Key":{"type":"string"}}}}
		}
	}`, nil)

	scalarSearch, err := SearchResponseSchema(document, "instance-id")
	require.NoError(t, err)
	example, err := BuildResponseQueryExample(ResponseQueryContext{
		Document: HelpResponseSchema{Schema: scalarSearch.Schema, Components: scalarSearch.Components},
		Product:  "ecs",
		API:      "DescribeInstances",
		Style:    ResponseCommandStylePascal,
	})
	require.NoError(t, err)
	assert.Nil(t, example)

	arraySearch, err := SearchResponseSchema(document, "tags")
	require.NoError(t, err)
	example, err = BuildResponseQueryExample(ResponseQueryContext{
		Document: HelpResponseSchema{Schema: arraySearch.Schema, Components: arraySearch.Components},
		Product:  "ecs",
		API:      "DescribeInstances",
		Style:    ResponseCommandStylePascal,
	})
	require.NoError(t, err)
	require.NotNil(t, example)
	assert.Equal(t, "Tags", example.Path)
}

func TestBuildResponseQueryExampleOmitsInvalidContexts(t *testing.T) {
	noArray := responseQueryDocument(`{"type":"object","properties":{"Value":{"type":"string"}}}`, nil)

	for _, context := range []ResponseQueryContext{
		{Document: noArray, Product: "ecs", API: "GetValue", Style: ResponseCommandStylePascal},
		{Document: responseQueryDocument(`{"type":"array","items":{"type":"string"}}`, nil), Product: "ecs", API: "ListValues", Style: ResponseCommandStylePascal},
		{Document: responseQueryDocument(`{"type":"object","properties":{"Items":{"type":"array","items":{"type":"string"}}}}`, nil), Product: "", API: "ListValues", Style: ResponseCommandStylePascal},
	} {
		example, err := BuildResponseQueryExample(context)
		require.NoError(t, err)
		assert.Nil(t, example)
	}
}

func TestBuildResponseQueryExampleRejectsMalformedSchema(t *testing.T) {
	_, err := BuildResponseQueryExample(ResponseQueryContext{
		Document: HelpResponseSchema{Schema: json.RawMessage(`{"type":`)},
		Product:  "ecs",
		API:      "ListThings",
		Style:    ResponseCommandStylePascal,
	})
	require.Error(t, err)
}

func responseQueryDocument(schema string, components map[string]string) HelpResponseSchema {
	document := HelpResponseSchema{
		Schema:     json.RawMessage(schema),
		Components: make(map[string]json.RawMessage, len(components)),
	}
	for name, component := range components {
		document.Components[name] = json.RawMessage(component)
	}
	return document
}
