package main

import (
	"fmt"
	"os"
)

var usageMsg = `
Example: esc file.go
`

func main() {
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
}

// Teach'em son
func usage() {
	fmt.Println(usageMsg)
}

// For now my sweet child
func handleError(err error) {
	panic(err)
}
