package editor

import (
	"fmt"
	"os"
	"time"
)

// ---------- Save / misc ----------

func (e *Editor) scroll() {
	textWidth := e.getTextWidth()
	visCursorScreenY, _ := e.getVisualCursorPos()
	visCursorScreenY--
	if e.cursorY < e.viewportY {
		e.viewportY = e.cursorY
		e.viewportWrapOffset = 0
	} else if e.cursorY >= e.viewportY {
		if e.cursorY == e.viewportY && visCursorScreenY < 0 {
			if e.viewportWrapOffset > 0 {
				e.viewportWrapOffset--
			} else if e.viewportY > 0 {
				e.viewportY--
				numVisualRows := e.countVisualRows(e.viewportY, textWidth)
				e.viewportWrapOffset = numVisualRows - 1
			}
		} else if visCursorScreenY >= e.termHeight {
			diff := visCursorScreenY - e.termHeight + 1
			for i := 0; i < diff; i++ {
				e.advanceViewport(textWidth)
			}
		}
	}
}

func (e *Editor) advanceViewport(textWidth int) {
	numVisualRows := e.countVisualRows(e.viewportY, textWidth)
	if e.viewportWrapOffset+1 < numVisualRows {
		e.viewportWrapOffset++
	} else {
		if e.viewportY+1 < e.buffer.LineCount() {
			e.viewportY++
			e.viewportWrapOffset = 0
		}
	}
}

func (e *Editor) save() error {
	if e.filename == "" {
		e.isSaveAs = true
		e.promptBuffer = ""
		e.statusMessage = "Save As: "
		return nil
	}

	f, err := os.Create(e.filename)
	if err != nil {
		e.setStatusMessage("Save error: %v", err)
		return err
	}
	defer f.Close()

	n, err := e.buffer.WriteTo(f)
	if err != nil {
		e.setStatusMessage("Write error: %v", err)
		return err
	}

	e.dirty = false
	e.setStatusMessage("%d bytes written to %s", n, e.filename)
	return nil
}

func (e *Editor) setStatusMessage(f string, a ...interface{}) {
	e.statusMessage = fmt.Sprintf(f, a...)
	e.statusTime = time.Now()
}

func (e *Editor) getRuneAt(y, x int) rune {
	if y >= e.buffer.LineCount() {
		return 0
	}
	line := e.buffer.GetLine(y)
	if x >= len([]rune(line)) {
		return 0
	}
	return []rune(line)[x]
}
