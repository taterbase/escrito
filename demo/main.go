package main

import (
	"fmt"
	"os"

	"tatersoft.com/escrito/term"
)

func main() {
	resetState, err := term.SetupTerminal(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer resetState()

	tty, err := os.Open("/dev/tty")
	if err != nil {
		panic(err)
	}
	defer tty.Close()

	var b [256]byte
	for {
		n, _ := tty.Read(b[:])
		fmt.Print(n, string(b[:n]), b[0] == byte(4))
	}
}
