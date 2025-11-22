package editor

import (
	"strconv"
	"strings"
	"time"
)

// ---------- Undo grouping helpers ----------

func (e *Editor) beginUndoGroup() {
	e.undoGrouping = true
	e.currentGroupID = e.lastGroupID
	e.lastGroupID++
}

func (e *Editor) endUndoGroup() {
	e.undoGrouping = false
}

func (e *Editor) flushTypingGroup() {
	if !e.typingActive {
		return
	}
	if len(e.typingEntries) > 0 {
		e.pushUndoInsertBlock(e.typingEntries)
	}
	e.typingEntries = nil
	e.typingActive = false
}

func (e *Editor) flushBackspaceGroup() {
	if !e.backspaceActive {
		return
	}
	if len(e.backspaceEntries) > 0 {
		e.pushUndoDeleteBlock(e.backspaceEntries, true)
	}
	e.backspaceEntries = nil
	e.backspaceActive = false
}

func (e *Editor) flushTypingAndBackspaceIfNeeded() {
	e.flushTypingGroup()
	e.flushBackspaceGroup()
}

func (e *Editor) flushDeleteGroup() {
	if !e.deleteActive {
		return
	}
	if len(e.deleteEntries) > 0 {
		e.pushUndoDeleteBlock(e.deleteEntries, false)
	}
	e.deleteEntries = nil
	e.deleteActive = false
}

func (e *Editor) flushEditGroups() {
	e.flushTypingGroup()
	e.flushBackspaceGroup()
	e.flushDeleteGroup()
}

// ---------- Input processing ----------

func (e *Editor) processInput() error {
	r, _, err := e.inputReader.ReadRune()
	if err != nil {
		return err
	}
	if r == '\x1b' {
		e.flushEditGroups()
		return e.handleEscape()
	}
	if e.isConfirmingReplace {
		return e.handleReplaceConfirm(r)
	}
	if e.isQuitting {
		return e.handleQuitPrompt(r)
	}
	if e.isGotoLine {
		return e.handleGotoLineInput(r)
	}
	if e.isSaveAs {
		return e.handleSaveAsInput(r)
	}
	if e.isReplacing {
		return e.handleReplaceInput(r)
	}
	if e.isFinding {
		return e.handleFindInput(r)
	}
	return e.handleKey(r)
}

func (e *Editor) handleFindInput(r rune) error {
	switch r {
	case '\x1b': // Escape
		return nil

	case '\x08': // Ctrl+H
		e.isReplacing = true
		e.promptFocus = 1
		e.replaceBuffer = ""
		e.replaceCursorX = 0
		return nil

	case '\r', '\x0e': // Enter or Ctrl+N (Find Next)
		e.findNext()
		return nil

	case '\x10': // Ctrl+P (Find Previous)
		e.findPrevious()
		return nil

	case '\x7f': // Backspace
		e.backspacePromptRune()
		e.lastSearchQuery = e.promptBuffer
		if e.promptBuffer == "" {
			e.findMatches = nil
			e.findCurrentMatch = -1
			e.selectionActive = false
		} else {
			e.findInitial()
		}

	default:
		if r < 32 {
			return nil
		}
		e.insertPromptRune(r)
		e.lastSearchQuery = e.promptBuffer
		e.findInitial()
	}
	return nil
}

func (e *Editor) handleReplaceConfirm(r rune) error {
	e.isConfirmingReplace = false
	switch r {
	case 'y', 'Y':
		e.replaceAll()
	default:
		e.setStatusMessage("Replace All cancelled.")
	}
	return nil
}

func (e *Editor) findAllMatches(query string) {
	e.findMatches = nil
	if query == "" {
		return
	}
	queryLower := strings.ToLower(query)
	matches := make([]findResult, 0)
	for y := 0; y < e.buffer.LineCount(); y++ {
		line := e.buffer.GetLine(y)
		lineLower := strings.ToLower(line)
		lineRunes := []rune(lineLower)
		offset := 0
		for {
			matchIndex := strings.Index(string(lineRunes[offset:]), queryLower)
			if matchIndex == -1 {
				break
			}
			matchX := offset + matchIndex
			matches = append(matches, findResult{y, matchX})
			offset = matchX + 1
			if offset >= len(lineRunes) {
				break
			}
		}
	}
	e.findMatches = matches
}

