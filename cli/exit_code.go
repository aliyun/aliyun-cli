package cli

import "os"

var (
	withExitCode = true
)

func EnableExitCode() {
	withExitCode = true
}

func DisableExitCode() {
	withExitCode = false
}

func Exit(code int) {
	if withExitCode {
		os.Exit(code)
	}
}
