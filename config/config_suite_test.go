/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"testing"
)

func TestConfigSuite(t *testing.T) {
	gomega.RegisterFailHandler(Fail)
	RunSpecs(t, "Cli Suite")
}
