# panka

A lightweight, console-based text editor for Windows PowerShell, designed with familiar Notepad and Word-style key bindings.

## Features

  * Standard file operations (Save, Quit)
  * Undo / Redo (Ctrl+U, Ctrl+Y)
  * Cut / Copy / Paste using the Windows clipboard (Ctrl+X, Ctrl+C, Ctrl+V)
  * Text selection (Shift + Arrows, Ctrl+A)
  * Find (Ctrl+F)
  * Go to Line (Ctrl+T)
  * Full document and line navigation (Home/End, Ctrl+Home/End, PageUp/Down)
  * Toggleable line numbers (Ctrl+L)

## Installation

These instructions will guide you through downloading the editor, adding it to your system's PATH, and setting a convenient PowerShell alias.

### 1\. Download the Binary

First, we will create a directory in your user profile and download the latest `panka.exe` binary to it.

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
wget -Uri $url -OutFile $output
```

### 2\. Add to PATH

Next, add this directory to your user's PATH environment variable so you can run `panka.exe` from any terminal.

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

**Important:** You must **restart your PowerShell session** after this step.

### 3\. (Optional) Set PowerShell Alias

After restarting PowerShell, you can create a permanent, short alias (e.g., `pk` or `edit`) for `panka.exe`. This command will add the alias to your PowerShell profile, creating the file if it doesn't exist.

```powershell
# 1. Ensure a PowerShell profile file exists
if (-not (Test-Path -Path $PROFILE)) {
    New-Item -ItemType File -Path $PROFILE -Force
}

# 2. Add the alias to your profile (using 'pk' as an example)
$aliasCommand = "Set-Alias -Name pk -Value panka.exe"
Add-Content -Path $PROFILE -Value $aliasCommand

# 3. Reload your profile to activate the alias immediately
. $PROFILE

Write-Host "Alias 'pk' created. You can now run the editor by typing 'pk'."
```

## Usage

Once installed, you can run the editor from your terminal.

```powershell
# With alias
pk my_file.txt

# Without alias
panka.exe my_file.txt

# Open without a file
pk
```

## Configuration

You can customize panka's settings by creating a `config.toml` file.

### Creating the Default Config

To create a default configuration file, run `panka` with the `--init-config` flag:

```powershell
panka --init-config
```

This will create a `config.toml` file in the standard user config directory (e.g., `C:\Users\YourUser\AppData\Roaming\panka\config.toml`).

### Available Settings

You can edit this `config.toml` file to change the following settings:

  * **`tabSize`**: The number of spaces to render for a tab character.
      * Default: `4`
  * **`showLineNumbers`**: Whether to display line numbers on startup. This can still be toggled with `Ctrl+L`.
      * Default: `true`
  * **`enableLogger`**: Set to `true` to enable debug logging. This will create a `panka.log` file in the same directory where you run the executable.
      * Default: `false`

**Example `config.toml`:**

```toml
# Number of spaces for a tab.
tabSize = 4

# Whether to show line numbers on startup.
showLineNumbers = true

# Set to true to enable debug logging.
enableLogger = false
```

## Key Bindings

### General

| Action | Key |
| --- | --- |
| Save File | Ctrl + S |
| Quit | Ctrl + Q |
| Undo | Ctrl + U |
| Redo | Ctrl + Y |
| Cut | Ctrl + X |
| Copy | Ctrl + C |
| Paste | Ctrl + V |

### Navigation

| Action | Key |
| --- | --- |
| Find | Ctrl + F |
| Go to Line | Ctrl + T |
| Toggle Line Numbers | Ctrl + L |
| Move to Doc Start | Ctrl + Home |
| Move to Doc End | Ctrl + End |
| Move to Line Start | Home |
| Move to Line End | End |
| Page Up | Page Up |
| Page Down | Page Down |

### Selection

| Action | Key |
| --- | --- |
| Select All | Ctrl + A |
| Select | Shift + Arrow Keys |

## License

This project is licensed under the [LICENSE\_NAME] License.