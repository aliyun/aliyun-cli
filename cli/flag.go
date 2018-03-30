/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
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
	Shorthand rune

	// Message print with --help command
	Usage *i18n.Text

	// If DefaultValue is not "" and Required is true, if flag is not assign
	//   GetValue() will return DefaultValue, and IsAssigned() will be false
	DefaultValue string

	// If Required is true, the flag must be assigned with `--flag value` or DefaultValue is not empty
	Required bool

	// Enable flag has alias names
	Aliases []string

	// Ref to AssignedMode
	// `AssignedNone`: 		flag only appear with `--flag1` `--flag2`
	// `AssignedDefault`: 	flag can appear with `--flag1` or `--flag1 value1`
	// `AssignedOnce`: 		flag only appear with `--flag1 value1`
	// `AssignedRepeatable`: flag can appear multi times sample: `--flag1 v1 v2 v3`
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
	form 	 string
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
func (f *Flag) GetValue() (string, bool) {
	if f.assigned {
		return f.value, true
	} else if f.Required {
		return f.DefaultValue, false
	} else {
		return "", false
	}
}

//
// for `AssignedMode == AssignedRepeatable` flag, return values
func (f *Flag) GetValues() []string {
	return f.values
}

//
// `AssignedMode == AssignedRepeatable` flag, return values, in Map
//func (f *Flag) GetValuesByMap() map[string]string {
//	r := make(map[string]string)
//	for _, v = range f.values {
//		if k2, v2, ok := SplitWith(v, "="); ok {
//			request.Headers[k2] = v2
//		} else {
//			return fmt.Errorf("invaild flag --header `%s` use `--header HeaderName=Value`", v)
//		}
//	}
//	return
//}

//
// return def if Flag is not assigned
func (f *Flag) GetValueOrDefault(def string) string {
	if f.assigned {
		return f.value
	} else {
		return def
	}
}

//
// get all appears forms, maybe {"--Name", "--Alias1", "-Shorthand"}
func (f *Flag) GetForms() []string {
	r := make([]string, 0)
	if f.Name != "" {
		r = append(r, "--" + f.Name)
	}
	for _, s := range f.Aliases {
		r = append(r, "--" + s)
	}
	if f.Shorthand != 0 {
		r = append(r, "-" + string(f.Shorthand))
	}
	return r
}


//
// if this flag is appeared set assigned = true
func (f *Flag) setIsAssigned() error {
	if !f.assigned {
		f.assigned = true
	} else {
		if f.AssignedMode != AssignedRepeatable {
			return fmt.Errorf("%s duplicated", f.form)
		}
	}
	return nil
}

//
// return true, if this flag need assigned with values
func (f *Flag) needValue() bool {
	switch f.AssignedMode {
	case AssignedNone:
		return false
	case AssignedDefault:
		return f.value == ""
	case AssignedOnce:
		return f.value == ""
	case AssignedRepeatable:
		return true
	default:
		panic(fmt.Errorf("unexpected Flag.AssignedMode %s", f.AssignedMode))
	}
}

//
// used in parser to put value to flag
func (f *Flag) putValue(v string) error {
	switch f.AssignedMode {
	case AssignedNone:
		return fmt.Errorf("flag --%s can't be assiged", f.Name)
	case AssignedDefault:
		f.setValue(v)
	case AssignedOnce:
		f.setValue(v)
	case AssignedRepeatable:
		f.setValue(v)
		f.values = append(f.values, v)
	default:
		return fmt.Errorf("unexpect assigned mode %v for flag --%s", f.AssignedMode, f.Name)
	}
	return nil
}

func (f *Flag) validate() error {
	if f.AssignedMode == AssignedOnce && f.value == "" {
		return fmt.Errorf("%s must be assigned with value", f.form)
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
