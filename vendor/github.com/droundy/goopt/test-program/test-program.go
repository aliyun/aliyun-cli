package main

// test out the goopt package...

import (
	"fmt"
	"strings"
	goopt "github.com/droundy/goopt"
)

var amVerbose = goopt.Flag([]string{"--verbose"}, []string{},
	"output verbosely", "")
var amHappy = goopt.Flag([]string{"-h", "--happy"}, []string{"-u", "--unhappy", "--sad"}, "be happy", "be unhappy")

var foo = goopt.String([]string{"--name"}, "anonymous", "pick your name")
var bar = goopt.String([]string{"-b"}, "BOO!", "pick your scary sound")
var baz = goopt.String([]string{"-o"}, "", "test whether a silent default works")
var speed = goopt.Alternatives([]string{"--speed", "--velocity"},
	[]string{"slow", "medium", "fast"},
	"set the speed")

var words = goopt.Strings([]string{"--word", "--saying", "-w", "-s"}, "word",
	"specify a word to speak")

var width = goopt.Int([]string{"-l", "--length"}, 1, "number of ?s")

func main() {
	goopt.Summary = "silly test program"
	goopt.Parse(nil)
	if *amVerbose {
		fmt.Println("I am verbose.")
	}
	if *amHappy {
		fmt.Println("I am happy")
	} else {
		fmt.Println("I am unhappy")
	}
	fmt.Println("Your name is", *foo)
	fmt.Println(*bar, "... Did I scare you?")
	fmt.Println("I am going so very", *speed, "!!!")
	fmt.Print("Here is the saying:")
	for _, w := range *words {
		fmt.Print(" ", w)
	}
	fmt.Println()
	fmt.Println(*baz)
	fmt.Print("Back in my day,")
	for _, w := range goopt.Args {
		fmt.Print(" ", w)
	}
	fmt.Println()
	fmt.Printf("What's up, man%s\n", strings.Repeat("?", *width))
}
