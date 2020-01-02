// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cli

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

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
	})

	ginkgo.It("3. can read next arg skip prev flag", func() {
		parser, fs := newTestParser("--prev", "s1", "s2")
		s, _, err := parser.ReadNextArg()

		Expect(err).NotTo(HaveOccurred())
		Expect(s).Should(Equal("s1"))
		Expect(fs.Get("prev")).ShouldNot(Equal(nil))
	})
})

func TestSpliString(t *testing.T) {
	sli := SplitString("nihao-Mrx", "-")
	assert.Len(t, sli, 2)
}

func TestUnquoteString(t *testing.T) {
	str := UnquoteString(`"nicai"`)
	assert.Equal(t, "nicai", str)
}
