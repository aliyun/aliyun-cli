package openapi

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/require"
)

func TestPricingListCommandFetchesAllPagesAndPropagatesFailures(t *testing.T) {
	for _, mode := range []string{"success", "filter", "invalid query", "server failure"} {
		t.Run(mode, func(t *testing.T) {
			requests := make(chan string, 2)
			servePricingFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, estimateCostListPath, r.URL.Path)
				requests <- r.URL.Query().Get("nextToken")
				w.Header().Set("Content-Type", "application/json")
				if mode == "server failure" {
					w.WriteHeader(http.StatusBadRequest)
					io.WriteString(w, `{"Code":"BadRequest","Message":"pricing unavailable"}`)
				} else if r.URL.Query().Get("nextToken") == "" {
					io.WriteString(w, `{"supportedApis":[{"popCode":"Ecs","popVersion":"2014-05-26","apiName":"RunInstances"}],"nextToken":"page-2"}`)
				} else {
					io.WriteString(w, `{"supportedApis":[{"popCode":"Oss","popVersion":"2019-05-17","apiName":"CreateBucket"}]}`)
				}
			}))
			t.Setenv(estimateCostHostEnv, "pricing-host.test")
			var output bytes.Buffer
			ctx := cli.NewCommandContext(&output, io.Discard)
			cmd := NewListSupportedPricingApisCommand()
			config.AddFlags(cmd.Flags())
			ctx.EnterCommand(cmd)
			writeOssTestConfig(t, ctx)
			config.SkipSecureVerify(ctx.Flags()).SetAssigned(true)
			if mode == "filter" {
				f := ctx.Flags().Get(PricingProductFlagName)
				f.SetAssigned(true)
				f.SetValue("Oss")
			}
			if mode == "invalid query" {
				QueryFlag(ctx.Flags()).SetAssigned(true)
				QueryFlag(ctx.Flags()).SetValue("[")
			}
			err := cmd.Run(ctx, nil)
			if mode == "invalid query" || mode == "server failure" {
				require.Error(t, err)
				require.Empty(t, output.String())
			} else {
				require.NoError(t, err)
				require.Contains(t, output.String(), "CreateBucket")
				if mode == "filter" {
					require.NotContains(t, output.String(), "RunInstances")
				} else {
					require.Contains(t, output.String(), "RunInstances")
				}
			}
			require.Equal(t, "", <-requests)
			if mode != "server failure" {
				require.Equal(t, "page-2", <-requests)
			}
		})
	}
}

func TestEstimateCostFiltersAndBusinessFailureRemainTerminal(t *testing.T) {
	_, ctx, output, _ := newCanonicalHelpTestContext(t)
	OutputFlag(ctx.Flags()).SetAssigned(true)
	OutputFlag(ctx.Flags()).Fields[0].SetAssigned(true)
	OutputFlag(ctx.Flags()).Fields[0].SetValue("Name:bad")
	OutputFlag(ctx.Flags()).Fields[1].SetAssigned(true)
	OutputFlag(ctx.Flags()).Fields[1].SetValue("Items")
	require.ErrorContains(t, printEstimateCostResult(ctx, `{"Items":[["value"]]}`), "string:number")
	require.Empty(t, output.String())
	filter := &TableOutputFilter{ctx: ctx}
	for _, column := range []string{"name:0:1", "name:x", "name:99999999999999999999999999"} {
		_, err := filter.FormatTable("@", []string{column}, []any{[]any{"value"}})
		require.ErrorContains(t, err, "string:number", fmt.Sprint(column))
	}
	require.False(t, isNumber("1a"))
}
