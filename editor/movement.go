package editor

import (
	"os"
	"strings"
	"time"
	"unicode"
)

func (e *Editor) clampCursorX() {
	lineLen := 0
	if e.cursorY < e.buffer.LineCount() {
		lineLen = len([]rune(e.buffer.GetLine(e.cursorY)))
	}
	if e.cursorX > lineLen {
		e.cursorX = lineLen
	}
}

func (e *Editor) movePageUp() {
	e.cursorY -= e.termHeight
	if e.cursorY < 0 {
		e.cursorY = 0
	}
	e.clampCursorX()
}

func (e *Editor) movePageDown() {
	lineCount := e.buffer.LineCount()
	e.cursorY += e.termHeight
	if e.cursorY >= lineCount {
		e.cursorY = max(lineCount-1, 0)
	}
	e.clampCursorX()
}

func (e *Editor) moveLineStart(isSelecting bool) {
	if isSelecting && !e.selectionActive {
		e.selectionActive = true
		e.selectionAnchorX = e.cursorX
		e.selectionAnchorY = e.cursorY
	} else if !isSelecting {
		e.selectionActive = false
	}
	e.cursorX = 0
}

func (e *Editor) moveLineEnd(isSelecting bool) {
	if isSelecting && !e.selectionActive {
		e.selectionActive = true
		e.selectionAnchorX = e.cursorX
		e.selectionAnchorY = e.cursorY
	} else if !isSelecting {
		e.selectionActive = false
	}
	if e.cursorY < e.buffer.LineCount() {
		e.cursorX = len([]rune(e.buffer.GetLine(e.cursorY)))
	} else {
		e.cursorX = 0
	}
}

func (e *Editor) moveDocStart() {
	e.cursorY = 0
	e.cursorX = 0
}

func (e *Editor) moveDocEnd() {
	e.cursorY = e.buffer.LineCount() - 1
	if e.cursorY < 0 {
		e.cursorY = 0
	}
	e.cursorX = len([]rune(e.buffer.GetLine(e.cursorY)))
}

func (e *Editor) toggleLineNumbers() {
	e.showLineNumbers = !e.showLineNumbers
	e.updateLineNumWidth()
	if e.showLineNumbers {
		e.lineNumWidth = 5 // Restore width
		e.setStatusMessage("Line numbers ON")
	} else {
		e.lineNumWidth = 0 // Remove width
		e.setStatusMessage("Line numbers OFF")
	}
}

