package openapi

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
)

func TestIsSensitiveHeader(t *testing.T) {
	assert.True(t, isSensitiveHeader("Authorization"))
	assert.True(t, isSensitiveHeader("authorization"))
	assert.True(t, isSensitiveHeader("AUTHORIZATION"))
	assert.True(t, isSensitiveHeader("x-acs-accesskey-id"))
	assert.True(t, isSensitiveHeader("X-Acs-Accesskey-Id"))
	assert.True(t, isSensitiveHeader("x-acs-security-token"))
	assert.True(t, isSensitiveHeader("x-acs-signature"))
	assert.False(t, isSensitiveHeader("Content-Type"))
	assert.False(t, isSensitiveHeader("x-acs-region-id"))
	assert.False(t, isSensitiveHeader("Host"))
}

func TestMaskValue(t *testing.T) {
	assert.Equal(t, "***", maskValue(""))
	assert.Equal(t, "***", maskValue("ab"))
	assert.Equal(t, "***", maskValue("abcd"))
	assert.Equal(t, "abcd***", maskValue("abcde"))
	assert.Equal(t, "Bear***", maskValue("Bearer token-value-here"))
}

func TestSanitizeHeaders(t *testing.T) {
	headers := map[string]string{
		"Authorization":        "Bearer sk-abcdefg12345",
		"Content-Type":         "application/json",
		"x-acs-accesskey-id":   "LTAI5tAbcDefGhi",
		"x-acs-security-token": "SecurityTokenValue123",
		"x-acs-signature":      "SignatureValue456",
		"x-acs-region-id":      "cn-hangzhou",
	}

	result := sanitizeHeaders(headers)
	assert.Equal(t, "Bear***", result["Authorization"])
	assert.Equal(t, "application/json", result["Content-Type"])
	assert.Equal(t, "LTAI***", result["x-acs-accesskey-id"])
	assert.Equal(t, "Secu***", result["x-acs-security-token"])
	assert.Equal(t, "Sign***", result["x-acs-signature"])
	assert.Equal(t, "cn-hangzhou", result["x-acs-region-id"])
}

func TestSanitizeHeaders_Empty(t *testing.T) {
	result := sanitizeHeaders(map[string]string{})
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestBuildActualPath(t *testing.T) {
	t.Run("no params", func(t *testing.T) {
		assert.Equal(t, "/2023-03-30/functions", buildActualPath("/2023-03-30/functions", nil))
	})

	t.Run("single param", func(t *testing.T) {
		params := map[string]string{"functionName": "my-func"}
		assert.Equal(t, "/2023-03-30/functions/my-func",
			buildActualPath("/2023-03-30/functions/[functionName]", params))
	})

	t.Run("multiple params", func(t *testing.T) {
		params := map[string]string{
			"functionName": "my-func",
			"aliasName":    "prod",
		}
		assert.Equal(t, "/2023-03-30/functions/my-func/aliases/prod",
			buildActualPath("/2023-03-30/functions/[functionName]/aliases/[aliasName]", params))
	})
}

func TestCopyMap(t *testing.T) {
	original := map[string]string{"a": "1", "b": "2"}
	copied := copyMap(original)
	assert.Equal(t, original, copied)

	copied["a"] = "modified"
	assert.Equal(t, "1", original["a"])
}

func TestMergeInto(t *testing.T) {
	t.Run("both nil", func(t *testing.T) {
		assert.Nil(t, mergeInto(nil, nil))
	})

	t.Run("dst nil src non-nil", func(t *testing.T) {
		src := map[string]string{"a": "1"}
		result := mergeInto(nil, src)
		assert.Equal(t, map[string]string{"a": "1"}, result)
	})

	t.Run("merge", func(t *testing.T) {
		dst := map[string]string{"a": "1"}
		src := map[string]string{"b": "2"}
		result := mergeInto(dst, src)
		assert.Equal(t, map[string]string{"a": "1", "b": "2"}, result)
	})
}

func TestBuildCliDryRunFromInvoker_RPC(t *testing.T) {
	req := requests.NewCommonRequest()
	req.Domain = "ecs.cn-hangzhou.aliyuncs.com"
	req.Method = "POST"
	req.ApiName = "DescribeInstances"
	req.Version = "2014-05-26"
	req.Product = "Ecs"
	req.Headers["Content-Type"] = "application/json"
	req.Headers["Authorization"] = "Bearer secret-token"
	req.QueryParams["RegionId"] = "cn-hangzhou"
	req.QueryParams["PageSize"] = "10"

	invoker := &ForceRpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: req,
		},
		method: "DescribeInstances",
	}

	out := buildCliDryRunFromInvoker(invoker)
	assert.Equal(t, "RPC", out.Style)
	assert.Equal(t, "ecs.cn-hangzhou.aliyuncs.com", out.Endpoint)
	assert.Equal(t, "POST", out.Method)
	assert.Equal(t, "DescribeInstances", out.Action)
	assert.Equal(t, "2014-05-26", out.Version)
	assert.Equal(t, "cn-hangzhou", out.Query["RegionId"])
	assert.Equal(t, "10", out.Query["PageSize"])
	assert.Equal(t, "application/json", out.Headers["Content-Type"])
	assert.Equal(t, "Bear***", out.Headers["Authorization"])
	assert.Empty(t, out.PathPattern)
	assert.Empty(t, out.Pathname)
	assert.Nil(t, out.PathParams)
}

