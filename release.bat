@echo off
REM release.bat - Windows Release Script

IF "%~1"=="" (
    echo Usage: release.bat ^<version^>
    echo Example: release.bat v1.0.0
    EXIT /B 1
)

SET VERSION=%~1

REM 1. Check if git is clean (Optional)
git status --porcelain > nul
REM Note: Checking exit code for dirty git in batch is tricky, skipping strict check for simplicity
REM but be aware: you should commit before running this.

REM 2. Create git tag
echo Creating tag %VERSION%...
git tag -a %VERSION% -m "Release %VERSION%"
IF %ERRORLEVEL% NEQ 0 (
    echo Error: Failed to create tag. It might already exist.
    EXIT /B 1
)

REM 3. Build with the tagged version
echo Building binary...
make release TAG=%VERSION%
IF %ERRORLEVEL% NEQ 0 (
    echo Error: Build failed.
    EXIT /B 1
)

echo ---------------------------------------
echo SUCCESS: Release %VERSION% created and built.
echo Don't forget to push the tag: git push origin %VERSION%
echo ---------------------------------------

EXIT /B 0