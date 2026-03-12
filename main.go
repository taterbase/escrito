package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

var (
	usageMsg = `
Example: esc file.go
`

	wg sync.WaitGroup
)

type state struct {
	termios unix.Termios
}

func setupTerminal(fd int) (func(), error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	termios, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return nil, err
	}

	// Turn post processing of output back on (for now)
	termios.Oflag |= unix.OPOST
	// Turn interrupt signal handling back on (ctrl-c, ctrl-d)
	termios.Lflag |= unix.ISIG
	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, termios); err != nil {
		return nil, err
	}

	return func() { term.Restore(int(os.Stdin.Fd()), oldState) }, nil
}

func main() {
	// Boilerplate from https://pkg.go.dev/golang.org/x/term#pkg-overview
	// Raw mode let's us worry about terminal sequences ourselves instead of
	// the terminal handling them.
	cleanupTerminal, err := setupTerminal(int(os.Stdin.Fd()))
	if err != nil {
		handleError(err)
	}
	defer cleanupTerminal()

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
	t := time.NewTicker(1 * time.Second)
	for {
		_, h, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			handleError(err)
		}
		fmt.Print("\033[2J")
		fmt.Print("\033[H")
		fmt.Print(e.file.contents[0])
		for i := 1; i < h; i++ {
			fmt.Print("\n" + e.file.contents[i])
		}

		select {
		case <-t.C:
			continue
		}
	}
}