func TestBuildCliDryRunFromInvoker_ROA(t *testing.T) {
	req := requests.NewCommonRequest()
	req.Domain = "cs.cn-hangzhou.aliyuncs.com"
	req.Method = "GET"
	req.PathPattern = "/clusters/[ClusterId]/nodepools"
	req.PathParams["ClusterId"] = "c12345"
	req.Version = "2015-12-15"
	req.Product = "CS"
	req.Headers["Content-Type"] = "application/json"
	req.QueryParams["pageSize"] = "20"

	invoker := &RestfulInvoker{
		BasicInvoker: &BasicInvoker{
			request: req,
		},
		method: "GET",
		path:   "/clusters/[ClusterId]/nodepools",
	}

	out := buildCliDryRunFromInvoker(invoker)
	assert.Equal(t, "ROA", out.Style)
	assert.Equal(t, "cs.cn-hangzhou.aliyuncs.com", out.Endpoint)
	assert.Equal(t, "GET", out.Method)
	assert.Equal(t, "/clusters/[ClusterId]/nodepools", out.PathPattern)
	assert.Equal(t, "/clusters/c12345/nodepools", out.Pathname)
	assert.Equal(t, "c12345", out.PathParams["ClusterId"])
	assert.Equal(t, "20", out.Query["pageSize"])
	assert.Empty(t, out.Action)
	assert.Empty(t, out.Version)
}

func TestBuildCliDryRunFromInvoker_WithBody(t *testing.T) {
	req := requests.NewCommonRequest()
	req.Domain = "ecs.cn-hangzhou.aliyuncs.com"
	req.Method = "POST"
	req.ApiName = "RunInstances"
	req.Version = "2014-05-26"
	req.Product = "Ecs"
	req.SetContent([]byte(`{"InstanceType":"ecs.g6.large"}`))

	invoker := &ForceRpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: req,
		},
		method: "RunInstances",
	}

	out := buildCliDryRunFromInvoker(invoker)
	assert.Equal(t, `{"InstanceType":"ecs.g6.large"}`, out.Body)
	assert.Equal(t, "raw", out.BodyFormat)
}

func TestBuildCliDryRunFromInvoker_WithFormParams(t *testing.T) {
	req := requests.NewCommonRequest()
	req.Domain = "ecs.cn-hangzhou.aliyuncs.com"
	req.Method = "POST"
	req.ApiName = "CreateInstance"
	req.Version = "2014-05-26"
	req.Product = "Ecs"
	req.FormParams["InstanceType"] = "ecs.g6.large"
	req.FormParams["RegionId"] = "cn-hangzhou"

	invoker := &ForceRpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: req,
		},
		method: "CreateInstance",
	}

	out := buildCliDryRunFromInvoker(invoker)
	assert.Equal(t, "form", out.BodyFormat)
	assert.Contains(t, out.Body, "InstanceType")
	assert.Contains(t, out.Body, "ecs.g6.large")
}

