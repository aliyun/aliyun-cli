/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFlagSet(t *testing.T) {
	fs := NewFlagSet()
	assert.NotNil(t, fs.flags)
	assert.NotNil(t, fs.index)
}

func TestFlags(t *testing.T) {
	fs := NewFlagSet()
	assert.NotNil(t, fs.flags)
	assert.Len(t, fs.flags, 0)
}

func TestAdd(t *testing.T) {
	defer func() {
		a := recover()
		err, _ := a.(error)
		assert.EqualError(t, err, "flag duplicated --MrX")
		//the err message has "---MrX" before, and it should be "--MrX"
		// assert.EqualError(t, err, "flag duplicated ---MrX")
	}()
	fs := NewFlagSet()
	f := &Flag{Name: "MrX"}
	fs.Add(f)
	assert.Subset(t, fs.flags, []*Flag{&Flag{Name: "MrX"}})
	assert.Len(t, fs.flags, 1)
	fs.Add(f)
}

func TestAddByName(t *testing.T) {
	fs := NewFlagSet()
	f, err := fs.AddByName("MrX2")
	assert.Equal(t, &Flag{Name: "MrX2"}, f)
	assert.Nil(t, err)

	f, err = fs.AddByName("MrX2")
	assert.Nil(t, f)
	assert.EqualError(t, err, "flag duplicated --MrX2")

}

func TestGet(t *testing.T) {
	fs := NewFlagSet()
	fs.AddByName("MrX")
	assert.Equal(t, &Flag{Name: "MrX", formation: "--MrX"}, fs.Get("MrX"))
	assert.Nil(t, fs.Get("MrX2"))
}

func TestGetByShorthand(t *testing.T) {
	fs := NewFlagSet()
	assert.Nil(t, fs.GetByShorthand('X'))
	fs.Add(&Flag{Name: "profile", Shorthand: 'p'})
	exf := &Flag{Name: "profile", Shorthand: 'p', formation: "-p"}
	fs.GetByShorthand('p')
	assert.Equal(t, exf, fs.GetByShorthand('p'))
}

func TestGetSuggestions(t *testing.T) {
	//TODO after prefected cli/suggestion.go testcase
}

func TestGetValue(t *testing.T) {
	fs := NewFlagSet()
	str, ok := fs.GetValue("NonExist")
	assert.False(t, ok)
	assert.Empty(t, str)
	fs.AddByName("MrX")
	str, ok = fs.GetValue("MrX")
	assert.False(t, ok)
	assert.Empty(t, str)
}

func TestAssignedCount(t *testing.T) {
	fs := NewFlagSet()
	fs.AddByName("MrX")
	assert.Zero(t, fs.assignedCount())
	fs.Flags()[0].assigned = true
	assert.Equal(t, 1, fs.assignedCount())
}

func TestPut(t *testing.T) {
	fs := NewFlagSet()
	fs.put(&Flag{Name: "profile", Shorthand: 'p'})
	assert.Len(t, fs.flags, 1)
	fs.put(&Flag{Name: "profile", Shorthand: 'r'})
	assert.Len(t, fs.flags, 1)
	assert.Equal(t, 'r', fs.flags[0].Shorthand)
	fs.put(&Flag{Name: "profil", Shorthand: 'a'})
	assert.Len(t, fs.flags, 2)
}

func TestMergeWith(t *testing.T) {
	var a = func(f *Flag) bool {
		//merge with the rule that you need
		//in this case , I need merge all
		return true
	}
	fs := NewFlagSet()
	assert.True(t, reflect.DeepEqual(fs, fs.mergeWith(nil, a)))
	fs2 := NewFlagSet()
	fs2.Add(&Flag{Name: "profile", Shorthand: 'p'})
	fs2.Add(&Flag{Name: "mode", Shorthand: 'm'})
	assert.Len(t, fs.mergeWith(fs2, a).flags, 2)
	fs.Add(&Flag{Name: "AK", Shorthand: 'a'})
	assert.Len(t, fs.mergeWith(fs2, a).flags, 3)
}
