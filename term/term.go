package term

import (
	"os"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func SetupTerminal(fd int) (func(), error) {
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