func TestBuildCliDryRunFromOpenapi(t *testing.T) {
	product := &meta.Product{
		Code:    "sls",
		Version: "2020-12-30",
	}
	api := &meta.Api{
		Name:    "GetLogStore",
		Product: product,
	}
	profile := &config.Profile{
		RegionId: "cn-hangzhou",
		Endpoint: "cn-hangzhou.log.aliyuncs.com",
	}

	oc := &OpenapiContext{
		HttpContext: &HttpContext{
			profile: profile,
			product: product,
			openapiRequest: &openapiutil.OpenApiRequest{
				Headers: map[string]*string{
					"Content-Type":         tea.String("application/json"),
					"x-acs-security-token": tea.String("token-value-here"),
				},
				Query: map[string]*string{
					"logstoreName": tea.String("my-store"),
				},
			},
			openapiParams: newOpenapiParams("GET", "/logstores/my-store", "GetLogStore", "2020-12-30"),
		},
		method: "GET",
		path:   "/logstores/[logstoreName]",
		api:    api,
	}

	out := buildCliDryRunFromOpenapi(oc)
	assert.Equal(t, "ROA", out.Style)
	assert.Equal(t, "cn-hangzhou.log.aliyuncs.com", out.Endpoint)
	assert.Equal(t, "GET", out.Method)
	assert.Equal(t, "/logstores/[logstoreName]", out.PathPattern)
	assert.Equal(t, "/logstores/my-store", out.Pathname)
	assert.Equal(t, "my-store", out.Query["logstoreName"])
	assert.Equal(t, "application/json", out.Headers["Content-Type"])
	assert.Equal(t, "toke***", out.Headers["x-acs-security-token"])
	assert.Equal(t, "GetLogStore", out.Action)
	assert.Equal(t, "2020-12-30", out.Version)
}

func TestBuildCliDryRunFromOpenapi_EndpointOverride(t *testing.T) {
	product := &meta.Product{Code: "sls", Version: "2020-12-30"}
	api := &meta.Api{Name: "ListProject", Product: product}
	profile := &config.Profile{RegionId: "cn-hangzhou"}

	oc := &OpenapiContext{
		HttpContext: &HttpContext{
			profile: profile,
			product: product,
			openapiRequest: &openapiutil.OpenApiRequest{
				Headers:          map[string]*string{},
				Query:            map[string]*string{},
				EndpointOverride: tea.String("custom.endpoint.com"),
			},
			openapiParams: newOpenapiParams("GET", "/", "ListProject", "2020-12-30"),
		},
		method: "GET",
		path:   "/",
		api:    api,
	}

	out := buildCliDryRunFromOpenapi(oc)
	assert.Equal(t, "custom.endpoint.com", out.Endpoint)
}

func TestBuildCliDryRunFromOpenapi_SLSEndpointFallback(t *testing.T) {
	product := &meta.Product{Code: "sls", Version: "2020-12-30"}
	api := &meta.Api{Name: "ListProject", Product: product}
	profile := &config.Profile{RegionId: "cn-shanghai"}

	oc := &OpenapiContext{
		HttpContext: &HttpContext{
			profile: profile,
			product: product,
			openapiRequest: &openapiutil.OpenApiRequest{
				Headers: map[string]*string{},
				Query:   map[string]*string{},
			},
			openapiParams: newOpenapiParams("GET", "/", "ListProject", "2020-12-30"),
		},
		method: "GET",
		path:   "/",
		api:    api,
	}

	out := buildCliDryRunFromOpenapi(oc)
	assert.Equal(t, "cn-shanghai.log.aliyuncs.com", out.Endpoint)
}

func TestBuildCliDryRunFromOpenapi_WithBody(t *testing.T) {
	product := &meta.Product{Code: "sls", Version: "2020-12-30"}
	api := &meta.Api{Name: "CreateLogStore", Product: product}
	profile := &config.Profile{RegionId: "cn-hangzhou", Endpoint: "cn-hangzhou.log.aliyuncs.com"}

	bodyContent := map[string]interface{}{
		"logstoreName": "test-store",
		"ttl":          30,
	}
	oc := &OpenapiContext{
		HttpContext: &HttpContext{
			profile: profile,
			product: product,
			openapiRequest: &openapiutil.OpenApiRequest{
				Headers: map[string]*string{},
				Query:   map[string]*string{},
				Body:    bodyContent,
			},
			openapiParams: newOpenapiParams("POST", "/logstores", "CreateLogStore", "2020-12-30"),
		},
		method: "POST",
		path:   "/logstores",
		api:    api,
	}

	out := buildCliDryRunFromOpenapi(oc)
	assert.Equal(t, "json", out.BodyFormat)
	assert.Contains(t, out.Body, "logstoreName")
	assert.Contains(t, out.Body, "test-store")
}

