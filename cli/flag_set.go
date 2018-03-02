/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
)

//
// to parser to
type FlagDetector interface {
	DetectFlag(name string) (*Flag, error)
}

type FlagSet struct {
	flags	[]Flag
}

func NewFlagSet() (*FlagSet) {
	return &FlagSet{
		flags: make([]Flag, 0),
	}
}

//
// traverse all values
func (a *FlagSet) Flags() ([]Flag) {
	return a.flags
}

//
// call from user program, if flag duplicated, panic!
func (a *FlagSet) Add(flag Flag) (*Flag) {
	for _, f := range a.flags {
		if f.Name == flag.Name {
			panic(fmt.Errorf("flag duplicated: --%s", f.Name))
		}
	}
	a.flags = append(a.flags, flag)
	return &a.flags[len(a.flags) - 1]
}

func (a *FlagSet) AddByName(name string) (*Flag, error) {
	for _, f := range a.flags {
		if f.Name == name {
			return nil, fmt.Errorf("flag duplicated --%s", f.Name)
		}
	}
	f := Flag {
		Name: name,
		Assignable: true,
	}
	a.flags = append(a.flags, f)
	return &a.flags[len(a.flags) - 1], nil
}

//
// put flags if duplicated replace
func (a *FlagSet) Put(flag Flag) {
	for i, f := range a.flags {
		if f.Name == flag.Name {
			a.flags[i] = flag
			return
		}
	}
	a.flags = append(a.flags, flag)
}

//
// get flag by name
func (a *FlagSet) Get(name string) *Flag {
	for i, v := range a.flags {
		if v.Name == name {
			return &a.flags[i]
		}
	}
	return nil
}

//
// get suggestions
func (a *FlagSet) GetSuggestions(name string, distance int) []string {
	sr := NewSuggester(name, distance)
	for _, v := range a.flags {
		sr.Apply(v.Name)
	}
	return sr.GetResults()
}

// check if the flag is assigned
func (a *FlagSet) IsAssigned(name string) bool {
	f := a.Get(name)
	if f == nil {
		return false
	} else {
		return f.IsAssigned()
	}
}

// get value by flag name
func (a *FlagSet) GetValue(name string) (string, bool) {
	for _, f := range a.flags {
		if f.Name == name {
			if f.IsAssigned() {
				return f.value, true
			} else {
				return f.DefaultValue, false
			}
		}
	}
	return "", false
}

// get value by flag name, if not assigned return default value
func (a *FlagSet) GetValueOrDefault(name string, def string) string {
	v, ok := a.GetValue(name)
	if ok {
		return v
	} else {
		return def
	}
}

// put value
// return: if duplicated return error
func (a *FlagSet) PutValue(name string, value string) (error) {
	f := a.Get(name)
	if f != nil {
		return f.PutValue(value)
	} else {
		f := Flag {
			Name:  name,
			value: value,
		}
		a.flags = append(a.flags, f)
		return nil
	}
}

func MergeFlagSet(to *FlagSet, from *FlagSet, applier func(flag Flag) bool) *FlagSet {
	result := &FlagSet {}
	result.flags = to.flags
	if from != nil {
		for _, rv := range from.Flags() {
			if applier(rv) {
				result.Put(rv)
			}
		}
	}
	return result
}

func (a *FlagSet) StringVar(p *string, name string, defaultValue string, usage *i18n.Text) (*Flag) {
	return a.Add(Flag{
		Name: name,
		Usage: usage,
		DefaultValue: defaultValue,
		Required: false,
		Assignable: true,
		Persistent: false,
		p: p,
	})
}

func (a *FlagSet) PersistentStringVar(p *string, name string, defaultValue string, usage *i18n.Text) (*Flag) {
	return a.Add(Flag{
		Name: name,
		Usage: usage,
		DefaultValue: defaultValue,
		Required: false,
		Assignable: true,
		Persistent: true,
		p: p,
	})
}

// get assigned count for flags
func (a *FlagSet) assignedCount() int {
	n := 0
	for _, f := range a.flags {
		if f.assigned {
			n++
		}
	}
	return n
}

type InvalidFlagError struct {
	Name string
	ctx *Context
}

func (e *InvalidFlagError) Error() string {
	return fmt.Sprintf("invalid flag --%s", e.Name)
}

func (e *InvalidFlagError) GetSuggestions() []string {
	distance := e.ctx.command.GetSuggestDistance()
	return e.ctx.Flags().GetSuggestions(e.Name, distance)
}