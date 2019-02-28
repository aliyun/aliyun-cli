package cli

import (
	"runtime"
)

func PlatformCompatible() {
	if runtime.GOOS == `windows` {
		DisableColor()
	}
}
