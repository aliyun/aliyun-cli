package openapi

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-cli/v3/sysconfig"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/stretchr/testify/require"
)

func TestRootHelpSearchFindsFlagsAliasesAndProductsTogether(t *testing.T) {
	doc := &machineHelpRootDocument{
		Commands:    []machineHelpCommandSummary{{Name: "utility", Group: string(RootGroupUtils), Path: []string{"aliyun", "utility"}, Aliases: []string{"shared"}}},
		GlobalFlags: []machineHelpFlagSummary{{Name: "--shared-flag", Shorthand: "-s", Aliases: []string{"shared"}}},
		Products:    []machineHelpProductSummary{{Code: "shared-product", Name: machineHelpLocalizedText{EN: "Shared service"}}},
	}
	applyRootHelpOptions(doc, helpOptions{Search: "shared", SearchAll: true}, true)
	require.Len(t, doc.Commands, 1)
	require.Len(t, doc.GlobalFlags, 1)
	require.Len(t, doc.Products, 1)
	require.Len(t, doc.Matches, 3)
	for _, match := range doc.Matches {
		if match.Kind == "command" {
			require.Empty(t, match.Aliases)
		} else if match.Kind == "global-flag" {
			require.Contains(t, match.Aliases, "-s")
		}
	}
	applyRootHelpOptions(nil, helpOptions{}, true)
	applyActionHelpOptions(nil, helpOptions{}, true, true)
	applyProductHelpOptions(nil, helpOptions{}, true)
	docAPI := &machineHelpAPIDocument{ActiveParameterSet: "kebab", ParameterSets: machineHelpParameterSets{
		Camel: []machineHelpParameter{{Name: "Camel"}}, Kebab: []machineHelpParameter{{Name: "kebab"}},
	}, GlobalParameters: []machineHelpParameter{{Name: "profile"}}}
	applyActionHelpOptions(docAPI, helpOptions{All: true}, true, true)
	require.Empty(t, docAPI.ParameterSets.Camel)
	require.Len(t, docAPI.ParameterSets.Kebab, 1)
	require.Equal(t, 2, docAPI.Result.Shown)
	responseDoc := &machineHelpAPIResponseDocument{}
	require.NoError(t, applyResponseHelpOptions(responseDoc, helpOptions{Search: "missing"}))
	require.Contains(t, responseDoc.Notice, "No Help entries")
}

func TestEngineHelpRepositoryReadsBaselineAndRejectsUnknownMetadata(t *testing.T) {
	entry, ok := engineProductEntry("ecs")
	require.True(t, ok)
	require.Equal(t, "ecs", entry.Code)
	require.NotEmpty(t, entry.Versions)
	index, err := engineVersionIndex("ecs", "2014-05-26")
	require.NoError(t, err)
	require.Equal(t, "2014-05-26", index.Version)
	require.Contains(t, index.APIs, "DescribeRegions")
	api, err := engineAPI("ecs", "2014-05-26", "DescribeRegions")
	require.NoError(t, err)
	require.Equal(t, "DescribeRegions", api.Name)
	_, ok = engineProductEntry("unknown-coverage-product")
	require.False(t, ok)
	_, err = engineVersionIndex("ecs", "invalid")
	require.Error(t, err)
	_, err = engineAPI("ecs", "2014-05-26", "UnknownCoverageAPI")
	require.Error(t, err)
}

type boundaryTransport func(*http.Request) (*http.Response, error)

