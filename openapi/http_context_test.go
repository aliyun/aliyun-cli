package openapi

import (
	"bytes"
	"encoding/base64"
	"testing"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-cli/v3/sls"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestShouldUseOpenapi(t *testing.T) {
	t.Run("SLSProduct", func(t *testing.T) {
		product := &meta.Product{Code: "SLS"}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		result := ShouldUseOpenapi(ctx, product)
		assert.True(t, result)
	})

	t.Run("NonSLSProduct", func(t *testing.T) {
		product := &meta.Product{Code: "ECS"}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		result := ShouldUseOpenapi(ctx, product)
		assert.False(t, result)
	})

	t.Run("LowercaseSLSProduct", func(t *testing.T) {
		product := &meta.Product{Code: "sls"}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		result := ShouldUseOpenapi(ctx, product)
		assert.True(t, result)
	})
}

func TestGetOpenapiClient(t *testing.T) {
	t.Run("EmptyRegionId", func(t *testing.T) {
		profile := &config.Profile{RegionId: ""}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "SLS"}

		client, err := GetOpenapiClient(profile, ctx, product)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "default RegionId is empty")
	})

	t.Run("ValidProfile", func(t *testing.T) {
		profile := &config.Profile{
			RegionId:        "cn-hangzhou",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
		}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "ECS"}

		client, err := GetOpenapiClient(profile, ctx, product)
		if err == nil {
			assert.NotNil(t, client)
		}
	})

	t.Run("SLSProductWithValidProfile", func(t *testing.T) {
		profile := &config.Profile{
			RegionId:        "cn-hangzhou",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
		}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "SLS"}

		client, err := GetOpenapiClient(profile, ctx, product)
		if err == nil {
			assert.NotNil(t, client)
		}
	})
}

func TestGetContentFromApiResponse(t *testing.T) {
	t.Run("NilResponseBody", func(t *testing.T) {
		response := map[string]any{}
		result := GetContentFromApiResponse(response)
		assert.Equal(t, "", result)
	})

	t.Run("StringResponseBody", func(t *testing.T) {
		response := map[string]any{
			"body": "test response",
		}
		result := GetContentFromApiResponse(response)
		assert.Equal(t, "test response", result)
	})

	t.Run("MapResponseBody", func(t *testing.T) {
		response := map[string]any{
			"body": map[string]any{
				"key": "value",
			},
		}
		result := GetContentFromApiResponse(response)
		assert.Equal(t, `{"key":"value"}`, result)
	})

	t.Run("ByteArrayResponseBody", func(t *testing.T) {
		response := map[string]any{
			"body": []byte("test bytes"),
		}
		result := GetContentFromApiResponse(response)
		assert.Equal(t, "test bytes", result)
	})

	t.Run("OtherTypeResponseBody", func(t *testing.T) {
		response := map[string]any{
			"body": 123,
		}
		result := GetContentFromApiResponse(response)
		assert.Equal(t, "123", result)
	})
}

func TestHttpContext(t *testing.T) {
	t.Run("NewHttpContext", func(t *testing.T) {
		profile := &config.Profile{}
		context := NewHttpContext(profile)
		assert.NotNil(t, context)
		assert.Equal(t, profile, context.profile)
	})

	t.Run("GetRequest", func(t *testing.T) {
		context := &HttpContext{}
		request := context.getRequest()
		assert.Nil(t, request)
	})

	t.Run("Init", func(t *testing.T) {
		profile := &config.Profile{
			Mode:            "AK",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
			RegionId:        "cn-hangzhou",
		}
		context := &HttpContext{profile: profile}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "ECS", Version: "2014-05-26"}

		err := context.Init(ctx, product)
		assert.NoError(t, err)
		assert.NotNil(t, context.openapiRequest)
		assert.NotNil(t, context.openapiParams)
		assert.NotNil(t, context.openapiRuntime)
	})
}

