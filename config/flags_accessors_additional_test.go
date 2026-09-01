package config

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
)

func TestPreviouslyUncalledFlagAccessors(t *testing.T) {
	flags := cli.NewFlagSet()
	for _, flag := range []*cli.Flag{
		{Name: SourceProfileFlagName},
		{Name: RegionIdFlagName},
		{Name: SkipSecureVerifyName},
		{Name: OAuthSiteTypeName},
	} {
		flags.Add(flag)
	}
	assert.Equal(t, SourceProfileFlagName, SourceProfileFlag(flags).Name)
	assert.Equal(t, RegionIdFlagName, RegionIdFlag(flags).Name)
	assert.Equal(t, SkipSecureVerifyName, SkipSecureVerify(flags).Name)
	assert.Equal(t, OAuthSiteTypeName, OAuthSiteTypeFlag(flags).Name)
}
