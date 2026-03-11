package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

var usageMsg = `
Example: esc file.go
`

func main() {
	// Boilerplate from https://pkg.go.dev/golang.org/x/term#pkg-overview
	// Raw mode let's us worry about terminal sequences ourselves instead of
	// the terminal handling them.
	//oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	//if err != nil {
	//	handleError(err)
	//}
	// Return to "cooked" terminal
	//defer term.Restore(int(os.Stdin.Fd()), oldState)

	//TODO: holding off on raw until we need it. Right now we're just a basic
	//			viewer.

	if len(os.Args) < 2 {
		usage()
		return
	}
	filename := (os.Args[1])
	file, err := os.Open(filename)
	if err != nil {
		handleError(err)
	}

	_, err = os.Stdout.ReadFrom(file)
	if err != nil {
		handleError(err)
	}

	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		handleError(err)
	}
	fmt.Println(w, h)
}

// Teach'em son
func usage() {
	fmt.Println(usageMsg)
}

// For now my sweet child
func handleError(err error) {
	panic(err)
}
