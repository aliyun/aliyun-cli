package openapi

import (
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/require"
)

func TestLegacyBackendTimeoutPrecedence(t *testing.T) {
	for _, tc := range []struct {
		product, action string
		metadata        int64
		user            int
		sse             bool
		want            time.Duration
	}{
		{"rds", "CreateDBInstance", 57000, 0, false, 57 * time.Second},
		{"rds", "CreateDBInstance", 57000, 2, false, 0},
		{"ecs", "CreateInstance", 62000, 0, false, 62 * time.Second},
		{"ecs", "DescribeInstances", 0, 0, false, 0},
		{"ecs", "CreateInstance", 62000, 2, false, 0},
		{"ecs", "CreateInstance", 62000, 120, false, 0},
		{"ecs", "CreateInstance", 0, 0, false, 0},
		{"rds", "CreateDBInstance", 0, 0, false, 0},
		{"rds", "CreateDBInstance", -1, 0, false, 0},
		{"rds", "CreateDBInstance", 57000, 0, true, 0},
	} {
		request := requests.NewCommonRequest()
		request.Product = tc.product
		invoker := &BasicInvoker{request: request, profile: &config.Profile{ReadTimeout: tc.user}}
		invoker.applyMetadataTimeout(&canonicalmeta.API{Name: tc.action, TimeoutMS: tc.metadata,
			Operation: &canonicalmeta.Operation{IsSSE: tc.sse}})
		if got := request.GetReadTimeout(); got != tc.want {
			t.Errorf("%s/%s metadata=%d user=%d sse=%v: got %v, want %v", tc.product, tc.action, tc.metadata, tc.user, tc.sse, got, tc.want)
		}
	}
}

func TestBackendTimeoutPreservesRequestTimeout(t *testing.T) {
	request := requests.NewCommonRequest()
	request.SetReadTimeout(3 * time.Second)
	invoker := &BasicInvoker{request: request}
	invoker.applyMetadataTimeout(&canonicalmeta.API{TimeoutMS: 57000})
	if got := request.GetReadTimeout(); got != 3*time.Second {
		t.Fatalf("request timeout = %v, want 3s", got)
	}
}

func TestBackendTimeoutDoesNotUseAnotherAPIVersion(t *testing.T) {
	request := requests.NewCommonRequest()
	request.Product, request.Version = "rds", "2025-01-01"
	invoker := &BasicInvoker{request: request, profile: &config.Profile{}}
	invoker.applyMetadataTimeout(&canonicalmeta.API{Name: "CreateDBInstance", TimeoutMS: 57000,
		Operation: &canonicalmeta.Operation{APIVersion: "2014-08-15"}})
	if request.GetReadTimeout() != 0 {
		t.Fatal("used a budget from another API version")
	}
}

