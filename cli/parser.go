/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"strings"
	"fmt"
)

type FlagDetector interface {
	DetectFlag(name string) (*Flag, error)
}

type Parser struct {
	current int
	args    []string
	detector func(string) (*Flag, error)
}

func NewParser(args []string, detector func(string) (*Flag, error)) *Parser {
	return &Parser {
		args:    args,
		current: 0,
		detector: detector,
	}
}

func (p *Parser) ReadNextArg() (arg string, more bool, err error) {
	for {
		arg, _, more, err = p.readNext()
		if err != nil {
			return
		}
		if !more {
			return
		}
		if arg != "" {
			return
		}
	}
}

func (p *Parser) GetRemains() []string {
	return p.args[p.current:]
}

func (p *Parser) ReadAll() ([]string, error) {
	r := make([]string, 0)
	for {
		arg, _, more, err := p.readNext()
		if err != nil {
			return r, err
		}
		if arg != "" {
			r = append(r, arg)
		}
		if !more {
			return r, nil
		}
	}
}

func (p *Parser) readNext() (arg string, flag *Flag, more bool, err error) {
	if p.current >= len(p.args) {
		more = false
		return
	}
	s := p.args[p.current]
	p.current++
	more = true

	if strings.HasPrefix(s, "--") {
		if name, value, ok := SplitWith(s[2:], "=:"); ok {
			flag, err = p.detector(name)
			if err != nil {
				return
			}
			err = flag.PutValue(value)
			if err != nil {
				return
			}
			return
		} else {
			flag, err = p.detector(name)
			if err != nil {
				return
			}
			if !flag.Assignable {
				flag.PutValue("")
				return
			} else {
				if value, ok := p.readNextValue(flag.Assignable); ok {
					flag.PutValue(value)
				}
				return
			}
		}
	} else if strings.HasPrefix(s,"-") {
		err = fmt.Errorf("not support single dash flag YET, next-version")
		return
	} else {
		arg = s
		return
	}
}

func (p *Parser) readNextValue(force bool) (string, bool) {
	if p.current >= len(p.args) {
		return "", false
	}
	s := p.args[p.current]
	if force {
		p.current++
		return s, true
	} else {
		if !strings.HasPrefix(s, "--") && !strings.HasPrefix(s, "-") {
			p.current++
			return s, true
		} else {
			return "", false
		}
	}
}