package cli

import "fmt"

//
type PrintableError interface {
	GetText(lang string) string
}


type InvalidCommandError struct {
	Name string
	ctx *Context
}

func NewInvalidCommandError(name string, ctx *Context) error {
	return &InvalidCommandError {
		Name: name,
		ctx: ctx,
	}
}

func (e *InvalidCommandError) Error() string {
	return fmt.Sprintf("'%s' is not a vaild command", e.Name)
}

func (e *InvalidCommandError) GetSuggestions() []string {
	cmd := e.ctx.command
	return cmd.GetSuggestions(e.Name)
}



