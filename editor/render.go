package editor

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bulga138/panka/runewidth"
	"github.com/bulga138/panka/version"
)

func (e *Editor) clampViewport() {
	if e.viewportCol < 0 {
		e.viewportCol = 0
	}
	if e.viewportY < 0 {
		e.viewportY = 0
	}
	if e.viewportWrapOffset < 0 {
		e.viewportWrapOffset = 0
	}
	if e.viewportY >= e.buffer.LineCount() {
		if e.buffer.LineCount() > 0 {
			e.viewportY = e.buffer.LineCount() - 1
		} else {
			e.viewportY = 0
		}
		e.viewportWrapOffset = 0
	}
}

func (e *Editor) render() {
	e.clampViewport()
	var ab bytes.Buffer
	ab.WriteString(ansiHideCursor)
	ab.WriteString(ansiMoveToHome)
	e.scroll()
	e.drawRows(&ab)
	e.drawStatusBar(&ab)
	e.drawCommandBar(&ab)
	e.drawMessageBar(&ab)

	if e.isGotoLine || e.isSaveAs || e.isFinding {
		var visualCursorOffset int
		var promptMsgLen int
		var cursorCol int
		var cursorRow int

		if e.isReplacing {
			if e.promptFocus == 0 { // Find line
				promptMsgLen = runewidth.StringWidth("Find: ")
				promptRunes := []rune(e.promptBuffer)
				if e.promptCursorX > len(promptRunes) {
					e.promptCursorX = len(promptRunes)
				}
				visualCursorOffset = runewidth.StringWidth(string(promptRunes[:e.promptCursorX]))
				cursorRow = e.termHeight + 2
				cursorCol = promptMsgLen + visualCursorOffset + 1
			} else { // Replace line
				promptMsgLen = runewidth.StringWidth("Replace: ")
				promptRunes := []rune(e.replaceBuffer)

				if e.isConfirmingReplace {
					separator := " | "
					prompt := fmt.Sprintf("Confirm Replace All (%d)? (Y/N)", len(e.findMatches))
					prefixLen := runewidth.StringWidth("Replace: ") + runewidth.StringWidth(e.replaceBuffer) + runewidth.StringWidth(separator)
					visualCursorOffset = runewidth.StringWidth(prompt)
					cursorCol = prefixLen + visualCursorOffset + 1
				} else {
					if e.replaceCursorX > len(promptRunes) {
						e.replaceCursorX = len(promptRunes)
					}
					visualCursorOffset = runewidth.StringWidth(string(promptRunes[:e.replaceCursorX]))
					cursorCol = promptMsgLen + visualCursorOffset + 1
				}
				cursorRow = e.termHeight + 3
			}
		} else {
			promptMsgLen = runewidth.StringWidth(e.statusMessage)
			promptRunes := []rune(e.promptBuffer)

			if e.isConfirmingReplace {
				prefixLen := runewidth.StringWidth("Replace: ") + runewidth.StringWidth(e.replaceBuffer) + 3
				msgLen := runewidth.StringWidth(e.statusMessage)
				cursorCol = prefixLen + msgLen
			} else {
				if e.promptCursorX > len(promptRunes) {
					e.promptCursorX = len(promptRunes)
				}
				visualCursorOffset = runewidth.StringWidth(string(promptRunes[:e.promptCursorX]))
				cursorCol = promptMsgLen + visualCursorOffset + 1
			}
			cursorRow = e.termHeight + 3
		}
		ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", cursorRow, cursorCol))
		ab.WriteString(ansiShowCursor)
	} else {
		visRow, visCol := e.calculateCursorScreenPosition()
		if visRow < 1 {
			visRow = 1
		}
		if visRow > e.termHeight {
			visRow = e.termHeight
		}
		if visCol < 1 {
			visCol = 1
		}
		if visCol > e.termWidth {
			visCol = e.termWidth
		}

		ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", visRow, visCol))
		ab.WriteString(ansiShowCursor)
	}

	os.Stdout.Write(ab.Bytes())
}

