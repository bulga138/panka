package editor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bulga138/panka/buffer"
	"github.com/bulga138/panka/config"
	"github.com/bulga138/panka/runewidth"
	"github.com/bulga138/panka/terminal"
)

// ANSI escape codes
const (
	ansiHideCursor     = "\x1b[?25l"
	ansiShowCursor     = "\x1b[?25h"
	ansiClearScreen    = "\x1b[2J"
	ansiMoveToHome     = "\x1b[H"
	ansiClearLine      = "\x1b[K"
	ansiReset          = "\x1b[m"
	ansiInvert         = "\x1b[7m"
	ansiDim            = "\x1b[2m" // Added Dim for non-printables
	ansiEnterAltScreen = "\x1b[?1049h"
	ansiExitAltScreen  = "\x1b[?1049l"
)

type findResult struct {
	y int
	x int
}

type Editor struct {
	term       terminal.Terminal
	buffer     buffer.Buffer
	config     config.Config
	filename   string
	termWidth  int
	termHeight int
	cursorX    int
	cursorY    int

	// Multi-cursor state
	// 0 = single cursor.
	// > 0 = extends downwards (e.g., 2 means current line + 2 lines below).
	// < 0 = extends upwards (e.g., -2 means current line + 2 lines above).
	extraCursorHeight int

	viewportWrapOffset int
	viewportY          int
	viewportCol        int
	lineNumWidth       int
	dirty              bool
	initialHash        string
	statusMessage      string
	statusTime         time.Time
	quit               bool
	inputReader        *bufio.Reader
	undoStack          []undoAction
	redoStack          []undoAction
	selectionActive    bool
	selectionAnchorX   int
	selectionAnchorY   int
	isQuitting         bool

	// Grouping mechanism
	undoGrouping   bool
	currentGroupID int
	lastGroupID    int

	// Typing grouping
	typingEntries      []opEntry
	typingActive       bool
	lastTypeTime       time.Time
	typeGroupThreshold time.Duration

	// Backspace grouping
	backspaceEntries   []opEntry
	backspaceActive    bool
	lastBackspaceTime  time.Time
	backspaceThreshold time.Duration

	// Line related
	showLineNumbers  bool
	showNonPrintable bool
	isGotoLine       bool

	// Prompt
	promptBuffer        string
	promptCursorX       int
	replaceBuffer       string
	replaceCursorX      int
	promptFocus         int
	isConfirmingReplace bool

	// Find related
	isFinding        bool
	isReplacing      bool
	lastSearchQuery  string
	findOrigCursorX  int
	findOrigCursorY  int
	findMatches      []findResult
	findCurrentMatch int

	// Delete
	deleteEntries   []opEntry
	deleteActive    bool
	lastDeleteTime  time.Time
	deleteThreshold time.Duration

	// Wrapped row
	lastTermWidth  int
	lastTermHeight int

	// Save
	isSaveAs bool
}

type opEntry struct {
	insertLine int
	insertCol  int
	delLine    int
	delCol     int
	r          rune
}

type undoAction struct {
	isInsert    bool
	ops         []opEntry
	groupID     int
	isBackspace bool
}

func NewEditor(term terminal.Terminal, cfg config.Config, file string) (*Editor, error) {
	e := &Editor{
		term:                term,
		config:              cfg,
		filename:            file,
		inputReader:         bufio.NewReader(term.Stdin()),
		lineNumWidth:        5,
		showLineNumbers:     cfg.ShowLineNumbers,
		showNonPrintable:    cfg.ShowNonPrintable,
		undoStack:           make([]undoAction, 0),
		redoStack:           make([]undoAction, 0),
		isQuitting:          false,
		lastGroupID:         1,
		typeGroupThreshold:  900 * time.Millisecond,
		backspaceThreshold:  900 * time.Millisecond,
		deleteThreshold:     900 * time.Millisecond,
		isGotoLine:          false,
		isFinding:           false,
		isReplacing:         false,
		viewportWrapOffset:  0,
		promptCursorX:       0,
		replaceBuffer:       "",
		replaceCursorX:      0,
		promptFocus:         0,
		isConfirmingReplace: false,
		initialHash:         "",
		extraCursorHeight:   0,
	}
	var content string
	if file != "" {
		var err error
		content, err = e.loadFileContent(file)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load file %s: %w", file, err)
		}
	}
	e.buffer = buffer.NewRope(content)
	e.initialHash = e.calculateBufferHash()

	e.refreshSize()
	e.updateLineNumWidth()
	e.lastTermWidth = e.termWidth
	e.lastTermHeight = e.termHeight + 3
	if !e.showLineNumbers {
		e.lineNumWidth = 0
	}
	return e, nil
}

