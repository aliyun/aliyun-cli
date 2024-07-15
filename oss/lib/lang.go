// This is for Condition Compling, which means it will be built on all non-windows platform.

//go:build !windows
// +build !windows

package lib

import (
	"os"
	"strings"
)

func getOsLang() string {
	lang := os.Getenv("LANG")
	langstr := strings.Split(lang, ".")

	if langstr[0] == "zh_CN" {
		return ChineseLanguage
	}
	return EnglishLanguage
}
