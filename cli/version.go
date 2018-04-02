/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

//
// This variable is replaced in compile time
// `-ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'"`
var (
	Version = "0.0.1"
)