func (e *Editor) findInitial() {
	e.findAllMatches(e.promptBuffer)
	if len(e.findMatches) == 0 {
		e.findCurrentMatch = -1
		e.selectionActive = false
		return
	}
	firstMatchAfterCursor := -1
	for i, match := range e.findMatches {
		if match.y > e.findOrigCursorY || (match.y == e.findOrigCursorY && match.x >= e.findOrigCursorX) {
			firstMatchAfterCursor = i
			break
		}
	}
	if firstMatchAfterCursor != -1 {
		e.findCurrentMatch = firstMatchAfterCursor
	} else {
		e.findCurrentMatch = 0
	}
	e.jumpToMatch(e.findCurrentMatch)
}

func (e *Editor) findNext() {
	if len(e.findMatches) == 0 {
		return
	}
	if e.findCurrentMatch == -1 {
		e.findCurrentMatch = 0
	} else {
		e.findCurrentMatch = (e.findCurrentMatch + 1) % len(e.findMatches)
	}
	e.jumpToMatch(e.findCurrentMatch)
}

func (e *Editor) findPrevious() {
	if len(e.findMatches) == 0 {
		return
	}
	e.findCurrentMatch--
	if e.findCurrentMatch < 0 {
		e.findCurrentMatch = len(e.findMatches) - 1
	}
	e.jumpToMatch(e.findCurrentMatch)
}

func (e *Editor) jumpToMatch(index int) {
	if index < 0 || index >= len(e.findMatches) {
		e.selectionActive = false
		return
	}
	match := e.findMatches[index]
	e.cursorY = match.y
	e.cursorX = match.x
	e.selectionActive = true
	e.selectionAnchorY = match.y
	e.selectionAnchorX = match.x
	e.cursorX += len([]rune(e.promptBuffer))
}

func (e *Editor) handleGotoLineInput(r rune) error {
	switch r {
	case '\x1b': // Escape
		return nil

	case '\r': // Enter
		e.isGotoLine = false
		lineNum, err := strconv.Atoi(e.promptBuffer)
		if err != nil || lineNum <= 0 || lineNum > e.buffer.LineCount() {
			if e.buffer.LineCount() == 0 && lineNum == 1 {
				e.cursorY = 0
				e.cursorX = 0
			} else {
				e.setStatusMessage("Invalid line number: %s", e.promptBuffer)
			}
		} else {
			e.cursorY = lineNum - 1
			e.cursorX = 0
			e.clampCursorX()
			e.setStatusMessage("Moved to line %d", lineNum)
		}
		e.promptBuffer = ""
		e.promptCursorX = 0
		return nil

	case '\x7f', '\b': // Backspace
		e.backspacePromptRune()

	default:
		if r >= '0' && r <= '9' {
			e.insertPromptRune(r)
		}
	}
	return nil
}

func (e *Editor) handleSaveAsInput(r rune) error {
	switch r {
	case '\x1b': // Escape
		return nil

	case '\r': // Enter
		e.isSaveAs = false
		filename := e.promptBuffer
		if filename == "" {
			e.setStatusMessage("Save As cancelled (empty filename).")
			e.promptBuffer = ""
			e.promptCursorX = 0
			return nil
		}
		e.filename = filename
		e.promptBuffer = ""
		e.promptCursorX = 0
		return e.save()

	case '\x7f', '\b': // Backspace
		e.backspacePromptRune()

	default:
		if r >= 32 || r == '\t' {
			e.insertPromptRune(r)
		}
	}
	return nil
}

func (e *Editor) handleQuitPrompt(r rune) error {
	switch r {
	case 'y', 'Y':
		if err := e.save(); err != nil {
			e.isQuitting = false
			return nil
		}
		e.quit = true
	case 'n', 'N':
		e.quit = true
	default:
		e.setStatusMessage("Quit cancelled.")
		e.isQuitting = false
	}
	return nil
}

