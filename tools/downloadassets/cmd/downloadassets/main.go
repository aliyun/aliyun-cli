package main

import (
	"os"

	"github.com/aliyun/aliyun-cli/v3/tools/downloadassets"
)

func main() {
	os.Exit(downloadassets.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
