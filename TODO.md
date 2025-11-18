Panka Editor - TODO & Roadmap

This file tracks the "Grow Up" features for Panka beyond the MVP.

Core Editing

[ ] Undo / Redo: Implement an undo stack. This could store inverse operations (e.g., "insert 'a' at 5" becomes "delete at 5") or use a persistent data structure.

[ ] Copy / Paste: Implement an internal clipboard.

[ ] System Clipboard: Integrate with the cross-platform system clipboard (this is difficult without Cgo or 3rd-party libs).

[ ] Search (Ctrl+F): Add a search prompt in the status bar and highlight matches.

[ ] Find & Replace: Extend search to support replacing text.

[ ] Select Text: Add a "selection mode" (e.g., with Shift + Arrow Keys).

Performance & Buffer

[ ] Rope Balancing: The rope currently splits but never re-balances. An unbalanced tree can degrade performance to $O(N)$. We should add a re-balancing function that can run in the background.

[ ] Rope Slicing: Replace GetLine's $O(K \log N)$ loop with a true $O(\log N + K)$ Slice() method that traverses the tree once.

[ ] Efficient WriteTo: The WriteTo can be made more efficient by using an in-order traversal with an io.Writer directly, avoiding string allocations.

[ ] Large File Loading: Currently, we read the entire file into the rope on startup. This should be streamed to build the rope in chunks for multi-GB files.

UI & Features

[ ] Line Numbers: Add an optional gutter for line numbers.

[ ] Syntax Highlighting: This is a major feature. It requires a regex-based or tree-sitter-based parsing engine.

[ ] Tabs / Multiple Buffers: Allow opening and switching between multiple files.

[ ] Split Views: Allow vertical and horizontal splits to view multiple files (or the same file) at once.

[ ] Mouse Support: Enable mouse-based cursor movement and scrolling.

[ ] File Tree: Add a toggleable file tree sidebar.

Configuration

[ ] TOML Support: If we relax the "no 3rd-party" rule, switch from JSON to TOML for a more user-friendly config file.

[ ] Keymap Configuration: Allow users to remap keybindings in the config file.