# Branch Office Backup Over Internet

This example shows you how to reliably send daily backups from your branch office to the main office over an internet connection.

## 📊 What We're Doing

**Situation**: You need to send daily backups from a branch office to your main data center over the internet
- **File Sizes**: Usually 10GB to 200GB compressed backup files
- **Network**: Regular internet connection, maybe 50-200 Mbps, speed can vary throughout the day
- **Challenges**: Internet can be slow or unreliable, connection might drop
- **Must Have**: Files must get through even if the connection is bad, and resume if interrupted

## 🎯 Best Settings for Internet Transfers

### Main Office Server Setup (Where Backups Go)
```cmd
rem Set up server at main office to receive branch backups
jdc.exe -server ^
    -listen 0.0.0.0:8000 ^
    -output "D:\BranchBackups" ^
    -workers 4 ^
    -buffer 524288 ^
    -timeout 8h ^
    -retries 10 ^
    -log-level info
```

### Branch Office Setup (Sending Backups)
```cmd
rem Send backup from branch office with internet-friendly settings
jdc.exe -file "C:\Backup\branch_backup_20250705.zip" ^
    -connect main-office.company.com:8000 ^
    -chunk 2097152 ^
    -buffer 524288 ^
    -workers 3 ^
    -adaptive ^
    -compress=false ^
    -verify=true ^
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
- **Timeout**: `8h` - Long timeout for overnight transfers

## 📋 Complete Branch Office Script

### Smart Branch Office Transfer Script
```batch
@echo off
setlocal enabledelayedexpansion

rem branch-backup-transfer.bat
rem Smart backup transfer that adapts to your internet connection

rem Setup
set BRANCH_NAME=%COMPUTERNAME%
set BACKUP_DIR=C:\Backup
set MAIN_OFFICE=main-office.company.com:8000
set LOG_FILE=C:\Logs\branch-backup.log
set TODAY=%date:~10,4%%date:~4,2%%date:~7,2%
set MAX_ATTEMPTS=3

rem Function to write log messages
:log
echo [%date% %time%] [%BRANCH_NAME%] %~1 >> "%LOG_FILE%"
echo [%date% %time%] [%BRANCH_NAME%] %~1
goto :eof

rem Function to test internet connection
:test_internet
call :log "Testing internet connection..."

rem Test if we can reach main office
ping -n 3 main-office.company.com >nul 2>&1
if !errorlevel! neq 0 (
    call :log "ERROR: Cannot reach main office server"
    exit /b 1
)

rem Get rough ping time to adjust settings
for /f "tokens=4 delims== " %%i in ('ping -n 1 main-office.company.com ^| find "time="') do (
    set ping_info=%%i
)

rem Set transfer settings based on ping
if defined ping_info (
    call :log "Connection looks good - using balanced settings"
    set CHUNK_SIZE=2097152
    set WORKERS=3
    set MAX_DELAY=500ms
) else (
    call :log "Connection might be slow - using conservative settings"
    set CHUNK_SIZE=1048576
    set WORKERS=2
    set MAX_DELAY=1000ms
)

exit /b 0

rem Function to transfer backup file with retries
:transfer_backup
set backup_file=%~1
set attempt=1

:retry_transfer
call :log "Transfer attempt !attempt! for %~nx1"

jdc.exe -file "%backup_file%" ^
        -connect %MAIN_OFFICE% ^
        -chunk !CHUNK_SIZE! ^
        -buffer 524288 ^
        -workers !WORKERS! ^
        -adaptive ^
        -verify=true ^
        -timeout 8h ^
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
        set /a wait_time=!attempt! * 60
        call :log "Waiting !wait_time! seconds before retry..."
        timeout /t !wait_time! /nobreak >nul
        goto :retry_transfer
    )
)

call :log "ERROR: All %MAX_ATTEMPTS% attempts failed for %~nx1"
exit /b 1

rem Function to check if backup file is good
:check_backup
set backup_file=%~1

call :log "Checking backup file: %~nx1"

rem Check if file exists and isn't empty
if not exist "%backup_file%" (
    call :log "ERROR: Backup file not found: %backup_file%"
    exit /b 1
)

rem Check file size (should be at least 1MB)
for %%f in ("%backup_file%") do set file_size=%%~zf
if !file_size! lss 1048576 (
    call :log "WARNING: Backup file seems very small: !file_size! bytes"
)

