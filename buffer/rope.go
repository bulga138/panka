// Package buffer provides efficient text storage and manipulation using a rope data structure.
// A rope is a binary tree that stores text in its leaves, providing O(log N) insertions and
// deletions while maintaining good performance for common operations.
package buffer

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Constants for node size, controlling performance and tree balance.
const (
	maxLeafSize = 1024 // Split leaf if it grows larger than this
	minLeafSize = 512  // Merge leaves if they fall below this (for Delete)
	// Rebalancing threshold: if the ratio of left/right subtree sizes exceeds this,
	// the tree is considered unbalanced and should be rebalanced.
	rebalanceThreshold = 3.0
)

// Rope is the main data structure for our text buffer.
// It uses a binary tree (rope) to store text efficiently, providing O(log N) insertions
// and deletions. The rope maintains a line index for fast line-based operations.
type Rope struct {
	root       *node
	lineStarts []int // Stores the rune-offset (index) of the *start* of each line.
}

// node is a node in the rope's binary tree.
// Internal nodes have nil data and store the weight (length) of the left subtree.
// Leaf nodes have non-nil data containing the actual text runes.
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
// If the text is empty, an empty rope is created.
// The line index is automatically built during initialization.
func NewRope(initialText string) *Rope {
	r := &Rope{
		root: &node{data: []rune(initialText)},
	}
	r.rebuildLineIndex()
	return r
}

// rebuildLineIndex scans the entire rope and rebuilds the line index.
// This is O(N) and should only be called during initialization.
// It uses an efficient in-order traversal to find all newline characters.
func (r *Rope) rebuildLineIndex() {
	r.lineStarts = []int{0} // Line 0 always starts at index 0
	if r.root == nil {
		return
	}

	// Use efficient in-order traversal instead of RuneAt
	r.root.rebuildLineIndexHelper(0, &r.lineStarts)
}

// --- Buffer Interface Implementation ---

// Insert inserts a rune at a given (line, col) position.
// The line and column are 0-indexed. After insertion, the rope may be rebalanced
// if it becomes too unbalanced. Time complexity: O(log N).
func (r *Rope) Insert(line, col int, ru rune) error {
	if r.root == nil {
		r.root = &node{data: []rune{}}
	}
	index, err := r.getIndex(line, col)
	if err != nil {
		return fmt.Errorf("invalid position (line %d, col %d): %w", line, col, err)
	}
	r.root = r.root.insert(index, ru)
	r.updateLineIndexOnInsert(index, ru)

	// Periodically rebalance if tree becomes too unbalanced
	if r.shouldRebalance() {
		r.rebalance()
	}
	return nil
}

// Delete deletes a rune at a given (line, col) position.
// Deleting "at" (line, col) means deleting the char *before* it (like backspace).
// Returns an error if the position is invalid or at the start of the document.
// Time complexity: O(log N).
func (r *Rope) Delete(line, col int) error {
	if r.root == nil {
		return fmt.Errorf("cannot delete from empty buffer")
	}

	// Cannot delete at start of document
	if col == 0 && line == 0 {
		return fmt.Errorf("cannot delete at start of document")
	}

	index, err := r.getIndex(line, col)
	if err != nil {
		return fmt.Errorf("invalid position (line %d, col %d): %w", line, col, err)
	}

	if index == 0 {
		return fmt.Errorf("nothing to delete at start of document")
	}

	// We want to delete the character *before* the index.
	deleteIndex := index - 1
	ru, err := r.RuneAt(deleteIndex)
	if err != nil {
		return fmt.Errorf("failed to get rune at delete position: %w", err)
	}

	r.root = r.root.delete(deleteIndex)
	r.updateLineIndexOnDelete(deleteIndex, ru)

	// Periodically rebalance if tree becomes too unbalanced
	if r.shouldRebalance() {
		r.rebalance()
	}
	return nil
}

// GetLine returns the content of a single line as a string.
// The line number is 0-indexed. Returns an empty string if the line is out of bounds.
// This method is optimized to O(log N + K) where K is the line length, using efficient
// tree traversal instead of repeated RuneAt calls.
func (r *Rope) GetLine(line int) string {
	if r.root == nil {
		return ""
	}
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

	// Adjust for newline characters (exclude \n and optional \r)
	if endIndex > startIndex {
		// Use optimized slice method to get the last rune
		if lastRune, err := r.sliceRuneAt(endIndex - 1); err == nil && lastRune == '\n' {
			endIndex-- // Exclude the newline
			if endIndex > startIndex {
				if prevRune, err := r.sliceRuneAt(endIndex - 1); err == nil && prevRune == '\r' {
					endIndex--
				}
			}
		}
	}

	// Use optimized slice method: O(log N + K) instead of O(K log N)
	result := r.slice(startIndex, endIndex)
	return result
}

// LineCount returns the total number of lines in the buffer.
// An empty buffer has 1 line. Time complexity: O(1).
func (r *Rope) LineCount() int {
	if len(r.lineStarts) == 0 {
		return 1
	}
	return len(r.lineStarts)
}

// WriteTo writes the entire contents of the buffer to an io.Writer.
// This is optimized to write directly during tree traversal, avoiding
// large string allocations. Time complexity: O(N).
func (r *Rope) WriteTo(w io.Writer) (int64, error) {
	if r.root == nil {
		return 0, nil
	}
	return r.root.writeTo(w)
}

