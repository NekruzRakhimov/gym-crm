@echo off
echo ============================================
echo  Gym CRM - Build
echo ============================================

echo.
echo [1/2] Building frontend...
cd /d "%~dp0gym-crm-front"
call npm install
call npm run build
if %errorlevel% neq 0 (
    echo ERROR: Frontend build failed!
    pause
    exit /b 1
)
echo Frontend built successfully.

echo.
echo [2/2] Building backend...
cd /d "%~dp0gym-crm-back"
go build -o gym-crm.exe ./cmd/server
if %errorlevel% neq 0 (
    echo ERROR: Backend build failed!
    pause
    exit /b 1
)
echo Backend built successfully.

echo.
echo ============================================
echo  Build complete! You can now run start.bat
echo ============================================
pause
