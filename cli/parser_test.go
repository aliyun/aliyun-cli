package cli

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parser", func() {
	It("can parse args", func() {
		r1, r2 := ParseArgs([]string{"1", "2", "3"})
		Expect(len(r1)).Should(Equal(3))
		Expect(r1[0]).Should(Equal("1"))
		Expect(r1[1]).Should(Equal("2"))
		Expect(r1[2]).Should(Equal("3"))
		Expect(len(r2)).Should(Equal(0))
	})
})