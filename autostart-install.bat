@echo off
echo ============================================
echo  Gym CRM - Installing Autostart
echo ============================================

:: Check for admin rights
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Run this script as Administrator!
    pause
    exit /b 1
)

set SCRIPT_DIR=%~dp0
set VBS_PATH=%SCRIPT_DIR%start-silent.vbs
set TASK_NAME=GymCRM

echo.
echo Registering task: %TASK_NAME%
echo Script: %VBS_PATH%
echo.

schtasks /create /tn "%TASK_NAME%" /tr "wscript.exe \"%VBS_PATH%\"" /sc onlogon /rl highest /f

if %errorlevel% equ 0 (
    echo.
    echo [OK] Autostart installed successfully!
    echo Gym CRM will start automatically on next login.
    echo.
    echo To remove autostart, run autostart-uninstall.bat
) else (
    echo.
    echo ERROR: Failed to register task.
)

echo.
pause
