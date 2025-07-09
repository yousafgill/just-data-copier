# Branch Office File Transfer Over Internet

This example shows you how to reliably send files from your branch office to the main office over an internet connection, optimized for Windows environments with common file types like ZIP archives, RAR files, and video content.

## 📊 What We're Doing

**Situation**: You need to send daily files from a branch office to your main data center over the internet
- **File Types**: ZIP archives, RAR files, video files (MP4, AVI, MKV), and various media content
- **File Sizes**: Few MBs to several GBs, sometimes TB-sized video collections
- **Network**: Regular internet connection, maybe 50-200 Mbps, speed can vary throughout the day
- **Challenges**: Internet can be slow or unreliable, connection might drop during large transfers
- **Must Have**: Files must get through even if the connection is bad, and resume if interrupted

## 🎯 Best Settings for Internet Transfers

### Main Office Server Setup (Where Files Go)
```cmd
rem Set up server at main office to receive branch files
jdc.exe -server ^
    -listen 0.0.0.0:8000 ^
    -output "D:\BranchFiles" ^
    -workers 4 ^
    -buffer 524288 ^
    -timeout 8h ^
    -retries 10 ^
    -verify ^
    -log-level info
```

### Branch Office Setup (Sending Files)
```cmd
rem Send files from branch office with internet-friendly settings
jdc.exe -file "C:\Files\project_archive_20250705.zip" ^
    -connect main-office.company.com:8000 ^
    -chunk 2097152 ^
    -buffer 524288 ^
    -workers 3 ^
    -adaptive ^
    -compress=false ^
    -verify ^
    -timeout 8h ^
    -retries 15 ^
    -min-delay 5ms ^
    -max-delay 500ms
```

## 🔧 Why These Settings Work for Internet

### Internet-Friendly Settings
- **Chunk Size**: `2MB` - Not too big, not too small for internet
- **Buffer Size**: `512KB` - Good for most internet connections
- **Workers**: `3` - Won't overwhelm your internet connection
- **Adaptive Mode**: `on` - Very important! Adjusts to your internet speed

### Extra Safety for Unreliable Internet
- **Min/Max Delay**: `5ms-500ms` - Gives room to adjust for network changes
- **Retries**: `15` - More retries because internet can be flaky
- **Timeout**: `8h` - Long timeout for large video files and overnight transfers
- **Compression**: `false` - ZIP/RAR files are already compressed, videos don't compress well
- **Verify**: Enabled on both client and server - Always verify file integrity after transfer

**Note**: Hash verification requires both server and client to have `-verify` flag enabled.

## 📋 Complete Branch Office Script

## 📋 Windows File Transfer Scripts

