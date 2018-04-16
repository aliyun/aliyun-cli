/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
)

//
//
type Plugin struct {
	Category string
	Name string
	Usage string
	flags *cli.FlagSet

	Run func(ctx *cli.Context) error
	Help func(ctx *cli.Context) error
	AutoComplete func(ctx *cli.Context) []string
}

func (a *Plugin) Flags() *cli.FlagSet {
	if a.flags == nil {
		a.flags = cli.NewFlagSet()
	}
	return a.flags
}
