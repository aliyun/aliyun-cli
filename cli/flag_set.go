// Package cli Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
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
	"strings"
)

type FlagSet struct {
	// flags
	flags []*Flag
	// key: "--name" "-c"
	index map[string]*Flag
}

func NewFlagSet() *FlagSet {
	return &FlagSet{
		flags: make([]*Flag, 0),
		index: make(map[string]*Flag),
	}
}

// traverse all values
func (fs *FlagSet) Flags() []*Flag {
	return fs.flags
}

// call from user program, if flag duplicated, panic!
func (fs *FlagSet) Add(f *Flag) {
	f.checkValid()
	for _, s := range f.GetFormations() {
		if _, ok := fs.index[s]; ok {
			panic(fmt.Errorf("flag duplicated %s", string(s)))
		}
		fs.index[s] = f
	}
	fs.flags = append(fs.flags, f)
}

// add by name, not support add by shorthand
func (fs *FlagSet) AddByName(name string) (*Flag, error) {
	if _, ok := fs.index["--"+name]; ok {
		return nil, fmt.Errorf("flag duplicated --%s", name)
	}
	f := &Flag{
		Name: name,
	}
	if strings.HasSuffix(name, "-FILE") {
		f.Aliases = append(f.Aliases, strings.TrimSuffix(name, "-FILE"))
	}
	fs.Add(f)
	return f, nil
}

// Get fetch flag by name, sample --name
func (fs *FlagSet) Get(name string) *Flag {
	if fs == nil || fs.index == nil {
		return nil
	}
	if f, ok := fs.index["--"+name]; ok {
		return f
	}
	return nil
}

// get flag by shorthand, sample -a
func (fs *FlagSet) GetByShorthand(c rune) *Flag {
	if f, ok := fs.index["-"+string(c)]; ok {
		return f
	}
	return nil
}

// get suggestions
func (fs *FlagSet) GetSuggestions(name string, distance int) []string {
	sr := NewSuggester(name, distance)
	for k := range fs.index {
		sr.Apply(k)
	}
	ss := make([]string, 0)
	ss = append(ss, sr.GetResults()...)

	return ss
}

// get value by flag name, not recommended
func (fs *FlagSet) GetValue(name string) (string, bool) {
	f := fs.Get(name)
	if f == nil {
		return "", false
	}
	return f.GetValue()
}

// merge FlagSet with from
func (fs *FlagSet) mergeWith(from *FlagSet, applier func(f *Flag) bool) *FlagSet {
	if from == nil {
		return fs
	}
	r := NewFlagSet()
	if fs != nil {
		for _, f := range fs.flags {
			r.Add(f)
		}
	}
	for _, rv := range from.Flags() {
		if applier(rv) {
			r.put(rv)
		}
	}
	return r
}

// put flag, replace old value if duplicated
func (fs *FlagSet) put(f *Flag) {

	for _, lv := range fs.flags {
		if lv == f {
			return
		}
	}
	fs.flags = append(fs.flags, f)
	for _, s := range f.GetFormations() {
		fs.index[s] = f
	}
}