func (e *Editor) drawRows(ab *bytes.Buffer) {
	e.clampViewport()
	textWidth := e.getTextWidth()
	fileLine := e.viewportY
	lineWrapOffset := e.viewportWrapOffset
	selStartL, selStartC, selEndL, selEndC := e.getSelectionCoordsSafe()

	mcStart, mcEnd := e.getMultiCursorRange()

	for screenRow := 0; screenRow < e.termHeight; screenRow++ {
		if fileLine >= e.buffer.LineCount() {
			ab.WriteString(e.drawTildeRow())
		} else {
			if e.showLineNumbers {
				lineNumStr := ""
				if lineWrapOffset == 0 {
					lineNumStr = fmt.Sprintf("%d", fileLine+1)
				}
				fmt.Fprintf(ab, "%s %*s %s", ansiInvert, e.lineNumWidth-2, lineNumStr, ansiReset)
			}
			lineContent := e.buffer.GetLine(fileLine)
			runes := []rune(lineContent)
			lineVisWidth := 0
			visCharPositions := make([]int, 0, len(runes)+1)
			visCharPositions = append(visCharPositions, 0)
			for _, r := range runes {
				var rWidth int
				if r == '\t' {
					rWidth = e.config.TabSize - (lineVisWidth % e.config.TabSize)
				} else {
					rWidth = runewidth.RuneWidth(r)
				}
				lineVisWidth += rWidth
				visCharPositions = append(visCharPositions, lineVisWidth)
			}

			totalVisualRows := 1
			if lineVisWidth > 0 {
				totalVisualRows = (lineVisWidth + textWidth - 1) / textWidth
				if totalVisualRows == 0 {
					totalVisualRows = 1
				}
			}

			if lineWrapOffset < totalVisualRows {
				var lineBuffer bytes.Buffer
				rowStartVisPos := lineWrapOffset * textWidth
				rowEndVisPos := rowStartVisPos + textWidth
				startChar := 0
				endChar := len(runes)
				for i := 0; i < len(visCharPositions)-1; i++ {
					if visCharPositions[i] <= rowStartVisPos && visCharPositions[i+1] > rowStartVisPos {
						startChar = i
						break
					}
				}
				for i := startChar; i < len(visCharPositions); i++ {
					if visCharPositions[i] >= rowEndVisPos {
						endChar = i
						break
					}
				}

				hasMultiCursor := false
				if fileLine != e.cursorY && fileLine >= mcStart && fileLine <= mcEnd {
					hasMultiCursor = true
				}

				renderedWidth := 0
				for i := startChar; i < endChar && renderedWidth < textWidth; i++ {
					if i >= len(runes) {
						break
					}
					r := runes[i]
					charStartVisPos := visCharPositions[i]
					visibleStart := max(charStartVisPos, rowStartVisPos)

					isUnderCursor := hasMultiCursor && i == e.cursorX
					isSelected := e.isRuneSelected(fileLine, i, selStartL, selStartC, selEndL, selEndC)

					if isUnderCursor {
						lineBuffer.WriteString(ansiInvert)
					} else if isSelected {
						lineBuffer.WriteString(ansiInvert)
					}

					if r == '\t' {
						spacesToRender := min(visibleWidth(charStartVisPos, visCharPositions[i+1], rowStartVisPos, rowEndVisPos), textWidth-renderedWidth)

						if e.showNonPrintable && spacesToRender > 0 {
							// Draw arrow for first char of tab
							if visibleStart == charStartVisPos {
								lineBuffer.WriteString(ansiDim)
								lineBuffer.WriteRune('→') // U+2192
								lineBuffer.WriteString(ansiReset)
								if isUnderCursor || isSelected {
									lineBuffer.WriteString(ansiInvert)
								}

								for j := 1; j < spacesToRender; j++ {
									lineBuffer.WriteRune(' ')
								}
							} else {
								for j := 0; j < spacesToRender; j++ {
									lineBuffer.WriteRune(' ')
								}
							}
						} else {
							for j := 0; j < spacesToRender; j++ {
								lineBuffer.WriteRune(' ')
							}
						}
						renderedWidth += spacesToRender
					} else if r == ' ' && e.showNonPrintable {
						lineBuffer.WriteString(ansiDim)
						lineBuffer.WriteRune('·') // U+00B7 Middle Dot
						lineBuffer.WriteString(ansiReset)
						if isUnderCursor || isSelected {
							lineBuffer.WriteString(ansiInvert) // Re-apply if needed
						}
						renderedWidth += 1
					} else {
						lineBuffer.WriteRune(r)
						renderedWidth += 1
					}

					if isUnderCursor || isSelected {
						lineBuffer.WriteString(ansiReset)
					}
				}

				isEOLUnderCursor := hasMultiCursor && e.cursorX >= len(runes)
				isEOLSelected := e.isRuneSelected(fileLine, len(runes), selStartL, selStartC, selEndL, selEndC)

				if endChar == len(runes) && renderedWidth < textWidth {
					if isEOLUnderCursor {
						lineBuffer.WriteString(ansiInvert)
						if e.showNonPrintable {
							lineBuffer.WriteString(ansiDim + "¶" + ansiReset + ansiInvert)
						} else {
							lineBuffer.WriteRune(' ')
						}
						lineBuffer.WriteString(ansiReset)
					} else if isEOLSelected {
						lineBuffer.WriteString(ansiInvert)
						if e.showNonPrintable {
							lineBuffer.WriteString(ansiDim + "¶" + ansiReset + ansiInvert)
						} else {
							lineBuffer.WriteRune(' ')
						}
						lineBuffer.WriteString(ansiReset)
					} else if e.showNonPrintable {
						// Draw newline char if visible mode is on (and not selected)
						lineBuffer.WriteString(ansiDim)
						lineBuffer.WriteRune('¶') // U+00B6 Pilcrow
						lineBuffer.WriteString(ansiReset)
					}
				}

				ab.Write(lineBuffer.Bytes())
			}
			ab.WriteString(ansiClearLine)
			ab.WriteString("\r\n")
		}
		if fileLine < e.buffer.LineCount() {
			numVisualRows := e.countVisualRows(fileLine, textWidth)
			if lineWrapOffset+1 < numVisualRows {
				lineWrapOffset++
			} else {
				fileLine++
				lineWrapOffset = 0
			}
		} else {
			fileLine++
			lineWrapOffset = 0
		}
	}
}

