/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
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
