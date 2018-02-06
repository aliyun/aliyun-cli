/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"testing"
	"gopkg.in/ini.v1"
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