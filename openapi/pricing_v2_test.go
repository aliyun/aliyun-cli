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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	pricingruntime "github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/stretchr/testify/require"
)

func TestPricingV2ListPaginationAndHost(t *testing.T) {
	ctx, _, profile := quoteServer(t, 200, `{}`)
	profile.Mode = config.StsToken
	profile.StsToken = "fixture-token"
	profile.Endpoint = "target.invalid"
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := calls.Add(1)
		if r.Method != "GET" || r.URL.Path != "/api/v1/price/supported-apis" || r.Host != "pricing.test" {
			t.Errorf("unexpected request: %s %s host=%s", r.Method, r.URL, r.Host)
		}
		if r.Header.Get("x-acs-version") != "2022-08-30" || r.Header.Get("x-acs-action") != "ListSupportedPricingApis" || r.Header.Get("x-acs-security-token") != "fixture-token" {
			t.Error("missing quote-service identity or STS credential")
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "acs test-ak:") {
			t.Error("missing ROA signature")
		}
		if page == 1 {
			if r.URL.RawQuery != "" {
				t.Errorf("first-page query=%s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"supportedApis":[{"popCode":"Ecs","apiName":"RunInstances","popVersion":"2014-05-26"}],"nextToken":"a+b/=","requestId":"r1"}`)
		} else {
			if r.URL.Query().Get("nextToken") != "a+b/=" {
				t.Errorf("pagination token changed: %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"supportedApis":[{"popCode":"Rds","apiName":"CreateDBInstance","popVersion":"2014-08-15"}],"nextToken":"","requestId":"r2"}`)
		}
	}))
	defer server.Close()
	t.Setenv(estimateCostEndpointEnv, server.Listener.Addr().String())
	t.Setenv(estimateCostHostEnv, "pricing.test")
	out, err := invokeListSupportedPricingApis(ctx, profile)
	require.NoError(t, err)
	require.EqualValues(t, 2, calls.Load())
	require.JSONEq(t, `{"supportedApis":[{"popCode":"Ecs","apiName":"RunInstances","popVersion":"2014-05-26"},{"popCode":"Rds","apiName":"CreateDBInstance","popVersion":"2014-08-15"}],"nextToken":"","requestId":"r2"}`, out)
}

func TestPricingV2ListSecondPageFailure(t *testing.T) {
	ctx, _, profile := quoteServer(t, 200, `{}`)
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			_, _ = io.WriteString(w, `{"supportedApis":[{"apiName":"Run"}],"nextToken":"next"}`)
			return
		}
		w.WriteHeader(403)
		_, _ = io.WriteString(w, `{"Code":"Forbidden","Message":"second page denied","RequestId":"r2"}`)
	}))
	defer server.Close()
	t.Setenv(estimateCostEndpointEnv, server.Listener.Addr().String())
	out, err := invokeListSupportedPricingApis(ctx, profile)
	require.ErrorContains(t, err, "Forbidden")
	require.ErrorContains(t, err, "r2")
	require.Empty(t, out, "partial pages must not be reported as a complete result")
	require.EqualValues(t, 2, calls.Load())
}

func TestPricingV2EntryPointsNeverInvokeTarget(t *testing.T) {
	for _, route := range []string{"RPC", "ROA", "OpenAPI", "no metadata", "OSS", "ossutil"} {
		t.Run(route, func(t *testing.T) {
			ctx, out, profile := quoteServer(t, 200, `{}`)
			var quotes, targets atomic.Int32
			target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targets.Add(1); w.WriteHeader(500) }))
			defer target.Close()
			profile.Endpoint = target.Listener.Addr().String()
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				quotes.Add(1)
				if r.URL.Path != "/api/v1/price/quote" {
					t.Errorf("wrong quote path %s", r.URL)
				}
				var payload pricingruntime.PricingRequest
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
					return
				}
				if payload.ApiName == "" || payload.PopVersion == "" || payload.Parameters["RegionId"] != "cn-hangzhou" {
					t.Errorf("incomplete quote: %+v", payload)
				}
				_, _ = io.WriteString(w, `{"success":true,"code":"Success","message":"ok","requestId":"r","price":{"success":true,"totalAmount":1}}`)
			}))
			defer server.Close()
			t.Setenv(estimateCostEndpointEnv, server.Listener.Addr().String())
			originalLegacy, originalOpenapi := hookdo, hookHttpContextCall
			defer func() { hookdo = originalLegacy; hookHttpContextCall = originalOpenapi }()
			hookdo = func(fn func() (*responses.CommonResponse, error)) func() (*responses.CommonResponse, error) {
				return func() (*responses.CommonResponse, error) { targets.Add(1); return nil, nil }
			}
			hookHttpContextCall = func(fn func() error) func() error { return func() error { targets.Add(1); return nil } }
			EstimateCostFlag(ctx.Flags()).SetAssigned(true)
			command := NewCommando(out, *profile)
			var err error
			switch route {
			case "RPC":
				err = command.processInvoke(ctx, "ecs", "DescribeRegions", "")
			case "ROA":
				err = command.processInvoke(ctx, "cs", "GET", "/clusters")
			case "OpenAPI":
				err = command.processApiInvoke(ctx, &meta.Product{Code: "sls", Version: "2020-12-30"}, &canonicalmeta.API{Name: "CreateLogStore"}, "POST", "/logstores")
			case "no metadata":
				BodyFlag(ctx.Flags()).SetAssigned(true)
				BodyFlag(ctx.Flags()).SetValue(`{"Nested":{"n":9007199254740993}}`)
				err = command.processEstimateCostByTriple(ctx, &meta.Product{Code: "tablestore"}, "2020-12-26", "CreateVCUInstance")
			case "OSS", "ossutil":
				configuration := config.Configuration{CurrentProfile: profile.Name, Profiles: []config.Profile{*profile}}
				data, e := json.Marshal(configuration)
				require.NoError(t, e)
				path := filepath.Join(t.TempDir(), "config.json")
				require.NoError(t, os.WriteFile(path, data, 0600))
				config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
				config.ConfigurePathFlag(ctx.Flags()).SetValue(path)
				if route == "OSS" {
					err = command.processOssEstimateCost(ctx, "CreateBucket")
				} else {
					err = EstimateOssCost(ctx, "CreateBucket", nil, nil)
				}
			}
			require.NoError(t, err)
			require.EqualValues(t, 1, quotes.Load())
			require.Zero(t, targets.Load(), "ordinary cost estimation must never call the target API")
			require.Contains(t, out.String(), "totalAmount")
		})
	}
}

