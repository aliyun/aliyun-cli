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
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/meta"
)

func GetClient(cp *config.Profile, ctx *cli.Context) (*sdk.Client, error) {
	conf := sdk.NewConfig()
	// get UserAgent from env
	conf.UserAgent = os.Getenv("ALIYUN_USER_AGENT")

	if cp.RetryCount > 0 {
		// when use --retry-count, enable auto retry
		conf.WithAutoRetry(true)
		conf.WithMaxRetryTime(cp.RetryCount)
	}
	var client *sdk.Client
	var err error
	switch cp.Mode {
	case config.AK:
		client, err = GetClientByAK(cp, conf)
	case config.StsToken:
		client, err = GetClientBySts(cp, conf)
	case config.RamRoleArn:
		client, err = GetClientByRoleArn(cp, conf)
	case config.EcsRamRole:
		client, err = GetClientByEcsRamRole(cp, conf)
	case config.RsaKeyPair:
		client, err = GetClientByPrivateKey(cp, conf)
	case config.RamRoleArnWithEcs:
		client, err = GetClientByRamRoleArnWithEcs(cp, conf)
	case config.ChainableRamRoleArn:
		return GetClientByChainableRamRoleArn(cp, conf, ctx)
	case config.External:
		return GetClientByExternal(cp, conf, ctx)
	case config.CredentialsURI:
		return GetClientByCredentialsURI(cp, conf, ctx)
	default:
		client, err = nil, fmt.Errorf("unexcepted certificate mode: %s", cp.Mode)
	}

	if client != nil {
		if cp.ReadTimeout > 0 {
			client.SetReadTimeout(time.Duration(cp.ReadTimeout) * time.Second)
		}
		if cp.ConnectTimeout > 0 {
			client.SetConnectTimeout(time.Duration(cp.ConnectTimeout) * time.Second)
		}
		if config.SkipSecureVerify(ctx.Flags()).IsAssigned() {
			client.SetHTTPSInsecure(true)
		}
	}
	return client, err
}