func (e *Editor) handleDeleteKey() {
	e.flushEditGroups()
	if e.selectionActive {
		e.beginUndoGroup()
		e.deleteSelectedText()
		e.endUndoGroup()
		return
	}
	now := time.Now()
	// if last delete was far, flush existing delete group
	if !e.deleteActive || now.Sub(e.lastDeleteTime) > e.deleteThreshold {
		e.flushDeleteGroup()
		e.deleteActive = true
		e.deleteEntries = make([]opEntry, 0, 8)
	}

	lineRunes := []rune(e.buffer.GetLine(e.cursorY))
	lineLen := len(lineRunes)

	// Check if we are at the end of the entire file
	if e.cursorY == e.buffer.LineCount()-1 && e.cursorX == lineLen {
		// Nothing to delete
		return
	}

	if e.cursorX == lineLen {
		// At end of line, merge with next line (delete the \n)
		// Record the \n at (cursorY, cursorX)
		e.deleteEntries = append(e.deleteEntries, opEntry{
			insertLine: e.cursorY,
			insertCol:  e.cursorX,
			r:          '\n',
		})
		// Delete the newline by deleting "before" (cursorY+1, 0)
		e.buffer.Delete(e.cursorY+1, 0)
	} else {
		// In middle of line, delete rune at (cursorY, cursorX)
		char := lineRunes[e.cursorX]

		// Record the rune at (cursorY, cursorX)
		e.deleteEntries = append(e.deleteEntries, opEntry{
			insertLine: e.cursorY,
			insertCol:  e.cursorX,
			r:          char,
		})
		// Delete the rune by deleting "before" (cursorY, cursorX + 1)
		e.buffer.Delete(e.cursorY, e.cursorX+1)
	}

	e.dirty = true
	e.lastDeleteTime = now
}

func (e *Editor) handleDeleteWordLeft() {
	e.flushEditGroups()
	if e.selectionActive {
		e.beginUndoGroup()
		e.deleteSelectedText()
		e.endUndoGroup()
		return
	}
	endY, endX := e.cursorY, e.cursorX
	e.moveWordLeft(false)
	startY, startX := e.cursorY, e.cursorX
	if startY == endY && startX == endX {
		return
	}
	e.cursorY = endY
	e.cursorX = endX
	e.selectionAnchorY = startY
	e.selectionAnchorX = startX
	e.selectionActive = true
	e.beginUndoGroup()
	e.deleteSelectedText()
	e.endUndoGroup()
}

func (e *Editor) handleDeleteWordRight() {
	e.flushEditGroups()
	if e.selectionActive {
		e.beginUndoGroup()
		e.deleteSelectedText()
		e.endUndoGroup()
		return
	}
	startY, startX := e.cursorY, e.cursorX
	e.moveWordRight(false)
	endY, endX := e.cursorY, e.cursorX
	if startY == endY && startX == endX {
		return
	}
	e.cursorY = startY
	e.cursorX = startX
	e.selectionAnchorY = startY
	e.selectionAnchorX = startX
	e.selectionActive = true
	e.cursorY = endY
	e.cursorX = endX
	e.beginUndoGroup()
	e.deleteSelectedText()
	e.endUndoGroup()
}

// Helper to get range of lines for multi-cursor
func (e *Editor) getMultiCursorRange() (int, int) {
	if e.extraCursorHeight == 0 {
		return e.cursorY, e.cursorY
	}
	if e.extraCursorHeight > 0 {
		return e.cursorY, e.cursorY + e.extraCursorHeight
	}
	return e.cursorY + e.extraCursorHeight, e.cursorY
}

