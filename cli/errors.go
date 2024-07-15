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

import "fmt"

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
	return fmt.Sprintf("'%s' is not a vaild command", e.Name)
}

func (e *InvalidCommandError) GetSuggestions() []string {
	cmd := e.ctx.command
	return cmd.GetSuggestions(e.Name)
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

func (e *InvalidFlagError) Error() string {
	return fmt.Sprintf("invalid flag %s", e.Flag)
}

func (e *InvalidFlagError) GetSuggestions() []string {
	distance := e.ctx.command.GetSuggestDistance()
	return e.ctx.Flags().GetSuggestions(e.Flag, distance)
}
