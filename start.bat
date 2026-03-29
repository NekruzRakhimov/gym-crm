@echo off
echo ============================================
echo  Gym CRM - Starting...
echo ============================================

echo.
echo Starting backend...
start "GymCRM Backend" /d "%~dp0gym-crm-back" /min gym-crm.exe

echo Starting frontend...
start "GymCRM Frontend" /d "%~dp0gym-crm-front" /min cmd /c "npm run dev"

echo.
echo Waiting for services to start...
timeout /t 4 /nobreak >nul

echo Opening browser...
start http://localhost:5173

echo.
echo Gym CRM is running!
echo Backend:  http://localhost:8080
echo Frontend: http://localhost:5173
echo.
echo To stop, run stop.bat
timeout /t 3 /nobreak >nul