// --- Rope-Specific Public Methods ---

// RuneAt finds the rune at a specific *global* rune offset (index).
// The index is 0-based and refers to the position in the entire document.
// Returns an error if the index is out of bounds. Time complexity: O(log N).
func (r *Rope) RuneAt(index int) (rune, error) {
	if r.root == nil {
		return 0, fmt.Errorf("index %d out of bounds: buffer is empty", index)
	}
	length := r.root.length()
	if index < 0 || index >= length {
		return 0, fmt.Errorf("index %d out of bounds (length %d)", index, length)
	}
	return r.root.runeAt(index)
}

// --- Internal Helper Methods ---

// getIndex converts a (line, col) pair to a *global* rune offset (index).
// Returns an error if the line is out of bounds. The column is clamped to valid range.
func (r *Rope) getIndex(line, col int) (int, error) {
	if r.root == nil {
		if line == 0 && col == 0 {
			return 0, nil
		}
		return 0, fmt.Errorf("line %d out of bounds: buffer is empty", line)
	}

	if line < 0 {
		return 0, fmt.Errorf("line %d is negative", line)
	}
	if line >= len(r.lineStarts) {
		return r.root.length(), fmt.Errorf("line %d out of bounds (max line: %d)", line, len(r.lineStarts)-1)
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
	return startIndex + col, nil
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

// --- Optimization Methods ---

// slice extracts a substring from startIndex to endIndex (exclusive) efficiently.
// Time complexity: O(log N + K) where K is the length of the slice.
func (r *Rope) slice(startIndex, endIndex int) string {
	if r.root == nil || startIndex >= endIndex {
		return ""
	}
	var result strings.Builder
	result.Grow(endIndex - startIndex)
	r.root.sliceHelper(startIndex, endIndex, 0, &result)
	return result.String()
}

// sliceRuneAt gets a single rune at the given index using the optimized slice method.
func (r *Rope) sliceRuneAt(index int) (rune, error) {
	if r.root == nil {
		return 0, fmt.Errorf("buffer is empty")
	}
	length := r.root.length()
	if index < 0 || index >= length {
		return 0, fmt.Errorf("index %d out of bounds (length %d)", index, length)
	}
	return r.root.runeAt(index)
}

// sliceHelper is a recursive helper that efficiently extracts a slice from the tree.
func (n *node) sliceHelper(startIndex, endIndex, offset int, result *strings.Builder) {
	if n.isLeaf() {
		leafStart := offset

		// Calculate the overlap
		sliceStart := max(0, startIndex-leafStart)
		sliceEnd := min(len(n.data), endIndex-leafStart)

		if sliceStart < sliceEnd {
			result.WriteString(string(n.data[sliceStart:sliceEnd]))
		}
		return
	}

	leftEnd := offset + n.weight

	// Recurse into left subtree if needed
	if startIndex < leftEnd && offset < endIndex {
		n.left.sliceHelper(startIndex, endIndex, offset, result)
	}

	// Recurse into right subtree if needed
	if endIndex > leftEnd && leftEnd < endIndex {
		n.right.sliceHelper(startIndex, endIndex, leftEnd, result)
	}
}

// rebuildLineIndexHelper efficiently rebuilds the line index using in-order traversal.
func (n *node) rebuildLineIndexHelper(offset int, lineStarts *[]int) {
	if n.isLeaf() {
		for i, r := range n.data {
			if r == '\n' {
				*lineStarts = append(*lineStarts, offset+i+1)
			}
		}
		return
	}

	if n.left != nil {
		n.left.rebuildLineIndexHelper(offset, lineStarts)
	}
	if n.right != nil {
		n.right.rebuildLineIndexHelper(offset+n.weight, lineStarts)
	}
}

// writeTo writes the rope contents directly to an io.Writer during tree traversal.
// This avoids creating large intermediate strings.
func (n *node) writeTo(w io.Writer) (int64, error) {
	if n.isLeaf() {
		n, err := w.Write([]byte(string(n.data)))
		return int64(n), err
	}

	var total int64
	if n.left != nil {
		written, err := n.left.writeTo(w)
		if err != nil {
			return total, err
		}
		total += written
	}
	if n.right != nil {
		written, err := n.right.writeTo(w)
		if err != nil {
			return total, err
		}
		total += written
	}
	return total, nil
}

// shouldRebalance checks if the rope tree is unbalanced and needs rebalancing.
// A tree is considered unbalanced if the ratio of left/right subtree sizes
// exceeds the rebalanceThreshold.
func (r *Rope) shouldRebalance() bool {
	if r.root == nil || r.root.isLeaf() {
		return false
	}

	leftLen := r.root.weight
	rightLen := r.root.length() - leftLen

	if leftLen == 0 || rightLen == 0 {
		return false
	}

	ratio := float64(max(leftLen, rightLen)) / float64(min(leftLen, rightLen))
	return ratio > rebalanceThreshold
}

// rebalance rebuilds the rope tree to ensure better balance.
// This is done by converting the tree to a flat string and rebuilding it.
// While this is O(N), it's only called when the tree becomes significantly unbalanced.
func (r *Rope) rebalance() {
	if r.root == nil {
		return
	}

	// Convert to string and rebuild
	var buf strings.Builder
	r.root.writeTo(&buf)
	content := buf.String()

	// Rebuild the tree
	r.root = &node{data: []rune(content)}
	r.rebuildLineIndex()
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
