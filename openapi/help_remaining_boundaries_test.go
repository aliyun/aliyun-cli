package openapi

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/stretchr/testify/require"
)

type failingHelpRepository struct {
	machineHelpRepository
	stage string
	cause error
}

func (r failingHelpRepository) GetVersionIndex(product, version string) (*canonicalmeta.VersionIndex, error) {
	if r.stage == "version" {
		return nil, r.cause
	}
	return r.machineHelpRepository.GetVersionIndex(product, version)
}
func (r failingHelpRepository) GetAPI(product, version, name string) (*canonicalmeta.API, error) {
	if r.stage == "api" {
		return nil, r.cause
	}
	return r.machineHelpRepository.GetAPI(product, version, name)
}

func TestMachineHelpPropagatesRepositoryFailuresAtEveryLevel(t *testing.T) {
	base := canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata"))
	cause := errors.New("metadata unavailable")
	for _, stage := range []string{"version", "api"} {
		service := newMachineHelpService(failingHelpRepository{machineHelpRepository: base, stage: stage, cause: cause})
		_, err := service.buildAPI("demo", "CreateReport", "2026-01-01")
		require.ErrorIs(t, err, cause)
		_, err = service.buildAPIResponse("demo", "CreateReport", "2026-01-01")
		require.ErrorIs(t, err, cause)
		if stage == "version" {
			_, err = service.buildProductForStyle("demo", "2026-01-01", "camel")
			require.ErrorIs(t, err, cause)
		}
	}
	service := newMachineHelpService(base)
	_, err := service.buildRoot(nil)
	require.ErrorContains(t, err, "nil")
	_, err = service.buildProductForStyle("demo", "2026-01-01", "unsupported")
	require.ErrorContains(t, err, "command style")
	_, err = service.buildProductForStyle("demo", "missing", "camel")
	require.Error(t, err)
	document, err := service.buildProductForStyle("demo", "2026-01-01", "pascal")
	require.NoError(t, err)
	require.Equal(t, "camel", document.Target.RequestedStyle)
	c, ctx, _, _ := newCanonicalHelpTestContext(t)
	require.ErrorContains(t, c.printMachineHelp(ctx, []string{"absent"}, "json", helpOptions{}), "unknown product")
	QueryFlag(ctx.Flags()).SetAssigned(true)
	QueryFlag(ctx.Flags()).SetValue("[")
	var queryErr *engine.QueryFilterError
	err = c.printMachineHelp(ctx, []string{"demo"}, "json", helpOptions{})
	require.ErrorAs(t, err, &queryErr)
	require.Contains(t, err.Error(), "[")
}

func TestMachineHelpIdentityAndNestedFieldsRetainCommandStyle(t *testing.T) {
	api := &canonicalmeta.API{Name: "CreateReport", CmdName: "create-report", Parameters: []canonicalmeta.Parameter{{Name: "report_id", Required: true}}}
	identity := projectMachineHelpAPI(api, "demo", "kebab")
	require.Equal(t, "demo create-report", identity.CmdFullName)
	require.Equal(t, []string{"--report-id"}, requiredResponseQueryFlags(api, "kebab"))
	require.Nil(t, requiredResponseQueryFlags(nil, "camel"))
	require.Nil(t, projectCanonicalResponseQueryExample(nil, "demo", "CreateReport", "camel", ""))
	require.Empty(t, projectCanonicalParameter(nil))
	require.Empty(t, projectCanonicalField(nil))
	field := projectCanonicalField(&canonicalmeta.Field{Name: "parent", Type: "object", Fields: []canonicalmeta.Field{{Name: "child", Type: "string"}}})
	require.Len(t, field.Fields, 1)
	require.Equal(t, "child", field.Fields[0].Name)
	require.Empty(t, projectGlobalParameters(nil))
	doc := &machineHelpAPIDocument{API: machineHelpAPI{Title: machineHelpLocalizedText{EN: "Create"}, Description: machineHelpLocalizedText{EN: "Long description"}, Operation: machineHelpOperation{APIStyle: "rpc", Method: "POST"}}}
	applyActionHelpOptions(doc, helpOptions{}, false, true)
	require.Empty(t, doc.API.Description)
	require.Empty(t, doc.API.Operation.Method)
}

