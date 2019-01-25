/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetField() *Field {
	return &Field{
		Key:    "first",
		values: make([]string, 0),
	}
}
func TestField(t *testing.T) {
	//assign
	field := resetField()
	field.assign("hello")
	assert.True(t, field.assigned)
	assert.Equal(t, "hello", field.value)
	assert.Len(t, field.values, 1)
	assert.Equal(t, "hello", field.values[0])

	//GetValue
	value, ok := field.getValue()
	assert.True(t, ok)
	assert.Equal(t, "hello", value)
	field.assigned = false
	value, ok = field.getValue()
	assert.False(t, ok)
	assert.Empty(t, value)
	field.DefaultValue = "default"
	value, ok = field.getValue()
	assert.False(t, ok)
	assert.Equal(t, "default", value)

	//check
	assert.Nil(t, field.check())
	field.assigned = false
	field.Required = true
	assert.EqualError(t, field.check(), "first= required")
	field.Key = ""
	assert.EqualError(t, field.check(), "value required")

	field.Required = false
	field.values = []string{"first", "second"}
	assert.EqualError(t, field.check(), "value duplicated")
	field.Key = "first"
	assert.EqualError(t, field.check(), "first= duplicated")

	field.SetAssigned(true)
	assert.True(t, field.assigned)

	field.SetValue("test")
	assert.Equal(t, "test", field.value)
}
