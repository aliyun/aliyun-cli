package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	runtime "github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductHelpEndpointModesAndText(t *testing.T) {
	product := canonicalmeta.ProductEntry{Code: "demo", Version: "2026-01-01",
		RegionalEndpoints:    map[string]string{"cn-beijing": "public.example.com"},
		RegionalVPCEndpoints: map[string]string{"cn-beijing": "vpc.example.com", "cn-hangzhou": "vpc-hz.example.com"},
	}
	assert.Equal(t, []machineHelpEndpoint{{RegionID: "cn-beijing", Endpoint: "public.example.com", VPCEndpoint: "vpc.example.com"}, {RegionID: "cn-hangzhou", VPCEndpoint: "vpc-hz.example.com"}}, productHelpEndpoints(product))

	service := newMachineHelpService(&stubMachineHelpRepository{products: &canonicalmeta.ProductsIndex{Products: []canonicalmeta.ProductEntry{product}}})
	doc, err := service.buildProductForStyle("demo", "", "camel")
	require.NoError(t, err)
	doc.setCurrentRegion("cn-shanghai")
	var output bytes.Buffer
	require.NoError(t, renderCanonicalProductText(&output, doc, ""))
	assert.Contains(t, output.String(), "ENDPOINTS")
	assert.Contains(t, output.String(), "public.example.com")
	assert.Contains(t, output.String(), "Note: current region cn-shanghai is not supported by this product.")
	output.Reset()
	require.NoError(t, renderCanonicalProductText(&output, doc, "missing"))
	assert.NotContains(t, output.String(), "ENDPOINTS")

	product.GlobalEndpoint = "global.example.com"
	doc.Endpoints = productHelpEndpoints(product)
	doc.unsupportedRegion = ""
	doc.setCurrentRegion("cn-shanghai")
	assert.Empty(t, doc.unsupportedRegion)
	assert.Equal(t, machineHelpEndpoint{Endpoint: "global.example.com"}, doc.Endpoints[2])
	assert.Empty(t, productHelpEndpoints(canonicalmeta.ProductEntry{}))
	applyProductHelpOptions(doc, helpOptions{Search: "memory"}, true)
	output.Reset()
	require.NoError(t, encodeMachineHelpJSON(&output, doc, true))
	assert.NotContains(t, output.String(), `"endpoints"`)
}

func TestProductHelpEndpointTopLevelLocalizedFourColumns(t *testing.T) {
	var product canonicalmeta.ProductEntry
	require.NoError(t, json.Unmarshal([]byte(`{"code":"demo","version":"2026-01-01",
		"regional_endpoints":{"cn-beijing":"public.example.com"},
		"regional_vpc_endpoints":{"cn-beijing":"vpc.example.com"},
		"region_names":{"cn-beijing":{"zh":"华北2（北京）","en":"China (Beijing)"}}}`), &product))
	service := newMachineHelpService(&stubMachineHelpRepository{products: &canonicalmeta.ProductsIndex{Products: []canonicalmeta.ProductEntry{product}}})
	oldLanguage := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(oldLanguage) })
	for _, language := range []string{"en", "zh"} {
		t.Setenv("ALIBABA_CLOUD_LANGUAGE", language)
		i18n.SetLanguage(language)
		doc, err := service.buildProductForStyle("demo", "", "camel")
		require.NoError(t, err)
		var output bytes.Buffer
		require.NoError(t, encodeMachineHelpJSON(&output, doc, false, "endpoints"))
		name := map[string]string{"en": "China (Beijing)", "zh": "华北2（北京）"}[language]
		assert.Contains(t, output.String(), `"regionName": "`+name+`"`)
		assert.Contains(t, output.String(), `"vpcEndpoint": "vpc.example.com"`)
		output.Reset()
		require.NoError(t, encodeMachineHelpJSON(&output, doc, false, "product.endpoints"))
		assert.Equal(t, "null\n", output.String())
		output.Reset()
		require.NoError(t, renderCanonicalProductText(&output, doc, ""))
		for _, value := range []string{"RegionId", "RegionName", "Endpoint", "VpcEndpoint", name, "public.example.com", "vpc.example.com"} {
			assert.Contains(t, output.String(), value)
		}
		lines := strings.Split(output.String(), "\n")
		var headerColumn, valueColumn int
		for _, line := range lines {
			if index := strings.Index(line, "Endpoint  "); index >= 0 {
				headerColumn = endpointCellWidth(line[:index])
			}
			if index := strings.Index(line, "public.example.com"); index >= 0 {
				valueColumn = endpointCellWidth(line[:index])
			}
		}
		assert.Positive(t, headerColumn)
		assert.Equal(t, headerColumn, valueColumn)
	}
}

