/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import "strings"

//
// This variable is replaced in compile time
// `-ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'"`
var (
	Version = "0.0.1"
)

func GetVersion() string {
	return strings.Replace(Version, " ", "-", -1)
}