func TestBuildCliDryRunFromOpenapi_WithBinaryBody(t *testing.T) {
	product := &meta.Product{Code: "sls", Version: "2020-12-30"}
	api := &meta.Api{Name: "PutLogs", Product: product}
	profile := &config.Profile{RegionId: "cn-hangzhou", Endpoint: "cn-hangzhou.log.aliyuncs.com"}

	binaryData := []byte("compressed-data-here")
	oc := &OpenapiContext{
		HttpContext: &HttpContext{
			profile: profile,
			product: product,
			openapiRequest: &openapiutil.OpenApiRequest{
				Headers: map[string]*string{},
				Query:   map[string]*string{},
				Body:    binaryData,
			},
			openapiParams: newOpenapiParams("POST", "/logstores/store/shards", "PutLogs", "2020-12-30"),
		},
		method: "POST",
		path:   "/logstores/store/shards",
		api:    api,
	}
	oc.openapiParams.ReqBodyType = tea.String("binary")

	out := buildCliDryRunFromOpenapi(oc)
	assert.Equal(t, "binary", out.BodyFormat)
	assert.Equal(t, "compressed-data-here", out.Body)
}

func TestBuildCliDryRunFromOpenapi_NilHeaders(t *testing.T) {
	product := &meta.Product{Code: "sls", Version: "2020-12-30"}
	api := &meta.Api{Name: "ListProject", Product: product}
	profile := &config.Profile{RegionId: "cn-hangzhou", Endpoint: "cn-hangzhou.log.aliyuncs.com"}

	oc := &OpenapiContext{
		HttpContext: &HttpContext{
			profile: profile,
			product: product,
			openapiRequest: &openapiutil.OpenApiRequest{
				Query: map[string]*string{},
			},
			openapiParams: newOpenapiParams("GET", "/", "ListProject", "2020-12-30"),
		},
		method: "GET",
		path:   "/",
		api:    api,
	}

	out := buildCliDryRunFromOpenapi(oc)
	assert.Nil(t, out.Headers)
}

func TestMarshalCliDryRunOutput(t *testing.T) {
	out := &CliDryRunOutput{
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
		Method:   "POST",
		Headers:  map[string]string{"Content-Type": "application/json"},
		Action:   "DescribeInstances",
		Version:  "2014-05-26",
		Query:    map[string]string{"RegionId": "cn-hangzhou"},
	}

	s, err := marshalCliDryRunOutput(out)
	assert.Nil(t, err)
	assert.Contains(t, s, "\"style\": \"RPC\"")
	assert.Contains(t, s, "\"action\": \"DescribeInstances\"")
	assert.Contains(t, s, "\"version\": \"2014-05-26\"")

	var parsed CliDryRunOutput
	err = json.Unmarshal([]byte(s), &parsed)
	assert.Nil(t, err)
	assert.Equal(t, "RPC", parsed.Style)
	assert.Equal(t, "DescribeInstances", parsed.Action)
}

func TestMarshalCliDryRunOutput_ROA(t *testing.T) {
	out := &CliDryRunOutput{
		Style:       "ROA",
		Endpoint:    "cs.cn-hangzhou.aliyuncs.com",
		Method:      "GET",
		Headers:     map[string]string{},
		PathPattern: "/clusters/[ClusterId]",
		Pathname:    "/clusters/c123",
		PathParams:  map[string]string{"ClusterId": "c123"},
	}

	s, err := marshalCliDryRunOutput(out)
	assert.Nil(t, err)
	assert.Contains(t, s, "\"style\": \"ROA\"")
	assert.Contains(t, s, "\"pathPattern\": \"/clusters/[ClusterId]\"")
	assert.Contains(t, s, "\"pathname\": \"/clusters/c123\"")

	var parsed CliDryRunOutput
	err = json.Unmarshal([]byte(s), &parsed)
	assert.Nil(t, err)
	assert.Equal(t, "ROA", parsed.Style)
	assert.Equal(t, "/clusters/[ClusterId]", parsed.PathPattern)
}

