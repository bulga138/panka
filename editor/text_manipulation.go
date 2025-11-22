package editor

import (
	"strings"
	"unicode"
)

// Case types
const (
	caseLower = iota
	caseTitle
	caseUpper
	caseMixed
)

// unindentLine removes indentation from the start of the line(s).
// It handles multi-cursor ranges.
func (e *Editor) unindentLine() {
	if e.buffer.LineCount() == 0 {
		return
	}

	e.beginUndoGroup()
	defer e.endUndoGroup()

	// Handle potential multi-cursor range
	startLine, endLine := e.getMultiCursorRange()

	// Keep track if we actually changed anything
	changed := false

	for i := startLine; i <= endLine; i++ {
		if i >= e.buffer.LineCount() {
			continue
		}

		line := e.buffer.GetLine(i)
		runes := []rune(line)
		if len(runes) == 0 {
			continue
		}

		removeCount := 0

		// Check what is at the start of the line
		if runes[0] == '\t' {
			removeCount = 1
		} else if runes[0] == ' ' {
			// Count spaces up to TabSize
			for j := 0; j < e.config.TabSize && j < len(runes); j++ {
				if runes[j] == ' ' {
					removeCount++
				} else {
					break
				}
			}
		}

		if removeCount > 0 {
			changed = true
			// Delete the characters from the start of the line
			for k := 0; k < removeCount; k++ {
				char := runes[k]
				// We are deleting at index 0 repeatedly.
				// For undo, we record the deletion at index 0.
				e.pushUndoDeleteIfExternalGrouping(i, 0, char)

				// FIX: buffer.Delete(i, col) deletes the char BEFORE col.
				// To delete the char at index 0, we must backspace from index 1.
				// Previous code used Delete(i, 0) which deleted the previous newline.
				e.buffer.Delete(i, 1)
			}

			// Adjust cursor if this is the main cursor line
			if i == e.cursorY {
				e.cursorX -= removeCount
				if e.cursorX < 0 {
					e.cursorX = 0
				}
			}
		}
	}

	if changed {
		e.dirty = true
	}
}

// duplicateLine duplicates the current line content to the next line.
func (e *Editor) duplicateLine() {
	if e.buffer.LineCount() == 0 {
		return
	}

	// 1. Save original state
	origX := e.cursorX
	origY := e.cursorY

	// 2. Get content to duplicate
	lineContent := e.buffer.GetLine(origY)

	// 3. Determine insertion strategy
	var textToInsert string

	e.beginUndoGroup()

	if origY == e.buffer.LineCount()-1 {
		// Last line case: we must append a newline before the content
		// and insert at the end of the current line.
		textToInsert = "\n" + lineContent
		e.cursorX = len([]rune(lineContent))
		// cursorY stays at origY
	} else {
		// Normal case: insert content + newline at the start of the NEXT line.
		// This pushes existing next lines down.
		textToInsert = lineContent + "\n"
		e.cursorY = origY + 1
		e.cursorX = 0
	}

	// 4. Perform insertion
	e.insertString(textToInsert)

	// 5. Restore cursor to original position
	e.cursorY = origY
	e.cursorX = origX
	e.clampCursorX()

	e.endUndoGroup()
	e.dirty = true
}

// moveLineUp moves the current line up by swapping it with the line above.
func (e *Editor) moveLineUp() {
	if e.cursorY == 0 {
		return
	}

	e.beginUndoGroup()
	defer e.endUndoGroup()

	// Save state
	origX := e.cursorX
	origY := e.cursorY

	// Get content of the two lines to swap
	prevY := origY - 1
	currY := origY

	prevContent := e.buffer.GetLine(prevY)
	currContent := e.buffer.GetLine(currY)

	// Check if the bottom line (current) is the last line of the file
	isLastLine := (currY == e.buffer.LineCount()-1)

	// Delete the current line first (to keep indices stable for the previous line)
	e.cursorY = currY
	e.cursorX = 0
	e.deleteCurrentLine()

	// Delete the previous line
	e.cursorY = prevY
	e.cursorX = 0
	e.deleteCurrentLine()

	// Cursor is now at prevY. Insert the lines in swapped order.
	// New order: currContent, then prevContent.

	// 1. Insert currContent (which moves UP)
	e.insertString(currContent)

	// Always add a newline after the first inserted line
	// FIX: Use insertString ensures this newline is recorded in undo history
	e.insertString("\n")

	// 2. Insert prevContent (which moves DOWN)
	e.insertString(prevContent)

	// If the original bottom line was NOT the last line, we need to ensure
	// the new bottom line (prevContent) has a newline after it.
	if !isLastLine {
		// FIX: Use insertString ensures this newline is recorded in undo history
		e.insertString("\n")
	}

	// Restore cursor (it moves up with the line)
	e.cursorY = origY - 1
	e.cursorX = origX
	e.clampCursorX()
	e.dirty = true
}

