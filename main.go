package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

var (
	usageMsg = `
Example: esc file.go
`

	wg sync.WaitGroup
)

func main() {
	// Boilerplate from https://pkg.go.dev/golang.org/x/term#pkg-overview
	// Raw mode let's us worry about terminal sequences ourselves instead of
	// the terminal handling them.
	//oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	//if err != nil {
	//	handleError(err)
	//}
	//// Return to "cooked" terminal
	//defer term.Restore(int(os.Stdin.Fd()), oldState)

	//TODO: holding off on raw until we need it. Right now we're just a basic
	//			viewer.

	if len(os.Args) != 2 {
		usage()
		return
	}
	filename := (os.Args[1])

	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		handleError(err)
	}

	editor := NewEditor(w, h)
	err = editor.OpenFile(filename)
	if err != nil {
		handleError(err)
	}
	wg.Add(1)
	editor.Display()
	wg.Wait()
}

// Teach'em son
func usage() {
	fmt.Println(usageMsg)
}

// For now my sweet child
func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

type File struct {
	raw      *os.File
	contents []string // starting with a slice of strings... rough
}

type Editor struct {
	// Width and height of the editor (the terminal window)
	width  int
	height int

	// The file currently being worked on. Only supports
	// one file for now. How quaint!
	file *File
}

func NewEditor(w, h int) *Editor {
	return &Editor{
		width:  w,
		height: h,
	}
}

func (e *Editor) OpenFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	contents, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("Error reading file: %w", err)
	}

	lines := strings.Split(string(contents), "\n")

	e.file = &File{
		raw:      file,
		contents: lines,
	}
	return nil
}

func (e *Editor) Display() {
	t := time.NewTicker(30 * time.Second)
	for {
		fmt.Print(e.file.contents[0])
		for i := 1; i < e.height; i++ {
			fmt.Print("\n" + e.file.contents[i])
		}

		select {
		case <-t.C:
			continue
		}
	}
}
