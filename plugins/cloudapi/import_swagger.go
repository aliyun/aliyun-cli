package cloudapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cloudapi"
	"github.com/aliyun/aliyun-cli/config"
	"strings"
)

var region string

func NewImportSwaggerCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "cloudapi_importSwagger",
		Short: i18n.T("cloudapi: mport swagger ", "cloudapi: 导入swagger定义"),
		Usage: "cloudapi_importSwagger xxx_swagger.json --endpoint <endpoint>",
		Run: func(ctx *cli.Context, args []string) error {
			return runImportSwagger(ctx, args[0])
		},
	}
	cmd.Flags().Add(GroupIdFlag)
	cmd.Flags().Add(AoneAppNameFlag)
	cmd.Flags().Add(DeleteAllFlag)
	return cmd
}

func runImportSwagger(ctx *cli.Context, f string) error {
	bytes, err := ioutil.ReadFile(f)
	if err != nil {
		return fmt.Errorf("failed read file %s %s", f, err)
	}

	var se SwaggerEntity
	err = json.Unmarshal(bytes, &se)
	if err != nil {
		return fmt.Errorf("unmarshal failed %s", err)
	}

	// fmt.Printf("%v\n", se)
	// update current `Profile` with flags
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return cli.NewErrorWithTip(err, "Configuration failed, use `aliyun configure` to configure it")
	}
	err = profile.Validate()
	if err != nil {
		return cli.NewErrorWithTip(err, "Configuration failed, use `aliyun configure` to configure it.")
	}
	i18n.SetLanguage(profile.Language)

	client, err := getClient(&profile)
	if err != nil {
		return fmt.Errorf("getClient() for cloudapi failed %s", err)
	}
	region = profile.RegionId
	processImport(client, &se)
	return nil
}

func processImport(client *cloudapi.Client, se *SwaggerEntity) {
	protocol := "http"
	for _, s := range se.Schemes {
		if strings.ToLower(s) == "https" {
			protocol = "https"
		}
	}

	groupId, ok := GroupIdFlag.GetValue()
	if !ok {
		cli.Errorf("need --GroupId ...")
		return
	}
	aoneAppName, ok := AoneAppNameFlag.GetValue()

	hostProtocol, hostAddress := parseHost(se.Host)
	if hostProtocol == "" {
		hostProtocol = protocol
	}

	for path, v := range se.Paths {
		for method, v2 := range v {
			_ = v2
			method = strings.ToUpper(method)
			if method == "X-AMAZON-APIGATEWAY-ANY-METHOD" {
				continue
			}

			apiName := getApiName(se.BasePath, path, method)

			apiId, err := findApi(client, groupId, apiName)
			if err != nil {
				cli.Errorf("findApi failed: %s\n", err)
				return
			}


			if DeleteAllFlag.IsAssigned() && apiId != "" {
				r := cloudapi.CreateDeleteApiRequest()
				r.ApiId = apiId
				r.GroupId = groupId
				fmt.Printf("delete api %s id=%s ... ", apiName, apiId)
				_, err := client.DeleteApi(r)
				if err != nil {
					fmt.Printf("failed %s\n", err)
				} else {
					fmt.Printf("done\n")
				}
				continue
			}

			requestParameters, bodyDesc, err := getRequestParameters(se, v2.Parameters)
			if err != nil {
				fmt.Printf("error importing ... %s %s\n", apiName, err)
				continue
			}
			requestConfig, _ := json.Marshal(&RequestConfig{
				RequestProtocol: protocol,
				RequestHttpMethod: method,
				RequestPath: se.BasePath + path,
				RequestMode: "PASSTHROUGH",
				PostBodyDescription: bodyDesc,
			})

			succ, failed := getResponseDescriptions(se, v2.Responses)

			vpcEnable := "FALSE"
			var vpcConfig VpcConfig
			if se.AliyunVpcId != "" {
				//
				vpcEnable = "TRUE"
				vpcConfig = VpcConfig{
					VpcId: se.AliyunVpcId,
					InstanceId: se.AliyunInstanceId,
					Port: se.AliyunPort,
				}
			}
			serviceConfig, _ := json.Marshal(&ServiceConfig{
				ServiceProtocol: hostProtocol,
				ServiceAddress: hostProtocol + "://" + hostAddress,
				ServicePath: se.BasePath + path,
				ServiceHttpMethod: method,
				// 	后端服务超时时间，单位：毫秒
				ServiceTimeout: "3000",
				//	调用后端服务HTTP服务时，ContentType头的取值策略：
				// 	DEFAULT：使用API网关默认的值
				// 	CUSTOM：自定义
				// 	CLIENT：使用客户端上行的ContentType的头
				ContentTypeCatagory: "CLIENT",
				// TRUE：启用Mock模式
				// FALSE：不启用Mock模式
				Mock: "FALSE",
				// 	如果启用Mock模式，返回的结果
				MockResult: "",
				// 是否启用VPC通道，目前可以取值：
				// TRUE：启用VPC通道
				// FALSE：不启用VPC通
				// 必须先添加VPC授权成功后才能启用
				ServiceVpcEnable: vpcEnable,
				//
				AoneAppName: aoneAppName,
				VpcConfig: vpcConfig,
			})

			if apiId == "" {
				r := cloudapi.CreateCreateApiRequest()
				r.RegionId = region
				r.GroupId = groupId
				r.ApiName = getApiName(se.BasePath, path, method)
				r.AuthType = "APP"
				r.RequestParameters = ""
				r.ServiceParameters = ""
				r.ServiceParametersMap = ""
				r.Visibility = "PRIVATE"
				r.ResultType = "JSON"
				r.ResultSample = "{}"
				r.RequestConfig = string(requestConfig)
				r.ServiceConfig = string(serviceConfig)
				s, _ := json.Marshal(requestParameters)
				r.RequestParameters = string(s)
				r.ResultSample = succ
				r.FailResultSample = failed

				fmt.Printf("creating %s %s ...", method, path)
				_, err := client.CreateApi(r)
				if err != nil {
					fmt.Printf(" failed %s %s\n", serviceConfig, err)
				}
				fmt.Printf("done.\n")
			} else {
				r := cloudapi.CreateModifyApiRequest()

				r.ApiId = apiId
				r.RegionId = region
				r.GroupId = groupId
				r.ApiName = getApiName(se.BasePath, path, method)
				r.AuthType = "APP"
				r.RequestParameters = ""
				r.ServiceParameters = ""
				r.ServiceParametersMap = ""
				r.Visibility = "PRIVATE"
				r.ResultType = "JSON"
				r.ResultSample = "{}"
				r.RequestConfig = string(requestConfig)
				r.ServiceConfig = string(serviceConfig)
				s, _ := json.Marshal(requestParameters)
				r.RequestParameters = string(s)
				r.ResultSample = succ
				r.FailResultSample = failed

				fmt.Printf("modifing %s %s ...", method, path)
				_, err := client.ModifyApi(r)
				if err != nil {
					fmt.Printf(" failed %s %s\n", serviceConfig, err)
				}
				fmt.Printf("done.\n")
			}
		}
	}
	return
}