func TestMarshalCliDryRunOutput_OmitsEmpty(t *testing.T) {
	out := &CliDryRunOutput{
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
		Method:   "GET",
		Headers:  map[string]string{},
		Action:   "DescribeRegions",
		Version:  "2014-05-26",
	}

	s, err := marshalCliDryRunOutput(out)
	assert.Nil(t, err)
	assert.NotContains(t, s, "pathPattern")
	assert.NotContains(t, s, "pathname")
	assert.NotContains(t, s, "pathParams")
	assert.NotContains(t, s, "body")
	assert.NotContains(t, s, "bodyFormat")
	assert.NotContains(t, s, "query")
}

func TestProcessCliDryRun(t *testing.T) {
	stdout := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, new(bytes.Buffer))
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	req := requests.NewCommonRequest()
	req.Domain = "ecs.cn-hangzhou.aliyuncs.com"
	req.Method = "POST"
	req.ApiName = "DescribeInstances"
	req.Version = "2014-05-26"
	req.Product = "Ecs"
	req.QueryParams["RegionId"] = "cn-hangzhou"

	invoker := &ForceRpcInvoker{
		BasicInvoker: &BasicInvoker{request: req},
		method:       "DescribeInstances",
	}

	err := processCliDryRun(ctx, invoker)
	assert.Nil(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Style:    RPC")
	assert.Contains(t, output, "Endpoint: ecs.cn-hangzhou.aliyuncs.com")
	assert.Contains(t, output, "Action:   DescribeInstances")
	assert.Contains(t, output, "Version:  2014-05-26")
	assert.Contains(t, output, "RegionId = cn-hangzhou")
}

func TestProcessCliDryRunJson(t *testing.T) {
	stdout := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, new(bytes.Buffer))
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	req := requests.NewCommonRequest()
	req.Domain = "ecs.cn-hangzhou.aliyuncs.com"
	req.Method = "POST"
	req.ApiName = "DescribeInstances"
	req.Version = "2014-05-26"
	req.Product = "Ecs"
	req.QueryParams["RegionId"] = "cn-hangzhou"

	invoker := &ForceRpcInvoker{
		BasicInvoker: &BasicInvoker{request: req},
		method:       "DescribeInstances",
	}

	err := processCliDryRunJson(ctx, invoker)
	assert.Nil(t, err)

	output := stdout.String()
	assert.Contains(t, output, "\"style\": \"RPC\"")
	assert.Contains(t, output, "\"action\": \"DescribeInstances\"")

	var parsed CliDryRunOutput
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed)
	assert.Nil(t, err)
	assert.Equal(t, "RPC", parsed.Style)
	assert.Equal(t, "ecs.cn-hangzhou.aliyuncs.com", parsed.Endpoint)
}

func TestProcessCliDryRunOpenapi(t *testing.T) {
	stdout := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, new(bytes.Buffer))
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	product := &meta.Product{Code: "sls", Version: "2020-12-30"}
	api := &meta.Api{Name: "GetLogStore", Product: product}
	profile := &config.Profile{RegionId: "cn-hangzhou", Endpoint: "cn-hangzhou.log.aliyuncs.com"}

	oc := &OpenapiContext{
		HttpContext: &HttpContext{
			profile: profile,
			product: product,
			openapiRequest: &openapiutil.OpenApiRequest{
				Headers: map[string]*string{
					"Content-Type": tea.String("application/json"),
				},
				Query: map[string]*string{},
			},
			openapiParams: newOpenapiParams("GET", "/logstores/my-store", "GetLogStore", "2020-12-30"),
		},
		method: "GET",
		path:   "/logstores/[logstoreName]",
		api:    api,
	}

	err := processCliDryRunOpenapi(ctx, oc)
	assert.Nil(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Style:    ROA")
	assert.Contains(t, output, "Endpoint: cn-hangzhou.log.aliyuncs.com")
	assert.Contains(t, output, "Action:   GetLogStore")
}

