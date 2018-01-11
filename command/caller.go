package command

import (
	"github.com/aliyun/aliyun-cli/cli"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/core"
	"github.com/aliyun/aliyun-cli/meta"
)

var products = meta.LoadProfile()

func InitOpenApiCaller(cmd *cli.Command) {
	cmd.Run = func(c *cli.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("usage aliyun [product] API")
		}

		product := args[0]
		api := args[1]
		parameters := make(map[string]string)

		for _, v := range c.UnknownFlags().Items() {
			parameters[v.Name] = v.GetValue()
		}

		return CallOpenApi(product, api, parameters)
	}
}

func CallOpenApi(product string, api string, parameters map[string]string) error {
	conf, err := core.LoadConfiguration()
	if err != nil {
		return err
	}
	cp := conf.GetCurrentProfile()

	client, err := sdk.NewClientWithAccessKey(cp.RegionId, cp.AccessKeyId, cp.AccessKeySecret)

	if err != nil {
		fmt.Errorf("failed with new client %v", err)
	}

	productInfo, ok := products[product]
	if !ok {
		return fmt.Errorf("unknown product %s", product)
	}

	request := requests.NewCommonRequest()
	request.RegionId = cp.RegionId
	request.Product = product
	request.ApiName = api
	request.Version = productInfo.MajorVersion

	for k, v := range parameters {
		switch k {
		case "region":
			request.RegionId = v
		case "endpoint":
			request.Domain = v
		case "version":
			request.Version = v
		default:
			request.QueryParams[k] = v
		}
	}

	if request.Domain == "" {
		request.Domain, err = productInfo.GetEndpoint(request.RegionId, client)
		if err != nil {
			return err
		}
	}

	resp, err := client.ProcessCommonRequest(request)
	if err != nil {
		return fmt.Errorf("failed with new call %v", err)
	}

	fmt.Println(resp.GetHttpContentString())
	return nil
}

//
//func NewOpenApiCommand() *cli.Command {
//	c := &cli.Command{
//		Name: "",
//		Usage: "",
//
//		Run: func(c *cli.Command, args []string) error {
//			func CallOpenApi()
//		}
//		Help: func(c *cli.Command, args []string, writer io.Writer) {
//
//		}
//	}
//	c.Flags().AddFlag()
//	c.Flags().AddFlag()
//	return c
//}
//
//func CallOpenApi(product string, api string, parameters map[string]string) {
//
//}

//
//import (
//	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
//	"fmt"
//	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
//	"github.com/aliyun/aliyun-cli/core"
//	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
//	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
//	"log"
//)
//
//type Caller struct {
//	config Configuration
//}
//
//
//
//func (Caller) Call(args Argument) {
//	config, err := cli.LoadConfigFromFile("/Users/ailan/.aliyuncli.conf")
//	fmt.Println("%v", config)
//
//	if err != nil {
//		fmt.Println("failed with load configuration %v", err)
//		return
//	}
//
//	client, err := sdk.NewClientWithAccessKey(config.RegionId, config.AccessKeyId, config.AccessKeySecret)
//	if err != nil {
//		fmt.Errorf("failed with new client %v", err)
//	}
//	// client := ecs.NewClient(config.AccessKeyId, config.AccessKeySecret)
//	// args := ecs.DescribeRegionsArgs{}
//
//	// regions, err := client.DescribeRegions()
//
//	request := requests.NewCommonRequest()
//	request.SetDomain("ecs.aliyuncs.com")
//	request.Version = "2014-05-26"
//	request.RegionId = config.RegionId
//	request.Product = "ECS"
//	request.ApiName = "DescribeRegions"
//	resp, err := client.ProcessCommonRequest(request)
//
//	if err != nil {
//		fmt.Println("failed with new call %v", err)
//		return
//	}
//
//	fmt.Println(resp.GetHttpContentString())
//
//	//for _, r := range regions {
//	//	fmt.Println("%v", r)
//	//}
//	//
//	fmt.Println("Alibaba Cloud Command Line Interface")
//}