### Smart File Transfer Script
```batch
@echo off
setlocal enabledelayedexpansion

rem branch-file-transfer.bat
rem Smart file transfer for common Windows file types

rem Setup
set BRANCH_NAME=%COMPUTERNAME%
set SOURCE_DIR=C:\Files
set MAIN_OFFICE=main-office.company.com:8000
set LOG_FILE=C:\Logs\branch-transfer.log
set TODAY=%date:~10,4%%date:~4,2%%date:~7,2%
set MAX_ATTEMPTS=3

rem Supported file extensions
set EXTENSIONS=*.zip *.rar *.7z *.mp4 *.avi *.mkv *.mov *.wmv *.flv *.webm *.m4v

rem Function to write log messages
:log
echo [%date% %time%] [%BRANCH_NAME%] %~1 >> "%LOG_FILE%"
echo [%date% %time%] [%BRANCH_NAME%] %~1
goto :eof

rem Function to test internet connection and get optimal settings
:test_connection
call :log "Testing internet connection..."

rem Test if we can reach main office
ping -n 3 main-office.company.com >nul 2>&1
if !errorlevel! neq 0 (
    call :log "ERROR: Cannot reach main office server"
    exit /b 1
)

rem Get connection quality and set transfer parameters
for /f "tokens=4 delims== " %%i in ('ping -n 3 main-office.company.com ^| find "Average"') do (
    set avg_ping=%%i
    set avg_ping=!avg_ping:ms=!
)

rem Set optimal settings based on ping time
if !avg_ping! gtr 100 (
    call :log "High latency connection detected (!avg_ping!ms) - using conservative settings"
    set CHUNK_SIZE=1048576
    set WORKERS=2
    set MAX_DELAY=1000ms
) else if !avg_ping! gtr 50 (
    call :log "Medium latency connection detected (!avg_ping!ms) - using balanced settings"
    set CHUNK_SIZE=2097152
    set WORKERS=3
    set MAX_DELAY=500ms
) else (
    call :log "Good connection detected (!avg_ping!ms) - using optimal settings"
    set CHUNK_SIZE=4194304
    set WORKERS=4
    set MAX_DELAY=200ms
)

exit /b 0

rem Function to get file size category and adjust settings
:get_file_settings
set file_path=%~1
for %%f in ("%file_path%") do set file_size=%%~zf

rem Adjust settings based on file size
if !file_size! gtr 10737418240 (
    rem Files larger than 10GB - use large chunk settings
    set CHUNK_SIZE=8388608
    set WORKERS=4
    set TIMEOUT=12h
    call :log "Large file detected (>10GB) - using large transfer settings"
) else if !file_size! gtr 1073741824 (
    rem Files larger than 1GB - use medium settings
    set CHUNK_SIZE=4194304
    set WORKERS=3
    set TIMEOUT=6h
    call :log "Medium file detected (>1GB) - using medium transfer settings"
) else (
    rem Small files - use standard settings
    set CHUNK_SIZE=2097152
    set WORKERS=2
    set TIMEOUT=2h
    call :log "Small file detected (<1GB) - using standard transfer settings"
)

exit /b 0

rem Function to transfer file with retries
:transfer_file
set file_path=%~1
set attempt=1

:retry_transfer
call :log "Transfer attempt !attempt! for %~nx1"

rem Get optimal settings for this file size
call :get_file_settings "%file_path%"

jdc.exe -file "%file_path%" ^
        -connect %MAIN_OFFICE% ^
        -chunk !CHUNK_SIZE! ^
        -buffer 524288 ^
        -workers !WORKERS! ^
        -adaptive ^
        -verify ^
        -timeout !TIMEOUT! ^
        -retries 15 ^
        -min-delay 5ms ^
        -max-delay !MAX_DELAY! ^
        -log-level info

if !errorlevel! equ 0 (
    call :log "SUCCESS: Transfer completed for %~nx1"
    exit /b 0
) else (
    call :log "FAILED: Transfer attempt !attempt! failed"
    set /a attempt+=1
    
    if !attempt! leq %MAX_ATTEMPTS% (
        set /a wait_time=!attempt! * 90
        call :log "Waiting !wait_time! seconds before retry..."
        timeout /t !wait_time! /nobreak >nul
        goto :retry_transfer
    )
)

call :log "ERROR: All %MAX_ATTEMPTS% attempts failed for %~nx1"
exit /b 1

rem Function to validate file before transfer
:validate_file
set file_path=%~1

call :log "Validating file: %~nx1"

rem Check if file exists and isn't empty
if not exist "%file_path%" (
    call :log "ERROR: File not found: %file_path%"
    exit /b 1
)

rem Check file size (should be at least 1KB)
for %%f in ("%file_path%") do set file_size=%%~zf
if !file_size! lss 1024 (
    call :log "WARNING: File seems very small: !file_size! bytes"
)

rem Test archive files for corruption
echo %file_path% | find /i ".zip" >nul
if !errorlevel! equ 0 (
    powershell -command "try { Add-Type -AssemblyName System.IO.Compression.FileSystem; [System.IO.Compression.ZipFile]::OpenRead('%file_path%').Dispose(); exit 0 } catch { exit 1 }"
    if !errorlevel! neq 0 (
        call :log "ERROR: ZIP file appears to be corrupted"
        exit /b 1
    )
)

echo %file_path% | find /i ".rar" >nul
if !errorlevel! equ 0 (
    "C:\Program Files\WinRAR\WinRAR.exe" t "%file_path%" >nul 2>&1
    if !errorlevel! neq 0 (
        call :log "WARNING: Could not verify RAR file (WinRAR not found or file corrupted)"
    )
)

call :log "File validation passed"
exit /b 0

rem Main program starts here
call :log "=== Branch Office File Transfer Started ==="
call :log "Branch: %BRANCH_NAME%"
call :log "Date: %TODAY%"

rem Clean up old temporary files
del /q "%TEMP%\*.justdatacopier.state" 2>nul

rem Test internet connection and set optimal settings
call :test_connection
if !errorlevel! neq 0 (
    call :log "Connection test failed, stopping transfer"
    exit /b 1
)

rem Find files to transfer
set file_count=0
set transfer_queue=

rem Search for supported file types
for %%e in (%EXTENSIONS%) do (
    for %%f in ("%SOURCE_DIR%\%%e") do (
        set transfer_queue=!transfer_queue! "%%f"
        set /a file_count+=1
    )
)

rem Also look for today's files specifically
for %%e in (%EXTENSIONS%) do (
    for %%f in ("%SOURCE_DIR%\*%TODAY%*%%e") do (
        echo !transfer_queue! | find /i "%%f" >nul
        if !errorlevel! neq 0 (
            set transfer_queue=!transfer_queue! "%%f"
            set /a file_count+=1
        )
    )
)

if !file_count! equ 0 (
    call :log "No files found to transfer"
    exit /b 0
)

call :log "Found !file_count! files to transfer"

rem Transfer each file
set failed_count=0
set success_count=0

for %%f in (%transfer_queue%) do (
    call :validate_file %%f
    if !errorlevel! equ 0 (
        call :transfer_file %%f
        if !errorlevel! equ 0 (
            set /a success_count+=1
        ) else (
            set /a failed_count+=1
        )
    ) else (
        call :log "Skipping invalid file: %%~nxf"
        set /a failed_count+=1
    )
)

rem Summary
call :log "=== Transfer Summary ==="
call :log "Total files: !file_count!"
call :log "Successful: !success_count!"
call :log "Failed: !failed_count!"

if !failed_count! equ 0 (
    call :log "=== All transfers completed successfully ==="
    exit /b 0
) else (
    call :log "=== Some transfers failed ==="
    exit /b 1
)
```