rem Test zip file if it's a zip
echo %backup_file% | find /i ".zip" >nul
if !errorlevel! equ 0 (
    "C:\Program Files\7-Zip\7z.exe" t "%backup_file%" >nul 2>&1
    if !errorlevel! neq 0 (
        call :log "ERROR: Backup file appears to be corrupted"
        exit /b 1
    )
)

call :log "Backup file looks good"
exit /b 0

rem Main program starts here
call :log "=== Branch Office Backup Transfer Started ==="
call :log "Branch: %BRANCH_NAME%"
call :log "Date: %TODAY%"

rem Clean up old temporary files
del /q "%TEMP%\*.justdatacopier.state" 2>nul

rem Test internet connection and set optimal settings
call :test_internet
if !errorlevel! neq 0 (
    call :log "Internet test failed, stopping transfer"
    exit /b 1
)

rem Find backup files for today
set "backup_files="
set file_count=0
for %%f in ("%BACKUP_DIR%\*%TODAY%*.zip" "%BACKUP_DIR%\*%TODAY%*.7z") do (
    set "backup_files=!backup_files! "%%f""
    set /a file_count+=1
)

if !file_count! equ 0 (
    call :log "No backup files found for today (%TODAY%)"
    exit /b 0
)

call :log "Found !file_count! backup files to transfer"

rem Transfer each backup file
set failed_count=0
set success_count=0

