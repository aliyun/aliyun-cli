package openapi

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderResponseHelpText(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildAPIResponse("demo", "CreateReport", "2026-01-01")
	require.NoError(t, err)
	doc.ResponseQuery = &machineHelpQueryExample{
		Path:         "Reports.Report",
		QueryCommand: "aliyun demo CreateReport --cli-query 'Reports.Report'",
	}

	var output bytes.Buffer
	require.NoError(t, renderResponseHelpText(&output, doc))
	assert.Contains(t, output.String(), "Response Schema (HTTP 200, application/json):")
	assert.Contains(t, output.String(), `"$ref": "#/components/schemas/ReportList"`)
	assert.Contains(t, output.String(), "Components:")
	assert.Contains(t, output.String(), `"ReportList": {`)
	assert.NotContains(t, output.String(), `"Unused":`)
	assert.Contains(t, output.String(), "Query this array directly:")
	assert.Contains(t, output.String(), "aliyun demo CreateReport --cli-query 'Reports.Report'")
}

func TestRenderResponseHelpTextWithoutSchema(t *testing.T) {
	service := testMachineHelpService(t)
	doc, err := service.buildAPIResponse("demo", "DescribeRegions", "2026-01-01")
	require.NoError(t, err)

	var output bytes.Buffer
	require.NoError(t, renderResponseHelpText(&output, doc))
	assert.Equal(t, "No response schema is available for this API.\n", output.String())
}
