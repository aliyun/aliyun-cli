/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"fmt"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newTestFlagSet() (*FlagSet) {
	fs := NewFlagSet()
	fs.Add(Flag{Name: "test", Assignable: AssignedOnce})
	fs.Add(Flag{Name: "test2", Assignable: AssignedOnce})
	fs.Add(Flag{Name: "prev", Assignable: AssignedNone})
	fs.Add(Flag{Name: "test-required", Required: true})
	return fs
}

func newTestParser(args ...string) (*Parser, *FlagSet) {
	fs := newTestFlagSet()
	parser := NewParser(args, func(s string) (*Flag, error) {
		f := fs.Get(s)
		if f != nil {
			return f, nil
		} else {
			return f, fmt.Errorf("unknown flag --%s", s)
		}
	})
	return parser, fs
}

var _ = ginkgo.Describe("Parser", func() {
	ginkgo.It("can parse args and flags", func() {
		parser, fs := newTestParser("s1", "s2", "--test", "aaa", "--test2=bbb")

		ginkgo.By("first arg")
		s, _, err := parser.ReadNextArg()
		Expect(err).NotTo(HaveOccurred())
		Expect(s).Should(Equal("s1"))
		Expect(fs.assignedCount()).Should(Equal(0))

		ginkgo.By("remain args")
		s2, err := parser.ReadAll()
		Expect(err).NotTo(HaveOccurred())
		Expect(len(s2)).Should(Equal(1))
		Expect(s2[0]).Should(Equal("s2"))
		v1, _ := fs.GetValue("test")
		Expect(v1).Should(Equal("aaa"))
		v2, _ := fs.GetValue("test2")
		Expect(v2).Should(Equal("bbb"))
	})

	ginkgo.It("can read next arg skip prev flag", func() {
		parser, fs := newTestParser("--prev", "s1", "s2")
		s, _, err := parser.ReadNextArg()

		Expect(err).NotTo(HaveOccurred())
		Expect(s).Should(Equal("s1"))
		Expect(fs.Get("prev")).ShouldNot(Equal(nil))
	})
})


//var _ = ginkgo.Describe("Parser", func() {
//	ginkgo.It("can parse args", func() {
//		// parser := NewParser([]string{"1", "2", "3"},str)
//		//	return &Flag {
//		//		Name: name,
//		//	}, nil
//		//})
//		//parser.ParseAll()
//		//Expect(len(parser.GetResultArgs())).Should(Equal(3))
//		parser := NewParser([]string{"1", "2", "3"}, func(s string) (*Flag, error) {
//			return nil, nil
//		})
//		a, more, err := parser.ReadNextArg()
//		Expect(err).NotTo(HaveOccurred())
//		Expect(more).Should(Equal(true))
//		Expect(a).Should(Equal("1"))
//	})
//	ginkgo.It("can parse args with flags", func() {
//
//
//	})
//})