func TestProcessCliDryRunOpenapiJson(t *testing.T) {
	stdout := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, new(bytes.Buffer))
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	product := &meta.Product{Code: "sls", Version: "2020-12-30"}
	api := &meta.Api{Name: "GetLogStore", Product: product}
	profile := &config.Profile{RegionId: "cn-hangzhou", Endpoint: "cn-hangzhou.log.aliyuncs.com"}

	oc := &OpenapiContext{
		HttpContext: &HttpContext{
			profile: profile,
			product: product,
			openapiRequest: &openapiutil.OpenApiRequest{
				Headers: map[string]*string{
					"Content-Type": tea.String("application/json"),
				},
				Query: map[string]*string{},
			},
			openapiParams: newOpenapiParams("GET", "/logstores/my-store", "GetLogStore", "2020-12-30"),
		},
		method: "GET",
		path:   "/logstores/[logstoreName]",
		api:    api,
	}

	err := processCliDryRunOpenapiJson(ctx, oc)
	assert.Nil(t, err)

	output := stdout.String()
	assert.Contains(t, output, "\"style\": \"ROA\"")
	assert.Contains(t, output, "GetLogStore")

	var parsed CliDryRunOutput
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed)
	assert.Nil(t, err)
	assert.Equal(t, "ROA", parsed.Style)
	assert.Equal(t, "cn-hangzhou.log.aliyuncs.com", parsed.Endpoint)
}

func TestProcessInvoke_CliDryRun_RPC(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	skipflag := config.NewSkipSecureVerify()
	skipflag.SetAssigned(true)
	ctx.Flags().Add(skipflag)

	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)

	endpointflag := config.NewEndpointFlag()
	endpointflag.SetAssigned(true)
	endpointflag.SetValue("ecs.cn-hangzhou.aliyuncs.com")
	ctx.Flags().Add(endpointflag)

	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("2014-05-26")

	ForceFlag(ctx.Flags()).SetAssigned(true)
	CliDryRunFlag(ctx.Flags()).SetAssigned(true)

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)

	err := command.processInvoke(ctx, "ecs", "DescribeRegions", "")
	assert.Nil(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Style:    RPC")
	assert.Contains(t, output, "Action:   DescribeRegions")
	assert.Contains(t, output, "Version:  2014-05-26")
	assert.Contains(t, output, "Endpoint: ecs.cn-hangzhou.aliyuncs.com")
}

func TestProcessInvoke_CliDryRunJson_RPC(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	skipflag := config.NewSkipSecureVerify()
	skipflag.SetAssigned(true)
	ctx.Flags().Add(skipflag)

	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)

	endpointflag := config.NewEndpointFlag()
	endpointflag.SetAssigned(true)
	endpointflag.SetValue("ecs.cn-hangzhou.aliyuncs.com")
	ctx.Flags().Add(endpointflag)

	VersionFlag(ctx.Flags()).SetAssigned(true)
	VersionFlag(ctx.Flags()).SetValue("2014-05-26")

	ForceFlag(ctx.Flags()).SetAssigned(true)
	DryRunJsonFlag(ctx.Flags()).SetAssigned(true)

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)

	err := command.processInvoke(ctx, "ecs", "DescribeRegions", "")
	assert.Nil(t, err)

	output := strings.TrimSpace(stdout.String())
	var parsed CliDryRunOutput
	err = json.Unmarshal([]byte(output), &parsed)
	assert.Nil(t, err)
	assert.Equal(t, "RPC", parsed.Style)
	assert.Equal(t, "DescribeRegions", parsed.Action)
	assert.Equal(t, "2014-05-26", parsed.Version)
	assert.Equal(t, "ecs.cn-hangzhou.aliyuncs.com", parsed.Endpoint)
}

func TestProcessInvoke_CliDryRun_ROA(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	skipflag := config.NewSkipSecureVerify()
	skipflag.SetAssigned(true)
	ctx.Flags().Add(skipflag)

	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)

	endpointflag := config.NewEndpointFlag()
	endpointflag.SetAssigned(true)
	endpointflag.SetValue("cs.cn-hangzhou.aliyuncs.com")
	ctx.Flags().Add(endpointflag)

	ForceFlag(ctx.Flags()).SetAssigned(true)
	CliDryRunFlag(ctx.Flags()).SetAssigned(true)

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)

	err := command.processInvoke(ctx, "cs", "GET", "/clusters")
	assert.Nil(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Style:    ROA")
	assert.Contains(t, output, "Method:   GET")
	assert.Contains(t, output, "Pathname: /clusters")
	assert.Contains(t, output, "Endpoint: cs.cn-hangzhou.aliyuncs.com")
}

