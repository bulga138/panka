Buffer Package

This package defines the core text-storage logic for the Panka editor.

buffer.go

Purpose: Defines the Buffer interface.

Design Decisions:

This interface is the "contract" between the editor and the data structure. It decouples the editor's UI logic from the complex text storage.

All operations use (line, col) coordinates. This makes the editor's job simple and pushes the complexity of coordinate-to-index translation into the buffer implementation.

WriteTo(io.Writer) is used for saving. This is a standard Go practice, allowing the buffer to write to a file, a network connection, or an in-memory buffer (for testing) without modification.

rope.go

Purpose: Implements the Buffer interface using a Rope data structure.

Design Decisions:

Why a Rope? A simple []string or [][]rune is fast for reading but terrible for editing. Inserting a character at the start of a 1MB line would require re-allocating and copying 1MB of data. A Rope is a binary tree that holds text in its leaves. Insertions and deletions are $O(\log N)$ operations (where $N$ is the document size) because they only involve creating a few new small nodes and changing pointers, not copying massive blocks of text.

lineStarts Index: The Rope itself only knows about a single stream of runes. To satisfy the (line, col) requirement, we maintain a []int slice that stores the global rune offset for the start of every line.

Incremental Updates: The most complex part of this file is updateLineIndexOnInsert and updateLineIndexOnDelete. Re-building the entire line index on every keystroke would be slow. These functions perform an "incremental" update. When a \n is added, they insert a new entry into lineStarts. When a character is added, they just increment the offsets of all subsequent lines. This keeps editing fast.

Node Splitting/Merging:

Insert: When a leaf node grows past maxLeafSize, it splits into two smaller leaves, and a new internal node is created to be their parent.

Delete: When a leaf node becomes empty (or below a minLeafSize threshold, see improvements), it's removed, and its parent may be simplified. This keeps the tree from becoming sparse.

Future Improvements:

Balancing: The tree is not currently self-balancing (like an AVL or Red-Black tree). A long series of patterned edits could unbalance it, degrading performance. A re-balancing function should be added.

Efficient Slicing: GetLine currently uses a $O(K \log N)$ loop (where $K$ is line length). A proper Slice(start, end) function would traverse the tree once, collecting all leaf data in $O(\log N + K)$ time, which is much faster.

Node Merging: The delete function is simple. A more robust implementation would merge leaf nodes when they become too small, not just when empty, to keep the tree compact.