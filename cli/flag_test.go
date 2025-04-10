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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func resetFlag() *Flag {
	f := &Flag{
		Category:     "config",
		Name:         "MrX",
		Shorthand:    'p',
		DefaultValue: "default",
		Persistent:   true,
		Short: i18n.T(
			"use `--profile <profileName>` to select profile",
			"使用 `--profile <profileName>` 指定操作的配置集",
		),
		Long:         nil,
		Required:     false,
		Aliases:      nil,
		AssignedMode: AssignedDefault,
		Hidden:       false,
		Validate:     nil,
		Fields:       nil,
		ExcludeWith:  nil,
	}
	return f
}
func TestFlag(t *testing.T) {

	f := resetFlag()

	assert.False(t, f.IsAssigned())

	//GetValue
	value, ok := f.GetValue()
	assert.False(t, ok)
	assert.Equal(t, "", value)
	f.Required = true
	value, ok = f.GetValue()
	assert.False(t, ok)
	assert.Equal(t, "default", value)

	//GetValues
	assert.Nil(t, f.GetValues())
	f.values = []string{"hello", "你好"}
	assert.Len(t, f.values, 2)
	assert.Subset(t, f.values, []string{"hello", "你好"})

	//GetField
	field, ok := f.getField("MrX")
	assert.Nil(t, field)
	assert.False(t, ok)

	f.Fields = []Field{{Key: "MrX"}, {Key: "你好"}}
	field, ok = f.getField("MrX")
	assert.NotNil(t, field)
	assert.True(t, ok)

	//GetFieldValue
	fieldValue, ok := f.GetFieldValue("MrX")
	assert.Empty(t, fieldValue)
	assert.False(t, ok)
	fieldValue, ok = f.GetFieldValue("NonExist")
	assert.Empty(t, fieldValue)
	assert.False(t, ok)

	//GetFieldValues
	fieldvalues := f.GetFieldValues("MrX")
	assert.Len(t, fieldvalues, 0)
	fieldvalues = f.GetFieldValues("NonExist")
	assert.Len(t, fieldvalues, 0)

	//GetStringOrDefault
	assert.Equal(t, "nihao", f.GetStringOrDefault("nihao"))
	f.assigned = true
	assert.Equal(t, "", f.GetStringOrDefault("nihao"))
	f = nil
	assert.Equal(t, "nihao", f.GetStringOrDefault("nihao"))

	//GetIntegerOrDefault
	f = resetFlag()
	f.value = "1"
	f.GetIntegerOrDefault(23)
	assert.Equal(t, 23, f.GetIntegerOrDefault(23))
	f.assigned = true
	assert.Equal(t, 1, f.GetIntegerOrDefault(23))

	//GetFormations
	assert.Len(t, f.GetFormations(), 2)
	assert.Subset(t, f.GetFormations(), []string{"--MrX", "-p"})

	//setIsAssigned
	assert.NotNil(t, f.setIsAssigned())
	f.assigned = false
	assert.Nil(t, f.setIsAssigned())

	//needValue
	f.AssignedMode = AssignedNone
	assert.False(t, f.needValue())
	f.AssignedMode = AssignedDefault
	assert.False(t, f.needValue())
	f.AssignedMode = AssignedOnce
	assert.False(t, f.needValue())
	f.AssignedMode = AssignedRepeatable
	assert.True(t, f.needValue())

	//validate
	assert.Nil(t, f.validate())
	f.AssignedMode = AssignedOnce
	f.value = ""
	assert.EqualError(t, f.validate(), "--MrX must be assigned with value")

	//assign
	assert.Nil(t, f.assign("who am i"))
	assert.Equal(t, f.value, "who am i")
	assert.True(t, f.assigned)
	f.AssignedMode = AssignedNone
	assert.EqualError(t, f.assign("who am i"), "flag --MrX can't be assiged")

	f.AssignedMode = AssignedRepeatable
	assert.Nil(t, f.assign("who am i"))
	assert.Len(t, f.values, 1)
	assert.Subset(t, f.values, []string{"who am i"})

	//assignField
	f = resetFlag()
	assert.EqualError(t, f.assignField("I am MrX"), "--MrX can't assign with value")
	assert.EqualError(t, f.assignField("MrX=Night556"), "--MrX can't assign with MrX=")
	f.Fields = []Field{{Key: "MrX"}}
	assert.Nil(t, f.assignField("MrX=Night556"))

	//checkFields
	f = resetFlag()
	assert.Nil(t, f.checkFields())
	f.Fields = []Field{{Key: "MrX"}}
	assert.Nil(t, f.checkFields())
	f.Fields[0].Required = true
	assert.EqualError(t, f.checkFields(), "bad flag format --MrX with field MrX= required")

	f.SetAssigned(true)
	assert.True(t, f.assigned)

	f.SetValue("test")
	assert.Equal(t, "test", f.value)

	f.SetValues([]string{"test", "test1"})
	assert.Equal(t, "test", f.values[0])
	assert.Equal(t, "test1", f.values[1])
}
