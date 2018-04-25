package main

// This example program is designed to demonstrate all the features of
// goopt.

import (
	"fmt"
	goopt "github.com/droundy/goopt"
)

// The Flag function creates a boolean flag, possibly with a negating
// alternative.  Note that you can specify either long or short flags
// naturally in the same list.
var amVerbose = goopt.Flag([]string{"-v", "--verbose"}, []string{"--quiet"},
	"output verbosely", "be quiet, instead")

// This is just a logging function that uses the verbosity flags to
// decide whether or not to log anything.
func log(x ...interface{}) {
	if *amVerbose {
		fmt.Println(x...)
	}
}

var color = goopt.Alternatives([]string{"--color", "--colour"},
	[]string{"default", "red", "green", "blue"},
	"determine the color of the output")

var repetitions = goopt.Int([]string{"-n", "--repeat"}, 1, "number of repetitions")

var username = goopt.String([]string{"-u", "--user"}, "User", "name of user")

var children = goopt.Strings([]string{"--child"}, "name of child", "specify child of user")

func main() {
	goopt.Description = func() string {
		return "Example program for using the goopt flag library."
	}
	goopt.Version = "1.0"
	goopt.Summary = "goopt demonstration program"
	goopt.Parse(nil)
	defer fmt.Print("\033[0m") // defer resetting the terminal to default colors
	switch *color {
	case "default":
	case "red": fmt.Print("\033[31m")
	case "green": fmt.Print("\033[32m")
	case "blue": fmt.Print("\033[34m")
	default: panic("Unrecognized color!") // this should never happen!
	}
	log("I have now set the color to", *color, ".")
	for i:=0; i<*repetitions; i++ {
		fmt.Println("Greetings,", *username)
		log("You have", *repetitions, "children.")
		for _,child := range *children {
			fmt.Println("I also greet your child, whose name is", child)
		}
	}
}
