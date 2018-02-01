package config

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

func DoHello(profile *Profile) {
	client, err := profile.GetClient()

	if err != nil {
		fmt.Println("-----------------------------------------------")
		fmt.Println("!!! Configure Failed please configure again !!!")
		fmt.Println("-----------------------------------------------")
		fmt.Println(err)
		fmt.Println("-----------------------------------------------")
		fmt.Println("!!! Configure Failed please configure again !!!")
		fmt.Println("-----------------------------------------------")
		return
	}
	request := ecs.CreateDescribeRegionsRequest()
	response := ecs.CreateDescribeRegionsResponse()
	err = client.DoAction(request, response)

	if err != nil {
		panic(err)
	}
	fmt.Printf(" available regions: \n")
	for _, region := range response.Regions.Region {
		fmt.Printf("  %s\n", region.RegionId)
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
.........+88888888 ..Command Line Interface(Reloaded) v0.31..O8888888D..........
.........+88888888...........................................88888888D..........
..........D888888888888DO+. ..........................?ND888888888888D..........
...........O8888888888888888888888...........D8888888888888888888888=...........
............ .:D8888888888888888888.........78888888888888888888O ..............`)
