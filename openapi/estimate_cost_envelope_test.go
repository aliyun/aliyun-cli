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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	pricingruntime "github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/stretchr/testify/require"
)

func TestEstimateCostServerEnvelopes(t *testing.T) {
	for _, test := range []struct {
		name, body   string
		status, exit int
	}{
		{"quoted", `{"success":true,"code":"Success","message":"server quoted","requestId":"req-priced","price":{"success":true,"totalAmount":0.084,"currency":"CNY"},"futureField":42}`, 200, 0},
		{"free", `{"success":true,"code":"PricingNotRequired","message":"server free","requestId":"req-free","price":null}`, 200, 0},
		{"unsupported", `{"success":false,"code":"PricingNotSupported","message":"server unsupported","requestId":"req-unsupported","price":null}`, 200, 3},
		{"free without price", `{"success":true,"code":"PricingNotRequired","message":"server free","requestId":"req-free"}`, 200, 0},
		{"unsupported without price", `{"success":false,"code":"PricingNotSupported","message":"server unsupported","requestId":"req-unsupported"}`, 200, 3},
		{"upstream failure", `{"success":false,"code":"UpstreamFailure","message":"server failure","requestId":"req-failure","price":{"success":true}}`, 200, 3},
		{"legacy free", `{"Code":"PricingNotRequired","Message":"legacy free","RequestId":"legacy-free"}`, 400, 0},
		{"legacy unsupported", `{"Code":"PricingNotSupported","Message":"legacy unsupported","RequestId":"legacy-unsupported"}`, 400, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, output, profile := quoteServer(t, test.status, test.body)
			raw, err := invokeEstimateCost(ctx, profile, "Ecs", "2014-05-26", "DescribeInstances", nil)
			require.NoError(t, err)
			require.JSONEq(t, test.body, raw)
			err = finishEstimateCost(ctx, raw)
			if test.exit == 0 {
				require.NoError(t, err)
			} else {
				var result *pricingruntime.PricingBusinessError
				require.ErrorAs(t, err, &result)
				require.Equal(t, test.exit, result.ExitCode())
				var stderr bytes.Buffer
				require.NoError(t, result.RenderError(&stderr))
				require.Empty(t, stderr.String())
			}
			require.JSONEq(t, test.body, output.String())
		})
	}
}

func TestEstimateCostUsesDefaultEnvelopeAndServerLanguage(t *testing.T) {
	ctx, _, profile := quoteServer(t, 200, `{}`)
	profile.Language = "zh"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/price/quote", r.URL.Path)
		var request map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.NotContains(t, request, "responseVersion")
		require.Equal(t, "zh", request["language"])
		require.Equal(t, "p", request["parameters"].(map[string]interface{})["PlanId"])
		_, _ = w.Write([]byte(`{"success":true,"code":"PricingNotRequired","message":"服务端文案","requestId":"r","price":null}`))
	}))
	defer server.Close()
	t.Setenv(estimateCostEndpointEnv, server.Listener.Addr().String())
	_, err := invokeEstimateCost(ctx, profile, "hbr", "2017-09-08", "EnableBackupPlan", map[string]interface{}{"PlanId": "p"})
	require.NoError(t, err)
}

func TestEstimateCostFailureSurvivesOutputFiltering(t *testing.T) {
	const body = `{"success":false,"code":"PricingNotSupported","message":"unsupported","requestId":"r"}`
	for _, quiet := range []bool{false, true} {
		ctx, output, _ := quoteServer(t, 200, body)
		QueryFlag(ctx.Flags()).SetAssigned(true)
		QueryFlag(ctx.Flags()).SetValue("requestId")
		if quiet {
			QuietFlag(ctx.Flags()).SetAssigned(true)
		}
		err := finishEstimateCost(ctx, body)
		var rendered cli.StructuredError
		require.ErrorAs(t, err, &rendered)
		require.Equal(t, 3, rendered.ExitCode())
		if quiet {
			require.Empty(t, output.String())
		} else {
			require.JSONEq(t, `"r"`, output.String())
		}
	}
}

func TestEstimateCostDoesNotTurnTransportErrorsIntoOutcomes(t *testing.T) {
	ctx, _, profile := quoteServer(t, 403, `{"Code":"Forbidden","Message":"denied","RequestId":"denied-r"}`)
	_, err := invokeEstimateCost(ctx, profile, "Ecs", "2014-05-26", "DescribeInstances", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Forbidden")
}
