/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
)

type FlagSet struct {
	flags []*Flag
	index map[string]*Flag
}

func NewFlagSet() *FlagSet {
	return &FlagSet{
		flags: make([]*Flag, 0),
		index: make(map[string]*Flag),
	}
}

//
// traverse all values
func (fs *FlagSet) Flags() []*Flag {
	return fs.flags
}

//
// call from user program, if flag duplicated, panic!
func (fs *FlagSet) Add(f *Flag) {
	for _, s := range f.GetFormations() {
		if _, ok := fs.index[s]; ok {
			panic(fmt.Errorf("flag duplicated -%s", string(s)))
		}
		fs.index[s] = f
	}
	fs.flags = append(fs.flags, f)
}

//
// add by name, not support add by shorthand
func (fs *FlagSet) AddByName(name string) (*Flag, error) {
	if _, ok := fs.index["--" + name]; ok {
		return nil, fmt.Errorf("flag duplicated --%s", name)
	}
	f := &Flag{
		Name: name,
	}
	fs.Add(f)
	return f, nil
}


//
// get flag by name, sample --name
func (fs *FlagSet) Get(name string) *Flag {
	if f, ok := fs.index["--" + name]; ok {
		f.formation = "--" + name
		return f
	} else {
		return nil
	}
}

//
// get flag by shorthand, sample -a
func (fs *FlagSet) GetByShorthand(c rune) *Flag {
	if f, ok := fs.index["-" + string(c)]; ok {
		f.formation = "-" + string(c)
		return f
	} else {
		return nil
	}
}


//
// get suggestions
func (fs *FlagSet) GetSuggestions(name string, distance int) []string {
	sr := NewSuggester(name, distance)
	for k := range fs.index {
		sr.Apply(k)
	}
	ss := make([]string, 0)
	for _, s := range sr.GetResults() {
		ss = append(ss, s)
	}
	return ss
}

// get value by flag name, not recommanded
func (fs *FlagSet) GetValue(name string) (string, bool) {
	f := fs.Get(name)
	if f == nil {
		return "", false
	} else {
		return f.GetValue()
	}
}


//
//// get value by flag name, if not assigned return default value
//func (a *FlagSet) GetValueOrDefault(name string, def string) string {
//	v, ok := a.GetValue(name)
//	if ok {
//		return v
//	} else {
//		return def
//	}
//}

//// put value
//// return: if duplicated return error
//func (a *FlagSet) PutValue(name string, value string) error {
//	f := a.Get(name)
//	if f != nil {
//		return f.putValue(value)
//	} else {
//		f := Flag{
//			Name:  name,
//			value: value,
//		}
//		a.flags = append(a.flags, f)
//		return nil
//	}
//}


func (fs *FlagSet) StringVar(p *string, name string, defaultValue string, usage *i18n.Text) *Flag {
	f := &Flag{
		Name:         name,
		Usage:        usage,
		DefaultValue: defaultValue,
		Required:     false,
		AssignedMode: AssignedOnce,
		Persistent:   false,
		p:            p,
	}
	fs.Add(f)
	return f
}

func (fs *FlagSet) PersistentStringVar(p *string, name string, defaultValue string, usage *i18n.Text) *Flag {
	f := &Flag{
		Name:         name,
		Usage:        usage,
		DefaultValue: defaultValue,
		Required:     false,
		AssignedMode: AssignedOnce,
		Persistent:   true,
		p:            p,
	}
	fs.Add(f)
	return f
}

// get assigned count for flags
func (fs *FlagSet) assignedCount() int {
	n := 0
	for _, f := range fs.flags {
		if f.assigned {
			n++
		}
	}
	return n
}

//
// merge FlagSet with from
func (fs *FlagSet) mergeWith(from *FlagSet, applier func(f *Flag) bool) (*FlagSet) {
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