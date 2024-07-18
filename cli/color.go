// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cli

import (
	"fmt"
	"io"
	"os"
)

const (
	ColorOff = "\033[0m" // Reset Color
	// Regular Colors
	Black  = "\033[0;30m" // Black
	Red    = "\033[0;31m" // Red
	Green  = "\033[0;32m" // Green
	Yellow = "\033[0;33m" // Yellow
	Blue   = "\033[0;34m" // Blue
	Purple = "\033[0;35m" // Purple
	Cyan   = "\033[0;36m" // Cyan
	White  = "\033[0;37m" // White

	// Bold
	BBlack  = "\033[1;30m" // Black
	BRed    = "\033[1;31m" // Red
	BGreen  = "\033[1;32m" // Green
	BYellow = "\033[1;33m" // Yellow
	BBlue   = "\033[1;34m" // Blue
	BPurple = "\033[1;35m" // Purple
	BCyan   = "\033[1;36m" // Cyan
	BWhite  = "\033[1;37m" // White

	// Underline
	UBlack  = "\033[4;30m" // Black
	URed    = "\033[4;31m" // Red
	UGreen  = "\033[4;32m" // Green
	UYellow = "\033[4;33m" // Yellow
	UBlue   = "\033[4;34m" // Blue
	UPurple = "\033[4;35m" // Purple
	UCyan   = "\033[4;36m" // Cyan
	UWhite  = "\033[4;37m" // White

	// Background
	OnBlack  = "\033[40m" // Black
	OnRed    = "\033[41m" // Red
	OnGreen  = "\033[42m" // Green
	OnYellow = "\033[43m" // Yellow
	OnBlue   = "\033[44m" // Blue
	OnPurple = "\033[45m" // Purple
	OnCyan   = "\033[46m" // Cyan
	OnWhite  = "\033[47m" // White

	// High Intensty
	IBlack  = "\033[0;90m" // Black
	IRed    = "\033[0;91m" // Red
	IGreen  = "\033[0;92m" // Green
	IYellow = "\033[0;93m" // Yellow
	IBlue   = "\033[0;94m" // Blue
	IPurple = "\033[0;95m" // Purple
	ICyan   = "\033[0;96m" // Cyan
	IWhite  = "\033[0;97m" // White

	// Bold High Intensty
	BIBlack  = "\033[1;90m" // Black
	BIRed    = "\033[1;91m" // Red
	BIGreen  = "\033[1;92m" // Green
	BIYellow = "\033[1;93m" // Yellow
	BIBlue   = "\033[1;94m" // Blue
	BIPurple = "\033[1;95m" // Purple
	BICyan   = "\033[1;96m" // Cyan
	BIWhite  = "\033[1;97m" // White

	// High Intensty backgrounds
	OnIBlack  = "\033[0;100m" // Black
	OnIRed    = "\033[0;101m" // Red
	OnIGreen  = "\033[0;102m" // Green
	OnIYellow = "\033[0;103m" // Yellow
	OnIBlue   = "\033[0;104m" // Blue
	OnIPurple = "\033[10;95m" // Purple
	OnICyan   = "\033[0;106m" // Cyan
	OnIWhite  = "\033[0;107m" // White
)

const (
	DebugColor   = White
	InfoColor    = Cyan
	NoticeColor  = BYellow
	WarningColor = BPurple
	ErrorColor   = BRed
)

func isNoColor() bool {
	var isTTY = os.FileMode(0)&os.ModeDevice != 0
	var NO_COLOR = os.Getenv("NO_COLOR")
	return isTTY || NO_COLOR == "true" || NO_COLOR == "1"
}

func Colorized(color string, a ...interface{}) string {
	if !isNoColor() && color != "" {
		return color + fmt.Sprint(a...) + ColorOff
	}
	return fmt.Sprint(a...)
}

func PrintWithColor(w io.Writer, color string, a ...interface{}) (n int, err error) {
	return Print(w, Colorized(color, a...))
}

func Notice(w io.Writer, a ...interface{}) (n int, err error) {
	return Print(w, Colorized(NoticeColor, a...))
}

func Error(w io.Writer, a ...interface{}) (n int, err error) {
	return Print(w, Colorized(ErrorColor, a...))
}

func Noticef(w io.Writer, format string, args ...interface{}) (n int, err error) {
	return Notice(w, fmt.Sprintf(format, args...))
}

func Errorf(w io.Writer, format string, args ...interface{}) (n int, err error) {
	return Error(w, fmt.Sprintf(format, args...))
}

func PrintfWithColor(w io.Writer, color string, format string, args ...interface{}) (n int, err error) {
	return PrintWithColor(w, color, fmt.Sprintf(format, args...))
}
