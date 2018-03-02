/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import "fmt"

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

func Debug(s string) {
	fmt.Print(DebugColor + s + ColorOff)
}

func Info(s string) {
	fmt.Print(InfoColor + s + ColorOff)
}

func Notice(s string) {
	fmt.Print(NoticeColor + s + ColorOff)
}

func Warning(s string) {
	fmt.Print(WarningColor + s + ColorOff)
}

func Error(s string) {
	fmt.Print(ErrorColor + s + ColorOff)
}

func Debugf(format string, args ...interface{}) {
	Debug(fmt.Sprintf(format, args...))
}

func Infof(format string, args ...interface{}) {
	Info(fmt.Sprintf(format, args...))
}

func Noticef(format string, args ...interface{}) {
	Notice(fmt.Sprintf(format, args...))
}

func Warningf(format string, args ...interface{}) {
	Warning(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}
