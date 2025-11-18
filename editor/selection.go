package editor

import (
	"strings"
)

// ---------- Selection / Delete ----------
func (e *Editor) deleteSelectedText() {
	if !e.selectionActive {
		return
	}
	startY, startX, endY, endX := e.getSelectionCoords()
	e.flushTypingAndBackspaceIfNeeded()
	entries := make([]opEntry, 0)
	if startY == endY {
		line := e.buffer.GetLine(startY)
		runes := []rune(line)
		actualEndX := min(endX, len(runes))
		for i := startX; i < actualEndX; i++ {
			entries = append(entries, opEntry{
				insertLine: startY,
				insertCol:  i,
				r:          runes[i],
			})
		}
		for i := actualEndX - 1; i >= startX; i-- {
			e.buffer.Delete(startY, i+1)
		}
	} else {
		firstLine := e.buffer.GetLine(startY)
		firstRunes := []rune(firstLine)
		for i := startX; i < len(firstRunes); i++ {
			entries = append(entries, opEntry{insertLine: startY, insertCol: i, r: firstRunes[i]})
		}
		entries = append(entries, opEntry{insertLine: startY, insertCol: len(firstRunes), r: '\n'})
		lineOffset := 1
		for y := startY + 1; y < endY; y++ {
			lineContent := e.buffer.GetLine(y)
			runes := []rune(lineContent)
			actualInsertLine := startY + lineOffset
			for i := 0; i < len(runes); i++ {
				entries = append(entries, opEntry{insertLine: actualInsertLine, insertCol: i, r: runes[i]})
			}
			entries = append(entries, opEntry{insertLine: actualInsertLine, insertCol: len(runes), r: '\n'})
			lineOffset++
		}
		lastLine := e.buffer.GetLine(endY)
		lastRunes := []rune(lastLine)
		actualEndX := min(endX, len(lastRunes))
		actualInsertLine := startY + lineOffset
		for i := 0; i < actualEndX; i++ {
			entries = append(entries, opEntry{insertLine: actualInsertLine, insertCol: i, r: lastRunes[i]})
		}
		for i := actualEndX - 1; i >= 0; i-- {
			e.buffer.Delete(endY, i+1)
		}
		for y := endY - 1; y > startY; y-- {
			e.buffer.Delete(y+1, 0)
			lineRunes := []rune(e.buffer.GetLine(y))
			for i := len(lineRunes) - 1; i >= 0; i-- {
				e.buffer.Delete(y, i+1)
			}
		}
		e.buffer.Delete(startY+1, 0)
		for i := len(firstRunes) - 1; i >= startX; i-- {
			e.buffer.Delete(startY, i+1)
		}
	}

	e.pushUndoDeleteBlock(entries, false)

	e.cursorY = startY
	e.cursorX = startX
	e.selectionActive = false
}

func (e *Editor) pushUndoDeleteIfExternalGrouping(line, col int, r rune) {
	action := undoAction{
		isInsert: false,
		ops: []opEntry{
			{insertLine: line, insertCol: col, r: r},
		},
	}
	if e.undoGrouping {
		action.groupID = e.currentGroupID
	}
	e.undoStack = append(e.undoStack, action)
}

func (e *Editor) getSelectedText() string {
	if !e.selectionActive {
		return ""
	}
	startY, startX, endY, endX := e.getSelectionCoords()
	if startY == endY {
		line := e.buffer.GetLine(startY)
		runes := []rune(line)
		if endX > len(runes) {
			endX = len(runes)
		}
		if startX > len(runes) {
			startX = len(runes)
		}
		return string(runes[startX:endX])
	}
	var result strings.Builder
	firstLine := e.buffer.GetLine(startY)
	firstRunes := []rune(firstLine)
	if startX < len(firstRunes) {
		result.WriteString(string(firstRunes[startX:]))
	}
	result.WriteString("\n")
	for y := startY + 1; y < endY; y++ {
		result.WriteString(e.buffer.GetLine(y))
		result.WriteString("\n")
	}
	lastLine := e.buffer.GetLine(endY)
	lastRunes := []rune(lastLine)
	if endX > len(lastRunes) {
		endX = len(lastRunes)
	}
	result.WriteString(string(lastRunes[:endX]))
	return result.String()
}

func (e *Editor) deleteCurrentLine() {
	e.flushTypingAndBackspaceIfNeeded()
	entries := make([]opEntry, 0)
	lineIdx := e.cursorY
	lineContent := e.buffer.GetLine(lineIdx)
	lineRunes := []rune(lineContent)
	for i := range lineRunes {
		entries = append(entries, opEntry{insertLine: lineIdx, insertCol: i, r: lineRunes[i]})
	}
	if e.cursorY < e.buffer.LineCount()-1 {
		entries = append(entries, opEntry{insertLine: lineIdx, insertCol: len(lineRunes), r: '\n'})
	}
	for i := len(lineRunes) - 1; i >= 0; i-- {
		e.buffer.Delete(lineIdx, i+1)
	}
	if e.cursorY < e.buffer.LineCount()-1 {
		e.buffer.Delete(e.cursorY+1, 0)
	}
	e.pushUndoDeleteBlock(entries, false)
	e.cursorX = 0
	if e.cursorY >= e.buffer.LineCount() && e.cursorY > 0 {
		e.cursorY--
		e.cursorX = len([]rune(e.buffer.GetLine(e.cursorY)))
	}
}
