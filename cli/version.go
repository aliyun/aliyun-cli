/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"github.com/aliyun/aliyun-cli/i18n"
	"io"
	"strings"
)

//
// This variable is replaced in compile time
// `-ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'"`
var (
	Version = "0.0.1"
)

func GetVersion() string {
	return strings.Replace(Version, " ", "-", -1)
}

func NewVersionCommand(w io.Writer) *Command {
	return &Command{
		Name:   "version",
		Short:  i18n.T("print current version", "打印当前版本号"),
		Hidden: true,
		Run: func(ctx *Context, args []string) error {
			Printf(w, "%s\n", Version)
			return nil
		},
	}
}
