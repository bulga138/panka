//go:build windows

package terminal

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/windows"
)

type Terminal interface {
	EnableRawMode() error
	DisableRawMode() error
	GetWindowSize() (width, height int, err error)
	Stdin() io.Reader
	Close() error
}

type stdTerminal struct {
	originalState *winState
	stdinFile     *os.File
}

type winState [2]uint32

func New() Terminal {
	conInHandle, err := windows.CreateFile(
		windows.StringToUTF16Ptr("CONIN$"),
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return &stdTerminal{stdinFile: os.Stdin}
	}

	stdinFile := os.NewFile(uintptr(conInHandle), "CONIN$")

	return &stdTerminal{
		stdinFile: stdinFile,
	}
}

func (t *stdTerminal) Close() error {
	if t.stdinFile != nil && t.stdinFile != os.Stdin {
		return t.stdinFile.Close()
	}
	return nil
}

func (t *stdTerminal) Stdin() io.Reader {
	return t.stdinFile
}

func (t *stdTerminal) EnableRawMode() error {
	inHandle := windows.Handle(t.stdinFile.Fd())
	outHandle := windows.Handle(os.Stdout.Fd())

	if inHandle == windows.InvalidHandle || outHandle == windows.InvalidHandle {
		return fmt.Errorf("invalid std handles")
	}

	var inMode, outMode uint32
	if err := windows.GetConsoleMode(inHandle, &inMode); err != nil {
		return fmt.Errorf("failed to get stdin console mode: %w", err)
	}
	if err := windows.GetConsoleMode(outHandle, &outMode); err != nil {
		return fmt.Errorf("failed to get stdout console mode: %w", err)
	}

	t.originalState = &winState{inMode, outMode}

	newInMode := inMode &^ (windows.ENABLE_ECHO_INPUT | windows.ENABLE_LINE_INPUT | windows.ENABLE_PROCESSED_INPUT)
	newInMode |= windows.ENABLE_VIRTUAL_TERMINAL_INPUT

	newOutMode := outMode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING

	if err := windows.SetConsoleMode(inHandle, newInMode); err != nil {
		return fmt.Errorf("failed to set stdin console mode: %w", err)
	}
	if err := windows.SetConsoleMode(outHandle, newOutMode); err != nil {
		windows.SetConsoleMode(inHandle, inMode) // Revertir
		return fmt.Errorf("failed to set stdout console mode: %w", err)
	}

	return nil
}

func (t *stdTerminal) DisableRawMode() error {
	if t.originalState == nil {
		return nil
	}

	inHandle := windows.Handle(t.stdinFile.Fd())
	outHandle := windows.Handle(os.Stdout.Fd())

	if inHandle == windows.InvalidHandle || outHandle == windows.InvalidHandle {
		return fmt.Errorf("invalid std handles")
	}

	windows.SetConsoleMode(inHandle, t.originalState[0])
	windows.SetConsoleMode(outHandle, t.originalState[1])

	return nil
}

func (t *stdTerminal) GetWindowSize() (width, height int, err error) {
	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr("CONOUT$"),
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get CONOUT$: %w", err)
	}
	defer windows.CloseHandle(handle)

	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(handle, &info); err != nil {
		return 0, 0, fmt.Errorf("failed to get console screen buffer info: %w", err)
	}
	width = int(info.Window.Right - info.Window.Left + 1)
	height = int(info.Window.Bottom - info.Window.Top + 1)
	return width, height, nil
}