### Simple Version (For Basic Use)
```batch
@echo off
rem simple-file-transfer.bat
rem Basic version for transferring common file types

set SOURCE_DIR=C:\Files
set MAIN_OFFICE=main-office.company.com:8000
set EXTENSIONS=*.zip *.rar *.mp4 *.avi *.mkv *.mov

echo Starting file transfer...

rem Transfer ZIP and RAR files
for %%e in (*.zip *.rar *.7z) do (
    for %%f in ("%SOURCE_DIR%\%%e") do (
        echo Transferring %%~nxf...
        
        jdc.exe -file "%%f" ^
                -connect %MAIN_OFFICE% ^
                -chunk 2097152 ^
                -workers 3 ^
                -adaptive ^
                -verify ^
                -timeout 6h ^
                -retries 10
        
        if !errorlevel! equ 0 (
            echo SUCCESS: %%~nxf transferred
        ) else (
            echo FAILED: %%~nxf transfer failed
        )
    )
)

rem Transfer video files with settings optimized for large files
for %%e in (*.mp4 *.avi *.mkv *.mov *.wmv *.flv) do (
    for %%f in ("%SOURCE_DIR%\%%e") do (
        echo Transferring video: %%~nxf...
        
        jdc.exe -file "%%f" ^
                -connect %MAIN_OFFICE% ^
                -chunk 8388608 ^
                -workers 2 ^
                -adaptive ^
                -verify ^
                -timeout 12h ^
                -retries 8
        
        if !errorlevel! equ 0 (
            echo SUCCESS: %%~nxf transferred
        ) else (
            echo FAILED: %%~nxf transfer failed
        )
    )
)

echo Transfer batch complete
pause
```

## 📊 Monitoring Your File Transfers

### Check if Branch Offices are Sending Files
```batch
@echo off
rem monitor-branch-transfers.bat
rem Run this at main office to check if branches are sending files

set BRANCH_FILES_DIR=D:\BranchFiles
set TODAY=%date:~10,4%%date:~4,2%%date:~7,2%
set EXPECTED_BRANCHES=Branch1 Branch2 Branch3 Branch4

echo Checking branch file transfers for %TODAY%...
echo.

for %%b in (%EXPECTED_BRANCHES%) do (
    set "found_files="
    
    rem Check for any file types from this branch today
    for %%f in ("%BRANCH_FILES_DIR%\%%b*%TODAY%*.*" "%BRANCH_FILES_DIR%\*%%b*.*") do (
        set "found_files=1"
        goto :found_%%b
    )
    
    echo ✗ %%b: No files received today
    goto :next_%%b
    
    :found_%%b
    echo ✓ %%b: Files received
    
    :next_%%b
)

echo.
echo Check complete
pause
```

## 📈 File Type Specific Optimizations

### Optimized Settings by File Type

