package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
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

type Mode int

const (
	NormalMode Mode = iota
	InsertMode
)

type Editor struct {
	// Width and height of the editor (the terminal window)
	width  int
	height int

	// The file currently being worked on. Only supports
	// one file for now. How quaint!
	file *File

	// topline is the top most visible line in the window
	topline int

	// cursor coordinates
	cursY     int
	cursX     int
	lastCursX int

	// visual cursor
	visCursX int

	// mode
	mode Mode

	workingLine []string
	clipboard   string

	isDeleting   bool
	isChanging   bool
	isYanking    bool
	isCommanding bool

	// This is super ghetto but just for now
	// A way to kill the infinite for loop looking for
	// key strokes.
	isOpen bool
}

func NewEditor(w, h int) *Editor {
	return &Editor{
		width:  w,
		height: h,

		cursY: 0,
		cursX: 0,

		visCursX: 1,

		mode: NormalMode,

		isOpen: true,
	}
}

func (e *Editor) CurLine() int {
	return e.topline + e.cursY
}

func (e *Editor) OpenFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_RDWR, 0)
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
	e.topline = 0
	return nil
}

func (e *Editor) drawLine(n int) {
	if n != e.topline {
		fmt.Print("\r\n")
	}
	//fmt.Printf("%3d: %s", n, e.file.contents[n])
	fmt.Print(e.file.contents[n])
}

func (e *Editor) redraw() {
	fmt.Print("\x1b[?25l") // hide cursor
	fmt.Print("\x1b[2J")   //clear screen
	fmt.Print("\x1b[H")    // reset to home
	e.drawLine(e.topline)
	for i, j := e.topline+1, 1; j < e.height && i < len(e.file.contents); i, j = i+1, j+1 {
		e.drawLine(i)
	}
	// place cursor
	fmt.Printf("\x1b[%d;%dH", e.cursY+1, e.visCursX) // reposition cursor
	fmt.Print("\x1b[?25h")                           // make cursor visible
}

func (e *Editor) saveFile() error {
	err := os.WriteFile(e.file.raw.Name(), []byte(strings.Join(e.file.contents, "\n")), 0)
	return err
}

func (e *Editor) resetVisualCursor() {
	e.visCursX = 1
	line := e.file.contents[e.CurLine()]
	xEnd := len(line)
	if e.mode == NormalMode {
		xEnd--
	}
	if e.cursX > xEnd {
		if len(line) > 0 {
			e.cursX = xEnd
		} else {
			e.cursX = 0
		}
	}
	for _, char := range line[:e.cursX] {
		if char == '\t' {
			// I don't really understand why this is necessary
			e.visCursX--
			e.visCursX += 8
			e.visCursX -= e.visCursX % 8
			e.visCursX++
		} else {
			e.visCursX++
		}
	}

	if e.CurLine()-e.topline >= e.height {
		e.topline++
		e.cursY--
	} else if e.CurLine() < e.topline {
		e.topline--
		e.cursY++
	}

	e.redraw()
}

