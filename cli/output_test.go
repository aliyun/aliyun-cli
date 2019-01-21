/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultWriter(t *testing.T) {
	w := DefaultWriter()
	writer, ok := w.(*os.File)
	assert.True(t, ok)
	assert.ObjectsAreEqual(os.Stdout, writer)

	buf := new(bytes.Buffer)
	n, err := Print(buf, "I am night")
	assert.Equal(t, 10, n)
	assert.Nil(t, err)

	buf.Reset()
	n, err = Println(buf, "How are you")
	assert.Equal(t, 12, n)
	assert.Nil(t, err)
	buf.Reset()
	n, err = Printf(buf, "I am %s", "fine")
	assert.Equal(t, 9, n)
	assert.Nil(t, err)
}

func TestOutput(t *testing.T) {
	w := new(bytes.Buffer)
	output := new(Output)
	index, err := output.Print(w, "who are you")
	assert.Equal(t, 11, index)
	assert.Nil(t, err)
	assert.Equal(t, "who are you", w.String())

	w.Reset()
	index, err = output.Println(w, "I am MrX")
	assert.Equal(t, 9, index)
	assert.Nil(t, err)
	assert.Equal(t, "I am MrX\n", w.String())

	w.Reset()
	index, err = output.Printf(w, "and you%s", "?")
	assert.Equal(t, 8, index)
	assert.Nil(t, err)
	assert.Equal(t, "and you?", w.String())
}
