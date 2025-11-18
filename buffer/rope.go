package buffer

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Constants for node size, controlling performance.
const (
	maxLeafSize = 1024 // Split leaf if it grows larger than this
	minLeafSize = 512  // Merge leaves if they fall below this (for Delete)
)

// --- Structs ---

// Rope is the main data structure for our text buffer.
type Rope struct {
	root       *node
	lineStarts []int // Stores the rune-offset (index) of the *start* of each line.
}

// node is a node in the rope's binary tree.
type node struct {
	left, right *node
	weight      int    // Length (in runes) of the *left* subtree
	data        []rune // nil for internal nodes, non-nil for leaves
}

// Statically check that *Rope implements the Buffer interface.
var _ Buffer = (*Rope)(nil)

// --- Node Helper Methods ---

func (n *node) isLeaf() bool {
	return n.data != nil
}

func (n *node) length() int {
	if n.isLeaf() {
		return len(n.data)
	}
	total := n.weight
	if n.right != nil {
		total += n.right.length()
	}
	return total
}

// --- Constructor ---

// NewRope creates a new Rope, initialized with the given text.
func NewRope(initialText string) *Rope {
	r := &Rope{
		root: &node{data: []rune(initialText)},
	}
	r.rebuildLineIndex()
	return r
}

// rebuildLineIndex scans the *entire* rope and rebuilds the line index.
// This is SLOW and only for initialization.
func (r *Rope) rebuildLineIndex() {
	r.lineStarts = []int{0} // Line 0 always starts at index 0

	// We need a proper iterator to walk the rope.
	// For now, we'll just use the (inefficient) RuneAt.
	// This is a key area for optimization.
	length := r.root.length()
	for i := range length {
		// This is the bottleneck, as RuneAt is O(log N).
		// Total rebuild is O(N log N).
		ru, err := r.RuneAt(i)
		if err != nil {
			break // Should not happen
		}
		if ru == '\n' {
			r.lineStarts = append(r.lineStarts, i+1)
		}
	}
}

// --- Buffer Interface Implementation ---

// Insert inserts a rune at a given (line, col) position.
func (r *Rope) Insert(line, col int, ru rune) {
	index := r.getIndex(line, col)
	r.root = r.root.insert(index, ru)
	r.updateLineIndexOnInsert(index, ru)
}

// Delete deletes a rune at a given (line, col) position.
func (r *Rope) Delete(line, col int) {
	// Deleting "at" (line, col) means deleting the char *before* it,
	// like backspace.
	if col == 0 && line == 0 {
		return // Cannot delete at start of document
	}

	index := r.getIndex(line, col)
	if index == 0 {
		return // Nothing to delete
	}

	// We want to delete the character *before* the index.
	deleteIndex := index - 1
	ru, err := r.RuneAt(deleteIndex)
	if err != nil {
		return // Should not happen
	}

	r.root = r.root.delete(deleteIndex)
	r.updateLineIndexOnDelete(deleteIndex, ru)
}

