package buffer

import "io"

type Buffer interface {
	// Insert a rune at a given (line, col) position.
	// Returns an error if the position is invalid.
	Insert(line, col int, r rune) error

	// Deletes a rune at a given (line, col) position.
	// Deleting "at" (line, col) means deleting the char *before* it (like backspace).
	// Returns an error if the position is invalid or at the start of the document.
	Delete(line, col int) error

	// GetLine returns the content of a single line.
	// Returns an empty string if the line is out of bounds.
	GetLine(line int) string

	// LineCount returns the total number of lines in the buffer.
	LineCount() int

	// WriteTo writes the entire contents of the buffer to an io.Writer.
	// Returns the number of bytes written and any error encountered.
	WriteTo(w io.Writer) (int64, error)
}
