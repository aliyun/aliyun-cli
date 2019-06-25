// Copyright 1999-2019 Alibaba Group Holding Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"fmt"
	"os"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/aliyun-cli/cli"
)

type Region struct {
	RegionId  string
	LocalName string
}

func GetRegions(ctx *cli.Context, profile *Profile) ([]Region, error) {
	client, err := profile.GetClient(ctx)

	regions := make([]Region, 0)
	if err != nil {
		return regions, err
	}

	request := ecs.CreateDescribeRegionsRequest()
	response := ecs.CreateDescribeRegionsResponse()
	if vendorEnv, ok := os.LookupEnv("ALIBABA_CLOUD_VENDOR"); ok {
		client.AppendUserAgent("vendor", vendorEnv)
	}
	client.AppendUserAgent("Aliyun-CLI", cli.GetVersion())
	err = client.DoAction(request, response)

	for _, region := range response.Regions.Region {
		regions = append(regions, Region{
			RegionId:  region.RegionId,
			LocalName: region.LocalName,
		})
	}
	return regions, nil
}

func DoHello(ctx *cli.Context, profile *Profile) {
	w := ctx.Writer()

	client, err := profile.GetClient(ctx)

	if err != nil {
		cli.Println(w, "-----------------------------------------------")
		cli.Println(w, "!!! Configure Failed please configure again !!!")
		cli.Println(w, "-----------------------------------------------")
		cli.Println(w, err)
		cli.Println(w, "-----------------------------------------------")
		cli.Println(w, "!!! Configure Failed please configure again !!!")
		cli.Println(w, "-----------------------------------------------")
		return
	}
	request := ecs.CreateDescribeRegionsRequest()
	response := ecs.CreateDescribeRegionsResponse()
	if vendorEnv, ok := os.LookupEnv("ALIBABA_CLOUD_VENDOR"); ok {
		client.AppendUserAgent("vendor", vendorEnv)
	}
	client.AppendUserAgent("Aliyun-CLI", cli.GetVersion())
	err = client.DoAction(request, response)

	if err != nil {
		cli.Println(w, "-----------------------------------------------")
		cli.Println(w, "!!! Configure Failed please configure again !!!")
		cli.Println(w, "-----------------------------------------------")
		cli.Println(w, err)
		cli.Println(w, "-----------------------------------------------")
		cli.Println(w, "!!! Configure Failed please configure again !!!")
		cli.Println(w, "-----------------------------------------------")
		return
	}
	fmt.Println(icon)
}

var icon = string(`
Configure Done!!!
..............888888888888888888888 ........=8888888888888888888D=..............
...........88888888888888888888888 ..........D8888888888888888888888I...........
.........,8888888888888ZI: ...........................=Z88D8888888888D..........
.........+88888888 ..........................................88888888D..........
.........+88888888 .......Welcome to use Alibaba Cloud.......O8888888D..........
.........+88888888 ............. ************* ..............O8888888D..........
.........+88888888 .... Command Line Interface(Reloaded) ....O8888888D..........
.........+88888888...........................................88888888D..........
..........D888888888888DO+. ..........................?ND888888888888D..........
...........O8888888888888888888888...........D8888888888888888888888=...........
............ .:D8888888888888888888.........78888888888888888888O ..............`)
