@echo off
echo ============================================
echo  Gym CRM - Stopping...
echo ============================================

echo.
echo Stopping backend...
taskkill /f /im gym-crm.exe >nul 2>&1
if %errorlevel% equ 0 (
    echo Backend stopped.
) else (
    echo Backend was not running.
)

echo Stopping frontend...
taskkill /f /im node.exe >nul 2>&1
if %errorlevel% equ 0 (
    echo Frontend stopped.
) else (
    echo Frontend was not running.
)

echo.
echo Gym CRM stopped.
timeout /t 2 /nobreak >nul
