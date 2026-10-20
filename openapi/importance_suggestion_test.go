package openapi

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
)

func TestImportanceStableOptionalRanking(t *testing.T) {
	for _, tc := range []struct {
		name   string
		scores map[string]float64
		want   []string
		active bool
	}{
		{"missing", nil, nil, false},
		{"partial", map[string]float64{"a": .2, "b": .9}, []string{"b", "a", "c"}, true},
		{"zero", map[string]float64{"b": 0}, []string{"b", "a", "c"}, true},
		{"equal", map[string]float64{"a": .2, "b": .2}, []string{"a", "b", "c"}, true},
		{"invalid", map[string]float64{"a": math.NaN(), "b": 2}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, active := rankParameterSuggestions([]string{"a", "b", "c"}, tc.scores, nil)
			if !reflect.DeepEqual(got, tc.want) || active != tc.active {
				t.Fatalf("got %v %v", got, active)
			}
		})
	}
	got, _ := rankParameterSuggestions([]string{"a", "flag", "b"}, map[string]float64{"a": .2, "b": .9, "flag": 1}, map[string]bool{"flag": true})
	if !reflect.DeepEqual(got, []string{"b", "flag", "a"}) {
		t.Fatal(got)
	}
}

func TestAPIImportanceKeepsTierAndQualityBeforePopularity(t *testing.T) {
	got, active := rankedAPISuggestions("DescribeThing", []string{"DescribeThings", "DescribeThingOptions"}, map[string]float64{"DescribeThings": 0, "DescribeThingOptions": 1}, true, "")
	if active || !reflect.DeepEqual(got, []string{"DescribeThings"}) {
		t.Fatalf("tier changed: %v", got)
	}
	got = apiTokenSuggestionsWithImportance("QueryCoupon", []string{"ListCouponAb", "ListCouponAc"}, 5, map[string]float64{"ListCouponAb": .1, "ListCouponAc": .9})
	if !reflect.DeepEqual(got, []string{"ListCouponAc", "ListCouponAb"}) {
		t.Fatalf("equal token quality: %v", got)
	}
	got = apiTokenSuggestionsWithImportance("QueryCoupons", []string{"ListCoupons", "ListCouponOptions"}, 5, map[string]float64{"ListCoupons": 0, "ListCouponOptions": 1})
	if len(got) != 2 || got[0] != "ListCoupons" {
		t.Fatalf("token quality lost: %v", got)
	}
}

func TestEqualAPIImportanceKeepsExistingFinalOrdering(t *testing.T) {
	names := []string{"ListCoupons", "ListCouponOptions"}
	cause := &InvalidApiError{Name: "QueryCoupons", product: &meta.Product{Code: "demo", ApiNames: names}}
	args := []string{"demo", "QueryCoupons"}
	before := normalizeAgentError(cause, args).(*cli.AgentError).Envelope()
	cause.ImportanceScores = map[string]float64{"ListCoupons": 0, "ListCouponOptions": 0}
	after := normalizeAgentError(cause, args).(*cli.AgentError).Envelope()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("equal scores changed output: %v -> %v", before, after)
	}
}

func TestAPIPrefixImportanceRanksBeforeLimitAndSurvivesJSON(t *testing.T) {
	names := []string{"DescribeAaaaa", "DescribeBbbbb", "DescribeCcccc", "DescribeDdddd", "DescribeEeeee", "DescribeFffff"}
	cause := &InvalidApiError{Name: "Describe", product: &meta.Product{Code: "demo", ApiNames: names}}
	args := []string{"demo", "Describe"}
	validator := func(RecoverySearchRequest) bool { return true }
	before := normalizeAgentErrorWithSearch(cause, args, validator).(*cli.AgentError).Envelope()
	cause.ImportanceScores = map[string]float64{"DescribeFffff": 1}
	after := normalizeAgentErrorWithSearch(cause, args, validator).(*cli.AgentError).Envelope()
	want := []string{"DescribeFffff", "DescribeAaaaa", "DescribeBbbbb", "DescribeCcccc", "DescribeDdddd"}
	if !reflect.DeepEqual(after.DidYouMean, want) {
		t.Fatalf("final order: %v", after.DidYouMean)
	}
	if after.Message != before.Message || after.Recovery != before.Recovery {
		t.Fatal("recovery or message changed")
	}
	text := cause.GetSuggestions()
	if len(text) != 6 || text[0] != "DescribeFffff" {
		t.Fatalf("text order/overflow: %v", text)
	}
	cause.ImportanceScores = nil
	missing := normalizeAgentErrorWithSearch(cause, args, validator).(*cli.AgentError).Envelope()
	if !reflect.DeepEqual(missing, before) {
		t.Fatalf("no-score output changed: %v", missing)
	}
	cause.ImportanceScores = map[string]float64{"DescribeFffff": math.NaN()}
	invalid := normalizeAgentErrorWithSearch(cause, args, validator).(*cli.AgentError).Envelope()
	if !reflect.DeepEqual(invalid, before) {
		t.Fatal("invalid scores changed output")
	}
}