func TestHelpRenderersPropagateEmptySearchAndWrappedTextFailures(t *testing.T) {
	require.Error(t, renderCanonicalRootText(&endpointFailWriter{}, &machineHelpRootDocument{}, "absent"))
	require.Error(t, renderCanonicalProductText(&endpointFailWriter{remaining: 1}, &machineHelpProductDocument{}, "absent"))
	t.Setenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH", "40")
	parameter := machineHelpParameter{Name: "long-name", Type: "string", Help: machineHelpLocalizedText{EN: "first paragraph contains enough words to wrap into several lines\n\nsecond paragraph"}}
	exerciseWriterFailures(t, func(w *failAfterWrites) error {
		return renderMachineHelpParameters(w, []machineHelpParameter{parameter})
	})
	for _, text := range []string{"Parameters:\n\nGlobal Parameters:\n", "Parameters:\n\n  --Name string\nGlobal Parameters:\n"} {
		out, listing := projectOriginalRequestHelpText(text, helpOptions{}, false)
		require.Equal(t, text, out)
		require.Nil(t, listing)
	}
	out, _ := projectOriginalRequestHelpText("Parameters:\n\n  --Name string\nGlobal Parameters:\n", helpOptions{Search: "absent"}, true)
	require.Contains(t, out, "No Help entries matched")
	require.NotContains(t, out, "--Name")
	require.Equal(t, []string{"plain", "text"}, wrapMachineHelpURLLine("plain text", 5))
	var output bytes.Buffer
	require.NoError(t, renderHelpProjectionResult(&output, "entries", HelpResult{Shown: 1, Total: 3, Truncated: true}, nil))
	require.Contains(t, output.String(), "Showing 1 of 3")
	require.Error(t, renderHelpProjectionResult(&endpointFailWriter{}, "entries", HelpResult{Shown: 1, Total: 3, Truncated: true}, nil))
}

func TestRecoveryCommandsRejectIncompleteAndUnsafeContext(t *testing.T) {
	require.Nil(t, normalizeServerAgentError(&tea.SDKError{}, recoveryContext{}))
	cause := errors.New("failure")
	var structured *cli.AgentError
	require.ErrorAs(t, serverAgentError(cause, serverErrorFacts{code: "Unavailable"}, recoveryContext{}), &structured)
	require.Equal(t, "server error Unavailable", structured.Envelope().Message)
	previous := i18n.GetLanguage()
	i18n.SetLanguage("zh")
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	require.Equal(t, "zh-CN", diagnoseAcceptLanguage())
	require.Nil(t, sanitizeNetworkTransportError(nil))
	transport := newTransportError(&url.Error{URL: "://invalid"})
	require.Equal(t, "network request failed: network transport failed", transport.Error())
	require.Equal(t, "--body-file: missing.json", bodyFileAgentMessage("missing.json", errors.New("--body-file:")))
	require.Equal(t, "--body-file: missing.json: failure", bodyFileAgentMessage("missing.json", cause))
	for _, flag := range []string{"--", "--Name.Child", "--bad;flag"} {
		require.Empty(t, topLevelRecoveryFlag(flag))
	}
	require.Empty(t, (recoveryContext{}).sectionSearchCommand("id"))
	require.Empty(t, (recoveryContext{product: "demo", api: "Create", section: "response", version: "v1", versionFlag: "--unsupported"}).sectionSearchCommand("id"))
	context := recoveryContext{product: "demo", api: "create-report", style: "kebab", section: "response"}
	require.Contains(t, context.sectionSearchCommand("id"), "create-report")
	require.Equal(t, "aliyun --help", (recoveryContext{}).parentHelpCommand(""))
	require.Equal(t, []string{"demo"}, (recoveryContext{args: []string{"demo", "--profile", "test"}}).parentCommandParts(""))
	require.Equal(t, []string{"demo"}, (recoveryContext{args: []string{"demo", "bad token"}}).parentCommandParts(""))
	require.Equal(t, "Create", searchRequestForHelpCommand("aliyun demo Create --help", "id").API)
	for _, args := range [][]string{{"--version"}, {"--api-version=bad value"}} {
		flag, version := explicitVersion(args)
		require.Empty(t, flag)
		require.Empty(t, version)
	}
	value, exists := recoveryOptionValue([]string{"--cli-section"}, "--cli-section")
	require.True(t, exists)
	require.Empty(t, value)
	require.Equal(t, []string{"id"}, parameterSearchKeywordCandidates("kebab", "", "id"))
	require.Equal(t, []string{"valid"}, orderedSearchCandidates("", "bad token", "valid"))
	require.Nil(t, crossStyleFlagSuggestions("--", nil))
	require.Nil(t, crossStyleFlagSuggestions("lowercase", []string{"other"}))
	require.Equal(t, []string{"--RegionId"}, crossStyleFlagSuggestions("region-id", []string{"RegionId"}))
}