func (e *Editor) handleKeyPress(b []byte) {
	switch e.mode {
	case NormalMode:
		keyString := string(b)
		if b[0] == 4 { // Ctrl-D
			e.topline = (e.topline + e.height/2)
			if e.topline >= len(e.file.contents) {
				e.topline = len(e.file.contents) - 1
			}
		} else if b[0] == 19 { // Ctrl-S
			err := e.saveFile()
			if err != nil {
				handleError(err)
			}
		} else if b[0] == 21 { // Ctrl-U
			e.topline = (e.topline - e.height/2)
			if e.topline < 0 {
				e.topline = 0
			}
		} else if keyString == "j" {
			if e.CurLine() < len(e.file.contents)-1 {
				e.cursY++
				e.cursX = e.lastCursX
				e.resetVisualCursor()
			}
		} else if keyString == "k" {
			if e.CurLine() > 0 {
				e.cursY--
				e.cursX = e.lastCursX
				e.resetVisualCursor()
			}
		} else if keyString == "h" {
			if e.cursX > 0 {
				e.cursX--
				e.lastCursX = e.cursX
				e.resetVisualCursor()
			}
		} else if keyString == "l" {
			if e.cursX < len(e.file.contents[e.CurLine()]) {
				e.cursX++
				e.lastCursX = e.cursX
				e.resetVisualCursor()
			}
		} else if keyString == "G" {
			e.topline = len(e.file.contents) - 5
		} else if keyString == "i" {
			if e.mode == NormalMode {
				//change cursor to bar
				e.mode = InsertMode
				fmt.Print("\x1b[5 q")
				e.workingLine = strings.Split(e.file.contents[e.CurLine()], "")
			}
		} else if keyString == "a" {
			//change cursor to bar
			e.mode = InsertMode
			fmt.Print("\x1b[5 q")
			e.workingLine = strings.Split(e.file.contents[e.CurLine()], "")
			e.cursX++
			e.resetVisualCursor()
		} else if keyString == "A" {
			e.mode = InsertMode
			fmt.Print("\x1b[5 q")
			e.workingLine = strings.Split(e.file.contents[e.CurLine()], "")
			e.cursX = len(e.workingLine)
			e.resetVisualCursor()
		} else if keyString == "c" {
			if e.isChanging {
				// Delete line and enter insert mode
				e.mode = InsertMode
				e.workingLine = e.workingLine[:0]
				e.file.contents[e.CurLine()] = ""
				e.cursX = 0
				e.resetVisualCursor()
				e.isChanging = false
			} else {
				e.isChanging = true
			}
		} else if keyString == "D" {
			e.file.contents[e.CurLine()] = e.file.contents[e.CurLine()][:e.cursX]
			e.cursX--
		} else if keyString == "d" {
			if e.isDeleting {
				e.clipboard = e.file.contents[e.CurLine()]
				e.file.contents = append(e.file.contents[:e.CurLine()], e.file.contents[e.CurLine()+1:]...)
				e.isDeleting = false
			} else {
				e.isDeleting = true
			}
		} else if keyString == "P" {
			if len(e.clipboard) > 0 {
				e.file.contents = slices.Insert(e.file.contents, e.CurLine(), e.clipboard)
				e.cursY++
			}
		} else if keyString == "p" {
			if len(e.clipboard) > 0 {
				e.file.contents = slices.Insert(e.file.contents, e.CurLine()+1, e.clipboard)
			}
		} else if keyString == "y" {
			if e.isYanking {
				e.clipboard = e.file.contents[e.CurLine()]
				e.isYanking = false
			}
			e.isYanking = true
		} else if keyString == "O" {
			if e.mode == NormalMode {
				e.mode = InsertMode
				fmt.Print("\x1b[5 q")
				e.workingLine = []string{}
				e.file.contents = slices.Insert(e.file.contents, e.CurLine(), "")
				e.resetVisualCursor()
			}
		} else if keyString == "o" {
			if e.mode == NormalMode {
				//change cursor to bar
				e.mode = InsertMode
				fmt.Print("\x1b[5 q")
				e.workingLine = []string{}
				e.file.contents = slices.Insert(e.file.contents, e.CurLine()+1, "")
				e.cursY++
				e.resetVisualCursor()
			}
		} else if keyString == "w" {
			if e.isCommanding {
				err := e.saveFile()
				if err != nil {
					panic(err)
				}
			}
		} else if keyString == "q" {
			if e.isCommanding {
				e.isOpen = false
			}
		} else if keyString == ":" {
			e.isCommanding = true
		}
		e.redraw()
	case InsertMode:
		if b[0] == 27 { // escape
			e.mode = NormalMode
			e.isCommanding = false
			e.isYanking = false
			e.isChanging = false
			e.isDeleting = false
			fmt.Printf("\x1b[0 q")
			e.resetVisualCursor()
		} else if b[0] == 13 { // enter/return
			newLine := e.workingLine[e.cursX:]
			e.workingLine = e.workingLine[:e.cursX]
			e.file.contents[e.CurLine()] = strings.Join(e.workingLine, "")
			e.file.contents = slices.Insert(e.file.contents, e.CurLine()+1,
				strings.Join(newLine, ""))
			e.workingLine = newLine
			e.cursY++
			e.cursX = 0
			e.resetVisualCursor()
		} else if b[0] == 127 { //TODO: this is not robust (backspace)
			e.delete(e.cursX - 1)
		} else if b[0] >= 32 || b[0] <= 126 {
			e.insert(string(b))
		}
	}
}

func (e *Editor) delete(idx int) {
	// Handling backspacing into the previous line
	if idx == -1 {
		if e.cursY == 0 {
			return
		}
		above := e.file.contents[e.CurLine()-1]
		e.cursX = len(above)
		e.file.contents[e.CurLine()-1] = above +
			strings.Join(e.workingLine, "")
		e.file.contents = append(e.file.contents[:e.CurLine()], e.file.contents[e.CurLine()+1:]...)
		e.cursY--
		e.workingLine = strings.Split(e.file.contents[e.CurLine()], "")
	} else {
		e.workingLine = append(e.workingLine[:idx], e.workingLine[idx+1:]...)
		e.file.contents[e.CurLine()] = strings.Join(e.workingLine, "")
		e.cursX--
	}
	e.resetVisualCursor()
}

func (e *Editor) insert(char string) {
	e.workingLine = slices.Insert(e.workingLine, e.cursX, char)
	e.file.contents[e.CurLine()] = strings.Join(e.workingLine, "")
	e.cursX++
	e.resetVisualCursor()
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
	for e.isOpen {
		n, err := tty.Read(b[:])
		if err != nil {
			panic(err)
		}
		e.handleKeyPress(b[:n])
	}
}