func TestProcessInvoke_CliDryRunJson_ROA(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	skipflag := config.NewSkipSecureVerify()
	skipflag.SetAssigned(true)
	ctx.Flags().Add(skipflag)

	regionflag := config.NewRegionFlag()
	regionflag.SetAssigned(true)
	regionflag.SetValue("cn-hangzhou")
	ctx.Flags().Add(regionflag)

	endpointflag := config.NewEndpointFlag()
	endpointflag.SetAssigned(true)
	endpointflag.SetValue("cs.cn-hangzhou.aliyuncs.com")
	ctx.Flags().Add(endpointflag)

	ForceFlag(ctx.Flags()).SetAssigned(true)
	DryRunJsonFlag(ctx.Flags()).SetAssigned(true)

	profile := config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "accesskeyid",
		AccessKeySecret: "accesskeysecret",
		RegionId:        "cn-hangzhou",
	}
	command := NewCommando(stdout, profile)

	err := command.processInvoke(ctx, "cs", "GET", "/clusters")
	assert.Nil(t, err)

	output := strings.TrimSpace(stdout.String())
	var parsed CliDryRunOutput
	err = json.Unmarshal([]byte(output), &parsed)
	assert.Nil(t, err)
	assert.Equal(t, "ROA", parsed.Style)
	assert.Equal(t, "GET", parsed.Method)
	assert.Equal(t, "/clusters", parsed.Pathname)
	assert.Equal(t, "cs.cn-hangzhou.aliyuncs.com", parsed.Endpoint)
}

func TestFormatCliDryRunHuman_RPC(t *testing.T) {
	out := &CliDryRunOutput{
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
		Method:   "POST",
		Headers:  map[string]string{"Content-Type": "application/json"},
		Action:   "DescribeInstances",
		Version:  "2014-05-26",
		Query:    map[string]string{"RegionId": "cn-hangzhou"},
	}

	s := formatCliDryRunHuman(out)
	assert.Contains(t, s, "Style:    RPC")
	assert.Contains(t, s, "Endpoint: ecs.cn-hangzhou.aliyuncs.com")
	assert.Contains(t, s, "Method:   POST")
	assert.Contains(t, s, "Action:   DescribeInstances")
	assert.Contains(t, s, "Version:  2014-05-26")
	assert.Contains(t, s, "Content-Type: application/json")
	assert.Contains(t, s, "RegionId = cn-hangzhou")
	assert.True(t, strings.HasPrefix(s, "---\n"))
	assert.True(t, strings.HasSuffix(s, "---"))
}

func TestFormatCliDryRunHuman_ROA(t *testing.T) {
	out := &CliDryRunOutput{
		Style:       "ROA",
		Endpoint:    "cs.cn-hangzhou.aliyuncs.com",
		Method:      "GET",
		Headers:     map[string]string{},
		PathPattern: "/clusters/[ClusterId]",
		Pathname:    "/clusters/c123",
		PathParams:  map[string]string{"ClusterId": "c123"},
	}

	s := formatCliDryRunHuman(out)
	assert.Contains(t, s, "Style:    ROA")
	assert.Contains(t, s, "PathPattern: /clusters/[ClusterId]")
	assert.Contains(t, s, "Pathname: /clusters/c123")
	assert.Contains(t, s, "ClusterId = c123")
	assert.NotContains(t, s, "Action:")
	assert.NotContains(t, s, "Version:")
}

func TestFormatCliDryRunHuman_WithBody(t *testing.T) {
	out := &CliDryRunOutput{
		Style:      "RPC",
		Endpoint:   "ecs.cn-hangzhou.aliyuncs.com",
		Method:     "POST",
		Headers:    map[string]string{},
		Action:     "RunInstances",
		Version:    "2014-05-26",
		Body:       `{"InstanceType":"ecs.g6.large"}`,
		BodyFormat: "raw",
	}

	s := formatCliDryRunHuman(out)
	assert.Contains(t, s, "Body[raw]:")
	assert.Contains(t, s, `{"InstanceType":"ecs.g6.large"}`)
}

