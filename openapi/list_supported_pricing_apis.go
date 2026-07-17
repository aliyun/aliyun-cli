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
	"encoding/json"
	"fmt"
	"os"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// Cross-product enumeration: asks CloudControl ListSupportedPricingApis
// (GET /api/v1/price/supported-apis) for every OpenAPI triple that has a
// pricing mapping registered, so Agents and humans can discover what supports
// --estimate-cost without probing each API.
//
// Exposed as a top-level `aliyun list-supported-pricing-apis` subcommand
// (not a flag): the operation is product-agnostic, takes no arguments, and
// is a standalone action — same shape as `aliyun configure`, `aliyun upgrade`
// etc. — so a subcommand reads more naturally than a global flag would.
const (
	estimateCostListPath = "/api/v1/price/supported-apis"
)

// NewListSupportedPricingApisCommand returns the top-level subcommand
// registration. Wired up from main/main.go alongside the other standalone
// subcommands (configure, plugin, upgrade, ...).
func NewListSupportedPricingApisCommand() *cli.Command {
	return &cli.Command{
		Name: "list-supported-pricing-apis",
		Short: i18n.T(
			"List every OpenAPI that supports --estimate-cost. Output is JSON.",
			"列出所有支持 --estimate-cost 的 OpenAPI 三元组，输出 JSON",
		),
		Run: func(ctx *cli.Context, args []string) error {
			profile, err := config.LoadProfileWithContext(ctx)
			if err != nil {
				return cli.NewErrorWithTip(err,
					"list-supported-pricing-apis needs a configured profile; run `aliyun configure` first")
			}
			out, err := invokeListSupportedPricingApis(ctx, &profile)
			if err != nil {
				return err
			}
			return printEstimateCostResult(ctx, out)
		},
	}
}

// pagedListResponse is the slice of the upstream response we care about for
// merging across pages. The full response carries more fields (maxResults,
// etc.) but those are per-page metadata that lose meaning after aggregation.
type pagedListResponse struct {
	SupportedApis []json.RawMessage `json:"supportedApis"`
	NextToken     string            `json:"nextToken"`
	RequestId     string            `json:"requestId"`
}

// invokeListSupportedPricingApis builds the authenticated single-page fetcher
// and hands it to paginateList. Pagination + JSON merge is fetcher-agnostic
// (see paginateList) so tests can swap in a stub fetcher without standing up
// an https server.
func invokeListSupportedPricingApis(ctx *cli.Context, profile *config.Profile) (string, error) {
	client, err := GetClient(profile, ctx)
	if err != nil {
		return "", fmt.Errorf("init list-supported-pricing-apis client failed: %s", err)
	}
	return paginateList(func(nextToken string) ([]byte, error) {
		request := requests.NewCommonRequest()
		request.Product = estimateCostProductCode
		request.Version = estimateCostApiVersion
		request.Method = "GET"
		request.PathPattern = estimateCostListPath
		request.Domain = estimateCostEndpoint()
		request.Scheme = "https"
		if host := os.Getenv(estimateCostHostEnv); host != "" {
			request.Headers["Host"] = host
		}
		// Pagination is via query string; the upstream contract uses camelCase
		// nextToken/maxResults (ROA style). We don't pass maxResults — the
		// server default (100) is fine for an enumeration that runs once.
		if nextToken != "" {
			request.QueryParams["nextToken"] = nextToken
		}
		request.SetContentType("application/json")

		resp, err := client.ProcessCommonRequest(request)
		if err != nil {
			return nil, err
		}
		return []byte(resp.GetHttpContentString()), nil
	})
}

// paginateList walks every page produced by getPage(nextToken) and emits a
// single merged JSON document. Users expect `list-*` commands to return the
// full set, not the first page; pagination is a backend implementation detail
// (ccapi caps each page at 100), not something to push onto the user.
//
// Loop terminates when nextToken comes back empty. The aggregate response keeps
// the same top-level shape as a single page (supportedApis + nextToken +
// requestId) so existing JMESPath filters via --cli-query don't have to special-
// case the merged form; nextToken is fixed to "" since the merged set is by
// definition complete.
//
// Decoupled from the SDK call (getPage is a stub-friendly fetcher) so the
// looping/merge logic can be tested without setting up an https server.
func paginateList(getPage func(nextToken string) ([]byte, error)) (string, error) {
	var aggregated []json.RawMessage
	var lastRequestId string
	nextToken := ""

	for {
		raw, err := getPage(nextToken)
		if err != nil {
			return "", err
		}

		var page pagedListResponse
		if err := json.Unmarshal(raw, &page); err != nil {
			// Upstream returned non-JSON or unexpected shape; surface raw to
			// preserve diagnosability rather than swallowing the body.
			return "", fmt.Errorf("list-supported-pricing-apis: failed to parse page: %v\nraw response:\n%s", err, string(raw))
		}

		aggregated = append(aggregated, page.SupportedApis...)
		lastRequestId = page.RequestId
		if page.NextToken == "" {
			break
		}
		nextToken = page.NextToken
	}

	merged := map[string]interface{}{
		"supportedApis": aggregated,
		"nextToken":     "", // by definition: merged result IS the full set
		"requestId":     lastRequestId,
	}
	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
