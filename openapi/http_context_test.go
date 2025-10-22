package openapi

import (
	"bytes"
	"encoding/base64"
	"os"
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

	t.Run("InValidProfile", func(t *testing.T) {
		profile := &config.Profile{
			RegionId:        "cn-hangzhou",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
		}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "ECS"}

		client, err := GetOpenapiClient(profile, ctx, product)
		assert.Nil(t, client)
		assert.NotEmpty(t, err.Error(), "empty error message")
	})

	t.Run("ValidProfile", func(t *testing.T) {
		profile := &config.Profile{
			Mode:            "AK",
			RegionId:        "cn-hangzhou",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
		}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "ECS"}

		client, err := GetOpenapiClient(profile, ctx, product)
		assert.NotNil(t, client)
		assert.Nil(t, err)
	})

	t.Run("SLSProductWithValidProfile", func(t *testing.T) {
		profile := &config.Profile{
			Mode:            "AK",
			RegionId:        "cn-hangzhou",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
		}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "SLS"}

		client, err := GetOpenapiClient(profile, ctx, product)
		assert.NotNil(t, client)
		assert.Nil(t, err)
	})

	t.Run("UserAgentWithVendorEnv", func(t *testing.T) {
		os.Setenv("ALIBABA_CLOUD_VENDOR", "test-vendor")
		profile := &config.Profile{
			Mode:            "AK",
			RegionId:        "cn-hangzhou",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
		}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "ECS"}

		client, err := GetOpenapiClient(profile, ctx, product)
		assert.NotNil(t, client)
		assert.Nil(t, err)

	})

	t.Run("TimeoutConfigurations", func(t *testing.T) {
		profile := &config.Profile{
			Mode:            "AK",
			RegionId:        "cn-hangzhou",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
			ReadTimeout:     30,
			ConnectTimeout:  10,
		}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "ECS"}

		client, err := GetOpenapiClient(profile, ctx, product)
		assert.NotNil(t, client)
		assert.Nil(t, err)
	})

	t.Run("ZeroTimeoutConfigurations", func(t *testing.T) {
		profile := &config.Profile{
			Mode:            "AK",
			RegionId:        "cn-hangzhou",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
			ReadTimeout:     0,
			ConnectTimeout:  0,
		}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "ECS"}

		client, err := GetOpenapiClient(profile, ctx, product)
		assert.NotNil(t, client)
		assert.Nil(t, err)
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

	t.Run("InitWithRetryCount", func(t *testing.T) {
		profile := &config.Profile{
			Mode:            "AK",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
			RegionId:        "cn-hangzhou",
			RetryCount:      3,
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

	t.Run("InitWithHeaders", func(t *testing.T) {
		profile := &config.Profile{
			Mode:            "AK",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
			RegionId:        "cn-hangzhou",
		}
		context := &HttpContext{profile: profile}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "ECS", Version: "2014-05-26"}
		headerflag := NewHeaderFlag()
		headerflag.SetValues([]string{"Accept=json", "Content-Type=json", "testfail"})
		ctx.Flags().Add(headerflag)
		skipflag := config.NewSkipSecureVerify()
		skipflag.SetAssigned(true)
		ctx.Flags().Add(skipflag)
		err := context.Init(ctx, product)
		assert.Contains(t, err.Error(), "invaild flag --header `testfail`")
	})

	t.Run("InitWithEndpoint", func(t *testing.T) {
		profile := &config.Profile{
			Mode:            "AK",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
			RegionId:        "cn-hangzhou",
		}
		context := &HttpContext{profile: profile}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "ECS", Version: "2014-05-26"}
		endpointflag := NewEndpointFlag()
		endpointflag.SetAssigned(true)
		endpointflag.SetValue("ecs.cn-hangzhou.aliyuncs")
		ctx.Flags().Add(endpointflag)
		err := context.Init(ctx, product)
		assert.Nil(t, err)
	})

	t.Run("InitWithEmptyRegion", func(t *testing.T) {
		profile := &config.Profile{
			Mode:            "AK",
			AccessKeyId:     "test-access-key-id",
			AccessKeySecret: "test-access-key-secret",
		}
		context := &HttpContext{profile: profile}
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		product := &meta.Product{Code: "ECS", Version: "2014-05-26"}
		err := context.Init(ctx, product)
		assert.Contains(t, err.Error(), "init openapi client failed")
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
		assert.NoError(t, err)
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
		context.openapiResponse = map[string]any{
			"body": encodedBody,
			"headers": map[string]any{
				"x-log-count":  "1",
				"x-log-cursor": "test-cursor",
			},
		}
		context.product = &meta.Product{Code: "sls"}
		context.api = &meta.Api{Name: "PullLogs"}

		result, _ := context.CheckResponseForPullLogs(context.openapiResponse)
		assert.NotEmpty(t, result)
	})

	t.Run("CheckResponseForPullLogsFail", func(t *testing.T) {
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

		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiResponse = map[string]any{
			"body": bodyBytes,
			"headers": map[string]any{
				"x-log-count":  "1",
				"x-log-cursor": "test-cursor",
			},
		}
		context.product = &meta.Product{Code: "sls"}
		context.api = &meta.Api{Name: "PullLogs"}

		_, err = context.CheckResponseForPullLogs(context.openapiResponse)
		assert.Contains(t, err.Error(), "invalid response body for pulllogs parsing, please check")
	})

	t.Run("CheckResponseForPullLogsDecodeFail", func(t *testing.T) {
		logGroup := &sls.LogGroup{
			Logs: []*sls.Log{{
				Time: proto.Uint32(1234567890),
			}},
			Topic: proto.String("test-topic"),
		}

		logGroupList := &sls.LogGroupList{
			LogGroups: []*sls.LogGroup{logGroup},
		}

		bodyBytes, err := proto.Marshal(logGroupList)
		assert.NoError(t, err)

		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiResponse = map[string]any{
			"body": string(bodyBytes),
			"headers": map[string]any{
				"x-log-count":  "1",
				"x-log-cursor": "test-cursor",
			},
		}
		context.product = &meta.Product{Code: "sls"}
		context.api = &meta.Api{Name: "PullLogs"}

		_, err = context.CheckResponseForPullLogs(context.openapiResponse)
		assert.Contains(t, err.Error(), "illegal base64 data")
	})

	t.Run("CheckResponseForPullLogsEmpty", func(t *testing.T) {
		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiResponse = map[string]any{
			"body": "",
			"headers": map[string]any{
				"x-log-count":  "1",
				"x-log-cursor": "test-cursor",
			},
		}
		context.product = &meta.Product{Code: "sls"}
		context.api = &meta.Api{Name: "PullLogs"}

		result, err := context.CheckResponseForPullLogs(context.openapiResponse)
		assert.Nil(t, err)
		assert.Equal(t, result, "")
	})

	t.Run("GetResponseForPullLogs", func(t *testing.T) {
		httpContext := &HttpContext{}
		context := &OpenapiContext{HttpContext: httpContext}
		context.openapiResponse = map[string]any{}
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
		context.openapiResponse = map[string]any{
			"body": "test response",
		}
		context.product = &meta.Product{Code: "ecs"}
		context.api = &meta.Api{Name: "DescribeInstances"}

		result, err := context.GetResponse()
		assert.NoError(t, err)
		assert.Equal(t, "test response", result)
	})
}

func TestProcessPutLogsBodyDataFail(t *testing.T) {
	jsonData := `{
		"Logs": [
			{
				"Time",
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
	assert.Contains(t, err.Error(), "parse json failed")
}

func TestProcessPutLogsBodyFile(t *testing.T) {
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
	BodyFileFlag(ctx.Flags()).SetAssigned(true)
	BodyFileFlag(ctx.Flags()).SetValue("../integration/logexample.json")

	err := context.ProcessPutLogsBody(ctx)
	assert.Nil(t, err)
}

func TestProcessRegularBodyFile(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Headers: map[string]*string{},
	}
	context.openapiParams = &openapiClient.Params{}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{Name: "TestApi"}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	AddFlags(ctx.Flags())
	ctx.SetUnknownFlags(cli.NewFlagSet())
	BodyFileFlag(ctx.Flags()).SetAssigned(true)
	BodyFileFlag(ctx.Flags()).SetValue("../integration/logexample.json")

	err := context.ProcessBody(ctx)
	assert.Nil(t, err)
}

func TestRequestProcessors(t *testing.T) {
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
}

func TestRequestProcessorsFail(t *testing.T) {
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
	err := context.RequestProcessors(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required parameter missing")
}

func TestPrepare(t *testing.T) {
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
}

func TestPrepareNoApi(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext, method: "GET", path: "/instances", api: nil}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Query:   map[string]*string{},
		Headers: map[string]*string{},
		HostMap: map[string]*string{},
	}
	context.openapiParams = &openapiClient.Params{}
	context.product = &meta.Product{Code: "ecs"}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	err := context.Prepare(ctx)
	assert.Equal(t, err.Error(), "api not found, should not happen")
}

func TestPrepareMissingArg(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext, method: "GET", path: "/instances", api: nil}
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

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	err := context.Prepare(ctx)
	assert.Equal(t, err.Error(), "no parameters provided, please check")
}

func TestPrepareSecureFlag(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext, method: "GET", path: "/instances", api: nil}
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

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	secureFlag := NewSecureFlag()
	secureFlag.SetAssigned(true)
	ctx.Flags().Add(secureFlag)
	err := context.Prepare(ctx)
	assert.Equal(t, err.Error(), "no parameters provided, please check")
}

func TestPrepareInSecureFlag(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext, method: "GET", path: "/instances", api: nil}
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

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	inSecureFlag := NewInsecureFlag()
	inSecureFlag.SetAssigned(true)
	ctx.Flags().Add(inSecureFlag)
	err := context.Prepare(ctx)
	assert.Equal(t, err.Error(), "no parameters provided, please check")
}

func TestProcessHeadersWithValidParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Headers: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestHeader",
				Position: "header",
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("TestHeader")
	ctx.UnknownFlags().Get("TestHeader").SetAssigned(true)
	ctx.UnknownFlags().Get("TestHeader").SetValue("test-value")

	err := context.ProcessHeaders(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test-value", *context.openapiRequest.Headers["TestHeader"])
}

func TestProcessHeadersWithInvalidParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Headers: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestHeader",
				Position: "header",
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("InvalidHeader")
	ctx.UnknownFlags().Get("InvalidHeader").SetAssigned(true)
	ctx.UnknownFlags().Get("InvalidHeader").SetValue("test-value")

	err := context.ProcessHeaders(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a valid parameter")
}

func TestProcessHeadersWithRequiredParameterMissing(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Headers: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestHeader",
				Position: "header",
				Required: true,
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("TestHeader")
	// Not assigning a value to simulate missing required parameter

	err := context.ProcessHeaders(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required parameter missing")
}

func TestProcessQueryWithInvalidParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Query: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestQueryParam",
				Position: "Query",
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("aaa")
	ctx.UnknownFlags().Get("aaa").SetAssigned(true)
	ctx.UnknownFlags().Get("aaa").SetValue("test-value")

	err := context.ProcessQuery(ctx)
	assert.Contains(t, err.Error(), "'--aaa' is not a valid parameter or flag")
}

func TestProcessQueryMissingParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Query: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestQueryParam",
				Position: "Query",
				Required: true,
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("TestQueryParam")
	ctx.UnknownFlags().Get("TestQueryParam").SetAssigned(true)

	err := context.ProcessQuery(ctx)
	assert.Contains(t, err.Error(), "required parameter missing")
}

func TestProcessQueryWithValidParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Query: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestQueryParam",
				Position: "Query",
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("TestQueryParam")
	ctx.UnknownFlags().Get("TestQueryParam").SetAssigned(true)
	ctx.UnknownFlags().Get("TestQueryParam").SetValue("test-value")

	err := context.ProcessQuery(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test-value", *context.openapiRequest.Query["TestQueryParam"])
}

func TestProcessQueryWithOtherParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Query: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestQueryParam",
				Position: "Query",
			},
			{
				Name:     "aaa",
				Position: "Path",
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("aaa")
	ctx.UnknownFlags().Get("aaa").SetAssigned(true)
	ctx.UnknownFlags().Get("aaa").SetValue("test-value")

	err := context.ProcessQuery(ctx)
	assert.Nil(t, err)
}

func TestProcessHostWithValidParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		HostMap: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestHost",
				Position: "Host",
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("TestHost")
	ctx.UnknownFlags().Get("TestHost").SetAssigned(true)
	ctx.UnknownFlags().Get("TestHost").SetValue("test-host.aliyuncs.com")

	err := context.ProcessHost(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test-host.aliyuncs.com", *context.openapiRequest.HostMap["testhost"])
}

func TestProcessHostWithInvalidParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		HostMap: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestHost",
				Position: "Host",
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("aaa")
	ctx.UnknownFlags().Get("aaa").SetAssigned(true)
	err := context.ProcessHost(ctx)
	assert.Contains(t, err.Error(), "'--aaa' is not a valid parameter or flag")
}

func TestProcessHostMissingRequiredParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		HostMap: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestHost",
				Position: "Host",
				Required: true,
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("TestHost")
	ctx.UnknownFlags().Get("TestHost").SetAssigned(true)
	err := context.ProcessHost(ctx)
	assert.Contains(t, err.Error(), "required parameter missing")
}

func TestProcessPathWithValidParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.path = "/instances/[InstanceId]"
	context.openapiParams = &openapiClient.Params{}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "InstanceId",
				Position: "Path",
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("InstanceId")
	ctx.UnknownFlags().Get("InstanceId").SetAssigned(true)
	ctx.UnknownFlags().Get("InstanceId").SetValue("i-test123")

	err := context.ProcessPath(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "/instances/i-test123", *context.openapiParams.Pathname)
}

func TestProcessPathWithInvalidParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.path = "/instances/[InstanceId]"
	context.openapiParams = &openapiClient.Params{}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "InstanceId",
				Position: "Path",
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("aaa")
	ctx.UnknownFlags().Get("aaa").SetAssigned(true)

	err := context.ProcessPath(ctx)
	assert.Contains(t, err.Error(), "'--aaa' is not a valid parameter or flag")
}

func TestProcessPathMissingRequiredParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.path = "/instances/[InstanceId]"
	context.openapiParams = &openapiClient.Params{}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "InstanceId",
				Position: "Path",
				Required: true,
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("InstanceId")
	ctx.UnknownFlags().Get("InstanceId").SetAssigned(true)

	err := context.ProcessPath(ctx)
	assert.Contains(t, err.Error(), "required parameter missing")
}

func TestProcessBodyWithValidParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Headers: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestBodyParam",
				Position: "Body",
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("TestBodyParam")
	ctx.UnknownFlags().Get("TestBodyParam").SetAssigned(true)
	ctx.UnknownFlags().Get("TestBodyParam").SetValue("test-value")

	err := context.ProcessBody(ctx)
	assert.NoError(t, err)
	body, ok := context.openapiRequest.Body.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "test-value", body["TestBodyParam"])

}
func TestProcessBodyWithNoRequiredParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Headers: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestBodyParam",
				Position: "Body",
				Required: true,
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("TestBodyParam")
	ctx.UnknownFlags().Get("TestBodyParam").SetAssigned(true)

	err := context.ProcessBody(ctx)
	assert.Contains(t, err.Error(), "required parameter missing")
}

func TestProcessBodyInvalidParameter(t *testing.T) {
	httpContext := &HttpContext{}
	context := &OpenapiContext{HttpContext: httpContext}
	context.openapiRequest = &openapiutil.OpenApiRequest{
		Headers: map[string]*string{},
	}
	context.product = &meta.Product{Code: "ecs"}
	context.api = &meta.Api{
		Name:    "DescribeInstances",
		Product: &meta.Product{Version: "2014-05-26"},
		Parameters: []meta.Parameter{
			{
				Name:     "TestBodyParam",
				Position: "Body",
				Required: true,
			},
		},
	}

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().AddByName("aaa")
	ctx.UnknownFlags().Get("aaa").SetAssigned(true)

	err := context.ProcessBody(ctx)
	assert.Contains(t, err.Error(), "'--aaa' is not a valid parameter or flag")
}
