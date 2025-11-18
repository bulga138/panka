package editor

// ---------- Undo/Redo push helpers ----------

func (e *Editor) pushUndoInsertBlock(entries []opEntry) {
	if len(entries) == 0 {
		return
	}
	action := undoAction{
		isInsert: true,
		ops:      entries,
	}
	if e.undoGrouping {
		action.groupID = e.currentGroupID
	}
	e.undoStack = append(e.undoStack, action)
}

func (e *Editor) pushUndoDeleteBlock(entries []opEntry, isBackspace bool) {
	if len(entries) == 0 {
		return
	}
	action := undoAction{
		isInsert:    false,
		isBackspace: isBackspace,
		ops:         entries,
	}
	if e.undoGrouping {
		action.groupID = e.currentGroupID
	}
	e.undoStack = append(e.undoStack, action)
}

// ---------- Undo/Redo execution ----------

func (e *Editor) performUndo(action undoAction) {
	// If action.isInsert == true, undo means: remove the inserted runes (reverse order)
	// If action.isInsert == false, undo means: re-insert the deleted runes (forward order)
	if action.isInsert {
		// delete inserted runes in reverse order using the recorded del positions
		for i := len(action.ops) - 1; i >= 0; i-- {
			op := action.ops[i]
			// Delete(op.delLine, op.delCol) removes the rune inserted earlier.
			e.buffer.Delete(op.delLine, op.delCol)
		}
		// Set cursor where the insertion started (convention: after undo, caret at insertion start)
		if len(action.ops) > 0 {
			e.cursorY = action.ops[0].insertLine
			e.cursorX = action.ops[0].insertCol
		}
	} else {
		// Re-insert deleted runes in forward order at their original insert positions
		for _, op := range action.ops {
			e.buffer.Insert(op.insertLine, op.insertCol, op.r)
		}
		// Position the cursor based on the type of deletion
		if len(action.ops) > 0 {
			if action.isBackspace {
				// For backspace, put cursor at the END of the re-inserted block
				last := action.ops[len(action.ops)-1]
				if last.r == '\n' {
					e.cursorY = last.insertLine + 1
					e.cursorX = 0
				} else {
					e.cursorY = last.insertLine
					e.cursorX = last.insertCol + 1
				}
			} else {
				// For Delete/Cut, put cursor at the START of the re-inserted block
				first := action.ops[0]
				e.cursorY = first.insertLine
				e.cursorX = first.insertCol
			}
		}
	}
	e.dirty = true
}

func (e *Editor) performRedo(action undoAction) {
	// Redo an insert => re-insert the recorded runes (forward order)
	// Redo a delete => delete the recorded runes again (reverse order)
	if action.isInsert {
		// Re-insert the runes in forward order at the recorded insert positions
		for _, op := range action.ops {
			e.buffer.Insert(op.insertLine, op.insertCol, op.r)
		}
		// Put cursor at end of inserted block (like Notepad/Word)
		if len(action.ops) > 0 {
			last := action.ops[len(action.ops)-1]
			if last.r == '\n' {
				e.cursorY = last.insertLine + 1
				e.cursorX = 0
			} else {
				e.cursorY = last.insertLine
				e.cursorX = last.insertCol + 1
			}
		}
	} else {
		// Delete the runes in reverse order using insert positions
		for i := len(action.ops) - 1; i >= 0; i-- {
			op := action.ops[i]
			// Delete at position (insertLine, insertCol+1) deletes the rune originally at insertCol
			e.buffer.Delete(op.insertLine, op.insertCol+1)
		}
		// Place cursor at the location of first deletion (insertLine, insertCol)
		if len(action.ops) > 0 {
			e.cursorY = action.ops[0].insertLine
			e.cursorX = action.ops[0].insertCol
		}
	}
	e.dirty = true
}

func (e *Editor) undo() {
	// Before undoing, flush typing/backspace groups to ensure everything is committed
	e.flushEditGroups()

	if len(e.undoStack) == 0 {
		e.setStatusMessage("Nothing to undo")
		return
	}

	action := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.redoStack = append(e.redoStack, action)
	e.performUndo(action)

	// For grouped operations (groupID > 0), process all with same groupID
	if action.groupID > 0 {
		groupID := action.groupID
		for len(e.undoStack) > 0 {
			next := e.undoStack[len(e.undoStack)-1]
			if next.groupID != groupID {
				break
			}
			e.undoStack = e.undoStack[:len(e.undoStack)-1]
			e.redoStack = append(e.redoStack, next)
			e.performUndo(next)
		}
	}

	e.setStatusMessage("Undid last action")
}

func (e *Editor) redo() {
	// Before redo, flush typing/backspace groups
	e.flushEditGroups()
	if len(e.redoStack) == 0 {
		e.setStatusMessage("Nothing to redo")
		return
	}
	action := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.undoStack = append(e.undoStack, action)
	e.performRedo(action)

	if action.groupID > 0 {
		groupID := action.groupID
		for len(e.redoStack) > 0 {
			next := e.redoStack[len(e.redoStack)-1]
			if next.groupID != groupID {
				break
			}
			e.redoStack = e.redoStack[:len(e.redoStack)-1]
			e.undoStack = append(e.undoStack, next)
			e.performRedo(next)
		}
	}

	e.setStatusMessage("Redid last action")
}
