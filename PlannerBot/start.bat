@echo off
setlocal enabledelayedexpansion

title PlannerBot - Sovereign Cognitive Engine
echo ======================================================================
echo    PlannerBot - Sovereign Cognitive Engine (v0.3.0)
echo    100% Offline, Air-Gapped AI Daily Planner & Focus Coach
echo ======================================================================
echo.

set MODEL_DIR=%LOCALAPPDATA%\planner_bot\models
set MODEL_FILE=%MODEL_DIR%\qwen2.5-0.5b-instruct-q4_k_m.gguf
set MODEL_URL=https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf

if not exist "%MODEL_DIR%" (
    echo [*] Creating model directory at "%MODEL_DIR%"...
    mkdir "%MODEL_DIR%"
)

if not exist "%MODEL_FILE%" (
    echo [*] Model file missing. Downloading Qwen2.5-0.5B Instruct GGUF (~398MB)...
    echo [*] Downloading from Hugging Face...
    curl.exe -L -o "%MODEL_FILE%" "%MODEL_URL%"
    if errorlevel 1 (
        echo [!] Curl download failed. Retrying with PowerShell...
        powershell -Command "Invoke-WebRequest -Uri '%MODEL_URL%' -OutFile '%MODEL_FILE%'"
    )
)

if not exist "PlannerBot.exe" (
    where go >nul 2>nul
    if not errorlevel 1 (
        echo [*] Go compiler detected. Building PlannerBot binary...
        go build -o PlannerBot.exe ./cmd/planner
    ) else (
        echo [*] PlannerBot.exe missing and Go not installed.
        echo [*] Downloading pre-compiled PlannerBot.exe from GitHub Releases...
        set EXE_URL=https://github.com/Anikesh0415/PlannerBot/releases/download/v0.3.0/PlannerBot.exe
        curl.exe -L -o "PlannerBot.exe" "!EXE_URL!"
        if errorlevel 1 (
            powershell -Command "Invoke-WebRequest -Uri '!EXE_URL!' -OutFile 'PlannerBot.exe'"
        )
    )
    if not exist "PlannerBot.exe" (
        echo [!] Error: Could not obtain PlannerBot.exe.
        echo [!] You can manually download it from: https://github.com/Anikesh0415/PlannerBot/releases
        pause
        exit /b 1
    )
)

echo [*] Stopping any existing instances on port 8088...
taskkill /F /IM PlannerBot.exe >nul 2>nul
taskkill /F /IM planner.exe >nul 2>nul

echo [*] Launching PlannerBot sovereign engine...
start "" "PlannerBot.exe"

echo [*] Waiting for engine to initialize...
set TIMEOUT=30
:WAIT_LOOP
powershell -Command "try { $res = Invoke-RestMethod -Uri 'http://127.0.0.1:8088/api/status' -TimeoutSec 1; if ($res.ai_online) { exit 0 } else { exit 1 } } catch { exit 1 }" >nul 2>nul
if %errorlevel% equ 0 goto LAUNCH_BROWSER

timeout /t 1 /nobreak >nul
set /a TIMEOUT-=1
if %TIMEOUT% gtr 0 goto WAIT_LOOP

echo [!] Engine initialization timed out. Opening browser anyway...

:LAUNCH_BROWSER
echo.
echo ======================================================================
echo    SUCCESS: PlannerBot is online at http://127.0.0.1:8088
echo    All conversation & task data is 100%% private on this device.
echo ======================================================================
echo.
start http://127.0.0.1:8088

pause
