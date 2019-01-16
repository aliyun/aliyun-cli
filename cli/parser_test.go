/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"fmt"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testContext struct {
	fs *FlagSet
}

func (tc *testContext) detectFlag(name string) (*Flag, error) {
	f := tc.fs.Get(name)
	if f != nil {
		return f, nil
	}
	return nil, fmt.Errorf("unknown flag --%s", name)

}

func (tc *testContext) detectFlagByShorthand(ch rune) (*Flag, error) {
	f := tc.fs.GetByShorthand(ch)
	if f != nil {
		return f, nil
	}
	return nil, fmt.Errorf("unknown flag -%c", ch)

}

func newTestContext() *testContext {
	fs := NewFlagSet()
	fs.Add(&Flag{Name: "test", AssignedMode: AssignedOnce})
	fs.Add(&Flag{Name: "test2", Shorthand: 't', AssignedMode: AssignedOnce})
	fs.Add(&Flag{Name: "prev", AssignedMode: AssignedNone})
	fs.Add(&Flag{Name: "test-required", Required: true})
	return &testContext{fs: fs}
}

func newTestParser(args ...string) (*Parser, *FlagSet) {
	tc := newTestContext()
	parser := NewParser(args, tc)
	return parser, tc.fs
}

var _ = ginkgo.Describe("Parser", func() {
	ginkgo.It("1. can parse command args", func() {
		parser, _ := newTestParser()

		flag, v, err := parser.parseCommandArg("--test")
		Expect(err).NotTo(HaveOccurred())
		Expect(flag.Name).Should(Equal("test"))
		Expect(v).Should(Equal(""))

		flag, v, err = parser.parseCommandArg("-t")
		Expect(err).NotTo(HaveOccurred())
		Expect(flag.Name).Should(Equal("test2"))
		Expect(v).Should(Equal(""))

		flag, v, err = parser.parseCommandArg("-t=ccc")
		Expect(err).NotTo(HaveOccurred())
		Expect(flag.Name).Should(Equal("test2"))
		Expect(v).Should(Equal("ccc"))

		flag, v, err = parser.parseCommandArg("-t:ccc")
		Expect(err).NotTo(HaveOccurred())
		Expect(flag.Name).Should(Equal("test2"))
		Expect(v).Should(Equal("ccc"))

		flag, v, err = parser.parseCommandArg("--test2:ccc")
		Expect(err).NotTo(HaveOccurred())
		Expect(flag.Name).Should(Equal("test2"))
		Expect(v).Should(Equal("ccc"))

		flag, v, err = parser.parseCommandArg("ccc")
		Expect(err).NotTo(HaveOccurred())
		Expect(flag).Should(BeNil())
		Expect(v).Should(Equal("ccc"))

		flag, v, err = parser.parseCommandArg("ccc=aaa")
		Expect(err).NotTo(HaveOccurred())
		Expect(flag).Should(BeNil())
		Expect(v).Should(Equal("ccc=aaa"))
	})

	ginkgo.It("2. can parse args and flags", func() {
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
		//v1, _ := fs.G("test")
		//Expect(v1).Should(Equal("aaa"))
		//v2, _ := fs.GetValue("test2")
		//Expect(v2).Should(Equal("bbb"))
	})

	ginkgo.It("3. can read next arg skip prev flag", func() {
		parser, fs := newTestParser("--prev", "s1", "s2")
		s, _, err := parser.ReadNextArg()

		Expect(err).NotTo(HaveOccurred())
		Expect(s).Should(Equal("s1"))
		Expect(fs.Get("prev")).ShouldNot(Equal(nil))
	})

	//	Testcase TODO
	// ginkgo.It("4. can read fields", func() {
	// 	parser, fs := newTestParser("--waiter", "expr=aaa", "to=bbb")
	// 	s, _, err := parser.ReadNextArg()

	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(s).Should(Equal(""))
	// 	Expect(fs.Get("prev")).ShouldNot(Equal(nil))
	// })
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
