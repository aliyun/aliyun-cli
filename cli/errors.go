package cli

import "fmt"

type InvalidCommandError struct {
	Name string
	Suggestions []*Command
}

func (e *InvalidCommandError) Error() string {
	return fmt.Sprintf("invalid command %s", e.Name)
}

type InvalidFlagError struct {
	Name string
	Suggestions []*Flag
}

func (e *InvalidFlagError) Error() string {
	return fmt.Sprintf("invalid flag --%s", e.Name)
}