// moveLineDown moves the current line down by swapping it with the line below.
func (e *Editor) moveLineDown() {
	if e.cursorY >= e.buffer.LineCount()-1 {
		return
	}

	// Moving line Y down is exactly the same as moving line Y+1 UP.
	// We just need to adjust the final cursor position to follow the line down.

	// Save cursor X
	origX := e.cursorX
	// Target Y is the line below
	targetY := e.cursorY + 1

	// Temporarily move cursor to the line below so we can use moveLineUp logic
	e.cursorY = targetY

	// Call moveLineUp on the line below (swaps it with current)
	e.moveLineUp()

	// moveLineUp moves the cursor to targetY - 1 (which is our original Y).
	// But since we effectively moved our line DOWN, we want cursor at origY + 1.
	e.cursorY = targetY
	e.cursorX = origX
	e.clampCursorX()
}

// toggleCaseAtCursor cycles the casing of the word under the cursor.
// Cycle: Lower -> Title -> Upper -> Lower.
// Mixed case words reset to Lower.
func (e *Editor) toggleCaseAtCursor() {
	if e.buffer.LineCount() == 0 {
		return
	}

	lineContent := e.buffer.GetLine(e.cursorY)
	runes := []rune(lineContent)
	if len(runes) == 0 {
		return
	}

	originalCursorX := e.cursorX

	idx := e.cursorX
	if idx >= len(runes) {
		idx = len(runes) - 1
	}

	if !isWordChar(runes[idx]) {
		if idx > 0 && isWordChar(runes[idx-1]) {
			idx--
		} else {
			return
		}
	}

	start := idx
	for start > 0 && isWordChar(runes[start-1]) {
		start--
	}

	end := idx
	for end < len(runes) && isWordChar(runes[end]) {
		end++
	}

	word := string(runes[start:end])
	if word == "" {
		return
	}

	currentCase := detectCase(word)
	var nextWord string

	switch currentCase {
	case caseLower:
		nextWord = toTitleCase(word)
	case caseTitle:
		nextWord = strings.ToUpper(word)
	case caseUpper:
		nextWord = strings.ToLower(word)
	default:
		nextWord = strings.ToLower(word)
	}

	e.beginUndoGroup()

	e.selectionActive = true
	e.selectionAnchorY = e.cursorY
	e.selectionAnchorX = start
	e.cursorY = e.cursorY
	e.cursorX = end

	e.deleteSelectedText()
	e.insertString(nextWord)

	e.selectionActive = false

	newLineLen := len([]rune(e.buffer.GetLine(e.cursorY)))
	if originalCursorX > newLineLen {
		e.cursorX = newLineLen
	} else {
		e.cursorX = originalCursorX
	}

	e.endUndoGroup()
	e.dirty = true
}

func detectCase(s string) int {
	if s == "" {
		return caseLower
	}
	if s == strings.ToLower(s) {
		return caseLower
	}
	if s == strings.ToUpper(s) {
		return caseUpper
	}

	runes := []rune(s)
	if unicode.IsUpper(runes[0]) && len(runes) > 1 {
		rest := string(runes[1:])
		if rest == strings.ToLower(rest) {
			return caseTitle
		}
	}

	return caseMixed
}

func toTitleCase(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		runes[i] = unicode.ToLower(runes[i])
	}
	return string(runes)
}
