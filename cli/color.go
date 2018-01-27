package cli

import "fmt"

const (
	InfoColor    = "\033[1;34m%s\033[0m"
	NoticeColor  = "\033[1;36m%s\033[0m"
	WarningColor = "\033[1;33m%s\033[0m"
	ErrorColor   = "\033[1;31m%s\033[0m"
	DebugColor   = "\033[0;36m%s\033[0m"
)

func Info(s string) string {
	return fmt.Sprintf(InfoColor, s)
}

func Notice(s string) string {
	return fmt.Sprintf(NoticeColor, s)
}

func Warning(s string) string {
	return fmt.Sprintf(WarningColor, s)
}
func Error(s string) string {
	return fmt.Sprintf(ErrorColor, s)
}

func Debug(s string) string {
	return fmt.Sprintf(DebugColor, s)
}

func Errorf(msg string, args ...interface{}) {
	fmt.Printf(Error(msg), args)
}