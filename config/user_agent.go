package config

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
)

var userAgent = fmt.Sprintf("Aliyun-CLI-V%s", cli.GetVersion())

func GetUserAgent() string {
	return userAgent
}

func SetUserAgent(agent string) {
	userAgent = agent
}