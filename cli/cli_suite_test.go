/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestCliSuite(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Cli Suite")
}