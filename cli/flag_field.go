// Copyright 1999-2019 Alibaba Group Holding Limited
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

	"github.com/aliyun/aliyun-cli/i18n"
)

type Field struct {
	//
	// appear in `--flag key1=value1, key2=value2`
	// if Key assigned with "", it can used with `--flag value1 value2`
	Key string

	//
	// if Required is true, this field must be assigned
	Required bool

	//
	// if Repeatable is true, this field can appear multiply times, eg: "--flag key1=value1 key2=value2"
	Repeatable bool

	//
	// if field not appear, use this value, not used with Required
	DefaultValue string

	//
	// Message show
	Short *i18n.Text

	assigned bool
	value    string
	values   []string
}

func (f *Field) assign(v string) {
	f.assigned = true
	f.value = v
	f.values = append(f.values, v)
}

func (f *Field) SetAssigned(istrue bool) {
	f.assigned = istrue
}

func (f *Field) SetValue(value string) {
	f.value = value
}

func (f *Field) getValue() (string, bool) {
	if f.assigned {
		return f.value, true
	} else if f.DefaultValue != "" {
		return f.DefaultValue, false
	} else {
		return "", false
	}
}

func (f *Field) check() error {
	if f.Required && !f.assigned {
		if f.Key != "" {
			return fmt.Errorf("%s= required", f.Key)
		}
		return fmt.Errorf("value required")

	}
	if !f.Repeatable && len(f.values) > 1 {
		if f.Key != "" {
			return fmt.Errorf("%s= duplicated", f.Key)
		}
		return fmt.Errorf("value duplicated")

	}
	return nil
}
