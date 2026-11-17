package openapi

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/require"
)

func TestLegacyDryRunResolvesRegionIdAndRejectsUnknownEndpointLocally(t *testing.T) {
	for _, source := range []string{"registered RegionId", "API RegionId"} {
		t.Run(source, func(t *testing.T) {
			_, ctx, _, _ := newCanonicalHelpTestContext(t)
			config.AddFlags(ctx.Flags())
			writeOssTestConfig(t, ctx)
			ctx.SetUnknownFlags(cli.NewFlagSet())
			CliDryRunFlag(ctx.Flags()).SetAssigned(true)
			flag := config.RegionIdFlag(ctx.Flags())
			if source == "API RegionId" {
				var err error
				flag, err = ctx.UnknownFlags().AddByName("RegionId")
				require.NoError(t, err)
			}
			flag.SetAssigned(true)
			flag.SetValue("cn-shanghai")
			invoker := NewBasicInvoker(&config.Profile{})
			product := &meta.Product{Code: "demo", Version: "v1", GlobalEndpoint: "fixture.invalid"}
			require.NoError(t, invoker.Init(ctx, product))
			require.Equal(t, "cn-shanghai", invoker.request.RegionId)
			require.Equal(t, "fixture.invalid", invoker.request.Domain)
			require.Nil(t, invoker.client)
			product.GlobalEndpoint = ""
			err := invoker.Init(ctx, product)
			require.ErrorContains(t, err, "unknown endpoint for demo/cn-shanghai")
			var endpointError *meta.InvalidEndpointError
			require.ErrorAs(t, err, &endpointError)
			require.Nil(t, invoker.client)
		})
	}
}
