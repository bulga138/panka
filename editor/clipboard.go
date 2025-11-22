package editor

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---------- Clipboard / Paste / Cut ----------

func (e *Editor) pasteFromClipboard() error {
	text, err := e.getClipboardText()
	if err != nil {
		e.setStatusMessage("Paste failed: %v", err)
		return nil
	}
	if text == "" {
		e.setStatusMessage("Clipboard is empty")
		return nil
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	e.flushTypingAndBackspaceIfNeeded()
	
	// Always group paste operations as a single undo action
	e.beginUndoGroup()
	defer e.endUndoGroup()
	
	entries := make([]opEntry, 0, len([]rune(text)))
	for _, r := range []rune(text) {
		insertLine := e.cursorY
		insertCol := e.cursorX
		if r == '\n' {
			if err := e.buffer.Insert(e.cursorY, e.cursorX, '\n'); err != nil {
				e.setStatusMessage("Paste error: %v", err)
				return err
			}
			e.cursorY++
			e.cursorX = 0
		} else {
			if err := e.buffer.Insert(e.cursorY, e.cursorX, r); err != nil {
				e.setStatusMessage("Paste error: %v", err)
				return err
			}
			e.cursorX++
		}
		delLine := e.cursorY
		delCol := e.cursorX
		entries = append(entries, opEntry{
			insertLine: insertLine, insertCol: insertCol,
			delLine: delLine, delCol: delCol,
			r: r,
		})
	}
	// Push all entries as a single grouped undo action
	e.pushUndoInsertBlock(entries)
	e.dirty = true
	e.setStatusMessage("Pasted from clipboard")
	return nil
}

func (e *Editor) getClipboardText() (string, error) {
	return getClipboardTextWindows()
}

// Windows clipboard implementation for getting text
func getClipboardTextWindows() (string, error) {
	user32 := windows.NewLazyDLL("user32.dll")
	kernel32 := windows.NewLazyDLL("kernel32.dll")

	// Get required functions
	openClipboard := user32.NewProc("OpenClipboard")
	closeClipboard := user32.NewProc("CloseClipboard")
	getClipboardData := user32.NewProc("GetClipboardData")
	globalLock := kernel32.NewProc("GlobalLock")
	globalUnlock := kernel32.NewProc("GlobalUnlock")

	// Open clipboard
	hwnd := uintptr(0) // NULL
	ret, _, _ := openClipboard.Call(hwnd)
	if ret == 0 {
		return "", fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard.Call()

	// Get clipboard data (CF_UNICODETEXT = 13)
	cfUnicodeText := uintptr(13)
	hMem, _, _ := getClipboardData.Call(cfUnicodeText)
	if hMem == 0 {
		return "", fmt.Errorf("failed to get clipboard data")
	}

	// Lock memory
	ptr, _, _ := globalLock.Call(hMem)
	if ptr == 0 {
		return "", fmt.Errorf("failed to lock global memory")
	}
	defer globalUnlock.Call(hMem)

	// Convert UTF-16 to Go string
	var result []uint16
	for i := 0; ; i++ {
		c := *(*uint16)(unsafe.Pointer(ptr + uintptr(i*2)))
		if c == 0 {
			break
		}
		result = append(result, c)
	}

	return windows.UTF16ToString(result), nil
}

func (e *Editor) selectAll() error {
	if e.buffer.LineCount() == 0 {
		return nil
	}
	e.selectionActive = true
	e.selectionAnchorX = 0
	e.selectionAnchorY = 0
	lastLine := e.buffer.LineCount() - 1
	lastLineContent := e.buffer.GetLine(lastLine)
	e.cursorY = lastLine
	e.cursorX = len([]rune(lastLineContent))
	e.setStatusMessage("Selected all text")
	return nil
}

func (e *Editor) setClipboardText(text string) error {
	return setClipboardTextWindows(text)
}

// Windows clipboard implementation using Windows API
func setClipboardTextWindows(text string) error {
	kernel32 := windows.NewLazyDLL("kernel32.dll")
	user32 := windows.NewLazyDLL("user32.dll")

	// Get required functions
	globalAlloc := kernel32.NewProc("GlobalAlloc")
	globalLock := kernel32.NewProc("GlobalLock")
	globalUnlock := kernel32.NewProc("GlobalUnlock")
	openClipboard := user32.NewProc("OpenClipboard")
	emptyClipboard := user32.NewProc("EmptyClipboard")
	setClipboardData := user32.NewProc("SetClipboardData")
	closeClipboard := user32.NewProc("CloseClipboard")

	// Convert string to Windows UTF-16
	utf16Text, err := windows.UTF16FromString(text)
	if err != nil {
		return err
	}

	// Allocate global memory
	GMEM_MOVEABLE := uintptr(0x0002)
	size := uintptr((len(utf16Text) + 1) * 2) // +1 for null terminator, *2 for UTF-16
	hMem, _, _ := globalAlloc.Call(GMEM_MOVEABLE, size)
	if hMem == 0 {
		return fmt.Errorf("failed to allocate global memory")
	}

	// Lock memory
	ptr, _, _ := globalLock.Call(hMem)
	if ptr == 0 {
		return fmt.Errorf("failed to lock global memory")
	}
	defer globalUnlock.Call(hMem)

	// Copy text to memory
	dst := (*[1 << 30]byte)(unsafe.Pointer(ptr))[:size:size]
	for i, v := range utf16Text {
		dst[i*2] = byte(v)
		dst[i*2+1] = byte(v >> 8)
	}

	// Open clipboard
	hwnd := uintptr(0) // NULL
	ret, _, _ := openClipboard.Call(hwnd)
	if ret == 0 {
		return fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard.Call()

	// Empty clipboard
	emptyClipboard.Call()

	// Set clipboard data (CF_UNICODETEXT = 13)
	cfUnicodeText := uintptr(13)
	setClipboardData.Call(cfUnicodeText, hMem)

	return nil
}

func (e *Editor) copyToClipboard() error {
	content := e.getSelectedText()
	if content == "" {
		// When copying a single line, include the newline character
		// so that pasting it will create a new line
		content = e.buffer.GetLine(e.cursorY) + "\n"
	}
	content = strings.ReplaceAll(content, "\n", "\r\n")
	err := e.setClipboardText(content)
	if err != nil {
		e.setStatusMessage("Copy failed: %v", err)
		return nil
	}
	e.setStatusMessage("Copied to clipboard")
	return nil
}

func (e *Editor) cutToClipboard() error {
	e.flushTypingAndBackspaceIfNeeded()
	content := e.getSelectedText()
	if content == "" {
		// When cutting a single line, include the newline character
		// so that pasting it will create a new line
		content = e.buffer.GetLine(e.cursorY) + "\n"
		e.beginUndoGroup()
		e.deleteCurrentLine()
		e.endUndoGroup()
	} else if e.selectionActive {
		e.beginUndoGroup()
		e.deleteSelectedText()
		e.endUndoGroup()
	}
	content = strings.ReplaceAll(content, "\n", "\r\n")
	err := e.setClipboardText(content)
	if err != nil {
		e.setStatusMessage("Cut failed: %v", err)
		return nil
	}
	e.setStatusMessage("Cut to clipboard")
	e.dirty = true
	return nil
}
