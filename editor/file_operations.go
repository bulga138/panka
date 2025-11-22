package editor

import (
	"crypto/sha256"
	"encoding/hex"
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
	// Update the hash after a successful save
	e.initialHash = e.calculateBufferHash()

	e.setStatusMessage("%d bytes written to %s", n, e.filename)
	return nil
}

func (e *Editor) setStatusMessage(f string, a ...interface{}) {
	e.statusMessage = fmt.Sprintf(f, a...)
	e.statusTime = time.Now()
}

// handleBufferError handles errors from buffer operations and displays them to the user.
func (e *Editor) handleBufferError(err error) {
	if err != nil {
		e.setStatusMessage("Error: %v", err)
	}
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

// calculateBufferHash computes the SHA-256 hash of the current buffer content.
// This allows for exact content comparison to check if the file was actually modified.
func (e *Editor) calculateBufferHash() string {
	hasher := sha256.New()
	// Rope.WriteTo works with any io.Writer, so we pipe it directly to the hasher.
	// This is memory efficient as it doesn't create a full string copy.
	if _, err := e.buffer.WriteTo(hasher); err != nil {
		// In the unlikely event of a hashing error, return a value that won't match.
		return "error_calculating_hash"
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// isContentUnchanged checks if the current buffer content exactly matches
// the content when the file was loaded or last saved.
func (e *Editor) isContentUnchanged() bool {
	currentHash := e.calculateBufferHash()
	return currentHash == e.initialHash
}
