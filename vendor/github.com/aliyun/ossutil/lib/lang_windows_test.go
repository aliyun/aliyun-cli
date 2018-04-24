// +build windows

package lib

import (
	. "gopkg.in/check.v1"
	"syscall"
)

func (s *OssutilCommandSuite) TestGetOsLang(c *C) {
	var mod = syscall.NewLazyDLL("kernel32.dll")
	var proc = mod.NewProc("GetUserDefaultUILanguage")
	oriLang, _, _ := proc.Call()

	var sproc = mod.NewProc("SetUserDefaultUILanguage")
	sproc.Call(1033)
	c.Assert(getOsLang(), Equals, EnglishLanguage)

	sproc.Call(2052)
	c.Assert(getOsLang(), Equals, ChineseLanguage)

	sproc.Call(oriLang)
}
