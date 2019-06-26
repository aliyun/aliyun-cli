// Copyright 1999-2019 Alibaba Group Holding Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cli

import (
	"fmt"
	"strings"
)

type flagDetector interface {
	detectFlag(name string) (*Flag, error)
	detectFlagByShorthand(ch rune) (*Flag, error)
}

type Parser struct {
	current     int
	args        []string
	detector    flagDetector
	currentFlag *Flag
}

func NewParser(args []string, detector flagDetector) *Parser {
	return &Parser{
		args:        args,
		current:     0,
		detector:    detector,
		currentFlag: nil,
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

	value := ""
	flag, value, err = p.parseCommandArg(s)
	if err != nil {
		return
	}
	if flag != nil {
		err = flag.setIsAssigned()
		if err != nil {
			return
		}
	}

	// fmt.Printf(">>> current=%v\n    flag=%v\n    value=%s\n    err=%s\n", p.currentFlag, flag, value, err)
	if flag == nil { // parse with value xxxx
		if p.currentFlag != nil { // value need to feed to previous flag xxx
			err = p.currentFlag.assign(value)
			if err != nil {
				return
			}
			if !p.currentFlag.needValue() { // if current flag is feeds close it
				// fmt.Printf("$$$ clear %s\n", p.currentFlag.AssignedMode)
				p.currentFlag = nil
			}
		} else {
			arg = value // this is a arg
		}
	} else { // parse with flag	--xxx or -x
		if p.currentFlag != nil {
			err = p.currentFlag.validate()
			if err != nil {
				return
			}
			p.currentFlag = nil
		}

		if value != "" { // pattern --xx=aa, -x:aa, -xxx=bb
			err = flag.assign(value)
			if err != nil {
				return
			}
		} else { // pattern --xx -- yy
			if flag.needValue() {
				p.currentFlag = flag
			}
		}
	}
	return
}

func (p *Parser) parseCommandArg(s string) (flag *Flag, value string, err error) {
	prefix, v, ok := SplitStringWithPrefix(s, "=:")

	if ok {
		value = v
	}

	if strings.HasPrefix(prefix, "--") {
		if len(prefix) > 2 {
			flag, err = p.detector.detectFlag(prefix[2:])
		} else {
			err = fmt.Errorf("not support '--' in command line")
		}
	} else if strings.HasPrefix(prefix, "-") {
		if len(prefix) == 2 {
			flag, err = p.detector.detectFlagByShorthand(rune(prefix[1]))
		} else {
			err = fmt.Errorf("not support flag form %s", prefix)
		}
	} else {
		value = s
	}
	return
}

//SplitStringWithPrefix TODO can use function string.SplitN to replace
func SplitStringWithPrefix(s string, splitters string) (string, string, bool) {
	i := strings.IndexAny(s, splitters)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+1:], true

}

func SplitString(s string, sep string) []string {
	return strings.Split(s, sep)
}

func UnquoteString(s string) string {
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") && len(s) >= 2 {
		return s[1 : len(s)-1]
	}
	return s
}