for %%f in (%backup_files%) do (
    call :check_backup "%%~f"
    if !errorlevel! equ 0 (
        call :transfer_backup "%%~f"
        if !errorlevel! equ 0 (
            set /a success_count+=1
        ) else (
            set /a failed_count+=1
        )
    ) else (
        call :log "Skipping bad backup file: %%~nxf"
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
rem simple-branch-backup.bat
rem Basic version for simple setups

set TODAY=%date:~10,4%%date:~4,2%%date:~7,2%
set BACKUP_DIR=C:\Backup
set MAIN_OFFICE=main-office.company.com:8000

echo Starting backup transfer for %TODAY%...

for %%f in ("%BACKUP_DIR%\*%TODAY%*.zip") do (
    echo Transferring %%~nxf...
    
    jdc.exe -file "%%f" ^
            -connect %MAIN_OFFICE% ^
            -chunk 2097152 ^
            -workers 3 ^
            -adaptive ^
            -verify=true ^
            -timeout 8h ^
            -retries 15
    
    if !errorlevel! equ 0 (
        echo SUCCESS: %%~nxf transferred
    ) else (
        echo FAILED: %%~nxf transfer failed
    )
)

echo Transfer batch complete
pause
```

## � Monitoring Your Transfers

### Check if Branch Offices are Sending Backups
```batch
@echo off
rem monitor-branch-transfers.bat
rem Run this at main office to check if branches are sending backups

set BRANCH_BACKUP_DIR=D:\BranchBackups
set TODAY=%date:~10,4%%date:~4,2%%date:~7,2%
set EXPECTED_BRANCHES=Branch1 Branch2 Branch3 Branch4

echo Checking branch backups for %TODAY%...
echo.

for %%b in (%EXPECTED_BRANCHES%) do (
    set "found_backup="
    for %%f in ("%BRANCH_BACKUP_DIR%\%%b*%TODAY%*.*") do set "found_backup=1"
    
    if defined found_backup (
        echo ✓ %%b: Backup received
    ) else (
        echo ✗ %%b: No backup found
        rem You could send an email alert here
    )
)

echo.
echo Check complete
pause
```

## 📈 Making It Work Better

### Auto-Adjust Based on Internet Speed
```batch
rem Add this to your transfer script to auto-adjust settings

rem Quick speed test (downloads a small file)
powershell -command "Measure-Command { Invoke-WebRequest -Uri 'http://speedtest.ftp.otenet.gr/files/test1Mb.db' -OutFile '%TEMP%\speedtest.tmp' }" > "%TEMP%\speed_result.txt"

rem Read the result and adjust settings
rem (This is simplified - you'd parse the actual time)
if exist "%TEMP%\speedtest.tmp" (
    for %%f in ("%TEMP%\speedtest.tmp") do set test_size=%%~zf
    if !test_size! gtr 500000 (
        rem Fast connection
        set CHUNK_SIZE=4194304
        set WORKERS=4
    ) else (
        rem Slow connection  
        set CHUNK_SIZE=1048576
        set WORKERS=2
    )
    del "%TEMP%\speedtest.tmp"
)
```

## 🚨 Common Problems

### Internet Connection Keeps Dropping
```cmd
rem Use these settings for really bad internet
jdc.exe -file backup.zip ^
        -connect server:8000 ^
        -chunk 1048576 ^
        -workers 2 ^
        -adaptive ^
        -retries 25 ^
        -timeout 12h
```

### Transfers are Too Slow
```cmd
rem Check if other programs are using your internet
netstat -b

rem Try reducing workers
jdc.exe -file backup.zip -connect server:8000 -workers 2

rem Or try during off-peak hours (like 2 AM)
```

### Files Keep Getting Corrupted
```cmd
rem Always verify your backups first
"C:\Program Files\7-Zip\7z.exe" t backup.zip

rem Use smaller chunks for bad connections
jdc.exe -file backup.zip -connect server:8000 -chunk 524288
```

## 🔐 Security for Internet Transfers

### Use VPN When Possible
```batch
rem Check if VPN is connected before transferring sensitive data
ping vpn-gateway.company.com >nul 2>&1
if errorlevel 1 (
    echo VPN not connected! Connect VPN first for security.
    pause
    exit /b 1
)

rem Then do your transfer to internal address
jdc.exe -file backup.zip -connect internal-backup.company.local:8000
```

## 📅 Schedule with Windows Task Scheduler

### Setting Up Automatic Daily Transfers
1. Open **Task Scheduler** (type it in Start menu)
2. Click **Create Basic Task**
3. Name it "Branch Office Backup Transfer"
4. Set trigger to **Daily** at **2:00 AM** (when internet is usually less busy)
5. Set action to **Start a program**: `C:\Scripts\branch-backup-transfer.bat`
6. Check **Run whether user is logged on or not**
7. Check **Run with highest privileges**

### Advanced PowerShell Version
```powershell
# branch-backup-transfer.ps1
# More advanced version with better error handling

param(
    [string]$BackupDir = "C:\Backup",
    [string]$MainOffice = "main-office.company.com:8000",
    [int]$MaxRetries = 3
)

$Today = Get-Date -Format "yyyyMMdd"
$LogFile = "C:\Logs\branch-backup-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"

function Write-Log($Message) {
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logEntry = "[$timestamp] $Message"
    Write-Host $logEntry
    Add-Content -Path $LogFile -Value $logEntry
}

function Test-InternetConnection {
    Write-Log "Testing connection to main office..."
    $ping = Test-Connection -ComputerName "main-office.company.com" -Count 3 -Quiet
    if (-not $ping) {
        Write-Log "ERROR: Cannot reach main office"
        return $false
    }
    Write-Log "Connection test passed"
    return $true
}

function Transfer-Backup($FilePath) {
    $fileName = Split-Path $FilePath -Leaf
    Write-Log "Starting transfer: $fileName"
    
    $args = @(
        "-file", $FilePath,
        "-connect", $MainOffice,
        "-chunk", "2097152",
        "-workers", "3",
        "-adaptive",
        "-verify=true",
        "-timeout", "8h",
        "-retries", "15"
    )
    
    $process = Start-Process -FilePath "jdc.exe" -ArgumentList $args -Wait -PassThru -NoNewWindow
    
    if ($process.ExitCode -eq 0) {
        Write-Log "SUCCESS: $fileName"
        return $true
    } else {
        Write-Log "FAILED: $fileName"
        return $false
    }
}

# Main script
Write-Log "=== Branch Office Backup Transfer Started ==="

if (-not (Test-InternetConnection)) {
    Write-Log "Internet connection failed, aborting"
    exit 1
}

$backupFiles = Get-ChildItem -Path $BackupDir -Filter "*$Today*.zip"

if ($backupFiles.Count -eq 0) {
    Write-Log "No backup files found for today"
    exit 0
}

$successCount = 0
$failCount = 0

foreach ($file in $backupFiles) {
    if (Transfer-Backup $file.FullName) {
        $successCount++
    } else {
        $failCount++
    }
}

Write-Log "=== Transfer Summary ==="
Write-Log "Successful: $successCount"
Write-Log "Failed: $failCount"

if ($failCount -eq 0) {
    Write-Log "All transfers completed successfully"
    exit 0
} else {
    Write-Log "Some transfers failed"
    exit 1
}
```

This setup ensures your branch office backups get to the main office reliably, even over sometimes-flaky internet connections!

## 📋 Complete Example Script

### Branch Office Transfer Script
```bash
#!/bin/bash
# branch-backup-transfer.sh

set -euo pipefail

# Configuration
BRANCH_ID="$(hostname -s)"
BACKUP_DIR="/backup"
DATACENTER_SERVER="datacenter.company.com:8000"
LOG_FILE="/var/log/branch-backup.log"
DATE=$(date +%Y%m%d)
MAX_RETRIES=3

# Network testing
SPEED_TEST_FILE="/tmp/speedtest.dat"
PING_TARGET="datacenter.company.com"

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$BRANCH_ID] $1" | tee -a "$LOG_FILE"
}

# Function to test network conditions
test_network() {
    log "Testing network conditions..."
    
    # Test connectivity
    if ! ping -c 3 "$PING_TARGET" >/dev/null 2>&1; then
        log "ERROR: Cannot reach datacenter server"
        return 1
    fi
    
    # Get average ping time
    avg_ping=$(ping -c 5 "$PING_TARGET" | tail -1 | awk -F '/' '{print $5}' | cut -d' ' -f1)
    log "Average ping time: ${avg_ping}ms"
    
    # Determine optimal settings based on ping
    if (( $(echo "$avg_ping > 100" | bc -l) )); then
        export JDC_CHUNK_SIZE=1048576  # 1MB for high latency
        export JDC_WORKERS=2
        export JDC_MAX_DELAY=1000ms
        log "High latency detected - using conservative settings"
    elif (( $(echo "$avg_ping > 50" | bc -l) )); then
        export JDC_CHUNK_SIZE=2097152  # 2MB for medium latency
        export JDC_WORKERS=3
        export JDC_MAX_DELAY=500ms
        log "Medium latency detected - using balanced settings"
    else
        export JDC_CHUNK_SIZE=4194304  # 4MB for low latency
        export JDC_WORKERS=4
        export JDC_MAX_DELAY=200ms
        log "Low latency detected - using aggressive settings"
    fi
}

# Function to transfer backup with retry logic
transfer_backup() {
    local backup_file="$1"
    local attempt=1
    
    while [[ $attempt -le $MAX_RETRIES ]]; do
        log "Transfer attempt $attempt for $(basename "$backup_file")"
        
        if jdc -file "$backup_file" \
               -connect "$DATACENTER_SERVER" \
               -chunk "${JDC_CHUNK_SIZE:-2097152}" \
               -buffer 524288 \
               -workers "${JDC_WORKERS:-3}" \
               -adaptive \
               -verify=true \
               -timeout 8h \
               -retries 15 \
               -min-delay 5ms \
               -max-delay "${JDC_MAX_DELAY:-500ms}" \
               -log-level info; then
            log "SUCCESS: Transfer completed for $(basename "$backup_file")"
            return 0
        else
            log "FAILED: Transfer attempt $attempt failed"
            ((attempt++))
            
            if [[ $attempt -le $MAX_RETRIES ]]; then
                sleep_time=$((attempt * 60))  # Progressive backoff
                log "Waiting ${sleep_time}s before retry..."
                sleep $sleep_time
            fi
        fi
    done
    
    log "ERROR: All $MAX_RETRIES attempts failed for $(basename "$backup_file")"
    return 1
}

# Function to cleanup old state files
cleanup_state_files() {
    log "Cleaning up old state files..."
    find /tmp -name "*.justdatacopier.state" -mtime +7 -delete 2>/dev/null || true
}

# Function to validate backup before transfer
validate_backup() {
    local backup_file="$1"
    
    log "Validating backup file: $(basename "$backup_file")"
    
    # Check if file exists and is readable
    if [[ ! -r "$backup_file" ]]; then
        log "ERROR: Backup file not readable: $backup_file"
        return 1
    fi
    
    # Check file size (minimum 1MB)
    local file_size=$(stat -c%s "$backup_file")
    if [[ $file_size -lt 1048576 ]]; then
        log "WARNING: Backup file unusually small: $file_size bytes"
    fi
    
    # Test archive integrity if it's a compressed file
    case "$backup_file" in
        *.tar.gz|*.tgz)
            if ! tar -tzf "$backup_file" >/dev/null 2>&1; then
                log "ERROR: Backup archive integrity check failed"
                return 1
            fi
            ;;
        *.zip)
            if ! unzip -t "$backup_file" >/dev/null 2>&1; then
                log "ERROR: Backup archive integrity check failed"
                return 1
            fi
            ;;
    esac
    
    log "Backup validation: PASSED"
    return 0
}

