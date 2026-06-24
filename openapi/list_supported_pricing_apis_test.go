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
	"bytes"
	"encoding/json"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/assert"
)

func TestNewListSupportedPricingApisCommand(t *testing.T) {
	// Top-level subcommand metadata — the Run hook itself touches credentials
	// + network so we just lock the command shape here. Run-time behavior is
	// covered by TestInvokeListSupportedPricingApis below.
	cmd := NewListSupportedPricingApisCommand()
	assert.Equal(t, "list-supported-pricing-apis", cmd.Name)
	assert.NotNil(t, cmd.Run)
	assert.NotEmpty(t, cmd.Short)
}

func TestInvokeListSupportedPricingApisTransport(t *testing.T) {
	// Endpoint pointed at an unresolvable host: the call must reach the
	// signed transport (proving the request shape is built correctly) and
	// fail on DNS, not silently succeed or short-circuit. Anchors the
	// pagination loop's first iteration and the surrounding request setup.
	t.Setenv(estimateCostEndpointEnv, "list.test.invalid")

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	profile := config.NewProfile("test-list")
	profile.Mode = "AK"
	profile.AccessKeyId = "test-ak"
	profile.AccessKeySecret = "test-secret"
	profile.RegionId = "cn-hangzhou"

	_, err := invokeListSupportedPricingApis(ctx, &profile)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "list.test.invalid")
}

func TestPagedListResponseUnmarshal(t *testing.T) {
	// The merge loop depends on this JSON shape being parsed correctly. Lock
	// it here so a backend rename of supportedApis/nextToken/requestId would
	// break this test before silently corrupting the merge.
	raw := []byte(`{
		"supportedApis": [
			{"popCode":"Ecs","popVersion":"2014-05-26","apiName":"RunInstances"},
			{"popCode":"Rds","popVersion":"2014-08-15","apiName":"RunRCInstances"}
		],
		"nextToken": "20",
		"requestId": "req-1"
	}`)
	var page pagedListResponse
	err := json.Unmarshal(raw, &page)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(page.SupportedApis))
	assert.Equal(t, "20", page.NextToken)
	assert.Equal(t, "req-1", page.RequestId)
}

func TestPagedListResponseUnmarshalEmptyTokenMeansLastPage(t *testing.T) {
	// Loop termination relies on NextToken == ""; verify zero-value semantics
	// against an upstream response that omits the field entirely.
	raw := []byte(`{
		"supportedApis": [{"popCode":"Ecs","popVersion":"2014-05-26","apiName":"RunInstances"}],
		"requestId": "req-final"
	}`)
	var page pagedListResponse
	err := json.Unmarshal(raw, &page)
	assert.Nil(t, err)
	assert.Equal(t, "", page.NextToken)
	assert.Equal(t, "req-final", page.RequestId)
}

func TestPaginateListSinglePage(t *testing.T) {
	// One page, nextToken empty → loop exits after first fetch. Sanity check
	// for the trivial case and that requestId from the only page flows
	// through to the merged document.
	calls := 0
	getPage := func(nextToken string) ([]byte, error) {
		calls++
		assert.Equal(t, "", nextToken, "first fetch should send empty token")
		return []byte(`{
			"supportedApis":[{"popCode":"Ecs","popVersion":"2014-05-26","apiName":"RunInstances"}],
			"nextToken":"",
			"requestId":"req-single"
		}`), nil
	}
	out, err := paginateList(getPage)
	assert.Nil(t, err)
	assert.Equal(t, 1, calls)
	assert.Contains(t, out, "RunInstances")
	assert.Contains(t, out, `"requestId": "req-single"`)
	assert.Contains(t, out, `"nextToken": ""`)
}

