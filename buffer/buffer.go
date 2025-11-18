package buffer

import "io"

type Buffer interface {
	// Inserts a rune at a given (line, col) position.
	Insert(line, col int, r rune)

	// Deletes a rune at a given (line, col) position.
	// This will have to be smart about joining lines if we delete a newline.
	Delete(line, col int)

	// GetLine returns the content of a single line.
	GetLine(line int) string

	// LineCount returns the total number of lines.
	LineCount() int

	// WriteTo writes the entire contents of the buffer to an io.Writer.
	// This is the primary method for saving the file, returning the number of bytes written and any error.
	WriteTo(w io.Writer) (int64, error)
}
