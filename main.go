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

const ioctlReadTermios = unix.TCGETS
const ioctlWriteTermios = unix.TCSETS

type state struct {
	termios unix.Termios
}

func rawDog(fd int) error {
	termios, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return err
	}

	// This is copy/paste job of the make raw behavior of the term lib from
	// golang.org/x/term but with some modifications by me.
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	//termios.Oflag &^= unix.OPOST
	//termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, termios); err != nil {
		return err
	}

	return nil
}

func main() {
	// Boilerplate from https://pkg.go.dev/golang.org/x/term#pkg-overview
	// Raw mode let's us worry about terminal sequences ourselves instead of
	// the terminal handling them.
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		handleError(err)
	}

	err = rawDog(int(os.Stdin.Fd()))
	if err != nil {
		handleError(err)
	}
	//// Return to "cooked" terminal
	defer term.Restore(int(os.Stdin.Fd()), oldState)

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