func (e *Editor) refreshSize() {
	w, h, err := e.term.GetWindowSize()
	if err != nil {
		e.termWidth = 80
		e.termHeight = 24
	} else {
		e.termWidth = w
		e.termHeight = h
	}
	if e.termHeight < 3 {
		e.termHeight = 3
	}
	e.termHeight -= 3
}

func (e *Editor) Run() error {
	if err := e.term.EnableRawMode(); err != nil {
		return err
	}
	os.Stdout.WriteString(ansiEnterAltScreen)
	defer func() {
		e.term.DisableRawMode()
		os.Stdout.WriteString(ansiExitAltScreen)
	}()
	for !e.quit {
		e.checkResize()
		e.render()
		if err := e.processInput(); err != nil {
			break
		}
	}
	return nil
}

func (e *Editor) getVisualX(lineY int, runeX int) int {
	if lineY >= e.buffer.LineCount() {
		return 0
	}

	runes := []rune(e.buffer.GetLine(lineY))
	if runeX > len(runes) {
		runeX = len(runes)
	}

	visX := 0
	for i := 0; i < runeX && i < len(runes); i++ {
		r := runes[i]
		if r == '\t' {
			visX += e.config.TabSize - (visX % e.config.TabSize)
		} else {
			visX += runewidth.RuneWidth(r)
		}
	}
	return visX
}

func (e *Editor) checkResize() {
	w, h, err := e.term.GetWindowSize()
	if err != nil {
		return
	}
	if w == e.lastTermWidth && h == e.lastTermHeight {
		return
	}
	e.lastTermWidth = w
	e.lastTermHeight = h
	e.termWidth = w
	e.termHeight = h
	if e.termHeight < 3 {
		e.termHeight = 3
	}
	e.termHeight -= 3
	e.updateLineNumWidth()
	e.setStatusMessage("Window resized to %d x %d", e.termWidth, e.termHeight)
}

func (e *Editor) updateLineNumWidth() {
	if e.showLineNumbers {
		e.lineNumWidth = 5
	} else {
		e.lineNumWidth = 0
	}
}

func (e *Editor) getTextWidth() int {
	textWidth := e.termWidth - e.lineNumWidth
	if textWidth < 1 {
		return 1
	}
	return textWidth
}

func (e *Editor) countVisualRows(fileLine int, textWidth int) int {
	if fileLine >= e.buffer.LineCount() {
		return 1
	}
	line := e.buffer.GetLine(fileLine)
	if len([]rune(line)) == 0 {
		return 1
	}
	lineVisWidth := e.getVisualX(fileLine, len([]rune(line)))
	if lineVisWidth == 0 {
		return 1
	}
	numVisualRows := (lineVisWidth + textWidth - 1) / textWidth
	if numVisualRows == 0 {
		return 1
	}
	return numVisualRows
}

func (e *Editor) loadFileContent(filename string) (string, error) {
	const streamingThreshold = 1024 * 1024

	info, err := os.Stat(filename)
	if err != nil {
		return "", err
	}

	if info.Size() < streamingThreshold {
		b, err := os.ReadFile(filename)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var result strings.Builder
	result.Grow(int(info.Size()))

	buf := make([]byte, 64*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error reading file: %w", err)
		}
	}

	return result.String(), nil
}

func (e *Editor) getVisualCursorPos() (int, int) {
	textWidth := e.getTextWidth()
	visRow := 1
	for fileLine := e.viewportY; fileLine < e.cursorY; fileLine++ {
		if fileLine >= e.buffer.LineCount() {
			break
		}
		visRow += e.countVisualRows(fileLine, textWidth)
	}
	visCursorX := e.getVisualX(e.cursorY, e.cursorX)
	visRow += (visCursorX / textWidth)
	visRow -= e.viewportWrapOffset
	visColOnLine := visCursorX - e.viewportCol
	visCol := visColOnLine + e.lineNumWidth + 1
	return visRow, visCol
}