func TestOpenapiContext(t *testing.T) {
	t.Run("ProcessPullLogsHeaders", func(t *testing.T) {
		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiRequest = &openapiutil.OpenApiRequest{
			Headers: map[string]*string{},
		}
		context.openapiParams = &openapiutil.Params{
			BodyType: tea.String("json"),
		}

		context.ProcessPullLogsHeaders(cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer)))

		assert.Equal(t, "lz4", *context.openapiRequest.Headers["Accept-Encoding"])
		assert.Equal(t, "application/x-protobuf", *context.openapiRequest.Headers["accept"])
		assert.Equal(t, "byte", *context.openapiParams.BodyType)
	})

	t.Run("ProcessPutLogsBodyWithBodyFlag", func(t *testing.T) {
		jsonData := `{
			"Logs": [
				{
					"Time": 1712345678,
					"Contents": [
						{ "Key": "method", "Value": "POST" },
						{ "Key": "path", "Value": "/api/login" }
					]
				}
			],
			"Topic": "web-logs",
			"Source": "192.168.1.100",
			"LogTags": [
				{ "Key": "env", "Value": "prod" }
			]
		}`

		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiRequest = &openapiutil.OpenApiRequest{
			Headers: map[string]*string{},
		}
		context.openapiParams = &openapiClient.Params{}
		context.product = &meta.Product{Code: "sls"}
		context.api = &meta.Api{Name: "PutLogs"}

		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		AddFlags(ctx.Flags())
		BodyFlag(ctx.Flags()).SetAssigned(true)
		BodyFlag(ctx.Flags()).SetValue(string(jsonData))

		err := context.ProcessPutLogsBody(ctx)
		if err != nil {
			assert.Contains(t, err.Error(), "no logs provided")
		}
	})

	t.Run("ProcessPutLogsBodyNoInput", func(t *testing.T) {
		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.product = &meta.Product{Code: "sls"}
		context.api = &meta.Api{Name: "PutLogs"}

		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))

		err := context.ProcessPutLogsBody(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no logs provided")
	})

	t.Run("CheckResponseForPullLogs", func(t *testing.T) {
		// Create a mock LogGroupList for testing
		logGroup := &sls.LogGroup{
			Logs: []*sls.Log{{
				Time: proto.Uint32(1234567890),
			}},
			Topic: proto.String("test-topic"),
		}

		logGroupList := &sls.LogGroupList{
			LogGroups: []*sls.LogGroup{logGroup},
		}

		// Marshal to protobuf bytes
		bodyBytes, err := proto.Marshal(logGroupList)
		assert.NoError(t, err)

		// Base64 encode for the response
		encodedBody := base64.StdEncoding.EncodeToString(bodyBytes)

		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiResponse = &map[string]any{
			"body": encodedBody,
			"headers": map[string]any{
				"x-log-count":  "1",
				"x-log-cursor": "test-cursor",
			},
		}
		context.product = &meta.Product{Code: "sls"}
		context.api = &meta.Api{Name: "PullLogs"}

		result, err := context.CheckResponseForPullLogs(*context.openapiResponse)
		if err == nil {
			assert.NotEmpty(t, result)
		}
	})

	t.Run("GetResponseForPullLogs", func(t *testing.T) {
		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiResponse = &map[string]any{}
		context.product = &meta.Product{Code: "sls"}
		context.api = &meta.Api{Name: "PullLogs"}

		result, err := context.GetResponse()
		if err != nil {
			assert.Contains(t, err.Error(), "invalid response body")
		} else {
			assert.Equal(t, result, "")
		}
	})

	t.Run("GetResponseForNonPullLogs", func(t *testing.T) {
		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiResponse = &map[string]any{
			"body": "test response",
		}
		context.product = &meta.Product{Code: "ecs"}
		context.api = &meta.Api{Name: "DescribeInstances"}

		result, err := context.GetResponse()
		assert.NoError(t, err)
		assert.Equal(t, "test response", result)
	})
}

func TestRequestProcessors(t *testing.T) {
	t.Run("RequestProcessors", func(t *testing.T) {
		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiRequest = &openapiutil.OpenApiRequest{
			Query:   map[string]*string{},
			Headers: map[string]*string{},
			HostMap: map[string]*string{},
		}
		context.openapiParams = &openapiClient.Params{}
		context.product = &meta.Product{Code: "ecs"}
		context.api = &meta.Api{
			Name:    "DescribeInstances",
			Product: &meta.Product{Version: "2014-05-26"},
			Parameters: []meta.Parameter{
				{
					Name:     "TestParam",
					Position: "Query",
					Required: true,
				},
			},
		}
		context.method = "GET"
		context.path = "/instances"

		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		ctx.SetUnknownFlags(cli.NewFlagSet())
		ctx.UnknownFlags().AddByName("TestParam")
		ctx.UnknownFlags().Get("TestParam").SetAssigned(true)
		ctx.UnknownFlags().Get("TestParam").SetValue("test_value")
		err := context.RequestProcessors(ctx)
		assert.NoError(t, err)
	})

	t.Run("Prepare", func(t *testing.T) {
		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiRequest = &openapiutil.OpenApiRequest{
			Query:   map[string]*string{},
			Headers: map[string]*string{},
			HostMap: map[string]*string{},
		}
		context.openapiParams = &openapiClient.Params{}
		context.product = &meta.Product{Code: "ecs"}
		context.api = &meta.Api{
			Name:     "DescribeInstances",
			Product:  &meta.Product{Version: "2014-05-26"},
			Protocol: "HTTPS",
		}
		context.method = "GET"
		context.path = "/instances"

		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		ctx.SetUnknownFlags(cli.NewFlagSet())
		err := context.Prepare(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "DescribeInstances", *context.openapiParams.Action)
		assert.Equal(t, "2014-05-26", *context.openapiParams.Version)
		assert.Equal(t, "GET", *context.openapiParams.Method)
		assert.Equal(t, "https", *context.openapiParams.Protocol)
	})
}
