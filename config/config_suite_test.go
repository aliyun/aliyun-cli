/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	. "github.com/onsi/ginkgo"
	"testing"
)

func TestConfigSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cli Suite")
}