func (e *Editor) handleEscape() error {
	var b byte
	var err error
	if f, ok := e.term.Stdin().(*os.File); ok {
		f.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		b, err = e.inputReader.ReadByte()
		f.SetReadDeadline(time.Time{})
	} else {
		return nil
	}
	if err != nil {
		goto CANCEL_MODE
	}

	{
		seq := make([]byte, 0, 8)
		paramBuf := make([]byte, 0, 8)

		if b == '\x7f' || b == '\b' {
			if !e.isSaveAs && !e.isGotoLine && !e.isFinding && !e.isReplacing {
				e.handleDeleteWordLeft()
			}
			return nil
		}

		if b != '[' {
			e.inputReader.UnreadByte()
			goto CANCEL_MODE
		}

		readTimeout := time.After(10 * time.Millisecond)
		for {
			select {
			case <-readTimeout:
				return nil
			default:
				if e.inputReader.Buffered() == 0 {
					time.Sleep(1 * time.Millisecond)
					if e.inputReader.Buffered() == 0 {
						goto PARSE
					}
				}

				b, err := e.inputReader.ReadByte()
				if err != nil {
					goto PARSE
				}

				if b >= '0' && b <= '9' || b == ';' {
					paramBuf = append(paramBuf, b)
				} else {
					seq = append(seq, paramBuf...)
					seq = append(seq, b)
					goto PARSE
				}
			}
		}

	PARSE:
		if len(seq) == 0 {
			return nil
		}
		cmd := seq[len(seq)-1]
		params := string(paramBuf)

		// --- PROMPT NAVIGATION ---
		if e.isSaveAs || e.isGotoLine || e.isFinding || e.isReplacing {
			var curCursor *int
			var maxLen int

			if e.isReplacing && e.promptFocus == 1 {
				curCursor = &e.replaceCursorX
				maxLen = len([]rune(e.replaceBuffer))
			} else {
				curCursor = &e.promptCursorX
				maxLen = len([]rune(e.promptBuffer))
			}

			switch cmd {
			case 'D': // Left
				e.movePromptCursor(-1)
			case 'C': // Right
				e.movePromptCursor(1)
			case 'H', '1', 'A': // Home / Up
				if cmd == 'A' && e.isReplacing {
					e.promptFocus = 0 // Up arrow goes to Find input
				} else {
					*curCursor = 0
				}
			case 'F', '4', 'B': // End / Down
				if cmd == 'B' && e.isReplacing {
					e.promptFocus = 1 // Down arrow goes to Replace input
				} else {
					*curCursor = maxLen
				}
			case '~': // Delete
				if params == "3" {
					e.deletePromptRune()
				}
			}
			return nil
		}

		// --- MAIN EDITOR NAVIGATION ---
		switch cmd {
		case 'A', 'B', 'C', 'D': // Arrow keys
			isShift := false
			isCtrl := false
			isCtrlShift := false

			if strings.Contains(params, ";2") {
				isShift = true
			}
			if strings.Contains(params, ";5") {
				isCtrl = true
			}
			if strings.Contains(params, ";6") {
				isCtrl = true
				isShift = true
				isCtrlShift = true
			}

			if isCtrl {
				switch cmd {
				case 'C': // Ctrl+Right
					e.moveWordRight(isShift || isCtrlShift)
				case 'D': // Ctrl+Left
					e.moveWordLeft(isShift || isCtrlShift)
				default:
					e.handleArrowKey(cmd, isShift || isCtrlShift)
				}
			} else {
				e.handleArrowKey(cmd, isShift)
			}

		case 'H': // Home
			isCtrl := strings.Contains(params, "5")
			isShift := strings.Contains(params, "2")
			if isCtrl {
				e.moveDocStart()
			} else {
				e.moveLineStart(isShift)
			}

		case 'F': // End
			isCtrl := strings.Contains(params, "5")
			isShift := strings.Contains(params, "2")
			if isCtrl {
				e.moveDocEnd()
			} else {
				e.moveLineEnd(isShift)
			}

		case '~': // PageUp, PageDown, Delete, etc.
			switch params {
			case "1": // Home
				e.moveLineStart(false)
			case "1;2": // Shift+Home
				e.moveLineStart(true)
			case "4": // End
				e.moveLineEnd(false)
			case "4;2": // Shift+End
				e.moveLineEnd(true)
			case "5": // Page Up
				e.movePageUp()
			case "6": // Page Down
				e.movePageDown()
			case "3": // Delete key
				e.handleDeleteKey()
			case "3;5": // Ctrl+Delete
				e.handleDeleteWordRight()
			}
		}
		return nil
	}

CANCEL_MODE:
	if e.isConfirmingReplace {
		e.isConfirmingReplace = false
		e.setStatusMessage("Replace All cancelled.")
		return nil
	}
	if e.isReplacing {
		e.isReplacing = false
		e.isFinding = false
		e.findMatches = nil
		e.selectionActive = false
		e.setStatusMessage("Replace cancelled.")
		return nil
	}
	if e.isSaveAs {
		e.isSaveAs = false
		e.promptBuffer = ""
		e.setStatusMessage("Save As cancelled.")
		return nil
	}
	if e.isGotoLine {
		e.isGotoLine = false
		e.promptBuffer = ""
		e.setStatusMessage("Go to line cancelled.")
		return nil
	}
	if e.isFinding {
		e.isFinding = false
		e.promptBuffer = ""
		e.findMatches = nil
		e.findCurrentMatch = -1
		e.cursorX = e.findOrigCursorX
		e.cursorY = e.findOrigCursorY
		e.selectionActive = false
		e.setStatusMessage("Find cancelled.")
		return nil
	}
	return nil
}

