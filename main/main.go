package main

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/command"
	"os"
)

var usage = `
	Alibaba Cloud CLI(Command Line Interface)
	Usage:
`


/**
## Configure

$ aliyuncli configure
	Aliyun Access Key ID [****************wQ7v]:
	Aliyun Access Key Secret [****************fxGu]:
	Default Region Id [cn-hangzhou]:
	Default output format [json]:

## OpenApi mode
	$ aliyuncli Ecs DescribeInstances
	$ aliyuncli Ecs StartInstance --InstanceId your_instance_id
	$ aliyuncli Rds DescribeDBInstances

## 用HTTPS(SSL/TLS)通信

	$ aliyuncli Ecs DescribeInstances --secure


 */
func main() {
	rootCmd := &cli.Command{
		Name: "aliyun",
		Short: "Alibaba Cloud Command Line Interface",
	}
	rootCmd.AddSubCommand(command.NewConfigureCommand())
	rootCmd.EnableUnknownFlags()
 	command.InitOpenApiCaller(rootCmd)

	rootCmd.Execute(os.Args[1:])
}