func (f boundaryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLegacyInvokerClientSettingsAndUserAgent(t *testing.T) {
	_, ctx, _, _ := newCanonicalHelpTestContext(t)
	config.AddFlags(ctx.Flags())
	config.ConfigurePathFlag(ctx.Flags()).SetValue(t.TempDir())
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.SkipSecureVerify(ctx.Flags()).SetAssigned(true)
	profile := &config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou", Endpoint: "local.invalid", RetryCount: 2, ReadTimeout: 3, ConnectTimeout: 4}
	client, err := GetClient(profile, ctx)
	require.NoError(t, err)
	require.Equal(t, 3*time.Second, client.GetReadTimeout())
	require.Equal(t, 4*time.Second, client.GetConnectTimeout())
	require.True(t, client.GetConfig().AutoRetry)
	require.Equal(t, 2, client.GetConfig().MaxRetryTime)
	profile.RegionId = ""
	_, err = GetClient(profile, ctx)
	require.ErrorContains(t, err, "RegionId is empty")
	profile.RegionId = "cn-hangzhou"
	t.Setenv("ALIBABA_CLOUD_VENDOR", "local-vendor")
	t.Setenv(sysconfig.EnvUserAgent, "EnvAgent/1.0")
	UserAgentFlag(ctx.Flags()).SetValue("FlagAgent/2.0")
	UserAgentFlag(ctx.Flags()).SetAssigned(true)
	invoker := NewBasicInvoker(profile)
	require.NoError(t, invoker.Init(ctx, &meta.Product{Code: "Ecs", Version: "2014-05-26"}))
	var userAgent string
	invoker.client.SetTransport(boundaryTransport(func(r *http.Request) (*http.Response, error) {
		userAgent = r.Header.Get("User-Agent")
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	}))
	invoker.request.ApiName = "DescribeRegions"
	_, err = invoker.client.ProcessCommonRequest(invoker.request)
	require.NoError(t, err)
	for _, segment := range []string{"vendor/local-vendor", "EnvAgent/1.0", "FlagAgent/2.0"} {
		require.Contains(t, userAgent, segment)
	}
}

type boundaryInvoker struct {
	request *requests.CommonRequest
	bodies  []string
	err     error
}

func (*boundaryInvoker) getClient() *sdk.Client                { return nil }
func (i *boundaryInvoker) getRequest() *requests.CommonRequest { return i.request }
func (*boundaryInvoker) Prepare(*cli.Context) error            { return nil }
func (i *boundaryInvoker) Call() (*responses.CommonResponse, error) {
	if i.err != nil {
		return nil, i.err
	}
	body := i.bodies[0]
	if len(i.bodies) > 1 {
		i.bodies = i.bodies[1:]
	}
	response := responses.NewCommonResponse()
	err := responses.Unmarshal(response, &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, "JSON")
	return response, err
}

func TestPagerCallWithMergesPagesAndPropagatesInvalidResponses(t *testing.T) {
	invoker := &boundaryInvoker{request: requests.NewCommonRequest(), bodies: []string{
		`{"PageNumber":1,"PageSize":1,"TotalCount":2,"Items":[1]}`,
		`{"PageNumber":2,"PageSize":1,"TotalCount":2,"Items":[2]}`,
	}}
	pager := &Pager{PageNumberFlag: "PageNumber", PageNumberExpr: "PageNumber", PageSizeExpr: "PageSize", TotalCountExpr: "TotalCount", collectionPath: "Items"}
	output, err := pager.CallWith(invoker)
	require.NoError(t, err)
	require.JSONEq(t, `{"Items":[1,2]}`, output)
	require.Equal(t, "2", invoker.request.QueryParams["PageNumber"])
	invoker.bodies = []string{"invalid"}
	_, err = pager.CallWith(invoker)
	require.ErrorContains(t, err, "call failed")
	pager = &Pager{NextTokenExpr: "["}
	require.ErrorContains(t, pager.FeedResponse(`{}`), "jmespath")
}

func TestWaiterCallWithSuccessfulAndMalformedResponses(t *testing.T) {
	previous := WaiterFlag
	copyFlag := *previous
	copyFlag.Fields = append([]cli.Field(nil), previous.Fields...)
	WaiterFlag = &copyFlag
	t.Cleanup(func() { WaiterFlag = previous })
	WaiterFlag.SetAssigned(true)
	for i := range WaiterFlag.Fields {
		WaiterFlag.Fields[i].SetAssigned(false)
	}
	invoker := &boundaryInvoker{bodies: []string{`{"State":"Ready"}`}}
	output, err := (&Waiter{expr: "State", to: "Ready"}).CallWith(invoker)
	require.NoError(t, err)
	require.JSONEq(t, `{"State":"Ready"}`, output)
	invoker.bodies = []string{"invalid"}
	_, err = (&Waiter{expr: "State", to: "Ready"}).CallWith(invoker)
	require.Error(t, err)
	invoker.err = errors.New("local failure")
	_, err = (&Waiter{}).CallWith(invoker)
	require.ErrorIs(t, err, invoker.err)
}

func TestHelpTargetRejectsConflictingControlsAndUnsafeTokens(t *testing.T) {
	for _, tc := range []struct {
		target HelpTarget
		match  string
	}{
		{HelpTarget{Level: HelpLevelRoot, SearchQuery: "query"}, "only valid"},
		{HelpTarget{Level: HelpLevelRoot, Operation: "unsupported"}, "operation"},
		{HelpTarget{Level: HelpLevelRoot, Provider: "unsupported"}, "provider"},
		{HelpTarget{Level: HelpLevelRoot, CommandStyle: CommandStyleCamel}, "command style"},
		{HelpTarget{Level: HelpLevelRoot, Section: "unsupported"}, "section"},
		{HelpTarget{Level: HelpLevelRoot, SectionExplicit: true}, "action target"},
		{HelpTarget{Level: HelpLevelRoot, Version: "v1"}, "API version"},
		{HelpTarget{Level: HelpLevelRoot, VersionFlag: VersionFlagAPI}, "requires a version"},
		{HelpTarget{Level: HelpLevelParameter, Product: "", Action: "Get", Parameter: "ID"}, "product"},
		{HelpTarget{Level: HelpLevelParameter, Product: "ecs", Action: "", Parameter: "ID"}, "action"},
		{HelpTarget{Level: HelpLevelAction, Product: "ecs product", Action: "Get"}, "one command token"},
		{HelpTarget{Level: HelpLevelAction, Product: "--ecs", Action: "Get"}, "cannot be an option"},
	} {
		require.ErrorContains(t, tc.target.Validate(), tc.match)
	}
	target := HelpTarget{Level: HelpLevelAction, Product: "ecs", Action: "describe-regions"}
	require.NoError(t, target.Validate())
	require.Equal(t, CommandStyleKebab, target.normalized().CommandStyle)
}

func TestLegacyAndRuntimeErrorsKeepRecoveryContext(t *testing.T) {
	for _, cause := range []error{
		&InvalidUnifiedApiError{Name: "DescribeRegons", product: &meta.Product{Code: "Ecs", ApiNames: []string{"DescribeRegions"}}},
		&engine.UnknownCommandError{Product: "ecs", Command: "describe-regons"},
		&InvalidArgumentError{Parameter: "RegionId", Flag: "--RegionId", FieldPath: "RegionId", Err: errors.New("expected string")},
		&InvalidOptionCombinationError{Options: []string{"--pager", "--waiter"}},
	} {
		err := normalizeAgentErrorWithSearch(cause, nil, func(RecoverySearchRequest) bool { return false })
		var structured *cli.AgentError
		require.ErrorAs(t, err, &structured)
		require.ErrorIs(t, structured, cause)
		require.NotEmpty(t, structured.Envelope().Message)
		require.NotEmpty(t, structured.Envelope().Recovery.Hint)
	}
	existing := cli.NewAgentError(cli.AgentErrorEnvelope{Message: "message", Recovery: cli.AgentErrorRecovery{Action: "inspect", Hint: "hint"}}, nil)
	require.Same(t, existing, normalizeAgentErrorWithSearch(existing, nil, nil))
}