func findApi(client *cloudapi.Client, groupId string, apiName string) (string, error) {
	r := cloudapi.CreateDescribeApisRequest()
	r.GroupId = groupId
	r.ApiName = apiName
	r.RegionId = region
	resp, err := client.DescribeApis(r)
	if err != nil {
		return "", err
	}
	if resp.TotalCount == 0 {
		return "", nil
	} else if resp.TotalCount == 1 {
		return resp.ApiSummarys.ApiSummary[0].ApiId, nil
	} else {
		return "", fmt.Errorf("ambigous api %s", apiName)
	}
}

func parseHost(s string) (protocol string, address string) {
	if strings.HasPrefix(s, "http://") {
		protocol = "http"
		address = s[len("http://"):]
	} else if strings.HasPrefix(s, "https://") {
		protocol = "https"
		address = s[len("https://"):]
	} else {
		address = s
	}
	return
}

func getRequestParameters(e *SwaggerEntity, params []SwaggerParameter) ([]RequestParameter, string, error) {
	r := make([]RequestParameter, 0)
	body := ""
	for _, p := range params {
		param := RequestParameter{
			ApiParameterName: p.Name,
		}
		if p.Required {
			param.Required = "REQUIRED"
		} else {
			param.Required = "OPTIONAL"
		}
		switch p.In {
		case "body":
			body = getSchemaDescription(e, p.Schema)
		case "path":
			param.Location = "PATH"
			r = append(r, param)
		case "query":
			param.Location = "QUERY"
			r = append(r, param)
		case "header":
			param.Location = "HEAD"
			r = append(r, param)
		case "cookie":
			return r, body, fmt.Errorf("cookie parameter type not supported")
		}
	}
	return r, body, nil
}

func getResponseDescriptions(e *SwaggerEntity, resps map[string]SwaggerResponse) (success string, failed string) {
	for code, resp := range resps {
		if code == "200" {
			success = getSchemaDescription(e, resp.Schema)
		} else {
			failed += fmt.Sprintf(`"%s": %s\n`, code, getSchemaDescription(e, resp.Schema))
		}
	}
	return
}

func getSchemaDescription(e *SwaggerEntity, schema SwaggerDefinition) string {
	if schema.Ref != "" {
		if ref, ok := removePrefix(schema.Ref, "#/definitions/"); ok {
			if s2, ok := e.Definitions[ref]; ok {
				schema = s2
			}
		}
	}
	j, _ := json.Marshal(schema)
	return string(j)
}