# Main execution
main() {
    log "=== Branch Office Backup Transfer Started ==="
    log "Branch ID: $BRANCH_ID"
    log "Date: $DATE"
    
    # Clean up old state files
    cleanup_state_files
    
    # Test network conditions
    if ! test_network; then
        log "Network test failed, aborting transfer"
        exit 1
    fi
    
    # Find backup files
    backup_files=$(find "$BACKUP_DIR" -name "*${DATE}*.tar.gz" -o -name "*${DATE}*.zip" | head -5)
    
    if [[ -z "$backup_files" ]]; then
        log "No backup files found for date $DATE"
        exit 0
    fi
    
    # Transfer each backup file
    failed_transfers=0
    total_transfers=0
    
    for backup_file in $backup_files; do
        ((total_transfers++))
        
        if validate_backup "$backup_file"; then
            if ! transfer_backup "$backup_file"; then
                ((failed_transfers++))
            fi
        else
            log "Skipping invalid backup: $(basename "$backup_file")"
            ((failed_transfers++))
        fi
    done
    
    # Summary
    log "=== Transfer Summary ==="
    log "Total files: $total_transfers"
    log "Failed transfers: $failed_transfers"
    log "Success rate: $(( (total_transfers - failed_transfers) * 100 / total_transfers ))%"
    
    if [[ $failed_transfers -eq 0 ]]; then
        log "=== All transfers completed successfully ==="
        exit 0
    else
        log "=== Transfer completed with failures ==="
        exit 1
    fi
}