func TestParameterImportanceAliasesAndRecoveryStayCompatible(t *testing.T) {
	cause := &InvalidParameterError{Name: "UploadA", ProductCode: "demo", ApiName: "Demo", ParameterNames: []string{"UploadAa", "UploadAb"}}
	before := normalizeAgentError(cause, []string{"demo", "Demo"}).(*cli.AgentError).Envelope()
	cause.ImportanceScores = map[string]float64{"UploadAa": .1, "UploadAb": .9}
	after := normalizeAgentError(cause, []string{"demo", "Demo"}).(*cli.AgentError).Envelope()
	if !reflect.DeepEqual(after.DidYouMean, []string{"--UploadAb", "--UploadAa"}) {
		t.Fatal(after.DidYouMean)
	}
	if after.Recovery != before.Recovery || after.Message != before.Message {
		t.Fatal("error protocol changed")
	}
	unknown := &argparser.UnknownFlagError{Flag: "upload-a", Known: []string{"upload-aa", "upload-ab"}, ImportanceScores: map[string]float64{"upload-aa": .1, "upload-ab": .9}}
	envelope := normalizeAgentError(unknown, []string{"demo", "demo"}).(*cli.AgentError).Envelope()
	if !reflect.DeepEqual(envelope.DidYouMean, []string{"--upload-ab", "--upload-aa"}) {
		t.Fatal(envelope.DidYouMean)
	}
}

func TestImportanceFlowsFromVersionReaderThroughFinishCommandRun(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_CLI_AI_MODE", "1")
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	cmd := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
	config.AddFlags(cmd.Flags())
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	repo := canonicalmeta.NewRepository(fstest.MapFS{
		"canonical/demo/v1/version.json": &fstest.MapFile{Data: []byte(`{"version":"v1","apis":{"DescribeAaaaa":{"cmd_name":"describe-aaaaa","importance_score":0.1},"DescribeZzzzz":{"cmd_name":"describe-zzzzz","importance_score":0.9}}}`)},
	})
	c := &Commando{library: &Library{canonicalRepo: repo}, profile: config.Profile{Language: "en"}}
	err := &InvalidApiError{Name: "Describe", product: &meta.Product{Code: "demo", Version: "v1", ApiNames: []string{"DescribeAaaaa", "DescribeZzzzz"}}}
	result := c.finishCommandRun(ctx, []string{"demo", "Describe"}, err)
	agentErr, ok := result.(*cli.AgentError)
	if !ok {
		t.Fatalf("not structured: %T %v", result, result)
	}
	if !reflect.DeepEqual(agentErr.Envelope().DidYouMean, []string{"DescribeZzzzz", "DescribeAaaaa"}) {
		t.Fatal(agentErr.Envelope())
	}
	if agentErr.Envelope().SchemaVersion != "v1" {
		t.Fatal("schema version lost")
	}
	if _, err := repo.GetVersionIndex("demo", "v2"); err == nil {
		t.Fatal("borrowed another version")
	}
}

func TestHelpImportanceIsInternalAndUnknownParameterCanUseIt(t *testing.T) {
	var api canonicalmeta.API
	if err := json.Unmarshal([]byte(`{"name":"Demo","parameters":[{"name":"upload_aa","raw_name":"UploadAa","options":["--upload-aa"],"location":"query","type":"string","importance_score":0.1},{"name":"upload_ab","raw_name":"UploadAb","options":["--upload-ab"],"location":"query","type":"string","importance_score":0.9}]}`), &api); err != nil {
		t.Fatal(err)
	}
	params := []machineHelpParameter{projectCanonicalParameter(&api.Parameters[0]), projectCanonicalParameter(&api.Parameters[1])}
	encoded, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("importance")) {
		t.Fatalf("internal score exposed: %s", encoded)
	}
	for _, p := range params {
		if p.importanceScore == nil {
			t.Fatal("internal score lost")
		}
	}
}
