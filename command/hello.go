package command

import (
	"fmt"
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

func DoHello() {
	 s, err := CallOpenApi("Ecs", "DescribeRegions", map[string]string{})
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

	 fmt.Printf("Configuring...")
	 var resp ecs.DescribeRegionsResponse
	 err = json.Unmarshal([]byte(s), &resp)
	 if err != nil {
	 	panic(err)
	 }
	 fmt.Printf(" available regions: \n")
	 for _, region := range resp.Regions.Region {
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
.........+88888888 ..Command Line Interface(Reloaded) v0.15..O8888888D..........
.........+88888888...........................................88888888D..........
..........D888888888888DO+. ..........................?ND888888888888D..........
...........O8888888888888888888888...........D8888888888888888888888=...........
............ .:D8888888888888888888.........78888888888888888888O ..............`)