main "$@"
```

### Windows Branch Office Script
```batch
@echo off
setlocal enabledelayedexpansion

REM branch-backup-transfer.bat
REM Configuration
set BRANCH_ID=%COMPUTERNAME%
set BACKUP_DIR=C:\Backup
set DATACENTER_SERVER=datacenter.company.com:8000
set LOG_FILE=C:\Logs\branch-backup.log
set DATE=%date:~10,4%%date:~4,2%%date:~7,2%

echo [%date% %time%] [%BRANCH_ID%] === Branch Office Backup Transfer Started === >> "%LOG_FILE%"

REM Find backup files
for %%f in ("%BACKUP_DIR%\*%DATE%*.zip") do (
    echo [%date% %time%] [%BRANCH_ID%] Starting transfer of %%~nxf >> "%LOG_FILE%"
    
    jdc.exe -file "%%f" ^
            -connect %DATACENTER_SERVER% ^
            -chunk 2097152 ^
            -buffer 524288 ^
            -workers 3 ^
            -adaptive ^
            -verify=true ^
            -timeout 8h ^
            -retries 15 ^
            -min-delay 5ms ^
            -max-delay 500ms ^
            -log-level info
    
    if !errorlevel! equ 0 (
        echo [%date% %time%] [%BRANCH_ID%] SUCCESS: Transfer completed for %%~nxf >> "%LOG_FILE%"
    ) else (
        echo [%date% %time%] [%BRANCH_ID%] ERROR: Transfer failed for %%~nxf >> "%LOG_FILE%"
    )
)

echo [%date% %time%] [%BRANCH_ID%] === Transfer batch completed === >> "%LOG_FILE%"
```

## 🔍 Monitoring & Alerting

### Transfer Status Monitoring
```bash
#!/bin/bash
# monitor-branch-transfers.sh

# Configuration
BRANCHES=("branch1" "branch2" "branch3" "branch4")
ALERT_EMAIL="admin@company.com"
BACKUP_DIR="/datacenter/branch-backups"
DATE=$(date +%Y%m%d)

