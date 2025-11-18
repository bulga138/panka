Terminal Package

This package is the crucial, low-level, OS-specific layer that allows Panka to take control of the terminal.

Purpose: To enable "raw mode," disable echo, and get the window size.

Design Decisions:

Go Build Tags: This is the standard Go way to handle platform-specific code. The //go:build directive at the top of the files tells the compiler which file to use for which OS.

golang.org/x/sys: This is our only dependency. It's maintained by the Go team and is the official, idiomatic way to make OS-level system calls (syscalls) in Go. We use it to avoid Cgo.

terminal.go: Defines the simple, cross-platform interface (EnableRawMode, DisableRawMode, GetWindowSize) that the editor will use.

terminal_unix.go: (Linux & macOS) Uses golang.org/x/sys/unix to manipulate the termios struct. This is the POSIX-standard way to control terminal behavior.

terminal_windows.go: (Windows) Uses golang.org/x/sys/windows to get the console handle and set its mode. We set ENABLE_VIRTUAL_TERMINAL_PROCESSING (to enable ANSI escape codes) and disable ENABLE_ECHO_INPUT and ENABLE_LINE_INPUT.

Future Improvements:

More Features: This could be expanded to include functions for setting colors, mouse reporting, etc., if we decide not to use ANSI codes for them.