func (e *Editor) handleArrowKey(direction byte, modified bool) {
	switch direction {
	case 'A': // Up
		e.moveCursor(0, -1, modified)
	case 'B': // Down
		e.moveCursor(0, 1, modified)
	case 'C': // Right
		e.moveCursor(1, 0, modified)
	case 'D': // Left
		e.moveCursor(-1, 0, modified)
	}
}

func (e *Editor) moveCursor(dx, dy int, isSelecting bool) {
	if !isSelecting {
		e.selectionActive = false
	} else if !e.selectionActive {
		e.selectionActive = true
		e.selectionAnchorX = e.cursorX
		e.selectionAnchorY = e.cursorY
	}
	if dy != 0 {
		e.cursorY += dy
		if e.cursorY < 0 {
			e.cursorY = 0
		}
		if e.cursorY >= e.buffer.LineCount() {
			e.cursorY = max(e.buffer.LineCount()-1, 0)
		}
		e.clampCursorX()
		return
	}
	if dx == -1 && e.cursorX == 0 && e.cursorY > 0 {
		e.cursorY--
		e.cursorX = len([]rune(e.buffer.GetLine(e.cursorY)))
		return
	}
	currentLineLen := 0
	if e.cursorY < e.buffer.LineCount() {
		currentLineLen = len([]rune(e.buffer.GetLine(e.cursorY)))
	}
	if dx == 1 && e.cursorX == currentLineLen && e.cursorY < e.buffer.LineCount()-1 {
		e.cursorY++
		e.cursorX = 0
		return
	}
	e.cursorX += dx
	if e.cursorX < 0 {
		e.cursorX = 0
	}
	e.clampCursorX()
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_'
}

func isPunctChar(r rune) bool {
	return !isWordChar(r) && !unicode.IsSpace(r)
}

func (e *Editor) moveWordRight(isSelecting bool) {
	if isSelecting && !e.selectionActive {
		e.selectionActive = true
		e.selectionAnchorX = e.cursorX
		e.selectionAnchorY = e.cursorY
	} else if !isSelecting {
		e.selectionActive = false
	}

	y, x := e.cursorY, e.cursorX
	lineRunes := []rune(e.buffer.GetLine(y))
	lineLen := len(lineRunes)

	if x == lineLen {
		if y < e.buffer.LineCount()-1 {
			e.cursorY++
			e.cursorX = 0
			y = e.cursorY
			x = 0
			lineRunes = []rune(e.buffer.GetLine(y))
			lineLen = len(lineRunes)
		} else {
			return
		}
	}
	if x < lineLen {
		r := lineRunes[x]

		if isWordChar(r) {
			for x < lineLen && isWordChar(lineRunes[x]) {
				x++
			}
		} else if isPunctChar(r) {
			for x < lineLen && isPunctChar(lineRunes[x]) {
				x++
			}
		}
		for x < lineLen && unicode.IsSpace(lineRunes[x]) {
			x++
		}
	}

	e.cursorY = y
	e.cursorX = x
}

func (e *Editor) moveWordLeft(isSelecting bool) {
	if isSelecting && !e.selectionActive {
		e.selectionActive = true
		e.selectionAnchorX = e.cursorX
		e.selectionAnchorY = e.cursorY
	} else if !isSelecting {
		e.selectionActive = false
	}

	y, x := e.cursorY, e.cursorX

	if x == 0 {
		if y > 0 {
			e.cursorY--
			e.cursorX = len([]rune(e.buffer.GetLine(e.cursorY)))
		}
		return
	}
	x--
	lineRunes := []rune(e.buffer.GetLine(y))
	for x >= 0 && unicode.IsSpace(lineRunes[x]) {
		x--
	}
	if x < 0 {
		e.cursorY = y
		e.cursorX = 0
		return
	}
	if isWordChar(lineRunes[x]) {
		for x >= 0 && isWordChar(lineRunes[x]) {
			x--
		}
	} else if isPunctChar(lineRunes[x]) {
		for x >= 0 && isPunctChar(lineRunes[x]) {
			x--
		}
	}
	e.cursorY = y
	e.cursorX = x + 1
}
