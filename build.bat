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
echo [3/3] Adding firewall rule for port 8080...
netsh advfirewall firewall delete rule name="GymCRM" >nul 2>&1
netsh advfirewall firewall add rule name="GymCRM" dir=in action=allow protocol=TCP localport=8080 >nul 2>&1
if %errorlevel% equ 0 (
    echo Firewall rule added.
) else (
    echo WARNING: Could not add firewall rule. Run build.bat as Administrator.
)

echo.
echo ============================================
echo  Build complete! You can now run start.bat
echo ============================================
pause