func TestPaginateListMultiPageMergesAcrossPages(t *testing.T) {
	// Three pages → three fetcher calls, supportedApis aggregated in order,
	// final merged response carries the LAST page's requestId and a forced-
	// empty nextToken (the merged result IS the complete set).
	pages := []string{
		`{"supportedApis":[{"popCode":"Ecs","apiName":"RunInstances"},{"popCode":"Ecs","apiName":"CreateImage"}],"nextToken":"20","requestId":"req-1"}`,
		`{"supportedApis":[{"popCode":"Rds","apiName":"RunRCInstances"}],"nextToken":"40","requestId":"req-2"}`,
		`{"supportedApis":[{"popCode":"Alb","apiName":"CreateLoadBalancer"}],"nextToken":"","requestId":"req-3-last"}`,
	}
	tokensSeen := []string{}
	idx := 0
	getPage := func(nextToken string) ([]byte, error) {
		tokensSeen = append(tokensSeen, nextToken)
		page := pages[idx]
		idx++
		return []byte(page), nil
	}
	out, err := paginateList(getPage)
	assert.Nil(t, err)
	// Loop walks first→second→third using the token chain from each page.
	assert.Equal(t, []string{"", "20", "40"}, tokensSeen)
	// Aggregated set: 2 + 1 + 1 = 4 entries, original order preserved.
	assert.Contains(t, out, "RunInstances")
	assert.Contains(t, out, "CreateImage")
	assert.Contains(t, out, "RunRCInstances")
	assert.Contains(t, out, "CreateLoadBalancer")
	// Merged doc keeps the LAST page's requestId — that's the audit anchor
	// users would want to grep against if pagination misbehaved.
	assert.Contains(t, out, `"requestId": "req-3-last"`)
	// nextToken forced to "" — the merged set is complete by construction;
	// echoing the intermediate token would falsely imply "more pages".
	assert.Contains(t, out, `"nextToken": ""`)
}

func TestPaginateListFetcherErrorPropagates(t *testing.T) {
	// Mid-loop fetcher failure must surface upward, not be swallowed (e.g.
	// network blip on page 2 of 5: better to fail loud than silently truncate
	// the listing to the pages already fetched).
	calls := 0
	getPage := func(nextToken string) ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte(`{"supportedApis":[{"popCode":"Ecs"}],"nextToken":"20","requestId":"req-1"}`), nil
		}
		return nil, assert.AnError
	}
	_, err := paginateList(getPage)
	assert.NotNil(t, err)
	assert.Equal(t, assert.AnError, err)
	assert.Equal(t, 2, calls)
}

func TestPaginateListBadJSONShowsRawBody(t *testing.T) {
	// Unparseable upstream body must keep the raw text in the error message
	// so debugging doesn't require turning on transport logging — common case
	// is an HTML error page from a misrouted proxy.
	getPage := func(nextToken string) ([]byte, error) {
		return []byte(`<html>oops 502</html>`), nil
	}
	_, err := paginateList(getPage)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to parse page")
	assert.Contains(t, err.Error(), "<html>oops 502</html>")
}

func TestNewListSupportedPricingApisCommandRunWithoutProfileFailsCleanly(t *testing.T) {
	// Profile loading fails (no config file, no env creds) → the command
	// should return a tip pointing at `aliyun configure`, not panic and not
	// silently swallow the error.
	t.Setenv(estimateCostEndpointEnv, "should-not-be-reached")
	t.Setenv("ALIBABACLOUD_PROFILE", "no-such-profile-zz")
	// explicitly clear env creds so LoadProfileWithContext can't fall through
	// to env-credential mode
	t.Setenv("ALIBABACLOUD_ACCESS_KEY_ID", "")
	t.Setenv("ALIBABACLOUD_ACCESS_KEY_SECRET", "")
	t.Setenv("ALIBABACLOUD_REGION_ID", "")

	cmd := NewListSupportedPricingApisCommand()
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	parent := &cli.Command{}
	parent.EnableUnknownFlag = true
	config.AddFlags(parent.Flags())
	ctx.EnterCommand(parent)

	err := cmd.Run(ctx, []string{})
	// Either profile load fails (clear tip) or transport fails ("should-not-
	// be-reached" host) — both prove the Run closure executed and reported.
	// Silent nil here would mean the command did nothing without telling the user.
	assert.NotNil(t, err)
}

func TestNewListSupportedPricingApisCommandRunReachesTransport(t *testing.T) {
	// The Run closure has to wire LoadProfileWithContext + invoke + print
	// together correctly — easy to introduce a typo there that silently
	// returns nil. Drive it end-to-end against an unresolvable host so the
	// closure executes and the failure surfaces at the transport layer
	// (proving we reached invokeListSupportedPricingApis), not at config
	// load.
	t.Setenv(estimateCostEndpointEnv, "list-cmd.test.invalid")
	t.Setenv("ALIBABACLOUD_ACCESS_KEY_ID", "test-ak")
	t.Setenv("ALIBABACLOUD_ACCESS_KEY_SECRET", "test-secret")
	t.Setenv("ALIBABACLOUD_REGION_ID", "cn-hangzhou")

	cmd := NewListSupportedPricingApisCommand()
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	parent := &cli.Command{}
	parent.EnableUnknownFlag = true
	config.AddFlags(parent.Flags())
	ctx.EnterCommand(parent)

	err := cmd.Run(ctx, []string{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "list-cmd.test.invalid")
}
