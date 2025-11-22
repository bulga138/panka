# Undo/Redo Algorithm

This document describes the undo/redo system implementation in the Panka editor.

## Overview

The undo/redo system uses an operation-based approach, storing sequences of operations rather than document snapshots. This is memory-efficient and allows for unlimited undo/redo operations.

## Data Structures

### opEntry

Represents a single edit operation:

```go
type opEntry struct {
    insertLine int  // Line where rune was inserted
    insertCol  int  // Column where rune was inserted
    delLine    int  // Line where rune can be deleted (for undo)
    delCol     int  // Column where rune can be deleted (for undo)
    r          rune // The rune that was inserted/deleted
}
```

### undoAction

Represents a group of operations that can be undone/redone atomically:

```go
type undoAction struct {
    isInsert    bool      // true if this is an insert operation
    ops         []opEntry // Sequence of operations
    groupID     int       // For grouping related operations
    isBackspace bool      // Distinguishes backspace from delete
}
```

## Operation Grouping

### Time-Based Grouping

Operations are automatically grouped based on timing:

- **Typing:** Characters typed within 900ms are grouped together
- **Backspace:** Backspace operations within 900ms are grouped together
- **Delete:** Delete key operations within 900ms are grouped together

This provides intuitive undo behavior: typing a word and pressing undo removes the entire word, not individual characters.

### Manual Grouping

Operations can be manually grouped using `beginUndoGroup()` and `endUndoGroup()`:

- **Cut/Paste:** All operations in a cut or paste are grouped
- **Replace All:** All replacements are grouped together
- **Multi-line operations:** Complex operations spanning multiple lines

## Undo Algorithm

### performUndo()

**For Insert Operations:**
1. Delete inserted runes in **reverse order** using the recorded `delLine` and `delCol` positions
2. Position cursor at the start of the insertion (where typing began)

**For Delete Operations:**
1. Re-insert deleted runes in **forward order** at their original `insertLine` and `insertCol` positions
2. Position cursor based on operation type:
   - **Backspace:** Cursor at end of re-inserted block
   - **Delete/Cut:** Cursor at start of re-inserted block

**Grouped Operations:**
- If an action has a `groupID > 0`, all actions with the same `groupID` are undone together
- This ensures atomic undo for grouped operations

### undo()

1. Flush any pending typing/backspace groups
2. Pop the last action from `undoStack`
3. Push it to `redoStack`
4. Call `performUndo()` on the action
5. If the action is grouped, undo all actions with the same `groupID`

## Redo Algorithm

### performRedo()

**For Insert Operations:**
1. Re-insert runes in **forward order** at recorded positions
2. Position cursor at end of inserted block

**For Delete Operations:**
1. Delete runes in **reverse order** using `insertLine` and `insertCol+1`
2. Position cursor at start of deleted block

### redo()

1. Flush any pending typing/backspace groups
2. Pop the last action from `redoStack`
3. Push it to `undoStack`
4. Call `performRedo()` on the action
5. If the action is grouped, redo all actions with the same `groupID`

## Cursor Positioning

The undo/redo system carefully manages cursor position to provide intuitive behavior:

- **After undo insert:** Cursor at start of insertion (where you started typing)
- **After undo delete (backspace):** Cursor at end of re-inserted text
- **After undo delete (cut):** Cursor at start of re-inserted text
- **After redo insert:** Cursor at end of inserted text
- **After redo delete:** Cursor at start of deleted text

## Error Handling

All buffer operations (Insert/Delete) now return errors. The undo/redo system:

1. Checks for errors on each operation
2. Displays error message to user if operation fails
3. Stops undo/redo sequence if an error occurs
4. Maintains consistency: if an operation fails, the undo/redo state remains valid

## Example: Typing a Word

1. User types "hello" (5 characters in quick succession)
2. Each character creates an `opEntry` and is added to `typingEntries`
3. After 900ms of no typing, `flushTypingGroup()` is called
4. All 5 operations are grouped into a single `undoAction` with `groupID = 0`
5. Pressing Ctrl+U (undo) removes all 5 characters at once
6. Pressing Ctrl+Y (redo) re-inserts all 5 characters

## Example: Cut and Paste

1. User selects text and presses Ctrl+X (cut)
2. `beginUndoGroup()` is called
3. Text is deleted (creates delete action)
4. `endUndoGroup()` is called
5. User presses Ctrl+V (paste)
6. `beginUndoGroup()` is called
7. Text is inserted (creates insert action)
8. `endUndoGroup()` is called
9. Each operation (cut, paste) can be undone independently

## Memory Efficiency

- **No snapshots:** Only stores operations, not full document state
- **Bounded memory:** Each operation stores minimal data (line, col, rune)
- **Grouping reduces entries:** Related operations are stored as single actions

## Performance

- **Undo/Redo:** O(K) where K is the number of operations in the action
- **Grouping:** O(1) per operation
- **Memory:** O(N) where N is the number of operations (typically much less than document size)

## Future Improvements

1. **Limit undo stack size:** Prevent unbounded growth
2. **Compression:** Compress sequences of identical operations
3. **Persistent undo:** Save undo stack to disk for crash recovery
4. **Branching undo:** Support multiple undo branches

