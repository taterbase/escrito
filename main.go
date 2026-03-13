package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
	"golang.org/x/term"

	tterm "tatersoft.com/escrito/term"
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

func main() {
	// Boilerplate from https://pkg.go.dev/golang.org/x/term#pkg-overview
	// Raw mode let's us worry about terminal sequences ourselves instead of
	// the terminal handling them.
	cleanupTerminal, err := tterm.SetupTerminal(int(os.Stdin.Fd()))
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
	editor.Display()
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

	// curline is the current line of the file
	curline int

	cursX int
	cursY int
}

func NewEditor(w, h int) *Editor {
	return &Editor{
		width:  w,
		height: h,

		cursX: 1,
		cursY: 1,
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
	e.curline = 0
	return nil
}

func (e *Editor) drawLine(n int) {
	if n != e.curline {
		fmt.Print("\r\n")
	}
	//fmt.Printf("%3d: %s", n, e.file.contents[n])
	fmt.Print(e.file.contents[n])
}

func (e *Editor) redraw() {
	fmt.Print("\x1b[?25l") // hide cursor
	fmt.Print("\x1b[2J")   //clear screen
	fmt.Print("\x1b[H")    // reset to home
	e.drawLine(e.curline)
	for i, j := e.curline+1, 1; j < e.height && i < len(e.file.contents); i, j = i+1, j+1 {
		e.drawLine(i)
	}
	// place cursor
	fmt.Printf("\x1b[%d;%dH", e.cursX, e.cursY) // reposition cursor
	fmt.Print("\x1b[?25h")                      // make cursor visible
}

func (e *Editor) Display() {
	e.redraw()

	// Better to use /dev/tty as it will always be user input at terminal.
	// Pulling from stdin would not work if something was piped in
	tty, err := os.Open("/dev/tty")
	if err != nil {
		panic(err)
	}

	resize := make(chan os.Signal, 1)
	signal.Notify(resize, syscall.SIGWINCH)
	go func() {
		for range resize {
			_, h, err := term.GetSize(int(os.Stdin.Fd()))
			if err != nil {
				handleError(err)
			}
			e.height = h
			e.redraw()
		}
	}()

	var b [256]byte
	for {
		n, err := tty.Read(b[:])
		if err != nil {
			panic(err)
		}
		key := string(b[:n])
		if b[0] == 4 {
			e.curline = (e.curline + e.height/2)
			if e.curline >= len(e.file.contents) {
				e.curline = len(e.file.contents) - 1
			}
		} else if b[0] == 21 {
			e.curline = (e.curline - e.height/2)
			if e.curline < 0 {
				e.curline = 0
			}
		} else if key == "j" {
			e.cursX++
		} else if key == "k" {
			e.cursX--
		} else if key == "h" {
			e.cursY--
		} else if key == "l" {
			e.cursY++
		} else if key == "G" {
			e.curline = len(e.file.contents) - 5
		} else if key == "q" {
			return
		}
		e.redraw()
	}
}
