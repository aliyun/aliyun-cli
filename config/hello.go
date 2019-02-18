/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"fmt"
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
	client.AppendUserAgent("Aliyun-CLI", cli.GetVersion())
	err = client.DoAction(request, response)

	if err != nil {
		panic(err)
	}
	cli.Printf(w, " available regions: \n")
	for _, region := range response.Regions.Region {
		cli.Printf(w, "  %s\n", region.RegionId)
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
