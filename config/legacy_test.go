/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/ini.v1"
	"testing"
)

var lcfgText = []byte(`
[default]
aliyun_access_key_secret = *************************
aliyun_access_key_id = **************************
[profile aaa]
aliyun_access_key_secret = ddf
aliyun_access_key_id = sdf
`)

func TestLoadLegacyConfiguration(t *testing.T) {
	cfg, err := ini.Load(lcfgText)
	if err != nil {
		t.Errorf(" parse failed: %v\n", err)
	}
	for _, s := range cfg.Sections() {
		t.Logf("Section: %s\n", s.Name())
		k, e := s.GetKey("aliyun_access_key_id")
		if e == nil {
			t.Logf("AccessKeyId=%s\n", k.String())
		}
		k2, e := s.GetKey("aliyun_access_key_secret")
		if e == nil {
			t.Logf("AccessKeySecret=%s\n", k2.String())
		}
	}
	//err := gcfg.ReadStringInto(&cfg, string(lcfgText))

	//t.Logf("%v\n", cfg)
}

func TestMigrateCredentials(t *testing.T) {
	exProfiles := []Profile{
		Profile{
			Name:            "default",
			Mode:            AK,
			AccessKeyId:     "**************************",
			AccessKeySecret: "*************************",
			OutputFormat:    "json",
		},
		Profile{
			Name:            "aaa",
			Mode:            AK,
			AccessKeyId:     "sdf",
			AccessKeySecret: "ddf",
			OutputFormat:    "json",
		},
	}

	cp, err := MigrateCredentials("../figures/credentials.ini")
	assert.Nil(t, err)
	assert.Equal(t, "default", cp.CurrentProfile)
	assert.Equal(t, "", cp.MetaPath)

	assert.Len(t, cp.Profiles, 2)
	assert.Subset(t, cp.Profiles, exProfiles)
}