func TestPricingV2ProfileRetryAndTimeout(t *testing.T) {
	t.Run("retry count", func(t *testing.T) {
		ctx, _, profile := quoteServer(t, 200, `{}`)
		var calls atomic.Int32
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) < 3 {
				w.WriteHeader(500)
				_, _ = io.WriteString(w, `{"Code":"InternalError","Message":"retry","RequestId":"r"}`)
				return
			}
			_, _ = io.WriteString(w, `{"price":{"success":true}}`)
		}))
		defer server.Close()
		t.Setenv(estimateCostEndpointEnv, server.Listener.Addr().String())
		profile.RetryCount = 2
		_, err := invokeEstimateCost(ctx, profile, "ecs", "2014-05-26", "RunInstances", nil)
		require.NoError(t, err)
		require.EqualValues(t, 3, calls.Load())
	})
	t.Run("read timeout seconds", func(t *testing.T) {
		ctx, _, profile := quoteServer(t, 200, `{}`)
		var calls atomic.Int32
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			time.Sleep(1200 * time.Millisecond)
			_, _ = io.WriteString(w, `{"price":{"success":true}}`)
		}))
		defer server.Close()
		t.Setenv(estimateCostEndpointEnv, server.Listener.Addr().String())
		profile.ReadTimeout = 1
		profile.ConnectTimeout = 1
		start := time.Now()
		_, err := invokeEstimateCost(ctx, profile, "ecs", "2014-05-26", "RunInstances", nil)
		require.Error(t, err)
		require.Greater(t, time.Since(start), 500*time.Millisecond, "seconds must not become milliseconds")
		require.EqualValues(t, 1, calls.Load())
	})
}

func TestPricingV2FailureSurvivesAllOutputModes(t *testing.T) {
	for _, body := range []string{
		`{"success":false,"code":"PricingNotSupported","message":"unsupported","requestId":"r"}`,
		`{"price":{"success":false,"errorCode":"BadParameter"},"requestId":"r"}`,
	} {
		for _, mode := range []string{"full", "quiet", "query", "table"} {
			t.Run(mode+body, func(t *testing.T) {
				ctx, out, profile := quoteServer(t, 200, body)
				switch mode {
				case "quiet":
					QuietFlag(ctx.Flags()).SetAssigned(true)
				case "query":
					QueryFlag(ctx.Flags()).SetAssigned(true)
					QueryFlag(ctx.Flags()).SetValue("requestId")
				case "table":
					OutputFlag(ctx.Flags()).SetAssigned(true)
					for i := range OutputFlag(ctx.Flags()).Fields {
						field := &OutputFlag(ctx.Flags()).Fields[i]
						if field.Key == "cols" {
							field.SetAssigned(true)
							field.SetValue("requestId")
						}
					}
				}
				raw, err := invokeEstimateCost(ctx, profile, "ecs", "2014-05-26", "RunInstances", nil)
				require.NoError(t, err)
				err = finishEstimateCost(ctx, raw)
				var business *pricingruntime.PricingBusinessError
				require.ErrorAs(t, err, &business)
				require.Equal(t, 3, business.ExitCode())
				if mode == "quiet" {
					require.Empty(t, out.String())
				} else {
					require.Contains(t, out.String(), "r")
				}
				var stderr bytes.Buffer
				require.NoError(t, business.RenderError(&stderr))
				require.Empty(t, stderr.String())
			})
		}
	}
}