func TestMetadataTimeoutThroughLegacyRoutes(t *testing.T) {
	for _, route := range []struct {
		name, style, version string
		force                bool
		want                 time.Duration
	}{
		{"rpc", "rpc", "", false, 57 * time.Second},
		{"force rpc", "rpc", "", true, 57 * time.Second},
		{"selected version", "rpc", "2026-02-01", false, 12001 * time.Millisecond},
		{"force selected version", "rpc", "2026-02-01", true, 12001 * time.Millisecond},
		{"version without metadata", "rpc", "2026-03-01", false, 0},
		{"roa", "restful", "", false, 57 * time.Second},
	} {
		for _, user := range []struct {
			name    string
			profile int
			flag    string
			want    time.Duration
		}{
			{"metadata", 0, "", 0},
			{"profile", 2, "", 2 * time.Second},
			{"flag overrides profile", 2, "120", 120 * time.Second},
		} {
			t.Run(route.name+"/"+user.name, func(t *testing.T) {
				c, ctx, _, _ := newCanonicalHelpTestContext(t)
				config.AddFlags(ctx.Flags())
				writeOssTestConfig(t, ctx)
				ctx.SetUnknownFlags(cli.NewFlagSet())
				ctx.Command().Name = "CreateReport"
				c.profile = config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou", ReadTimeout: user.profile}
				if user.flag != "" {
					config.ReadTimeoutFlag(ctx.Flags()).SetAssigned(true)
					config.ReadTimeoutFlag(ctx.Flags()).SetValue(user.flag)
				}
				c.profile.OverwriteWithFlags(ctx)
				ForceFlag(ctx.Flags()).SetAssigned(route.force)
				if route.version != "" {
					VersionFlag(ctx.Flags()).SetAssigned(true)
					VersionFlag(ctx.Flags()).SetValue(route.version)
					ctx.Flags().Add(&cli.Flag{Name: "style"})
					ctx.Flags().Get("style").SetAssigned(true)
					ctx.Flags().Get("style").SetValue(route.style)
				}
				product := meta.Product{Code: "demo", Version: "2026-01-01", ApiStyle: route.style, GlobalEndpoint: "fixture.invalid", ApiNames: []string{"CreateReport"}}
				repo, err := meta.MockLoadRepository([]meta.Product{product})
				require.NoError(t, err)
				c.library.builtinRepo = repo
				c.library.canonicalRepo = canonicalmeta.NewRepository(fstest.MapFS{
					"canonical/demo/2026-01-01/CreateReport.json": {Data: []byte(`{"name":"CreateReport","timeout_ms":57000,"operation":{"api_version":"2026-01-01","method":"POST","protocol":"HTTPS"}}`)},
					"canonical/demo/2026-02-01/CreateReport.json": {Data: []byte(`{"name":"CreateReport","timeout_ms":12001,"operation":{"api_version":"2026-02-01","method":"POST","protocol":"HTTPS"}}`)},
				})
				action, path := "CreateReport", ""
				if route.style == "restful" {
					action, path = "GET", "/reports"
				}
				invoker, err := c.createInvoker(ctx, product.Code, action, path)
				require.NoError(t, err)
				require.NoError(t, invoker.Prepare(ctx))
				if user.want > 0 {
					require.Zero(t, invoker.getRequest().GetReadTimeout(), "metadata must not override the SDK client configured by the user")
					require.Equal(t, user.want, invoker.getClient().GetReadTimeout())
				} else {
					require.Equal(t, route.want, invoker.getRequest().GetReadTimeout())
				}
			})
		}
	}
}

func TestMetadataTimeoutThroughOpenapiContext(t *testing.T) {
	for _, tc := range []struct {
		name     string
		metadata int64
		profile  int
		flag     string
		sse      bool
		dryRun   bool
		want     int
	}{
		{name: "final milliseconds", metadata: 12001, want: 12001},
		{name: "final cap", metadata: 62000, want: 62000},
		{name: "profile wins", metadata: 57000, profile: 2, want: 2000},
		{name: "flag wins without cap", metadata: 57000, profile: 2, flag: "120", want: 120000},
		{name: "missing field"},
		{name: "nonpositive field", metadata: -1},
		{name: "overflow", metadata: 1<<63 - 1},
		{name: "SSE default unchanged", metadata: 57000, sse: true},
		{name: "SSE user timeout preserved", metadata: 57000, sse: true, profile: 2, want: 2000},
		{name: "dry run without client", metadata: 57000, dryRun: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, ctx, _, _ := newCanonicalHelpTestContext(t)
			config.AddFlags(ctx.Flags())
			writeOssTestConfig(t, ctx)
			ctx.SetUnknownFlags(cli.NewFlagSet())
			c.profile = config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou", ReadTimeout: tc.profile}
			if tc.flag != "" {
				config.ReadTimeoutFlag(ctx.Flags()).SetAssigned(true)
				config.ReadTimeoutFlag(ctx.Flags()).SetValue(tc.flag)
			}
			c.profile.OverwriteWithFlags(ctx)
			CliDryRunFlag(ctx.Flags()).SetAssigned(tc.dryRun)
			field := ""
			if tc.metadata != 0 {
				field = `,"timeout_ms":` + strconv.FormatInt(tc.metadata, 10)
			}
			reader := canonicalmeta.NewReader(fstest.MapFS{
				"api.json": {Data: []byte(`{"name":"CreateReport","operation":{"method":"POST","protocol":"HTTPS","is_sse":` + strconv.FormatBool(tc.sse) + `}` + field + `}`)},
			})
			api, err := reader.ReadAPIFromPath("api.json")
			require.NoError(t, err)
			invoker, err := c.createHttpContext(ctx, &meta.Product{Code: "sls", Version: "2026-01-01", ApiStyle: "rpc"}, api, "", "")
			require.NoError(t, err)
			require.NoError(t, invoker.Prepare(ctx))
			client := invoker.(*OpenapiContext).openapiClient
			if tc.dryRun {
				require.Nil(t, client)
			} else if tc.want == 0 {
				require.Nil(t, client.ReadTimeout, "leave the SDK default unset")
			} else {
				require.Equal(t, tc.want, tea.IntValue(client.ReadTimeout))
			}
		})
	}
}
