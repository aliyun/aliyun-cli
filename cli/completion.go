/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"os"
	"strconv"
	"strings"
)

type Completion struct {
	Words []string
	Line string
	Point int
}

func ParseCompletion() *Completion {
	line := os.Getenv("COMP_LINE")
	if line == "" {
		return nil
	}

	point, _ := strconv.Atoi(os.Getenv("COMP_POINT"))
	words := os.Getenv("COMP_WORDS")
	return &Completion{
		Words: strings.Split(words, " "),
		Line: line,
		Point: point,
	}
}

func (c *Completion) GetArgs() []string {
	return []string{}
}