func GetClientByAK(cp *config.Profile, config *sdk.Config) (*sdk.Client, error) {
	if cp.AccessKeyId == "" || cp.AccessKeySecret == "" {
		return nil, fmt.Errorf("AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first")
	}
	if cp.RegionId == "" {
		return nil, fmt.Errorf("default RegionId is empty! run `aliyun configure` first")
	}
	cred := credentials.NewAccessKeyCredential(cp.AccessKeyId, cp.AccessKeySecret)
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func GetClientBySts(cp *config.Profile, config *sdk.Config) (*sdk.Client, error) {
	cred := credentials.NewStsTokenCredential(cp.AccessKeyId, cp.AccessKeySecret, cp.StsToken)
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func GetClientByRoleArn(cp *config.Profile, config *sdk.Config) (*sdk.Client, error) {
	cred := credentials.NewRamRoleArnCredential(cp.AccessKeyId, cp.AccessKeySecret, cp.RamRoleArn, cp.RoleSessionName, cp.ExpiredSeconds)
	cred.StsRegion = cp.StsRegion
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func GetClientByRamRoleArnWithEcs(cp *config.Profile, config *sdk.Config) (*sdk.Client, error) {
	client, err := GetClientByEcsRamRole(cp, config)
	if err != nil {
		return nil, err
	}
	accessKeyID, accessKeySecret, StsToken, err := cp.GetSessionCredential(client)
	if err != nil {
		return nil, err
	}
	cred := credentials.NewStsTokenCredential(accessKeyID, accessKeySecret, StsToken)
	return sdk.NewClientWithOptions(cp.RegionId, config, cred)
}

func GetClientByEcsRamRole(cp *config.Profile, config *sdk.Config) (*sdk.Client, error) {
	cred := credentials.NewEcsRamRoleCredential(cp.RamRoleName)
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func GetClientByPrivateKey(cp *config.Profile, config *sdk.Config) (*sdk.Client, error) {
	cred := credentials.NewRsaKeyPairCredential(cp.PrivateKey, cp.KeyPairName, cp.ExpiredSeconds)
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func GetClientByExternal(cp *config.Profile, config *sdk.Config, ctx *cli.Context) (*sdk.Client, error) {
	args := strings.Fields(cp.ProcessCommand)
	cmd := exec.Command(args[0], args[1:]...)
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, cp)
	if err != nil {
		fmt.Println(cp.ProcessCommand)
		fmt.Println(string(buf))
		return nil, err
	}
	return GetClient(cp, ctx)
}

func GetClientByCredentialsURI(cp *config.Profile, config *sdk.Config, ctx *cli.Context) (*sdk.Client, error) {
	uri := cp.CredentialsURI

	if uri == "" {
		uri = os.Getenv("ALIBABA_CLOUD_CREDENTIALS_URI")
	}

	if uri == "" {
		return nil, fmt.Errorf("invalid credentials uri")
	}

	res, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("get credentials from %s failed, status code %d", uri, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	type Response struct {
		Code            string
		AccessKeyId     string
		AccessKeySecret string
		SecurityToken   string
		Expiration      string
	}
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("unmarshal credentials failed, the body %s", string(body))
	}

	if response.Code != "Success" {
		return nil, fmt.Errorf("get sts token err, Code is not Success")
	}

	cred := credentials.NewStsTokenCredential(response.AccessKeyId, response.AccessKeySecret, response.SecurityToken)
	return sdk.NewClientWithOptions(cp.RegionId, config, cred)
}

func GetClientByChainableRamRoleArn(cp *config.Profile, config *sdk.Config, ctx *cli.Context) (*sdk.Client, error) {
	profileName := cp.SourceProfile

	// 从 configuration 中重新获取 source profile
	source, loaded := cp.GetParent().GetProfile(profileName)
	if !loaded {
		return nil, fmt.Errorf("can not load the source profile: " + profileName)
	}

	client, err := GetClient(&source, ctx)
	if err != nil {
		return nil, err
	}
	accessKeyID, accessKeySecret, StsToken, err := cp.GetSessionCredential(client)
	if err != nil {
		return nil, err
	}
	cred := credentials.NewStsTokenCredential(accessKeyID, accessKeySecret, StsToken)
	return sdk.NewClientWithOptions(cp.RegionId, config, cred)
}

// implementations:
// - RpcInvoker,
// - RpcForceInvoker
// - RestfulInvoker
type Invoker interface {
	getClient() *sdk.Client
	getRequest() *requests.CommonRequest
	Prepare(ctx *cli.Context) error
	Call() (*responses.CommonResponse, error)
}

// implementations
// - Waiter
// - Pager
type InvokeHelper interface {
	CallWith(invoker Invoker) (string, error)
}

// basic invoker to init common object and headers
type BasicInvoker struct {
	profile *config.Profile
	client  *sdk.Client
	request *requests.CommonRequest
	product *meta.Product
}

func NewBasicInvoker(cp *config.Profile) *BasicInvoker {
	return &BasicInvoker{profile: cp}
}

func (a *BasicInvoker) getClient() *sdk.Client {
	return a.client
}

func (a *BasicInvoker) getRequest() *requests.CommonRequest {
	return a.request
}

func (a *BasicInvoker) Init(ctx *cli.Context, product *meta.Product) error {
	var err error
	a.product = product
	a.client, err = GetClient(a.profile, ctx)
	if err != nil {
		return fmt.Errorf("init client failed %s", err)
	}
	if vendorEnv, ok := os.LookupEnv("ALIBABA_CLOUD_VENDOR"); ok {
		a.client.AppendUserAgent("vendor", vendorEnv)
	}
	a.client.AppendUserAgent("Aliyun-CLI", cli.GetVersion())
	a.request = requests.NewCommonRequest()
	a.request.Product = product.Code

	a.request.RegionId = a.profile.RegionId
	if v, ok := config.RegionFlag(ctx.Flags()).GetValue(); ok {
		a.request.RegionId = v
	} else if v, ok := config.RegionIdFlag(ctx.Flags()).GetValue(); ok {
		a.request.RegionId = v
	}

	a.request.Version = product.Version
	if v, ok := VersionFlag(ctx.Flags()).GetValue(); ok {
		a.request.Version = v
	}

	if v, ok := EndpointFlag(ctx.Flags()).GetValue(); ok {
		a.request.Domain = v
	}

	for _, s := range HeaderFlag(ctx.Flags()).GetValues() {
		if k, v, ok := cli.SplitStringWithPrefix(s, "="); ok {
			a.request.Headers[k] = v
			if k == "Accept" {
				if strings.Contains(v, "xml") {
					a.request.AcceptFormat = "XML"
				} else if strings.Contains(v, "json") {
					a.request.AcceptFormat = "JSON"
				}
			}
			if k == "Content-Type" {
				a.request.SetContentType(v)
			}
		} else {
			return fmt.Errorf("invaild flag --header `%s` use `--header HeaderName=Value`", s)
		}
	}

	hint := "you can find it on https://help.aliyun.com"
	if product.Version != "" {
		hint = fmt.Sprintf("please use `aliyun help %s` get more information.", product.GetLowerCode())
	}

	if a.request.Version == "" {
		return cli.NewErrorWithTip(fmt.Errorf("missing version for product %s", product.Code),
			"Use flag `--version <YYYY-MM-DD>` to assign version, "+hint)
	}

	if a.request.RegionId == "" {
		return cli.NewErrorWithTip(fmt.Errorf("missing region for product %s", product.Code),
			"Use flag --region <regionId> to assign region, "+hint)
	}

	if a.request.Domain == "" {
		a.request.Domain, err = product.GetEndpoint(a.request.RegionId, a.client)
		if err != nil {
			return cli.NewErrorWithTip(
				fmt.Errorf("unknown endpoint for %s/%s! failed %s", product.GetLowerCode(), a.request.RegionId, err),
				"Use flag --endpoint xxx.aliyuncs.com to assign endpoint, "+hint)
		}
	}

	return nil
}