func TestUtilityAndEarlyParameterHelpQuery(t *testing.T) {
	c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
	ctx.Command().AddSubCommand(&cli.Command{Name: "utils", Short: i18n.T("Utilities", "工具")})
	ctx.SetInvocationArgs([]string{"utils", "--help", "--cli-query", "helpLevel"})
	require.NoError(t, c.renderUtilityHelp(ctx, []string{"utils"}, cli.HelpOptions{}, false))
	assert.JSONEq(t, `"utility"`, stdout.String())
	stdout.Reset()
	args := []string{"demo", "CreateReport", "--ReportId", "--help", "--cli-query=parameter.name"}
	ctx.SetInvocationArgs(args)
	handled, err := c.beforeParseHelpRoute(ctx, args)
	require.NoError(t, err)
	require.True(t, handled)
	assert.JSONEq(t, `"ReportId"`, stdout.String())
}

func TestHelpQueryUsesLocalizedJSONAndPreservesNumbers(t *testing.T) {
	var output bytes.Buffer
	doc := map[string]any{
		"name":       machineHelpLocalizedText{EN: "Demo", ZH: "演示"},
		"id":         json.Number("9007199254740993"),
		"previousId": json.Number("9007199254740992"),
	}
	require.NoError(t, encodeMachineHelpJSON(&output, doc, false, "{name:name,id:id}"))
	assert.Contains(t, output.String(), "9007199254740993")
	assert.NotContains(t, output.String(), `"en"`)
	output.Reset()
	require.NoError(t, encodeMachineHelpJSON(&output, doc, false, "id == previousId"))
	assert.Equal(t, "false\n", output.String())
	output.Reset()
	require.NoError(t, encodeMachineHelpJSON(&output, doc, false, "id > previousId"))
	assert.Equal(t, "true\n", output.String())
	output.Reset()
	require.NoError(t, encodeMachineHelpJSON(&output, doc, false, "missing"))
	assert.Equal(t, "null\n", output.String())
}

func TestProductHelpEndpointQuery(t *testing.T) {
	for _, aiMode := range []bool{false, true} {
		t.Run(map[bool]string{false: "text mode", true: "AI mode"}[aiMode], func(t *testing.T) {
			c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
			c.library.helpRepo = &stubMachineHelpRepository{products: &canonicalmeta.ProductsIndex{Products: []canonicalmeta.ProductEntry{{
				Code: "demo", Version: "2026-01-01", PluginDefaultVersion: "2026-01-01",
				RegionalEndpoints: map[string]string{"cn-beijing": "demo.cn-beijing.aliyuncs.com", "ap-southeast-1": "demo.ap-southeast-1.aliyuncs.com"},
			}}}}
			c.library.baselineHelpRepo = c.library.helpRepo
			ctx.SetInvocationArgs([]string{"demo", "--help", "--cli-output", "json", "--cli-query", "endpoints"})
			err := c.renderHostHelpTarget(ctx, HelpTarget{Level: HelpLevelProduct, Product: "demo", CommandStyle: CommandStyleCamel, Output: HelpOutputJSON}, aiMode)
			require.NoError(t, err)
			assert.JSONEq(t, `[{"regionId":"ap-southeast-1","endpoint":"demo.ap-southeast-1.aliyuncs.com"},{"regionId":"cn-beijing","endpoint":"demo.cn-beijing.aliyuncs.com"}]`, stdout.String())
		})
	}
}

