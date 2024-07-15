//go:build !windows
// +build !windows

package lib

import (
	. "gopkg.in/check.v1"
	"os"
)

func (s *OssutilCommandSuite) TestGetOsLang(c *C) {
	lang := os.Getenv("LANG")

	os.Setenv("LANG", "zh_CN.GB2312")
	c.Assert(getOsLang(), Equals, ChineseLanguage)
	os.Setenv("LANG", "zh_CN")
	c.Assert(getOsLang(), Equals, ChineseLanguage)
	os.Setenv("LANG", "en_US.UTF-8")
	c.Assert(getOsLang(), Equals, EnglishLanguage)
	os.Setenv("LANG", "es_ES.UTF-8")
	c.Assert(getOsLang(), Equals, EnglishLanguage)
	os.Setenv("LANG", "da_DK")
	c.Assert(getOsLang(), Equals, EnglishLanguage)
	os.Setenv("LANG", "zh_TW")
	c.Assert(getOsLang(), Equals, EnglishLanguage)

	os.Setenv("LANG", lang)
}