func TestFormatCliDryRunHuman_Minimal(t *testing.T) {
	out := &CliDryRunOutput{
		Style:    "RPC",
		Endpoint: "ecs.cn-hangzhou.aliyuncs.com",
		Method:   "GET",
		Headers:  map[string]string{},
		Action:   "DescribeRegions",
		Version:  "2014-05-26",
	}

	s := formatCliDryRunHuman(out)
	assert.NotContains(t, s, "PathPattern:")
	assert.NotContains(t, s, "Pathname:")
	assert.NotContains(t, s, "PathParams:")
	assert.NotContains(t, s, "Query:")
	assert.NotContains(t, s, "Body[")
}

func TestCliDryRunFlag_Definition(t *testing.T) {
	flag := NewCliDryRunFlag()
	assert.Equal(t, "cli-dry-run", flag.Name)
	assert.Equal(t, cli.AssignedNone, flag.AssignedMode)
	assert.Contains(t, flag.ExcludeWith, DryRunFlagName)
	assert.Contains(t, flag.ExcludeWith, CliDryRunJsonFlagName)
}

func TestCliDryRunFlag_Accessor(t *testing.T) {
	fs := cli.NewFlagSet()
	fs.Add(NewCliDryRunFlag())
	f := CliDryRunFlag(fs)
	assert.NotNil(t, f)
	assert.Equal(t, CliDryRunFlagName, f.Name)
}

func TestDryRunFlag_ExcludesCliDryRun(t *testing.T) {
	flag := NewDryRunFlag()
	assert.Contains(t, flag.ExcludeWith, CliDryRunFlagName)
}

func TestDryRunJsonFlag_ExcludesCliDryRun(t *testing.T) {
	flag := NewDryRunJsonFlag()
	assert.Contains(t, flag.ExcludeWith, CliDryRunFlagName)
}

func TestEstimateCostFlag_ExcludesCliDryRun(t *testing.T) {
	flag := NewEstimateCostFlag()
	assert.Contains(t, flag.ExcludeWith, CliDryRunFlagName)
}

func TestQuietFlag_ExcludesCliDryRun(t *testing.T) {
	flag := NewQuietFlag()
	assert.Contains(t, flag.ExcludeWith, CliDryRunFlagName)
}

func TestBuildCliDryRunFromInvoker_NoQuery(t *testing.T) {
	req := requests.NewCommonRequest()
	req.Domain = "ecs.cn-hangzhou.aliyuncs.com"
	req.Method = "GET"
	req.ApiName = "DescribeRegions"
	req.Version = "2014-05-26"
	req.Product = "Ecs"

	invoker := &ForceRpcInvoker{
		BasicInvoker: &BasicInvoker{request: req},
		method:       "DescribeRegions",
	}

	out := buildCliDryRunFromInvoker(invoker)
	assert.Nil(t, out.Query)
}

func TestBuildCliDryRunFromOpenapi_EmptyQuery(t *testing.T) {
	product := &meta.Product{Code: "sls", Version: "2020-12-30"}
	api := &meta.Api{Name: "ListProject", Product: product}
	profile := &config.Profile{RegionId: "cn-hangzhou", Endpoint: "cn-hangzhou.log.aliyuncs.com"}

	oc := &OpenapiContext{
		HttpContext: &HttpContext{
			profile: profile,
			product: product,
			openapiRequest: &openapiutil.OpenApiRequest{
				Headers: map[string]*string{},
				Query:   map[string]*string{},
			},
			openapiParams: newOpenapiParams("GET", "/", "ListProject", "2020-12-30"),
		},
		method: "GET",
		path:   "/",
		api:    api,
	}

	out := buildCliDryRunFromOpenapi(oc)
	assert.Nil(t, out.Query)
}

// helper to build openapiParams for tests
func newOpenapiParams(method, pathname, action, version string) *openapiClient.Params {
	return &openapiClient.Params{
		Method:   tea.String(method),
		Pathname: tea.String(pathname),
		Action:   tea.String(action),
		Version:  tea.String(version),
	}
}
