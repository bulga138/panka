Config Package

This package manages loading the editor's configuration.

config.go

Purpose: Defines the Config struct and provides a LoadConfig function.

Design Decisions:

JSON, not TOML: The user request was contradictory ("use TOML" vs. "no 3rd-party packages"). TOML parsing requires a 3rd-party library. encoding/json is part of the standard library. We chose to respect the "no 3rd-party" rule, as it's a harder constraint.

File Location: The config is loaded from a user's home directory (.panka.json). This is a standard practice for user-specific configuration.

Defaults: If the config file is not found, LoadConfig returns a DefaultConfig() struct. This ensures the editor always has a valid configuration to run with.

Future Improvements:

Add more configuration options (e.g., theme, keybindings).

If 3rd-party libraries are allowed, switch to go-toml for a more human-readable config file.