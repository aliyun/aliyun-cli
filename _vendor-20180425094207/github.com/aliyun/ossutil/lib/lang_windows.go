// This filename is for Condition Compling, which means it will be built only on windows platform.

package lib

import (
	"syscall"
)

func getOsLang() string {
	var mod = syscall.NewLazyDLL("kernel32.dll")
	var proc = mod.NewProc("GetUserDefaultUILanguage")
	ret, _, _ := proc.Call()

	/* Refer following link about LanggId values
	 * https://msdn.microsoft.com/en-us/library/bb165625(v=vs.90).aspx
	 */
	if ret == 2052 { // 2052 is User Language ID, means Chinese (Simplified)
		return ChineseLanguage
	}
	return EnglishLanguage
}
