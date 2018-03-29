package cli

import "fmt"

type InvalidCommandError struct {
	Name string
	ctx  *Context
}

func NewInvalidCommandError(name string, ctx *Context) error {
	return &InvalidCommandError{
		Name: name,
		ctx:  ctx,
	}
}

func (e *InvalidCommandError) Error() string {
	return fmt.Sprintf("'%s' is not a vaild command", e.Name)
}

func (e *InvalidCommandError) GetSuggestions() []string {
	cmd := e.ctx.command
	return cmd.GetSuggestions(e.Name)
}

type InvalidFlagError struct {
	Name      string
	Shorthand string
	ctx       *Context
}

func NewInvalidFlagError(name, shorthand string, ctx *Context) error {
	return &InvalidFlagError{
		Name:      name,
		Shorthand: shorthand,
		ctx:       ctx,
	}
}

func (e *InvalidFlagError) Error() string {
	var param string
	if e.Name != "" {
		param = "--" + e.Name
	} else {
		param = "-" + e.Shorthand
	}
	return fmt.Sprintf("invalid flag %s", param)
}

func (e *InvalidFlagError) GetSuggestions() []string {
	distance := e.ctx.command.GetSuggestDistance()
	return e.ctx.Flags().GetSuggestions(e.Name, distance)
}
