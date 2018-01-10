package cli

import "strings"

type Parser struct {
	addArg func(arg string)
	addFlag func(name string, value string)
}

func NewParser(addArg func(arg string), addFlag func(name string, value string)) *Parser {
	return &Parser {
		addArg: addArg,
		addFlag: addFlag,
	}
}

func ParseArgs(args []string) ([]string, map[string]string) {
	r1 := []string{}
	r2 := make(map[string]string)
	parser := NewParser(
		func(arg string) {
			r1 = append(r1, arg)
		},
		func(name string, value string) {
			r2[name] = value
		},
	)
	parser.Parse(args)
	return r1, r2
}

func (p *Parser) Parse(args []string) {
	lastFlag := ""
	for _, s := range args {
		if strings.HasPrefix(s, "--") {
			name := s[2:]
			if lastFlag == "" {
				lastFlag = p.parseFlag(name)
			} else {
				p.addFlag(lastFlag, "")
				lastFlag = name
			}
		} else if strings.HasPrefix(s,"-") {
			name := s[1:]
			if lastFlag == "" {
				lastFlag = p.parseFlag(name)
			} else {
				p.addFlag(lastFlag, "")
				lastFlag = name
			}
		} else {
			if lastFlag != "" {
				p.addFlag(lastFlag, s)
				lastFlag = ""
			} else {
				p.addArg(s)
			}
		}
	}
}

func (p *Parser) parseFlag(s string) string {
	if name, value, ok := splitWithFirstChars(s, "=:"); ok {
		p.addFlag(name, value)
		return ""
	} else {
		return name
	}
}

func splitWithFirstChars(s string, splitters string) (string, string, bool) {
	i := strings.IndexAny(s, splitters)
	if i < 0 {
		return s, "", false
	} else {
		return s[:i], s[i + 1:], true
	}
}


func GetFirstArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	a := args[0]
	if strings.HasPrefix(a, "-") || strings.HasPrefix(a, "--") {
		return ""
	} else {
		return a
	}
}