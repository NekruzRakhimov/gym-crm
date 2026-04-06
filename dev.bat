@echo off
echo ============================================
echo  Gym CRM - Dev Mode
echo ============================================

echo.
echo Starting backend...
start "GymCRM Backend" cmd /k "cd /d "%~dp0gym-crm-back" && go run .\cmd\server\main.go"

echo.
echo Starting frontend...
start "GymCRM Frontend" cmd /k "cd /d "%~dp0gym-crm-front" && npm run dev"

echo.
echo ============================================
echo  Dev servers running:
echo  Backend:  http://localhost:8080
echo  Frontend: http://localhost:5173
echo ============================================
