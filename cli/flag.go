package cli

import (
	"fmt"
)

type Flag struct {
	Name			string
	ShortName		string
	Usage			string
	DefaultValue	string
	Required 		bool
	Assignable		bool
	Persistent		bool

	assigned		bool
	value			string
	p				*string
}

func (f *Flag) IsAssigned() (bool) {
	return f.assigned
}

func (f *Flag) GetValue() (string) {
	return f.value
}

//
// FlagSet
type FlagSet struct {
	flags	[]Flag
}

func NewFlagSet() (*FlagSet) {
	return &FlagSet{}
}

func (a *FlagSet) Items() ([]Flag) {
	return a.flags
}

func (a *FlagSet) Add(f Flag) {
	a.flags = append(a.flags, f)
}

func (a *FlagSet) PutValue(name string, value string) {
	for _, v := range a.flags {
		if v.Name == name {
			v.value = value
			if v.p != nil {
				*v.p = value
			}
		}
	}
	a.flags = append(a.flags, Flag {
		Name: name,
		value: value,
	})
}

func (a *FlagSet) Get(name string) *Flag {
	for _, v := range a.flags {
		if v.Name == name {
			return &v
		}
	}
	return nil
}

func (a *FlagSet) GetValue(name string) (string, bool) {
	for _, v := range a.flags {
		if v.Name == name {
			return v.value, true
		}
	}
	return "", false
}

func (a *FlagSet) ParseArgs(args []string, unknownFlags *FlagSet) ([]string, error) {
	a2, ff := ParseArgs(args)
	for k, v := range ff {
		f2 := a.Get(k)
		if f2 != nil {
			a.PutValue(k, v)
		} else if unknownFlags == nil {
			return a2, fmt.Errorf("unknown flag --" + k)
		} else {
			unknownFlags.PutValue(k, v)
		}
	}
	return a2, nil
}

func (a *FlagSet) StringVar(p *string, name string, defaultValue string, usage string) {
	a.Add(Flag{
		Name: name,
		Usage: usage,
		DefaultValue: defaultValue,
		Required: false,
		Assignable: true,
		Persistent: false,
		p: p,
	})
}

func (a *FlagSet) PersistentStringVar(p *string, name string, defaultValue string, usage string) {
	a.Add(Flag{
		Name: name,
		Usage: usage,
		DefaultValue: defaultValue,
		Required: false,
		Assignable: true,
		Persistent: true,
		p: p,
	})
}

