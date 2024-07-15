//go:build windows
// +build windows

package lib

import (
	"fmt"
	"syscall"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestGetOsLang(c *C) {
	var mod = syscall.NewLazyDLL("kernel32.dll")

	var proc = mod.NewProc("GetUserDefaultUILanguage")
	err := proc.Find()
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}

	var sproc = mod.NewProc("SetUserDefaultUILanguage")
	err = sproc.Find()
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}

	oriLang, _, _ := proc.Call()

	sproc.Call(1033)
	c.Assert(getOsLang(), Equals, EnglishLanguage)

	sproc.Call(2052)
	c.Assert(getOsLang(), Equals, ChineseLanguage)

	sproc.Call(oriLang)
}