// Helper to calculate visible width of a char/tab split across rows
func visibleWidth(start, end, rowStart, rowEnd int) int {
	vStart := max(start, rowStart)
	vEnd := min(end, rowEnd)
	if vEnd > vStart {
		return vEnd - vStart
	}
	return 0
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (e *Editor) getSelectionCoords() (startY, startX, endY, endX int) {
	if e.selectionAnchorY < e.cursorY || (e.selectionAnchorY == e.cursorY && e.selectionAnchorX < e.cursorX) {
		return e.selectionAnchorY, e.selectionAnchorX, e.cursorY, e.cursorX
	}
	return e.cursorY, e.cursorX, e.selectionAnchorY, e.selectionAnchorX
}

func (e *Editor) drawStatusBar(ab *bytes.Buffer) {
	ab.WriteString(ansiInvert)
	name := e.filename
	if name == "" {
		name = "[No Name]"
	}
	left := fmt.Sprintf(" %.20s", name)
	if e.dirty {
		left += " (modified)"
	}
	versionInfo := " v" + version.GetVersion()
	right := fmt.Sprintf("Ln %d, Col %d %s", e.cursorY+1, e.cursorX+1, versionInfo)
	totalLen := len(left) + len(right)
	padding := max(e.termWidth-totalLen, 0)
	ab.WriteString(left)
	ab.WriteString(strings.Repeat(" ", padding))
	ab.WriteString(right)
	ab.WriteString(ansiReset)
	ab.WriteString("\r\n")
}

func (e *Editor) drawCommandBar(ab *bytes.Buffer) {
	ab.WriteString(ansiClearLine)
	if e.isReplacing {
		findLabel := "Find: "
		if e.promptFocus == 0 {
			findLabel = ansiInvert + findLabel + ansiReset
		}
		hints := " [TAB Switch | ^R Repl | ^A All | ESC Cancel]"
		countStr := ""
		if e.promptBuffer != "" {
			if len(e.findMatches) == 0 {
				countStr = " (0)"
			} else if e.findCurrentMatch == -1 {
				countStr = fmt.Sprintf(" (%d)", len(e.findMatches))
			} else {
				countStr = fmt.Sprintf(" (%d/%d)", e.findCurrentMatch+1, len(e.findMatches))
			}
		}
		prefixLen := runewidth.StringWidth("Find: ") + runewidth.StringWidth(e.promptBuffer) + runewidth.StringWidth(countStr)
		hintsLen := runewidth.StringWidth(hints)
		padding := max(1, e.termWidth-prefixLen-hintsLen)
		ab.WriteString(findLabel)
		ab.WriteString(e.promptBuffer)
		ab.WriteString(countStr)
		ab.WriteString(strings.Repeat(" ", padding))
		ab.WriteString(hints) // Draw hints aligned to right
	} else {
		cmdStr := " ^S Save | ^Q Quit | ^U Undo | ^Y Redo | ^X Cut | ^C Copy | ^V Paste | ^T Go to | ^F Find | ^H Replace | ^K Toggle case | ^O Non-printable"
		if len(cmdStr) > e.termWidth {
			cmdStr = cmdStr[:e.termWidth]
		}
		ab.WriteString(cmdStr)
	}
	ab.WriteString("\r\n")
}

func (e *Editor) drawMessageBar(ab *bytes.Buffer) {
	ab.WriteString(ansiClearLine)

	if e.isReplacing {
		replaceLabel := "Replace: "
		if e.promptFocus == 1 && !e.isConfirmingReplace {
			replaceLabel = ansiInvert + replaceLabel + ansiReset
		}
		ab.WriteString(replaceLabel)
		ab.WriteString(e.replaceBuffer)
		if e.isConfirmingReplace {
			separator := " | "
			prompt := fmt.Sprintf("Confirm Replace All (%d)? (Y/N)", len(e.findMatches))
			ab.WriteString(separator + ansiInvert + prompt + ansiReset)
		}
	} else if e.isFinding {
		prompt := e.statusMessage + e.promptBuffer
		countStr := ""
		if e.promptBuffer != "" {
			if len(e.findMatches) == 0 {
				countStr = "(0 of 0)"
			} else if e.findCurrentMatch == -1 {
				countStr = fmt.Sprintf("(%d matches)", len(e.findMatches))
			} else {
				countStr = fmt.Sprintf("(%d of %d)", e.findCurrentMatch+1, len(e.findMatches))
			}
		}
		padding := max(0, e.termWidth-runewidth.StringWidth(prompt)-runewidth.StringWidth(countStr))
		ab.WriteString(prompt + strings.Repeat(" ", padding) + countStr)
	} else if e.isQuitting || e.isSaveAs || e.isGotoLine {
		ab.WriteString(e.statusMessage)
		if e.isSaveAs || e.isGotoLine {
			ab.WriteString(e.promptBuffer)
		}
	} else if time.Since(e.statusTime) < 5*time.Second {
		ab.WriteString(e.statusMessage)
	}
}

func (e *Editor) drawTildeRow() string {
	var sb strings.Builder
	if e.showLineNumbers {
		fmt.Fprintf(&sb, "%s %*s %s", ansiInvert, e.lineNumWidth-2, "~", ansiReset)
	}
	sb.WriteString(ansiClearLine)
	sb.WriteString("\r\n")
	return sb.String()
}

func (e *Editor) getSelectionCoordsSafe() (int, int, int, int) {
	if !e.selectionActive {
		return -1, -1, -1, -1
	}
	return e.getSelectionCoords()
}

func (e *Editor) isRuneSelected(fileLine, runeIdx, selStartL, selStartC, selEndL, selEndC int) bool {
	if !e.selectionActive {
		return false
	}
	if fileLine > selStartL && fileLine < selEndL {
		return true
	}
	if fileLine == selStartL && fileLine == selEndL {
		return runeIdx >= selStartC && runeIdx < selEndC
	}
	if fileLine == selStartL {
		return runeIdx >= selStartC
	}
	if fileLine == selEndL {
		return runeIdx < selEndC
	}
	return false
}

func (e *Editor) calculateCursorScreenPosition() (int, int) {
	textWidth := e.getTextWidth()
	screenRow := 1
	fileLine := e.viewportY
	lineWrapOffset := e.viewportWrapOffset
	cursorVisX := e.getVisualX(e.cursorY, e.cursorX)
	cursorVisRowInLine := cursorVisX / textWidth
	if e.cursorY == e.viewportY {
		screenRow += cursorVisRowInLine - e.viewportWrapOffset
	} else if e.cursorY > e.viewportY {
		if fileLine < e.buffer.LineCount() {
			totalVisRows := e.countVisualRows(fileLine, textWidth)
			screenRow += (totalVisRows - lineWrapOffset)
			fileLine++
		}
		for fileLine < e.cursorY && fileLine < e.buffer.LineCount() {
			screenRow += e.countVisualRows(fileLine, textWidth)
			fileLine++
		}

		if fileLine == e.cursorY && fileLine < e.buffer.LineCount() {
			screenRow += cursorVisRowInLine
		}
	} else {
		screenRow = 1
	}
	visColOnLine := cursorVisX % textWidth
	visCol := visColOnLine + e.lineNumWidth + 1
	return screenRow, visCol
}
