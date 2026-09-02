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
	"strings"
)

// StructuredError renders a machine-readable error without the decorations
// used by the interactive error path.
type StructuredError interface {
	error
	RenderError(io.Writer) error
	ExitCode() int
}

// InvalidOptionCombinationError identifies flags that cannot be used together
// while preserving the existing human-readable error text.
type InvalidOptionCombinationError struct {
	Options []string
	Err     error
}

func (e *InvalidOptionCombinationError) Error() string {
	if e == nil || e.Err == nil {
		return "invalid option combination"
	}
	return e.Err.Error()
}

func (e *InvalidOptionCombinationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (*InvalidOptionCombinationError) AIRecoveryEligible() {}

// If command.Execute return Noticeable error, print i18n Notice under error information
type ErrorWithTip interface {
	GetTip(lang string) string
}

type errorWithTip struct {
	err error
	tip string
}

func NewErrorWithTip(err error, tipFormat string, args ...interface{}) error {
	return &errorWithTip{
		err: err,
		tip: fmt.Sprintf(tipFormat, args...),
	}
}

func (e *errorWithTip) Error() string {
	return e.err.Error()
}

func (e *errorWithTip) Unwrap() error {
	return e.err
}

func (e *errorWithTip) GetTip(lang string) string {
	return e.tip
}

// OUTPUT:
// Error: "'%s' is not a valid command
//
// {Hint}
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
	return fmt.Sprintf("%q is not a valid command", e.Name)
}

func (*InvalidCommandError) AIRecoveryEligible() {}

func (e *InvalidCommandError) GetSuggestions() []string {
	if e == nil || e.ctx == nil || e.ctx.command == nil {
		return nil
	}
	cmd := e.ctx.command
	return cmd.GetSuggestions(e.Name)
}

func (e *InvalidCommandError) GetTip(lang string) string {
	if e.ctx != nil && e.ctx.command != nil {
		return fmt.Sprintf("Use `%s --help` for more information.", e.ctx.command.getName())
	}
	return "Use `--help` for more information."
}

type InvalidFlagError struct {
	Flag string
	ctx  *Context
}

func NewInvalidFlagError(name string, ctx *Context) error {
	return &InvalidFlagError{
		Flag: name,
		ctx:  ctx,
	}
}

func (e *InvalidFlagError) flagDisplay() string {
	if strings.HasPrefix(e.Flag, "-") {
		return e.Flag
	}
	return "--" + e.Flag
}

func (e *InvalidFlagError) Error() string {
	display := e.flagDisplay()
	suggestions := e.closeSuggestions()
	if len(suggestions) > 0 {
		return fmt.Sprintf("invalid flag %s, did you mean %s?", display, strings.Join(suggestions, " or "))
	}
	available := e.availableFlags()
	if len(available) > 0 {
		return fmt.Sprintf("invalid flag %s; available flags: %s", display, strings.Join(available, ", "))
	}
	return fmt.Sprintf("invalid flag %s", display)
}

// AgentMessage keeps recovery instructions out of the structured error
// message. Human Error() output remains unchanged.
func (e *InvalidFlagError) AgentMessage() string {
	return fmt.Sprintf("invalid flag %s", e.flagDisplay())
}

// AgentHelpCommand returns the ordinary Help entry for the command that
// rejected this CLI flag. Built-in commands do not support Canonical
// --help-search/--cli-section options.
func (e *InvalidFlagError) AgentHelpCommand() string {
	if e == nil || e.ctx == nil || e.ctx.command == nil {
		return "aliyun help"
	}
	path := strings.TrimSpace(e.ctx.command.getName())
	if path == "aliyun" {
		return "aliyun help"
	}
	path = strings.TrimSpace(strings.TrimPrefix(path, "aliyun "))
	if path == "" {
		return "aliyun help"
	}
	return "aliyun help " + path
}

func (*InvalidFlagError) AIRecoveryEligible() {}

// AgentSuggestions returns the same close flag matches used by Error without
// exposing Context internals to the OpenAPI recovery adapter.
func (e *InvalidFlagError) AgentSuggestions() []string {
	return e.closeSuggestions()
}

func (e *InvalidFlagError) closeSuggestions() []string {
	if e.ctx == nil || e.ctx.command == nil || e.ctx.Flags() == nil {
		return nil
	}
	distance := e.ctx.command.GetSuggestDistance()
	return e.ctx.Flags().GetSuggestions(e.Flag, distance)
}

func (e *InvalidFlagError) availableFlags() []string {
	if e.ctx == nil || e.ctx.Flags() == nil {
		return nil
	}
	return e.ctx.Flags().AvailableFlagNames()
}

func (e *InvalidFlagError) GetSuggestions() []string {
	// Suggestions are already embedded in Error() to keep the hint on stderr
	// (PrintSuggestions writes to stdout and is easy to miss).
	return nil
}
