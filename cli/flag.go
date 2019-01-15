/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
	"strconv"
)

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
	Short *i18n.Text

	// Message print with `help --flag` command
	Long *i18n.Text

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
	// `AssignedRepeatable`: flag can appear multi times sample: `--flag1 [v1] [v2] [v3] [field1=value1]`, use with Fields
	AssignedMode AssignedMode

	// If Persistent is true, the flag can apply to child commands
	Persistent bool

	// If Hidden is true, it will not appear in --help mode
	Hidden bool

	// Using in FlagSet.GetByCategory()...
	Category string

	// Use to validate flag is in correct format
	Validate func(f *Flag) error

	// Flag can assigned with --flag field1=value1 field2=value2 value3 ...
	// must used with AssignedMode=AssignedRepeatable
	Fields []Field

	// Flag can't appear with other flags, use Flag.Name
	ExcludeWith []string

	assigned  bool
	value     string
	values    []string
	formation string
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
	if f.IsAssigned() {
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
// for `AssignedMode == AssignedRepeatable` flag, return fields, multiply assignable
// Sample: --output abc bbc acd bb=2 cc=3
func (f *Flag) getField(key string) (*Field, bool) {
	for i, field := range f.Fields {
		if field.Key == key {
			return &(f.Fields[i]), true
		}
	}
	return nil, false
}

//
// --flag field1=value1
func (f *Flag) GetFieldValue(key string) (string, bool) {
	if field, ok := f.getField(key); ok {
		return field.getValue()
	} else {
		return "", false
	}
}

//
//
func (f *Flag) GetFieldValues(key string) []string {
	if field, ok := f.getField(key); ok {
		return field.values
	} else {
		return make([]string, 0)
	}
}

//
// return def if Flag is not assigned
func (f *Flag) GetStringOrDefault(def string) string {
	if f == nil {
		return def
	}
	if f.assigned {
		return f.value
	} else {
		return def
	}
}

// TODO: flag support integer validate
// return def if Flag is not assign or assign failed
func (f *Flag) GetIntegerOrDefault(def int) int {
	if f == nil {
		return def
	}
	if f.assigned {
		if i, err := strconv.Atoi(f.value); err != nil {
			return i
		}
	}
	return def
}

//
// get all appears forms, maybe {"--Name", "--Alias1", "-Shorthand"}
func (f *Flag) GetFormations() []string {
	r := make([]string, 0)
	if f.Name != "" {
		r = append(r, "--"+f.Name)
	}
	for _, s := range f.Aliases {
		r = append(r, "--"+s)
	}
	if f.Shorthand != 0 {
		r = append(r, "-"+string(f.Shorthand))
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
			return fmt.Errorf("%s duplicated", f.formation)
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
		panic(fmt.Errorf("unexpected Flag.AssignedMode %s", strconv.Itoa(int(f.AssignedMode))))
	}
}

//
// make check valid
func (f *Flag) checkValid() {
	if len(f.Fields) > 0 {
		if f.AssignedMode != AssignedRepeatable {
			panic(fmt.Errorf("flag %s with fields must use AssignedRepeatable", f.Name))
		}
	}
}

//
// validate flag value
func (f *Flag) validate() error {
	if f.AssignedMode == AssignedOnce && f.value == "" {
		return fmt.Errorf("%s must be assigned with value", f.formation)
	}
	return nil
}

//
// assign value
func (f *Flag) assign(v string) error {
	if f.AssignedMode == AssignedNone {
		return fmt.Errorf("flag --%s can't be assiged", f.Name)
	}

	f.assigned = true
	f.value = v

	if f.AssignedMode == AssignedRepeatable {
		f.values = append(f.values, v)
		if len(f.Fields) > 0 {
			f.assignField(v)
		}
	}
	return nil
}

//
// assign field
func (f *Flag) assignField(s string) error {
	if k, v, ok := SplitStringWithPrefix(s, "="); ok {
		field, ok2 := f.getField(k)
		if ok2 {
			field.assign(v)
		} else {
			return fmt.Errorf("--%s can't assign with %s=", f.Name, k)
		}
	} else {
		field, ok2 := f.getField("")
		if ok2 {
			field.assign(v)
		} else {
			return fmt.Errorf("--%s can't assign with value", f.Name)
		}
	}
	return nil
}

//
func (f *Flag) checkFields() error {
	if len(f.Fields) == 0 {
		return nil
	}
	for _, field := range f.Fields {
		if err := field.check(); err != nil {
			return fmt.Errorf("bad flag format --%s with field %s", f.Name, err)
		}
	}
	return nil
}

//func (f *Flag) useDefaultValue() bool {
//	if f.DefaultValue != "" {
//		f.value = f.DefaultValue
//		if f.p != nil {
//			*f.p = f.DefaultValue
//		}
//		return true
//	} else {
//		return false
//	}
//}