| File Type | Chunk Size | Workers | Timeout | Notes |
|-----------|------------|---------|---------|--------|
| **ZIP/RAR Archives** | 4MB | 3 | 6h | Good compression ratio, medium priority |
| **Video Files (MP4/AVI)** | 8MB | 2 | 12h | Large files, already compressed |
| **High-Res Videos (4K)** | 16MB | 2 | 24h | Very large files, stable connection needed |
| **Small Media Files** | 2MB | 4 | 2h | Quick transfers, multiple connections |

### Auto-File-Type Detection Script
```batch
rem Add this to your transfer script for automatic optimization

:get_optimal_settings
set file_path=%~1
set file_ext=%~x1

rem Set defaults
set CHUNK_SIZE=2097152
set WORKERS=3
set TIMEOUT=6h

rem Optimize based on file extension
if /i "%file_ext%"==".mp4" set CHUNK_SIZE=8388608 & set WORKERS=2 & set TIMEOUT=12h
if /i "%file_ext%"==".avi" set CHUNK_SIZE=8388608 & set WORKERS=2 & set TIMEOUT=12h
if /i "%file_ext%"==".mkv" set CHUNK_SIZE=8388608 & set WORKERS=2 & set TIMEOUT=12h
if /i "%file_ext%"==".mov" set CHUNK_SIZE=8388608 & set WORKERS=2 & set TIMEOUT=12h
if /i "%file_ext%"==".wmv" set CHUNK_SIZE=8388608 & set WORKERS=2 & set TIMEOUT=12h

if /i "%file_ext%"==".zip" set CHUNK_SIZE=4194304 & set WORKERS=3 & set TIMEOUT=6h
if /i "%file_ext%"==".rar" set CHUNK_SIZE=4194304 & set WORKERS=3 & set TIMEOUT=6h
if /i "%file_ext%"==".7z" set CHUNK_SIZE=4194304 & set WORKERS=3 & set TIMEOUT=6h

rem Check file size for fine-tuning
for %%f in ("%file_path%") do set file_size=%%~zf
if !file_size! gtr 5368709120 (
    rem Files > 5GB get bigger chunks and longer timeout
    set CHUNK_SIZE=16777216
    set TIMEOUT=24h
)

exit /b 0
```

## 🚨 Common Problems

### Internet Connection Keeps Dropping
```cmd
rem Use these settings for really bad internet with large video files
jdc.exe -file big_video.mp4 ^
        -connect server:8000 ^
        -chunk 1048576 ^
        -workers 1 ^
        -adaptive ^
        -retries 25 ^
        -timeout 24h
```

### Transfers are Too Slow
```cmd
rem Check if other programs are using your internet
netstat -b

rem Try reducing workers for video files
jdc.exe -file large_video.mp4 -connect server:8000 -workers 1

rem Or schedule transfers during off-peak hours (like 2 AM)
rem Use Task Scheduler for this
```

### Files Keep Getting Corrupted
```cmd
rem Always verify your files first
"C:\Program Files\7-Zip\7z.exe" t archive.zip
"C:\Program Files\WinRAR\WinRAR.exe" t archive.rar

rem For video files, check if they play properly
rem Use MediaInfo tool: mediainfo.exe video.mp4

rem Use smaller chunks for bad connections
jdc.exe -file large_video.mp4 -connect server:8000 -chunk 2097152
```

## 🔐 Security for Internet Transfers

### Use VPN When Possible
```batch
rem Check if VPN is connected before transferring sensitive files
ping internal-vpn.company.com >nul 2>&1
if errorlevel 1 (
    echo VPN not connected! Connect VPN first for security.
    echo Connect to company VPN and try again.
    pause
    exit /b 1
)

rem Then do your transfer to internal address
jdc.exe -file sensitive_archive.zip -connect internal-fileserver.company.local:8000
```

## 📅 Schedule with Windows Task Scheduler

### Setting Up Automatic Daily Transfers
1. Open **Task Scheduler** (type it in Start menu)
2. Click **Create Basic Task**
3. Name it "Branch Office File Transfer"
4. Set trigger to **Daily** at **2:00 AM** (when internet is usually less busy)
5. Set action to **Start a program**: `C:\Scripts\branch-file-transfer.bat`
6. Check **Run whether user is logged on or not**
7. Check **Run with highest privileges**

### Multiple Transfer Schedule
```batch
rem Create separate tasks for different file types

rem Task 1: Small files (ZIP/RAR) - Daily at 1:00 AM
rem C:\Scripts\transfer-archives.bat

rem Task 2: Video files - Daily at 2:00 AM  
rem C:\Scripts\transfer-videos.bat

rem Task 3: Weekly large file cleanup - Sunday at 3:00 AM
rem C:\Scripts\cleanup-transferred-files.bat
```

