/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
	"strings"
)

//
type AssignedMode int

const (
	AssignedNone       = AssignedMode(-1)
	AssignedDefault    = AssignedMode(0)
	AssignedOnce       = AssignedMode(1)
	AssignedRepeatable = AssignedMode(9)
)

type Flag struct {
	// Name of flag --{Name}
	Name string

	// Flag is the single characters
	Shorthand string

	// Message print with --help command
	Usage *i18n.Text

	// If DefaultValue is not "" and Required is true, if flag is not assign
	//   GetValue() will return DefaultValue, and IsAssigned() will be false
	DefaultValue string

	// If Required is true, the flag must be assigned with `--flag value` or DefaultValue is not empty
	Required bool

	// Ref to AssignedMode
	// `AssignedNone`: 		flag only appear with `--flag1` `--flag2`
	// `AssignedDefault`: 	flag can appear with `--flag1` or `--flag1 value1`
	// `AssignedOnce`: 		flag only appear with `--flag1 value1`
	// `AssignedRepeatable`: flag can appear multi times sample: `--flag1 v1 --flag1 v2`
	AssignedMode AssignedMode

	// If Persistent is true, the flag can apply to child commands
	Persistent bool

	// If Hidden is true, it will not appear in --help mode
	Hidden bool

	// Using in FlagSet.GetByCategory()...
	Category string

	assigned bool
	value    string
	values   []string
	p        *string
}

//
// return true if flag appeared, either `--flag1` or `--flag1 value1`
func (f *Flag) IsAssigned() bool {
	return f.assigned
}

//
// return flag value, if not assigned return f.DefaultValue
//   for `AssignedMode == AssignedRepeatable`. Use GetValues() to get all values
func (f *Flag) GetValue() string {
	if f.assigned {
		return f.value
	} else if f.Required {
		return f.DefaultValue
	} else {
		return ""
	}
}

//
// for `AssignedMode == AssignedRepeatable` flag, return values
func (f *Flag) GetValues() []string {
	return f.values
}

func (f *Flag) GetValueOrDefault(ctx *Context, def string) string {
	if f == nil {
		return def
	}
	v, ok := ctx.Flags().GetValue(f.Name)
	if ok {
		return v
	} else {
		return def
	}
}

//
// used in parser to put value to flag
func (f *Flag) putValue(v string) error {
	switch f.AssignedMode {
	case AssignedNone:
		if f.assigned {
			return fmt.Errorf("flag duplucated: --%s", f.Name)
		}
		if v != "" {
			return fmt.Errorf("flag --%s can't be assiged", f.Name)
		}
		f.setValue(v)
	case AssignedDefault:
		if f.assigned {
			return fmt.Errorf("flag duplicated: --%s", f.Name)
		}
		f.setValue(v)
	case AssignedOnce:
		if f.assigned {
			return fmt.Errorf("flag duplicated: --%s", f.Name)
		}
		f.setValue(v)
	case AssignedRepeatable:
		f.setValue(v)
		f.values = append(f.values, v)
	default:
		return fmt.Errorf("unexpect assigned mode %v for flag --%s", f.AssignedMode, f.Name)
	}
	return nil
}

//
// set value
func (f *Flag) setValue(v string) {
	f.assigned = true
	f.value = v
	if f.p != nil {
		*f.p = v
	}
}

func (f *Flag) useDefaultValue() bool {
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
		return s[:i], s[i+1:], true
	}
}
