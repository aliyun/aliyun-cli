// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

// 1. can parse command args
func TestParser1(t *testing.T) {
	parser, _ := newTestParser()

	flag, v, err := parser.parseCommandArg("--test")
	assert.Nil(t, err)
	assert.Equal(t, "test", flag.Name)
	assert.Equal(t, "", v)

	flag, v, err = parser.parseCommandArg("-t")
	assert.Nil(t, err)
	assert.Equal(t, "test2", flag.Name)
	assert.Equal(t, "", v)

	flag, v, err = parser.parseCommandArg("-t=ccc")
	assert.Nil(t, err)
	assert.Equal(t, "test2", flag.Name)
	assert.Equal(t, "ccc", v)

	flag, v, err = parser.parseCommandArg("-t:ccc")
	assert.Nil(t, err)
	assert.Equal(t, "test2", flag.Name)
	assert.Equal(t, "ccc", v)

	flag, v, err = parser.parseCommandArg("--test2:ccc")
	assert.Nil(t, err)
	assert.Equal(t, "test2", flag.Name)
	assert.Equal(t, "ccc", v)

	flag, v, err = parser.parseCommandArg("ccc")
	assert.Nil(t, err)
	assert.Nil(t, flag)
	assert.Equal(t, "ccc", v)

	flag, v, err = parser.parseCommandArg("ccc=aaa")
	assert.Nil(t, err)
	assert.Nil(t, flag)
	assert.Equal(t, "ccc=aaa", v)

	_, _, err = parser.parseCommandArg("--")
	assert.NotNil(t, err)
	assert.Equal(t, "not support '--' in command line", err.Error())

	_, _, err = parser.parseCommandArg("-")
	assert.NotNil(t, err)
	assert.Equal(t, "not support flag form -", err.Error())

	// more than two dashes, treat as value
	_, v, err = parser.parseCommandArg("---a")
	assert.Nil(t, err)
	assert.Equal(t, "---a", v)

	// contain ==
	_, v, err = parser.parseCommandArg("----a==2")
	assert.Nil(t, err)
	assert.Equal(t, "----a==2", v)
}

// 2. can parse args and flags
func TestParser2(t *testing.T) {
	parser, _ := newTestParser("s1", "s2", "--test", "aaa", "--test2=bbb")

	s, _, err := parser.ReadNextArg()
	assert.Nil(t, err)
	assert.Equal(t, "s1", s)

	s2, err := parser.ReadAll()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(s2))
	assert.Equal(t, "s2", s2[0])
}

// 3. can read next arg skip prev flag
func TestParser3(t *testing.T) {
	parser, fs := newTestParser("--prev", "s1", "s2")
	s, _, err := parser.ReadNextArg()

	assert.Nil(t, err)
	assert.Equal(t, "s1", s)
	assert.NotNil(t, fs.Get("prev"))
}

func TestParser4(t *testing.T) {
	parser, _ := newTestParser("oss", "ls")
	s, f, err := parser.ReadNextArg()

	assert.Nil(t, err)
	assert.Equal(t, "oss", s)
	assert.True(t, f)
	remains := parser.GetRemains()
	assert.Equal(t, []string{"ls"}, remains)

	s, f, err = parser.ReadNextArg()
	assert.Nil(t, err)
	assert.Equal(t, "ls", s)
	assert.True(t, f)
	remains = parser.GetRemains()
	assert.Equal(t, []string{}, remains)

	s, f, err = parser.ReadNextArg()
	assert.Nil(t, err)
	assert.Equal(t, "", s)
	assert.False(t, f)
	remains = parser.GetRemains()
	assert.Equal(t, []string{}, remains)
}

// aliyun oss cp -r oss-url local-path
// aliyun oss cp -r local-path oss-url
// aliyun oss -e oss-cn-beijing.aliyuncs.com ls oss://bucket-name