// GetLine returns the content of a single line as a string.
func (r *Rope) GetLine(line int) string {
	if line < 0 || line >= len(r.lineStarts) {
		return ""
	}

	startIndex := r.lineStarts[line]
	var endIndex int
	if line+1 < len(r.lineStarts) {
		endIndex = r.lineStarts[line+1]
	} else {
		endIndex = r.root.length()
	}

	if endIndex > startIndex {
		lastRune, err := r.RuneAt(endIndex - 1)
		if err == nil && lastRune == '\n' {
			endIndex-- // Exclude the newline
			if endIndex > startIndex {
				prevRune, err := r.RuneAt(endIndex - 1)
				if err == nil && prevRune == '\r' {
					endIndex--
				}
			}
		}
	}

	// This is inefficient (O(K log N)).
	// A proper Rope iterator or Slice() method is needed for optimization.
	var sb strings.Builder
	sb.Grow(endIndex - startIndex)
	for i := startIndex; i < endIndex; i++ {
		r, err := r.RuneAt(i)
		if err != nil {
			return "[ERROR]"
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// LineCount returns the total number of lines in the buffer.
func (r *Rope) LineCount() int {
	return len(r.lineStarts)
}

// WriteTo writes the entire contents of the buffer to an io.Writer.
func (r *Rope) WriteTo(w io.Writer) (int64, error) {
	// This can be optimized by passing the io.Writer down
	// during the traversal.
	s := r.root.toString()
	n, err := w.Write([]byte(s))
	return int64(n), err
}

// --- Rope-Specific Public Methods ---

// RuneAt finds the rune at a specific *global* rune offset (index).
func (r *Rope) RuneAt(index int) (rune, error) {
	if r.root == nil || index < 0 || index >= r.root.length() {
		return 0, fmt.Errorf("index %d out of bounds (length %d)", index, r.root.length())
	}
	return r.root.runeAt(index)
}

// --- Internal Helper Methods ---

// getIndex converts a (line, col) pair to a *global* rune offset (index).
func (r *Rope) getIndex(line, col int) int {
	if r.root == nil {
		return 0
	}
	if line < 0 {
		line = 0
	}
	if line >= len(r.lineStarts) {
		return r.root.length()
	}

	startIndex := r.lineStarts[line]
	var lineCharLength int
	if line+1 < len(r.lineStarts) {
		lineCharLength = (r.lineStarts[line+1] - startIndex) - 1
	} else {
		lineCharLength = r.root.length() - startIndex
	}

	if col < 0 {
		col = 0
	}
	if col > lineCharLength {
		col = lineCharLength
	}
	return startIndex + col
}

// runeAt is the recursive helper for the node.
func (n *node) runeAt(index int) (rune, error) {
	if n.isLeaf() {
		if index < 0 || index >= len(n.data) {
			return 0, fmt.Errorf("internal error: leaf index out of bounds")
		}
		return n.data[index], nil
	}

	if index < n.weight {
		if n.left == nil {
			return 0, fmt.Errorf("internal error: nil left child with weight > 0")
		}
		return n.left.runeAt(index)
	} else {
		if n.right == nil {
			return 0, fmt.Errorf("internal error: nil right child")
		}
		return n.right.runeAt(index - n.weight)
	}
}

func (n *node) insert(index int, ru rune) *node {
	if n.isLeaf() {
		n.data = append(n.data[:index], append([]rune{ru}, n.data[index:]...)...)
		if len(n.data) > maxLeafSize {
			// Split the node
			mid := len(n.data) / 2
			leftData := make([]rune, mid)
			copy(leftData, n.data[:mid])
			rightData := make([]rune, len(n.data)-mid)
			copy(rightData, n.data[mid:])
			newLeftLeaf := &node{data: leftData}
			newRightLeaf := &node{data: rightData}
			return &node{
				left:   newLeftLeaf,
				right:  newRightLeaf,
				weight: len(newLeftLeaf.data),
			}
		}
		return n
	}

	if index < n.weight {
		n.left = n.left.insert(index, ru)
		n.weight++
	} else {
		n.right = n.right.insert(index-n.weight, ru)
	}
	return n
}

// delete is the recursive helper for node deletion.
func (n *node) delete(index int) *node {
	if n.isLeaf() {
		n.data = append(n.data[:index], n.data[index+1:]...)
		return n // Node merging logic would go here
	}

	if index < n.weight {
		n.left = n.left.delete(index)
		n.weight--
	} else {
		n.right = n.right.delete(index - n.weight)
	}

	// Optional: Add logic to merge nodes if children become too small or empty
	if n.left != nil && n.left.length() == 0 {
		// Si el izquierdo está vacío, simplemente promueve el derecho
		return n.right
	}
	if n.right != nil && n.right.length() == 0 {
		// Si el derecho está vacío, simplemente promueve el izquierdo
		return n.left
	}

	return n
}

// toString is a recursive helper to convert the rope to a string.
func (n *node) toString() string {
	if n.isLeaf() {
		return string(n.data)
	}
	var sb strings.Builder
	if n.left != nil {
		sb.WriteString(n.left.toString())
	}
	if n.right != nil {
		sb.WriteString(n.right.toString())
	}
	return sb.String()
}

// findLine uses binary search to find the line containing the global 'index'.
func (r *Rope) findLine(index int) int {
	i := sort.SearchInts(r.lineStarts, index)
	if i == 0 {
		return 0
	}
	if i < len(r.lineStarts) && r.lineStarts[i] == index {
		return i
	}
	return i - 1
}

// updateLineIndexOnInsert incrementally updates the lineStarts array.
func (r *Rope) updateLineIndexOnInsert(index int, ru rune) {
	line := r.findLine(index)

	if ru == '\n' {
		newLineStartIndex := index + 1
		// Shift all subsequent lines
		for i := line + 1; i < len(r.lineStarts); i++ {
			r.lineStarts[i]++
		}
		// Insert new line start
		r.lineStarts = append(r.lineStarts[:line+1], append([]int{newLineStartIndex}, r.lineStarts[line+1:]...)...)
	} else {
		// Just shift all subsequent lines
		for i := line + 1; i < len(r.lineStarts); i++ {
			r.lineStarts[i]++
		}
	}
}

// updateLineIndexOnDelete incrementally updates the lineStarts array.
func (r *Rope) updateLineIndexOnDelete(index int, ru rune) {
	line := r.findLine(index)

	if ru == '\n' {
		// Hard Case: A line was merged
		// The line *after* the deleted '\n' is removed.
		if line+1 < len(r.lineStarts) {
			r.lineStarts = append(r.lineStarts[:line+1], r.lineStarts[line+2:]...)
		}
		// Shift all subsequent lines
		for i := line + 1; i < len(r.lineStarts); i++ {
			r.lineStarts[i]--
		}
	} else {
		// Easy Case: Just a regular character
		// Shift all subsequent lines
		for i := line + 1; i < len(r.lineStarts); i++ {
			r.lineStarts[i]--
		}
	}
}