func (e *Editor) handleKey(r rune) error {
	// Common: if key is not selection-related, we stop selection mode
	switch r {
	case '\x1b': // Escape key (arrows, handled by handleEscape)
	case '\x03': // Ctrl+C (Copy)
	case '\x18': // Ctrl+X (Cut)
	case '\x01': // Ctrl+A (Select All)
	case '\x7f': // Backspace
		// Do nothing
	default:
		e.selectionActive = false
	}

	// For most actions (except undo/redo/escape/copy/cut/select), new edits clear redo stack
	switch r {
	case '\x15', '\x19', '\x1b', '\x03', '\x18', '\x01': // Ctrl+U, Ctrl+Y, ESC, Ctrl+C, Ctrl+X, Ctrl+A
	default:
		e.redoStack = nil
	}

	switch r {
	case '\x01': // Ctrl+A - Select All
		e.flushEditGroups()
		e.extraCursorHeight = 0
		return e.selectAll()
	case '\x11': // Ctrl+Q
		e.flushEditGroups()
		if !e.dirty {
			e.quit = true
			return nil
		}
		if e.isContentUnchanged() {
			e.quit = true
			return nil
		}
		e.isQuitting = true
		e.setStatusMessage("Save modified buffer (Y/N)?")
	case '\x13': // Ctrl+S
		e.flushEditGroups()
		return e.save()
	case '\x05': // Ctrl+E (for "Save As")
		e.flushEditGroups()
		e.isSaveAs = true
		e.promptBuffer = e.filename
		e.promptCursorX = len([]rune(e.filename))
		e.setStatusMessage("Save As: ")
		return nil
	case '\x15': // Ctrl+U (Undo)
		e.flushEditGroups()
		e.undo()
	case '\x19': // Ctrl+Y (Redo)
		e.flushEditGroups()
		e.redo()
	case '\x03': // Ctrl+C - Copy
		e.flushEditGroups()
		return e.copyToClipboard()
	case '\x18': // Ctrl+X - Cut
		e.flushEditGroups()
		return e.cutToClipboard()
	case '\x16': // Ctrl+V - Paste
		e.flushEditGroups()
		return e.pasteFromClipboard()
	case '\x0c': // Ctrl+L
		e.flushEditGroups()
		e.toggleLineNumbers()
	case '\x14': // Ctrl+T
		e.flushEditGroups()
		e.isGotoLine = true
		e.promptBuffer = ""
		e.statusMessage = "Go to Line: "
	case '\x06': // Ctrl+F
		e.flushEditGroups()
		e.findOrigCursorX = e.cursorX
		e.findOrigCursorY = e.cursorY
		if e.lastSearchQuery != "" {
			e.promptBuffer = e.lastSearchQuery
			e.findInitial()
		} else {
			e.promptBuffer = ""
			e.findMatches = nil
		}
		e.promptCursorX = len([]rune(e.promptBuffer))
		e.isFinding = true
		e.findCurrentMatch = -1
		e.statusMessage = "Find (ESC:Cancel | Enter/Ctrl+N:Next | Ctrl+P:Prev): "
	case '\x08': // Ctrl+H
		e.flushEditGroups()
		e.findOrigCursorX = e.cursorX
		e.findOrigCursorY = e.cursorY
		e.isReplacing = true
		e.isFinding = true
		e.promptFocus = 0
		e.promptBuffer = e.lastSearchQuery
		e.promptCursorX = len([]rune(e.promptBuffer))
		e.replaceBuffer = ""
		e.replaceCursorX = 0
		if e.promptBuffer != "" {
			e.findInitial()
		}
		return nil
	case '\x0f': // Ctrl+O (Toggle Non-Printable)
		e.flushEditGroups()
		e.showNonPrintable = !e.showNonPrintable
		status := "Show non-printable: OFF"
		if e.showNonPrintable {
			status = "Show non-printable: ON"
		}
		e.setStatusMessage(status)
	case '\x04': // Ctrl+D
		e.flushEditGroups()
		e.extraCursorHeight = 0
		e.duplicateLine()

	case '\x0b': // Ctrl+K
		e.flushEditGroups()
		e.extraCursorHeight = 0
		e.toggleCaseAtCursor()

	case '\x17': // Ctrl+W
		e.handleDeleteWordLeft()
	case '\r': // Enter
		e.flushBackspaceGroup()
		e.flushDeleteGroup()
		e.flushTypingGroup()
		e.extraCursorHeight = 0

		now := time.Now()

		currentLine := e.buffer.GetLine(e.cursorY)
		indent := ""
		for _, char := range currentLine {
			if char == ' ' || char == '\t' {
				indent += string(char)
			} else {
				break
			}
		}

		textToInsert := "\n" + indent

		entries := make([]opEntry, 0, len(textToInsert))

		for _, char := range textToInsert {
			insertLine := e.cursorY
			insertCol := e.cursorX

			if err := e.buffer.Insert(e.cursorY, e.cursorX, char); err != nil {
				e.setStatusMessage("Insert error: %v", err)
				return nil
			}

			if char == '\n' {
				e.cursorY++
				e.cursorX = 0
			} else {
				e.cursorX++
			}

			entries = append(entries, opEntry{
				insertLine: insertLine,
				insertCol:  insertCol,
				delLine:    e.cursorY,
				delCol:     e.cursorX,
				r:          char,
			})
		}

		e.pushUndoInsertBlock(entries)
		e.lastTypeTime = now
		e.dirty = true

	case '\x7f': // Backspace
		if e.selectionActive {
			e.beginUndoGroup()
			e.deleteSelectedText()
			e.endUndoGroup()
			return nil
		}
		e.flushTypingGroup()
		e.flushDeleteGroup()

		// --- Multi-Cursor Backspace ---
		e.beginUndoGroup()
		defer e.endUndoGroup()

		startLine, endLine := e.getMultiCursorRange()

		// Process from bottom to top
		for i := endLine; i >= startLine; i-- {
			if i >= e.buffer.LineCount() {
				continue
			}

			lineRunes := []rune(e.buffer.GetLine(i))
			lineLen := len(lineRunes)

			targetX := e.cursorX
			if targetX > lineLen {
				targetX = lineLen
			}

			if targetX == 0 && i == 0 {
				continue
			} else {
				if targetX > 0 {
					delIndex := targetX - 1
					char := e.getRuneAt(i, delIndex)
					e.pushUndoDeleteIfExternalGrouping(i, delIndex, char)
					e.buffer.Delete(i, targetX)
				} else {
					// Handle join lines only if single cursor, or explicit decision.
					// For column block, joining lines shifts everything below up, breaking the block structure.
					// Let's DISABLE line joining in multi-cursor mode unless height is 0.
					if e.extraCursorHeight == 0 {
						prevLineIdx := i - 1
						prevLineContent := e.buffer.GetLine(prevLineIdx)
						expectedCursorX := len([]rune(prevLineContent))
						e.pushUndoDeleteIfExternalGrouping(prevLineIdx, expectedCursorX, '\n')
						e.cursorY = prevLineIdx
						e.buffer.Delete(i, 0) // Delete newline of prev line? No, buffer delete logic is (y+1, 0)
						// Actually logic is Delete(cursorY, cursorX).
						// If cursorX==0, we delete the previous newline.
						// Buffer.Delete(i, 0) -> deletes char BEFORE (i,0).
						// Which is the newline at end of i-1.

						// We only update main cursor if it's the primary line
						if i == e.cursorY {
							mergedLineContent := e.buffer.GetLine(e.cursorY)
							e.cursorX = len([]rune(mergedLineContent))
						}
					}
				}
				e.dirty = true
			}
		}
		// For normal typing backspace, we update cursorX *after* the loop if we didn't change lines
		if e.cursorX > 0 {
			e.cursorX--
		}

	default: // Typing
		e.flushBackspaceGroup()
		e.flushDeleteGroup()
		e.flushTypingGroup()

		// --- Multi-Cursor Typing ---
		e.beginUndoGroup()
		defer e.endUndoGroup()

		startLine, endLine := e.getMultiCursorRange()

		for i := startLine; i <= endLine; i++ {
			if i >= e.buffer.LineCount() {
				continue
			}

			lineRunes := []rune(e.buffer.GetLine(i))
			targetX := e.cursorX
			if targetX > len(lineRunes) {
				targetX = len(lineRunes)
			}

			if err := e.buffer.Insert(i, targetX, r); err != nil {
				continue
			}

			// Push undo op
			// Note: Undo logic uses 'delLine/Col' to know where to delete.
			// insertLine/Col is mostly for redo.
			e.pushUndoInsertBlock([]opEntry{{
				insertLine: i, insertCol: targetX,
				delLine: i, delCol: targetX,
				r: r,
			}})
		}

		e.cursorX++
		e.lastTypeTime = time.Now()
		e.dirty = true
	}
	return nil
}