# Check each branch backup
for branch in "${BRANCHES[@]}"; do
    expected_backup="${BACKUP_DIR}/${branch}_backup_${DATE}.tar.gz"
    
    if [[ -f "$expected_backup" ]]; then
        # Check if file was modified in last 24 hours
        if [[ $(find "$expected_backup" -mtime -1 | wc -l) -eq 1 ]]; then
            echo "✓ $branch: Backup received"
        else
            echo "⚠ $branch: Backup file old"
            echo "$branch backup is outdated" | mail -s "Branch Backup Alert" "$ALERT_EMAIL"
        fi
    else
        echo "✗ $branch: No backup received"
        echo "$branch backup missing for $DATE" | mail -s "Branch Backup Missing" "$ALERT_EMAIL"
    fi
done
```

## 📈 Performance Optimization

### Bandwidth-Based Configuration
```bash
# Function to auto-configure based on available bandwidth
configure_for_bandwidth() {
    local bandwidth_mbps="$1"
    
    if [[ $bandwidth_mbps -ge 100 ]]; then
        # High bandwidth (100+ Mbps)
        echo "chunk=4194304 workers=4 buffer=1048576"
    elif [[ $bandwidth_mbps -ge 50 ]]; then
        # Medium bandwidth (50-100 Mbps)
        echo "chunk=2097152 workers=3 buffer=524288"
    else
        # Low bandwidth (<50 Mbps)
        echo "chunk=1048576 workers=2 buffer=262144"
    fi
}

# Usage in transfer script
bandwidth=$(speedtest-cli --simple | grep Download | awk '{print $2}')
config=$(configure_for_bandwidth "${bandwidth%.*}")
eval "jdc -file backup.tar.gz -connect server:8000 -adaptive $config"
```

## 🚨 Troubleshooting Common Issues

### Network Interruption Recovery
```bash
# Check for incomplete transfers
find /tmp -name "*.justdatacopier.state" -exec ls -la {} \;

# Resume interrupted transfer
jdc -file /backup/branch_backup_20250705.tar.gz \
    -connect datacenter.company.com:8000 \
    -adaptive \
    # Same parameters as original transfer
```

### Bandwidth Throttling Detection
```bash
# Monitor transfer speed and adjust
tail -f /var/log/branch-backup.log | grep -i "transfer rate" | while read line; do
    rate=$(echo "$line" | grep -o '[0-9.]* MB/s')
    if (( $(echo "$rate < 5" | bc -l) )); then
        echo "Low transfer rate detected: $rate"
        # Consider reducing workers or chunk size
    fi
done
```

## 🔐 Security for WAN Transfers

### VPN Configuration
```bash
# Ensure VPN is active before transfer
if ! ip route | grep -q "10.0.0.0/8"; then
    echo "VPN not connected, starting..."
    systemctl start openvpn@company
    sleep 10
fi

# Then proceed with transfer
jdc -file backup.tar.gz -connect datacenter-internal.company.com:8000
```

### Certificate-based Authentication (Future Enhancement)
```bash
# Example of securing JDC with client certificates
jdc -file backup.tar.gz \
    -connect datacenter.company.com:8443 \
    -cert /etc/ssl/certs/branch-client.crt \
    -key /etc/ssl/private/branch-client.key \
    -ca /etc/ssl/certs/company-ca.crt
```

## 📅 Scheduling & Automation

### Systemd Timer (Linux)
```ini
# /etc/systemd/system/branch-backup-transfer.timer
[Unit]
Description=Branch Office Backup Transfer
Requires=branch-backup-transfer.service

[Timer]
OnCalendar=daily
RandomizedDelaySec=3600  # Random delay up to 1 hour

[Install]
WantedBy=timers.target
```

### Task Scheduler (Windows)
```xml
<!-- Import this XML into Windows Task Scheduler -->
<Task xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <Triggers>
    <CalendarTrigger>
      <DaysOfWeek>Daily</DaysOfWeek>
      <StartBoundary>2025-01-01T02:00:00</StartBoundary>
      <RandomDelay>PT1H</RandomDelay>
    </CalendarTrigger>
  </Triggers>
  <Actions>
    <Exec>
      <Command>C:\Scripts\branch-backup-transfer.bat</Command>
    </Exec>
  </Actions>
</Task>
```

This configuration ensures reliable, adaptive backup transfers from branch offices over variable WAN connections with comprehensive monitoring and error handling.
