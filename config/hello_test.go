/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/cli"
)

func TestGetRegions(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
	profile := NewProfile("default")
	ctx.Flags().AddByName("skip-secure-verify")
	regions, err := GetRegions(ctx, &profile)
	assert.Empty(t, regions)
	assert.EqualError(t, err, "AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first")

	profile.AccessKeyId = "AccessKeyId"
	profile.AccessKeySecret = "AccessKeySecret"
	profile.RegionId = "cn-hangzhou"
	regions, err = GetRegions(ctx, &profile)
	assert.Empty(t, regions)
	assert.Nil(t, err)
}

func TestDoHello(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
	ctx.Flags().AddByName("skip-secure-verify")
	profile := NewProfile("default")

	exw := "-----------------------------------------------\n" +
		"!!! Configure Failed please configure again !!!\n" +
		"-----------------------------------------------\n" +
		"AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first\n" +
		"-----------------------------------------------\n" +
		"!!! Configure Failed please configure again !!!\n" +
		"-----------------------------------------------\n"
	DoHello(ctx, &profile)
	assert.Equal(t, exw, w.String())

	w.Reset()
	profile.AccessKeyId = "AccessKeyId"
	profile.AccessKeySecret = "AccessKeySecret"
	profile.RegionId = "cn-hangzhou"
	defer func() {
		err := recover()
		assert.NotNil(t, err)

	}()
	DoHello(ctx, &profile)
}