func (e *Editor) handleReplaceInput(r rune) error {
	switch r {
	case '\x1b': // Escape
		return nil

	case '\t': // Tab to toggle focus
		e.promptFocus = (e.promptFocus + 1) % 2
		return nil

	case '\r', '\x0e': // Enter or Ctrl+N (Find Next)
		e.findNext()
		return nil

	case '\x10': // Ctrl+P (Find Previous)
		e.findPrevious()
		return nil

	case '\x12': // Ctrl+R (Replace Next)
		e.replaceNext()
		return nil

	case '\x01': // Ctrl+A (Replace All)
		if len(e.findMatches) > 0 {
			e.isConfirmingReplace = true
			e.setStatusMessage("Replace all %d instance(s)? (Y/N)", len(e.findMatches))
		} else {
			e.setStatusMessage("No matches found.")
		}
		return nil

	case '\x11': // Ctrl+Q (Cancel Replace)
		e.isReplacing = false
		e.isFinding = false
		e.findMatches = nil
		e.selectionActive = false
		e.setStatusMessage("Replace cancelled.")
		return nil

	case '\x7f', '\b': // Backspace
		e.backspacePromptRune()
		if e.promptFocus == 0 {
			e.lastSearchQuery = e.promptBuffer
			e.findInitial()
		}

	default:
		if r < 32 {
			return nil
		}
		e.insertPromptRune(r)
		if e.promptFocus == 0 {
			e.lastSearchQuery = e.promptBuffer
			e.findInitial()
		}
	}
	return nil
}

