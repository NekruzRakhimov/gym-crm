@echo off
echo ============================================
echo  Gym CRM - Starting...
echo ============================================

echo.
echo Starting backend...
start "GymCRM Backend" /d "%~dp0gym-crm-back" /min gym-crm.exe

echo.
echo Waiting for server to start...
timeout /t 3 /nobreak >nul

echo Opening browser...
start http://localhost:8080

echo.
echo Gym CRM is running!
echo URL: http://localhost:8080
echo.
echo To stop, run stop.bat
timeout /t 3 /nobreak >nul
