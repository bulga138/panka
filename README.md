# panka

![](./assets/banner.png)

> (aimara panka). s. 1. Bot. Dry leaf or bract that surrounds the ear of corn; husk. 2. Educ. Book, notebook, or physical medium for school reading and writing.

A lightweight, high-performance console-based text editor for Windows PowerShell. Designed with familiar key bindings, it includes modern features like multi-cursor editing, smart line manipulation, and infinite undo/redo.

## Features

- **Core Editing**: Fast typing, infinite Undo/Redo (`Ctrl+U`, `Ctrl+Y`), and Clipboard integration.
- **Search & Replace**: Full find and replace functionality with "Replace Next" and "Replace All" support.
- **Multi-Cursor**: Vertical column selection ("Block Mode") for editing multiple lines simultaneously.
- **Line Operations**: Move lines up/down, duplicate lines, and smart indentation.
- **Text Manipulation**: Toggle case (lowercase, UPPERCASE, Title Case).
- **Visual Aids**: Toggleable line numbers and non-printable characters (spaces, tabs, newlines).

## Installation

These instructions will guide you through downloading the editor, adding it to your system's **PATH**, and setting a convenient PowerShell alias.

### 1. Download the Binary

First, create a directory in your user profile and download the latest `panka.exe`.

Run the following commands in PowerShell:
```powershell
# Set the installation directory
$installDir = "$env:USERPROFILE\panka"

# Create the directory if it doesn't exist
if (-not (Test-Path -Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir
}

# Download the latest panka.exe
$url = "https://github.com/bulga138/panka/releases/latest/download/panka.exe"
$output = "$installDir\panka.exe"

# Use wget (alias for Invoke-WebRequest) to download the file
Invoke-WebRequest -Uri $url -OutFile $output
```

### 2. Add to PATH

Add this directory to your user's PATH environment variable so you can run `panka.exe` from any terminal.
```powershell
# Get the current user PATH
$oldPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")

# Append the new path if it's not already included
if ($oldPath -notlike "*$installDir*") {
    $newPath = $oldPath + ";" + $installDir
    [System.Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    
    Write-Host "panka has been added to your PATH."
    Write-Host "Please restart your PowerShell session for the changes to take effect."
} else {
    Write-Host "panka is already in your PATH."
}
```

**Important**: You must **restart** your PowerShell session after this step.

### 3. (Optional) Set PowerShell Alias

Create a permanent short alias (e.g., pk) for `panka.exe`.
```powershell
# 1. Ensure a PowerShell profile file exists
if (-not (Test-Path -Path $PROFILE)) {
    New-Item -ItemType File -Path $PROFILE -Force
}

# 2. Add the alias to your profile
$aliasCommand = "Set-Alias -Name pk -Value panka.exe"
Add-Content -Path $PROFILE -Value $aliasCommand

# 3. Reload your profile
. $PROFILE

Write-Host "Alias 'pk' created. You can now run the editor by typing 'pk'."
```

## Usage
```powershell
# Open an existing file or create a new one
pk my_file.txt

# Open empty editor
pk

# Display version
pk --version
```

## Configuration

You can customize panka's settings by creating a `config.toml` file. Run `panka --init-config` to generate a default file in your configuration directory.

Example `config.toml`:
```toml
# Number of spaces for a tab.
tabSize = 4

# Whether to show line numbers on startup.
showLineNumbers = true

# Whether to show non-printable characters (spaces as ·, tabs as →, newlines as ¶).
showNonPrintable = false

# Set to true to enable debug logging.
enableLogger = false
```

## Key Bindings

### General & File

|Action|Key
|---|----|
|**Save File**|`Ctrl` + `S`||
|**Save As**|`Ctrl` + `E`||
|**Quit**|`Ctrl` + `Q`||
|**Undo**|`Ctrl` + `U`||
|**Redo**|`Ctrl` + `Y`||
|**Toggle Line Numbers**|`Ctrl` + `L`||
|**Toggle Non-Printables**|`Ctrl` + `O`||


## Editing & Clipboard

Action|Key|
|---|---|
|**Cut**|`Ctrl` + `X`||
|**Copy**|`Ctrl` + `C`||
|**Paste**|`Ctrl` + `V`||
|**Duplicate Line**|`Ctrl` + `D`||
|**Move Line Up**|`Ctrl` + `Alt` + `Up`||
|**Move Line Down**|`Ctrl` + `Alt` + `Down`||
|**Toggle Case**|`Ctrl` + `K`||
|**Indent Line**|`Tab`||
|**Unindent Line**|`Shift` + `Tab`||

### Navigation & Selection

|Action|Key|
|---|---|
|**Go to Line**|`Ctrl` + `T`||
|**Select All**|`Ctrl` + `A`||
|**Select Text**|`Shift` + `Arrows`||
|**Move by Word**|`Ctrl` + `Left` / `Right`||
|**Doc Start/End**|`Ctrl` + `Home` / `End`||

### Search & Replace

|Action|Key|Description|
|---|---|---|
|**Find**|`Ctrl` + `F`|Open Find prompt||
**Replace**|`Ctrl` + `H`|Open Find & Replace prompt||
|**Find Next**|`Enter` or `Ctrl` + `N`|Jump to next match||
|**Find Previous**|`Ctrl` + `P`|Jump to previous match||
|**Replace Next**|`Ctrl` + `R`|Replace current match & find next||
|**Replace All**|`Ctrl` + `A`|Replace all matches (requires confirm)||
|**Switch Focus**|`Tab`|Switch between Find/Replace inputs||

### Multi-Cursor (Block Mode)

Use these keys to create a vertical block of cursors for simultaneous editing.

|Action|Key|
|---|---|
|Extend Cursor Down|`Ctrl` + `Alt` + `Right`||
|Extend Cursor Up|`Ctrl` + `Alt` + `Left`||
|Cancel Multi-Cursor|`Esc` or arrow keys without modifiers||

## License

This project is licensed under the MIT License.
