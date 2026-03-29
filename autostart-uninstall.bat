@echo off
echo ============================================
echo  Gym CRM - Removing Autostart
echo ============================================

:: Check for admin rights
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Run this script as Administrator!
    pause
    exit /b 1
)

set TASK_NAME=GymCRM

echo.
echo Removing task: %TASK_NAME%

schtasks /delete /tn "%TASK_NAME%" /f

if %errorlevel% equ 0 (
    echo.
    echo [OK] Autostart removed.
) else (
    echo.
    echo Task was not found or already removed.
)

echo.
pause
