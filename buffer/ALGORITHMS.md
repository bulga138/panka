# Rope Data Structure Algorithms

This document describes the key algorithms used in the rope implementation.

## Overview

A rope is a binary tree data structure optimized for efficient text editing. Unlike arrays or strings, ropes provide O(log N) insertions and deletions while maintaining good performance for common operations.

## Data Structure

### Node Structure

Each node in the rope can be either:
- **Leaf node**: Contains actual text data (`data []rune` is non-nil)
- **Internal node**: Contains pointers to left and right subtrees, with a `weight` field storing the length of the left subtree

### Line Index

The rope maintains a `lineStarts []int` array that stores the global rune offset for the start of each line. This allows O(1) line access and O(log N) line-to-index conversion.

## Key Algorithms

### 1. Insertion (O(log N))

**Algorithm:**
1. Convert (line, col) coordinates to global rune index using binary search on `lineStarts`
2. Traverse the tree to find the insertion point
3. If inserting into a leaf:
   - Insert the rune into the leaf's data array
   - If leaf exceeds `maxLeafSize`, split it into two leaves with a new internal node
4. If inserting into an internal node:
   - Recurse into left or right subtree based on index
   - Update weight if inserting into left subtree
5. Incrementally update `lineStarts` array:
   - If inserting a newline, insert a new entry
   - Shift all subsequent line offsets by +1

**Optimization:** The tree is periodically rebalanced if the left/right subtree size ratio exceeds `rebalanceThreshold` (3.0).

### 2. Deletion (O(log N))

**Algorithm:**
1. Convert (line, col) coordinates to global rune index
2. Get the rune to be deleted (at index - 1)
3. Traverse the tree to find the deletion point
4. If deleting from a leaf:
   - Remove the rune from the leaf's data array
   - Optionally merge with sibling if leaf becomes too small
5. If deleting from an internal node:
   - Recurse into left or right subtree
   - Decrement weight if deleting from left subtree
6. Incrementally update `lineStarts` array:
   - If deleting a newline, remove the corresponding entry
   - Shift all subsequent line offsets by -1

### 3. GetLine (O(log N + K))

**Optimized Algorithm:**
1. Use `lineStarts` array to get start and end indices for the line (O(1))
2. Use optimized `slice()` method to extract the substring:
   - Traverse tree once to find all relevant leaf nodes
   - Extract only the needed portions from each leaf
   - Concatenate results
3. Adjust for newline characters (exclude `\n` and optional `\r`)

**Previous Implementation:** Used repeated `RuneAt()` calls, resulting in O(K log N) complexity.

**Optimization:** Single tree traversal with direct leaf access reduces complexity to O(log N + K) where K is the line length.

### 4. Rebalancing

**Algorithm:**
1. Check if tree is unbalanced: `max(leftLen, rightLen) / min(leftLen, rightLen) > rebalanceThreshold`
2. If unbalanced:
   - Convert entire tree to string using efficient traversal
   - Rebuild tree from string (creates balanced structure)
   - Rebuild line index

**Complexity:** O(N), but only called when tree becomes significantly unbalanced.

**Trade-off:** Periodic rebalancing ensures good performance even after many insertions/deletions, at the cost of occasional O(N) operations.

### 5. Line Index Maintenance

The `lineStarts` array is maintained incrementally:

**On Insert:**
- If inserting newline: Insert new entry at position `line+1` with value `index+1`
- Shift all subsequent entries by +1

**On Delete:**
- If deleting newline: Remove entry at position `line+1`
- Shift all subsequent entries by -1

**On Rebuild:**
- Use efficient in-order traversal to find all newline characters
- Build array in single pass: O(N)

### 6. WriteTo (O(N))

**Optimized Algorithm:**
1. Traverse tree in-order
2. For each leaf node, write directly to `io.Writer`
3. Avoid creating large intermediate strings

**Previous Implementation:** Converted entire tree to string first, causing large memory allocations.

**Optimization:** Direct writing during traversal reduces memory usage and improves performance for large files.

## Performance Characteristics

| Operation | Time Complexity | Space Complexity |
|-----------|----------------|------------------|
| Insert    | O(log N)       | O(1)             |
| Delete    | O(log N)       | O(1)             |
| GetLine   | O(log N + K)   | O(K)             |
| LineCount | O(1)           | O(1)             |
| WriteTo   | O(N)           | O(1)             |
| RuneAt    | O(log N)       | O(1)             |

Where:
- N = total number of runes in the document
- K = length of the line being accessed

## Memory Management

- **Leaf nodes:** Store up to `maxLeafSize` (1024) runes before splitting
- **Node splitting:** Prevents individual leaves from growing too large
- **Rebalancing:** Prevents tree from becoming too unbalanced (which would degrade to O(N) performance)

## Future Optimizations

1. **Lazy rebalancing:** Rebalance in background thread
2. **Node merging:** Merge small sibling leaves to reduce tree depth
3. **Caching:** Cache frequently accessed lines
4. **Parallel operations:** Parallelize tree traversal for very large documents

