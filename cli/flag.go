/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"strings"
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
)

type Flag struct {
	// Name of flag --{Name}
	Name			string

	// Flag is the single characters
	SingleChars		string

	// Help message
	Usage			*i18n.Text

	// default value
	DefaultValue	string

	// true if required
	Required 		bool

	//
	Assignable		bool

	// can to child commands
	Persistent		bool

	//
	Repeatable		bool

	//
	Hidden			bool

	assigned		bool
	value			string
	values			[]string
 	p				*string
}

func (f *Flag) IsAssigned() (bool) {
	return f.assigned
}

func (f *Flag) GetValue() (string) {
	return f.value
}

//
// for Repeatable flag, return values
func (f *Flag) GetValues() []string {
	return f.values
}

func (f *Flag) PutValue(v string) error {
	if !f.assigned {
		f.assigned = true
		f.value = v
		if f.p != nil {
			*f.p = v
		}
	} else if !f.Repeatable {
		return fmt.Errorf("flag duplucated: --%s", f.Name)
	}
	if f.Repeatable {
		f.values = append(f.values, v)
	}
	return nil
}

func (f *Flag) UseDefaultValue() bool {
	if f.DefaultValue != "" {
		f.value = f.DefaultValue
		if f.p != nil {
			*f.p = f.DefaultValue
		}
		return true
	} else {
		return false
	}
}

func SplitWith(s string, splitters string) (string, string, bool) {
	i := strings.IndexAny(s, splitters)
	if i < 0 {
		return s, "", false
	} else {
		return s[:i], s[i + 1:], true
	}
}


