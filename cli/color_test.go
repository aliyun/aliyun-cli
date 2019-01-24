/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColor(t *testing.T) {
	assert.True(t, withColor)
	DisableColor()
	assert.False(t, withColor)
	EnableColor()
	assert.True(t, withColor)
	DisableColor()
	assert.Empty(t, ProductListColor())
	SetProductListColor(Red)
	assert.Equal(t, "\x1b[0;31m", productListColor)
	assert.Empty(t, APIListColor())
	SetAPIListColor("Red")
	assert.Equal(t, "Red", apiListColor)

	assert.Equal(t, "a", colorized("", "a"))
	EnableColor()
	assert.Equal(t, "reda\033[0m", colorized("red", "a"))
}

func TestCotainWriter(t *testing.T) {
	w := new(bytes.Buffer)
	n, err := PrintWithColor(w, "red", "a")
	assert.Equal(t, "reda\033[0m", w.String())
	assert.Equal(t, 8, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Debug(w, "a")
	assert.Equal(t, "\x1b[0;37ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Info(w, "a")
	assert.Equal(t, "\033[0;36ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Notice(w, "a")
	assert.Equal(t, "\033[1;33ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Warning(w, "a")
	assert.Equal(t, "\033[1;35ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Error(w, "a")
	assert.Equal(t, "\033[1;31ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Debugf(w, "%s", "a")
	assert.Equal(t, "\x1b[0;37ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Infof(w, "%s", "a")
	assert.Equal(t, "\033[0;36ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Noticef(w, "%s", "a")
	assert.Equal(t, "\033[1;33ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Warningf(w, "%s", "a")
	assert.Equal(t, "\033[1;35ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = Errorf(w, "%s", "a")
	assert.Equal(t, "\033[1;31ma\x1b[0m", w.String())
	assert.Equal(t, 12, n)
	assert.Nil(t, err)

	w.Reset()
	n, err = PrintfWithColor(w, "red", "%s", "a")
	assert.Equal(t, "reda\033[0m", w.String())
	assert.Equal(t, 8, n)
	assert.Nil(t, err)
}
