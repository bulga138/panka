package editor

func (e *Editor) insertString(s string) {
	runes := []rune(s)
	if len(runes) == 0 {
		return
	}
	entries := make([]opEntry, 0, len(runes))
	if !e.undoGrouping {
		e.beginUndoGroup()
		defer e.endUndoGroup()
	}
	for _, r := range runes {
		insertLine := e.cursorY
		insertCol := e.cursorX
		if r == '\n' {
			e.buffer.Insert(e.cursorY, e.cursorX, '\n')
			e.cursorY++
			e.cursorX = 0
		} else {
			e.buffer.Insert(e.cursorY, e.cursorX, r)
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
	e.pushUndoInsertBlock(entries)
	e.dirty = true
}

func (e *Editor) replaceNext() {
	if e.findCurrentMatch == -1 || len(e.findMatches) == 0 {
		e.findNext()
		return
	}
	e.beginUndoGroup()
	match := e.findMatches[e.findCurrentMatch]
	matchLen := len([]rune(e.promptBuffer))
	e.selectionActive = true
	e.selectionAnchorY = match.y
	e.selectionAnchorX = match.x
	e.cursorY = match.y
	e.cursorX = match.x + matchLen
	e.deleteSelectedText()
	e.insertString(e.replaceBuffer)
	e.endUndoGroup()
	e.findInitial()
}

func (e *Editor) replaceAll() {
	e.findAllMatches(e.promptBuffer)
	if len(e.findMatches) == 0 {
		e.setStatusMessage("No matches found to replace.")
		return
	}
	numReplaced := len(e.findMatches)
	e.beginUndoGroup()
	for i := len(e.findMatches) - 1; i >= 0; i-- {
		match := e.findMatches[i]
		matchLen := len([]rune(e.promptBuffer))
		e.selectionActive = true
		e.selectionAnchorY = match.y
		e.selectionAnchorX = match.x
		e.cursorY = match.y
		e.cursorX = match.x + matchLen
		e.deleteSelectedText()
		e.insertString(e.replaceBuffer)
	}
	e.endUndoGroup()
	e.isReplacing = false
	e.isFinding = false
	e.selectionActive = false
	e.findMatches = nil
	e.findCurrentMatch = -1
	e.promptFocus = 0
	e.lastSearchQuery = e.promptBuffer
	e.setStatusMessage("Replaced %d instance(s).", numReplaced)
}
