package main

import (
	"os"

	"github.com/aliyun/aliyun-cli/v3/tools/releasever"
)

func main() {
	os.Exit(releasever.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