// ---------- Prompt helpers ----------

func (e *Editor) movePromptCursor(dx int) {
	if e.promptFocus == 0 { // Find buffer
		e.promptCursorX += dx
		promptLen := len([]rune(e.promptBuffer))
		if e.promptCursorX < 0 {
			e.promptCursorX = 0
		}
		if e.promptCursorX > promptLen {
			e.promptCursorX = promptLen
		}
	} else { // Replace buffer
		e.replaceCursorX += dx
		promptLen := len([]rune(e.replaceBuffer))
		if e.replaceCursorX < 0 {
			e.replaceCursorX = 0
		}
		if e.replaceCursorX > promptLen {
			e.replaceCursorX = promptLen
		}
	}
}

func (e *Editor) insertPromptRune(r rune) {
	if e.promptFocus == 0 { // Find buffer
		runes := []rune(e.promptBuffer)
		e.promptBuffer = string(runes[:e.promptCursorX]) + string(r) + string(runes[e.promptCursorX:])
		e.promptCursorX++
	} else { // Replace buffer
		runes := []rune(e.replaceBuffer)
		e.replaceBuffer = string(runes[:e.replaceCursorX]) + string(r) + string(runes[e.replaceCursorX:])
		e.replaceCursorX++
	}
}

func (e *Editor) backspacePromptRune() {
	if e.promptFocus == 0 { // Find buffer
		if e.promptCursorX > 0 {
			runes := []rune(e.promptBuffer)
			e.promptBuffer = string(runes[:e.promptCursorX-1]) + string(runes[e.promptCursorX:])
			e.promptCursorX--
		}
	} else { // Replace buffer
		if e.replaceCursorX > 0 {
			runes := []rune(e.replaceBuffer)
			e.replaceBuffer = string(runes[:e.replaceCursorX-1]) + string(runes[e.replaceCursorX:])
			e.replaceCursorX--
		}
	}
}

func (e *Editor) deletePromptRune() {
	if e.promptFocus == 0 { // Find buffer
		runes := []rune(e.promptBuffer)
		promptLen := len(runes)
		if e.promptCursorX < promptLen {
			e.promptBuffer = string(runes[:e.promptCursorX]) + string(runes[e.promptCursorX+1:])
		}
	} else { // Replace buffer
		runes := []rune(e.replaceBuffer)
		promptLen := len(runes)
		if e.replaceCursorX < promptLen {
			e.replaceBuffer = string(runes[:e.replaceCursorX]) + string(runes[e.replaceCursorX+1:])
		}
	}
}