func TestHelpQueryAllHostLevels(t *testing.T) {
	for _, target := range []HelpTarget{
		{Level: HelpLevelRoot},
		{Level: HelpLevelProduct, Product: "demo"},
		{Level: HelpLevelAction, Product: "demo", Action: "CreateReport"},
		{Level: HelpLevelAction, Product: "demo", Action: "CreateReport", Section: HelpSectionResponse, SectionExplicit: true},
		{Level: HelpLevelParameter, Product: "demo", Action: "CreateReport", Parameter: "--ReportId"},
	} {
		t.Run(string(target.Level)+string(target.Section), func(t *testing.T) {
			c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
			if target.Level != HelpLevelRoot {
				target.CommandStyle = CommandStyleCamel
			}
			target.Output = HelpOutputJSON
			QueryFlag(ctx.Flags()).SetAssigned(true)
			QueryFlag(ctx.Flags()).SetValue("helpLevel")
			require.NoError(t, c.renderHostHelpTarget(ctx, target, false))
			var level string
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &level))
			expected := string(target.Level)
			if target.Level == HelpLevelAction {
				expected = "api"
			}
			assert.Equal(t, expected, level)
		})
	}
}

func TestInvalidHelpQueryProducesNoPartialOutput(t *testing.T) {
	c, ctx, stdout, _ := newCanonicalHelpTestContext(t)
	ctx.SetInvocationArgs([]string{"demo", "--help", "--cli-query", "invalid["})
	err := c.renderHostHelpTarget(ctx, HelpTarget{Level: HelpLevelProduct, Product: "demo", CommandStyle: CommandStyleCamel, Output: HelpOutputJSON}, true)
	var queryError *engine.QueryFilterError
	require.ErrorAs(t, err, &queryError)
	assert.Empty(t, stdout.String())
	t.Setenv("ALIBABA_CLOUD_CLI_AI_MODE", "1")
	var agentError *cli.AgentError
	require.ErrorAs(t, c.finishCommandRun(ctx, []string{"demo"}, err), &agentError)
	assert.Contains(t, agentError.Envelope().Recovery.Hint, "Help JSON")
}

func TestEndpointRecoveryUsesOfflineHelp(t *testing.T) {
	for _, action := range []string{"ListMemoryNodes", "list-memory-nodes"} {
		got := endpointDiagnosticsCommand(newRecoveryContext([]string{"bailian", action}))
		assert.Equal(t, "aliyun bailian --help --cli-output json --cli-query 'endpoints'", got)
	}
	assert.Empty(t, endpointDiagnosticsCommand(newRecoveryContext(nil)))
}

func TestEndpointTextRecoveryAcrossExecutionChains(t *testing.T) {
	for _, err := range []error{
		&meta.InvalidEndpointError{Region: "cn-shanghai", Product: &meta.Product{Code: "Bailian"}},
		&runtime.EndpointNotResolvedError{Region: "cn-shanghai", Product: "bailian"},
		cli.NewErrorWithTip(&meta.InvalidEndpointError{Region: "cn-shanghai", Product: &meta.Product{Code: "Bailian"}}, "old tip"),
	} {
		c, ctx, _, _ := newCanonicalHelpTestContext(t)
		got := c.finishCommandRun(ctx, []string{"bailian", "ListMemoryNodes"}, err)
		var tip cli.ErrorWithTip
		var originalTip cli.ErrorWithTip
		assert.Equal(t, errors.As(err, &originalTip), errors.As(got, &tip), "preserve the existing exit-code classification")
		text := got.Error()
		if tip != nil {
			text += tip.GetTip("en")
		}
		assert.Contains(t, text, "aliyun bailian --help --cli-output json --cli-query 'endpoints'")
		assert.ErrorIs(t, got, err)
	}
}
