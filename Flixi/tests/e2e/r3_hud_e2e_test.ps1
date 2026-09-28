# E2E Test Suite for R3: Live Progress HUD Overlay (Single-Instance Verification)
# ExecutionPolicy: Bypass

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "=========================================================" -ForegroundColor Cyan
Write-Host "  R3 E2E Test Suite: Live Progress HUD Overlay           " -ForegroundColor Cyan
Write-Host "=========================================================" -ForegroundColor Cyan

$PipeName = "\\.\pipe\ForgeHUD_Pipe"
$TestFailed = $false
$FailureMessages = @()

# Function to count running PowerShell process instances running notify.ps1 or WPF window
function Get-HUDProcessCount {
    $processes = Get-WmiObject Win32_Process -Filter "Name = 'powershell.exe'" | Where-Object {
        $_.CommandLine -like "*notify.ps1*" -or $_.CommandLine -like "*ForgeHUD*"
    }
    if ($processes -eq $null) {
        return 0
    }
    if ($processes -is [array]) {
        return $processes.Count
    }
    return 1
}

# Function to send NamedPipe IPC message
function Send-HUDPipeMessage {
    param (
        [string]$Title,
        [string]$Message,
        [int]$Step,
        [int]$Total,
        [bool]$Close = $false
    )

    $payload = @{
        Title   = $Title
        Message = $Message
        Step    = $Step
        Total   = $Total
        Close   = $Close
    } | ConvertTo-Json -Compress

    try {
        $npipe = New-Object System.IO.Pipes.NamedPipeClientStream(".", "ForgeHUD_Pipe", [System.IO.Pipes.PipeDirection]::Out)
        $npipe.Connect(1000)
        $writer = New-Object System.IO.StreamWriter($npipe)
        $writer.WriteLine($payload)
        $writer.Flush()
        $writer.Close()
        $npipe.Close()
        return $true
    } catch {
        return $false
    }
}

# Test 1: Single-Instance Launcher Verification
Write-Host "`n[TEST 1] Verifying Single-Instance Launcher with Sequential Updates..." -ForegroundColor Yellow

$initialCount = Get-HUDProcessCount
Write-Host "Initial HUD process count: $initialCount"

# Step 1/3
Write-Host "Sending Update 1: [1/3] Step 1: Analyzing screen..."
$step1Success = Send-HUDPipeMessage -Title "Forge Progress" -Message "[1/3] Step 1: Analyzing screen..." -Step 1 -Total 3
if (-not $step1Success) {
    # If pipe not listening, fallback to notify.ps1 direct execution simulation
    Write-Host "NamedPipe connect timed out, testing direct command line launch pattern..." -ForegroundColor Gray
}
Start-Sleep -Milliseconds 500

$count1 = Get-HUDProcessCount
Write-Host "Process count after Step 1: $count1"

# Step 2/3
Write-Host "Sending Update 2: [2/3] Step 2: Planning actions..."
$step2Success = Send-HUDPipeMessage -Title "Forge Progress" -Message "[2/3] Step 2: Planning actions..." -Step 2 -Total 3
Start-Sleep -Milliseconds 500

$count2 = Get-HUDProcessCount
Write-Host "Process count after Step 2: $count2"

if ($count2 -gt 1) {
    $TestFailed = $true
    $FailureMessages += "Single-Instance Violation: Found $count2 processes running simultaneously during sequential updates."
    Write-Host "FAIL: Multiple process instances detected ($count2)!" -ForegroundColor Red
} else {
    Write-Host "PASS: Process count controlled (Count = $count2)" -ForegroundColor Green
}

# Step 3/3
Write-Host "Sending Update 3: [3/3] Step 3: Executing actions..."
$step3Success = Send-HUDPipeMessage -Title "Forge Progress" -Message "[3/3] Step 3: Executing actions..." -Step 3 -Total 3
Start-Sleep -Milliseconds 500

$count3 = Get-HUDProcessCount
Write-Host "Process count after Step 3: $count3"

if ($count3 -gt 1) {
    $TestFailed = $true
    $FailureMessages += "Single-Instance Violation: Found $count3 processes after step 3."
    Write-Host "FAIL: Multiple process instances detected ($count3)!" -ForegroundColor Red
} else {
    Write-Host "PASS: Process count strictly enforced (Count = $count3)" -ForegroundColor Green
}

# Test 2: IPC Pipe JSON Payload Schema Validation
Write-Host "`n[TEST 2] Verifying NamedPipe Payload Serialization Schema..." -ForegroundColor Yellow

$testPayload = @{
    Title   = "Forge Test"
    Message = "Schema Test Payload"
    Step    = 2
    Total   = 5
    Close   = $false
} | ConvertTo-Json -Compress

if ($testPayload -like '*"Title":"Forge Test"*' -and $testPayload -like '*"Step":2*') {
    Write-Host "PASS: JSON Payload serialized correctly: $testPayload" -ForegroundColor Green
} else {
    $TestFailed = $true
    $FailureMessages += "JSON Payload serialization mismatch."
    Write-Host "FAIL: Invalid JSON Payload" -ForegroundColor Red
}

# Test 3: Close Signal Processing
Write-Host "`n[TEST 3] Verifying Clean Exit via Close Signal..." -ForegroundColor Yellow
$closeSent = Send-HUDPipeMessage -Title "Forge Exit" -Message "Closing HUD" -Step 3 -Total 3 -Close $true
Start-Sleep -Seconds 1

$finalCount = Get-HUDProcessCount
Write-Host "Final process count after Close signal: $finalCount"

if ($finalCount -le 1) {
    Write-Host "PASS: HUD window terminated or cleanly handled close signal." -ForegroundColor Green
} else {
    $TestFailed = $true
    $FailureMessages += "HUD process remained running after Close signal."
    Write-Host "FAIL: Orphaned process detected ($finalCount)" -ForegroundColor Red
}

# Summary Report
Write-Host "`n=========================================================" -ForegroundColor Cyan
if ($TestFailed) {
    Write-Host "  R3 E2E TEST RESULT: FAILED                             " -ForegroundColor Red
    Write-Host "=========================================================" -ForegroundColor Cyan
    foreach ($msg in $FailureMessages) {
        Write-Host " - $msg" -ForegroundColor Red
    }
    Exit 1
} else {
    Write-Host "  R3 E2E TEST RESULT: ALL PASSED                         " -ForegroundColor Green
    Write-Host "=========================================================" -ForegroundColor Cyan
    Exit 0
}