### Advanced PowerShell Version
```powershell
# branch-file-transfer.ps1
# Advanced file transfer with intelligent file type handling

param(
    [string]$SourceDir = "C:\Files",
    [string]$MainOffice = "main-office.company.com:8000",
    [int]$MaxRetries = 3,
    [string[]]$FileTypes = @("*.zip", "*.rar", "*.7z", "*.mp4", "*.avi", "*.mkv", "*.mov", "*.wmv")
)

$LogFile = "C:\Logs\branch-file-transfer-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"

function Write-Log($Message) {
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logEntry = "[$timestamp] $Message"
    Write-Host $logEntry
    Add-Content -Path $LogFile -Value $logEntry
}

function Test-InternetConnection {
    Write-Log "Testing connection to main office..."
    try {
        $ping = Test-NetConnection -ComputerName "main-office.company.com" -Port 8000 -InformationLevel Quiet
        if ($ping) {
            Write-Log "Connection test passed"
            return $true
        } else {
            Write-Log "ERROR: Cannot reach main office on port 8000"
            return $false
        }
    } catch {
        Write-Log "ERROR: Connection test failed - $($_.Exception.Message)"
        return $false
    }
}

function Get-OptimalSettings($FilePath) {
    $fileInfo = Get-Item $FilePath
    $fileSize = $fileInfo.Length
    $extension = $fileInfo.Extension.ToLower()
    
    # Default settings
    $settings = @{
        ChunkSize = 2097152
        Workers = 3
        Timeout = "6h"
    }
    
    # Optimize based on file type
    switch ($extension) {
        {$_ -in @(".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm")} {
            $settings.ChunkSize = 8388608
            $settings.Workers = 2
            $settings.Timeout = "12h"
            Write-Log "Video file detected - using video optimized settings"
        }
        {$_ -in @(".zip", ".rar", ".7z")} {
            $settings.ChunkSize = 4194304
            $settings.Workers = 3
            $settings.Timeout = "6h"
            Write-Log "Archive file detected - using archive optimized settings"
        }
    }
    
    # Adjust for file size
    if ($fileSize -gt 5GB) {
        $settings.ChunkSize = 16777216
        $settings.Timeout = "24h"
        $settings.Workers = 2
        Write-Log "Large file (>5GB) detected - using large file settings"
    } elseif ($fileSize -gt 1GB) {
        $settings.ChunkSize = 8388608
        $settings.Timeout = "12h"
        Write-Log "Medium file (>1GB) detected - using medium file settings"
    }
    
    return $settings
}

function Transfer-File($FilePath) {
    $fileName = Split-Path $FilePath -Leaf
    Write-Log "Starting transfer: $fileName ($(([math]::Round((Get-Item $FilePath).Length/1MB, 2))) MB)"
    
    $settings = Get-OptimalSettings $FilePath
    
    $args = @(
        "-file", $FilePath,
        "-connect", $MainOffice,
        "-chunk", $settings.ChunkSize,
        "-workers", $settings.Workers,
        "-adaptive",
        "-verify",
        "-timeout", $settings.Timeout,
        "-retries", "15",
        "-log-level", "info"
    )
    
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    $process = Start-Process -FilePath "jdc.exe" -ArgumentList $args -Wait -PassThru -NoNewWindow
    $stopwatch.Stop()
    
    if ($process.ExitCode -eq 0) {
        $speed = [math]::Round(((Get-Item $FilePath).Length / $stopwatch.Elapsed.TotalSeconds) / 1MB, 2)
        Write-Log "SUCCESS: $fileName transferred in $($stopwatch.Elapsed.ToString('hh\:mm\:ss')) at $speed MB/s"
        return $true
    } else {
        Write-Log "FAILED: $fileName (Exit code: $($process.ExitCode))"
        return $false
    }
}

function Test-FileIntegrity($FilePath) {
    $extension = [System.IO.Path]::GetExtension($FilePath).ToLower()
    
    switch ($extension) {
        ".zip" {
            try {
                Add-Type -AssemblyName System.IO.Compression.FileSystem
                $zip = [System.IO.Compression.ZipFile]::OpenRead($FilePath)
                $zip.Dispose()
                return $true
            } catch {
                Write-Log "WARNING: ZIP file integrity check failed for $(Split-Path $FilePath -Leaf)"
                return $false
            }
        }
        ".rar" {
            if (Test-Path "C:\Program Files\WinRAR\WinRAR.exe") {
                $result = & "C:\Program Files\WinRAR\WinRAR.exe" t $FilePath 2>$null
                return $LASTEXITCODE -eq 0
            } else {
                Write-Log "WARNING: WinRAR not found, skipping RAR integrity check"
                return $true
            }
        }
        default {
            # For video files and others, just check if file exists and has size
            return (Test-Path $FilePath) -and ((Get-Item $FilePath).Length -gt 0)
        }
    }
}

# Main script execution
Write-Log "=== Branch Office File Transfer Started ==="

if (-not (Test-InternetConnection)) {
    Write-Log "Internet connection failed, aborting"
    exit 1
}

# Find files to transfer
$allFiles = @()
foreach ($fileType in $FileTypes) {
    $files = Get-ChildItem -Path $SourceDir -Filter $fileType -File
    $allFiles += $files
}

if ($allFiles.Count -eq 0) {
    Write-Log "No files found to transfer"
    exit 0
}

Write-Log "Found $($allFiles.Count) files to transfer"

$successCount = 0
$failCount = 0

foreach ($file in $allFiles) {
    Write-Log "Processing: $($file.Name)"
    
    if (Test-FileIntegrity $file.FullName) {
        if (Transfer-File $file.FullName) {
            $successCount++
        } else {
            $failCount++
        }
    } else {
        Write-Log "Skipping corrupted file: $($file.Name)"
        $failCount++
    }
}

Write-Log "=== Transfer Summary ==="
Write-Log "Total files: $($allFiles.Count)"
Write-Log "Successful: $successCount"
Write-Log "Failed: $failCount"
Write-Log "Success rate: $(if ($allFiles.Count -gt 0) { [math]::Round(($successCount / $allFiles.Count) * 100, 1) } else { 0 })%"

if ($failCount -eq 0) {
    Write-Log "All transfers completed successfully"
    exit 0
} else {
    Write-Log "Some transfers failed"
    exit 1
}
```

## 📖 Command Reference for Branch Office Transfers

### Main Office Server (Receiver)
```cmd
rem Basic setup for receiving branch office files
jdc.exe -server -output "D:\BranchFiles" -verify

rem Full command with all options for internet transfers:
jdc.exe -server ^
    -listen <ip:port>          rem Default: 0.0.0.0:8000
    -output <directory>        rem Where to store received files
    -verify                    rem Enable hash verification (recommended)
    -workers <number>          rem Default: half CPU cores (use 4 for internet)
    -buffer <bytes>            rem Default: 512KB (good for internet)
    -timeout <duration>        rem Default: 2m (use 8h for large files over internet)
    -retries <number>          rem Default: 5 (use 10+ for internet)
    -progress                  rem Show progress (default: true)
```

### Branch Office Client (Sender)
```cmd
rem Basic file transfer over internet
jdc.exe -file "archive.zip" -connect main-office.company.com:8000 -verify -adaptive

rem Full command with all options for internet transfers:
jdc.exe -file <file_path> ^
    -connect <server:port>     rem Main office server address
    -verify                    rem Enable hash verification (recommended)
    -chunk <bytes>             rem Default: 2MB (good for internet, use 1MB for slow connections)
    -compress                  rem Enable compression (false for ZIP/RAR/videos)
    -workers <number>          rem Default: half CPU cores (use 2-3 for internet)
    -buffer <bytes>            rem Default: 512KB (good for internet)
    -timeout <duration>        rem Default: 2m (use 8h+ for large files over internet)
    -retries <number>          rem Default: 5 (use 15+ for internet)
    -progress                  rem Show progress (default: true)
    -adaptive                  rem Enable adaptive delays (highly recommended for internet)
    -delay <duration>          rem Chunk delay (default: 10ms)
    -min-delay <duration>      rem Minimum adaptive delay (default: 1ms, use 5ms for internet)
    -max-delay <duration>      rem Maximum adaptive delay (default: 100ms, use 500ms+ for internet)
```

### File Type Optimizations
```cmd
rem For ZIP/RAR archives (already compressed):
jdc.exe -file "archive.zip" -connect server:8000 -verify -chunk 4194304 -workers 3 -adaptive

rem For video files (large, already compressed):
jdc.exe -file "video.mp4" -connect server:8000 -verify -chunk 8388608 -workers 2 -timeout 12h -adaptive

rem For small files (documents, etc.):
jdc.exe -file "document.pdf" -connect server:8000 -verify -chunk 2097152 -workers 4 -adaptive
````
