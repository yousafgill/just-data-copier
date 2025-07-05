# Moving Database Backups Between Servers

This example shows you how to efficiently move large database backup files between Windows servers in your office or data center.

## 📊 What We're Doing

**Situation**: You need to copy nightly database backups from your main database server to your backup server
- **File Sizes**: Usually 100GB to 2TB (really big database dumps)
- **Network**: Fast office network (like 1 Gigabit connection) with good speed
- **How Often**: Every night automatically
- **Must Have**: Files must transfer correctly and be verified

## 🎯 Best Settings for This Job

### Setup the Backup Server (Where Files Go)
```cmd
rem Start the server to receive big database files
jdc.exe -server ^
    -listen 0.0.0.0:8000 ^
    -output "D:\Database_Backups" ^
    -workers 8 ^
    -buffer 1048576 ^
    -timeout 6h ^
    -log-level info
```

### Setup the Database Server (Where Files Come From)
```cmd
rem Send the database backup file
jdc.exe -file "C:\DatabaseBackups\prod_db_20250705.sql.gz" ^
    -connect backup-server:8000 ^
    -chunk 8388608 ^
    -buffer 1048576 ^
    -workers 8 ^
    -compress=false ^
    -verify=true ^
    -timeout 6h ^
    -retries 5
```

## 🔧 Why These Settings?

### Network Settings Explained
- **Chunk Size**: `8MB` - Good for fast office networks
- **Buffer Size**: `1MB` - Helps move data faster
- **Workers**: `8` - Uses multiple connections for speed
- **Compression**: `false` - Database dumps are already compressed

### Safety Settings
- **Verify**: `true` - Very important for database files!
- **Timeout**: `6h` - Gives plenty of time for big files
- **Retries**: `5` - Tries again if something goes wrong

## 📋 Complete Example (Batch File)

### Automatic Database Transfer Script
```batch
@echo off
setlocal enabledelayedexpansion

rem database-backup-transfer.bat
rem This script automatically transfers database backups

rem Setup
set DB_BACKUP_DIR=C:\DatabaseBackups
set BACKUP_SERVER=backup-server.company.local:8000
set LOG_FILE=C:\Logs\backup-transfer.log
set TODAY=%date:~10,4%%date:~4,2%%date:~7,2%

rem Function to write log messages
:log
echo [%date% %time%] %~1 >> "%LOG_FILE%"
echo [%date% %time%] %~1
goto :eof

call :log "=== Starting Database Backup Transfer ==="

rem Look for today's backup files
set "found_files="
for %%f in ("%DB_BACKUP_DIR%\*%TODAY%*.sql.gz") do (
    set "found_files=1"
    call :log "Found backup file: %%~nxf"
    
    rem Get file size for logging
    for %%s in ("%%f") do set file_size=%%~zs
    call :log "File size: !file_size! bytes"
    
    call :log "Starting transfer of %%~nxf"
    
    rem Transfer the file
    jdc.exe -file "%%f" ^
            -connect %BACKUP_SERVER% ^
            -chunk 8388608 ^
            -buffer 1048576 ^
            -workers 8 ^
            -verify=true ^
            -timeout 6h ^
            -retries 5 ^
            -log-level info
    
    if !errorlevel! equ 0 (
        call :log "SUCCESS: Transfer completed for %%~nxf"
    ) else (
        call :log "ERROR: Transfer failed for %%~nxf"
        set /a failed_transfers+=1
    )
)

rem Check if we found any files
if not defined found_files (
    call :log "ERROR: No backup files found for date %TODAY%"
    exit /b 1
)

rem Summary
if !failed_transfers! equ 0 (
    call :log "=== All transfers completed successfully ==="
    exit /b 0
) else (
    call :log "=== Transfer completed with !failed_transfers! failures ==="
    exit /b 1
)
```

### Server Startup Script
```batch
@echo off
setlocal

rem backup-server-start.bat
rem This starts the backup server

rem Setup
set LISTEN_ADDRESS=0.0.0.0:8000
set OUTPUT_DIR=D:\Database_Backups
set LOG_FILE=C:\Logs\backup-server.log

rem Create output folder if it doesn't exist
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

echo Starting JDC backup server...
echo Listen Address: %LISTEN_ADDRESS%
echo Output Directory: %OUTPUT_DIR%
echo Log File: %LOG_FILE%

rem Start the server
jdc.exe -server ^
    -listen %LISTEN_ADDRESS% ^
    -output "%OUTPUT_DIR%" ^
    -workers 8 ^
    -buffer 1048576 ^
    -timeout 6h ^
    -log-level info >> "%LOG_FILE%" 2>&1
```

## 🔍 Checking Everything Works

### Before Starting Transfer
```cmd
rem Check if you have enough disk space
dir "D:\Database_Backups"

rem Test if you can reach the backup server
ping backup-server.company.local
telnet backup-server.company.local 8000

rem Check if your backup file is good
rem (This checks if the compressed file isn't corrupted)
"C:\Program Files\7-Zip\7z.exe" t "C:\DatabaseBackups\prod_db_20250705.sql.gz"
```

### After Transfer Completes
```cmd
rem Compare file sizes to make sure they match
for %%f in ("C:\DatabaseBackups\prod_db_20250705.sql.gz") do set original_size=%%~zf
for %%f in ("D:\Database_Backups\prod_db_20250705.sql.gz") do set transferred_size=%%~zf

if %original_size% equ %transferred_size% (
    echo File size check: PASSED
) else (
    echo File size check: FAILED - sizes don't match!
)
```

## 📈 How Fast Should It Be?

### Expected Transfer Times
| File Size | Expected Time | Speed |
|-----------|---------------|-------|
| 100GB | ~15 minutes | ~110 MB/s |
| 500GB | ~75 minutes | ~110 MB/s |
| 1TB | ~2.5 hours | ~110 MB/s |
| 2TB | ~5 hours | ~110 MB/s |

*These times are for a good 1 Gigabit office network*

### What Your Computer Will Use
- **Network**: 80-90% of your available bandwidth
- **CPU**: 10-20% on both computers
- **Memory**: About 10MB plus buffer size per worker

## 🚨 When Things Go Wrong

### Transfer is Really Slow
```cmd
rem Try fewer workers if your CPU is maxed out
jdc.exe -file backup.sql.gz -connect server:8000 -workers 4

rem Try bigger chunks if your network is really fast
jdc.exe -file backup.sql.gz -connect server:8000 -chunk 16777216
```

### Connection Keeps Timing Out
```cmd
rem Give it more time for really big files
jdc.exe -file backup.sql.gz -connect server:8000 -timeout 12h

rem Use adaptive mode if your network is unpredictable
jdc.exe -file backup.sql.gz -connect server:8000 -adaptive
```

### File Verification Fails
```cmd
rem Check if your source file is corrupted first
"C:\Program Files\7-Zip\7z.exe" t backup.sql.gz

rem Try with different chunk size
jdc.exe -file backup.sql.gz -connect server:8000 -chunk 4194304
```

## 🔐 Keeping Things Secure

### Network Security
```cmd
rem If you're sending sensitive database backups, consider:
rem 1. Using a VPN connection
rem 2. Setting up firewall rules to only allow your database server

rem Example firewall rule (run as administrator):
netsh advfirewall firewall add rule name="JDC Backup Server" dir=in action=allow protocol=TCP localport=8000 remoteip=192.168.1.100
```

## 📅 Running This Automatically

### Using Windows Task Scheduler
1. Open Task Scheduler
2. Create Basic Task
3. Set trigger to "Daily" at 2:00 AM
4. Set action to start your batch file: `C:\Scripts\database-backup-transfer.bat`
5. Configure to run whether user is logged on or not

### PowerShell Version for Advanced Users
```powershell
# database-backup-transfer.ps1
param(
    [string]$BackupDir = "C:\DatabaseBackups",
    [string]$BackupServer = "backup-server.company.local:8000"
)

$Today = Get-Date -Format "yyyyMMdd"
$LogFile = "C:\Logs\backup-transfer-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"

function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] $Message"
    Write-Host $logMessage
    Add-Content -Path $LogFile -Value $logMessage
}

Write-Log "=== Database Backup Transfer Started ==="

$backupFiles = Get-ChildItem -Path $BackupDir -Filter "*$Today*.sql.gz"

if ($backupFiles.Count -eq 0) {
    Write-Log "No backup files found for today ($Today)"
    exit 1
}

foreach ($file in $backupFiles) {
    Write-Log "Transferring: $($file.Name) (Size: $([math]::Round($file.Length/1GB, 2)) GB)"
    
    $process = Start-Process -FilePath "jdc.exe" -ArgumentList @(
        "-file", $file.FullName,
        "-connect", $BackupServer,
        "-chunk", "8388608",
        "-buffer", "1048576", 
        "-workers", "8",
        "-verify=true",
        "-timeout", "6h",
        "-retries", "5"
    ) -Wait -PassThru
    
    if ($process.ExitCode -eq 0) {
        Write-Log "SUCCESS: $($file.Name)"
    } else {
        Write-Log "FAILED: $($file.Name)"
    }
}

Write-Log "=== Transfer Complete ==="
```

This setup gives you a reliable way to automatically move your database backups every night with proper error handling and logging!

## 📋 Complete Example Script

### Automated Backup Transfer Script
```bash
#!/bin/bash
# database-backup-transfer.sh

set -euo pipefail

# Configuration
DB_BACKUP_DIR="/var/backups"
BACKUP_SERVER="backup-server.company.com:8000"
LOG_FILE="/var/log/backup-transfer.log"
DATE=$(date +%Y%m%d)

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Function to transfer database backup
transfer_backup() {
    local backup_file="$1"
    local file_size=$(du -h "$backup_file" | cut -f1)
    
    log "Starting transfer of $backup_file (Size: $file_size)"
    
    # Start transfer with optimal settings
    if jdc -file "$backup_file" \
           -connect "$BACKUP_SERVER" \
           -chunk 8388608 \
           -buffer 1048576 \
           -workers 8 \
           -verify=true \
           -timeout 6h \
           -retries 5 \
           -log-level info; then
        log "SUCCESS: Transfer completed for $backup_file"
        return 0
    else
        log "ERROR: Transfer failed for $backup_file"
        return 1
    fi
}

# Main execution
main() {
    log "=== Database Backup Transfer Started ==="
    
    # Find today's backup files
    backup_files=$(find "$DB_BACKUP_DIR" -name "*${DATE}*.sql.gz" -type f)
    
    if [[ -z "$backup_files" ]]; then
        log "ERROR: No backup files found for date $DATE"
        exit 1
    fi
    
    # Transfer each backup file
    failed_transfers=0
    for backup_file in $backup_files; do
        if ! transfer_backup "$backup_file"; then
            ((failed_transfers++))
        fi
    done
    
    # Summary
    if [[ $failed_transfers -eq 0 ]]; then
        log "=== All transfers completed successfully ==="
        exit 0
    else
        log "=== Transfer completed with $failed_transfers failures ==="
        exit 1
    fi
}

main "$@"
```

### Server Startup Script
```bash
#!/bin/bash
# backup-server-start.sh

set -euo pipefail

# Configuration
LISTEN_ADDRESS="0.0.0.0:8000"
OUTPUT_DIR="/backup/databases"
LOG_FILE="/var/log/backup-server.log"

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Function to cleanup on exit
cleanup() {
    echo "Shutting down backup server..."
    # Any cleanup operations here
}
trap cleanup EXIT

# Start server
echo "Starting JDC backup server..."
echo "Listen Address: $LISTEN_ADDRESS"
echo "Output Directory: $OUTPUT_DIR"
echo "Log File: $LOG_FILE"

jdc -server \
    -listen "$LISTEN_ADDRESS" \
    -output "$OUTPUT_DIR" \
    -workers 8 \
    -buffer 1048576 \
    -timeout 6h \
    -log-level info 2>&1 | tee "$LOG_FILE"
```

## 🔍 Monitoring & Validation

### Pre-Transfer Validation
```bash
# Check disk space
df -h /backup/databases

# Test connectivity
ping -c 4 backup-server.company.com
nc -zv backup-server.company.com 8000

# Verify backup file integrity
if [[ -f /var/backups/prod_db_20250705.sql.gz ]]; then
    gzip -t /var/backups/prod_db_20250705.sql.gz
    echo "Backup file integrity: OK"
fi
```

### Post-Transfer Validation
```bash
# Compare file sizes
original_size=$(stat -c%s "/var/backups/prod_db_20250705.sql.gz")
transferred_size=$(stat -c%s "/backup/databases/prod_db_20250705.sql.gz")

if [[ $original_size -eq $transferred_size ]]; then
    echo "File size verification: PASSED"
else
    echo "File size verification: FAILED"
    exit 1
fi

# Hash verification (if needed)
md5sum "/var/backups/prod_db_20250705.sql.gz"
md5sum "/backup/databases/prod_db_20250705.sql.gz"
```

## 📈 Performance Expectations

### Transfer Times (Estimated)
| File Size | Expected Time | Throughput |
|-----------|---------------|------------|
| 100GB | ~15 minutes | ~110 MB/s |
| 500GB | ~75 minutes | ~110 MB/s |
| 1TB | ~2.5 hours | ~110 MB/s |
| 2TB | ~5 hours | ~110 MB/s |

*Times based on 1Gbps network with optimal configuration*

### Network Utilization
- **Expected Throughput**: 80-90% of available bandwidth
- **CPU Usage**: 10-20% on both client and server
- **Memory Usage**: ~10MB base + buffer size per worker

## 🚨 Troubleshooting

### Common Issues

**Slow Transfer Speed**
```bash
# Reduce workers if CPU-bound
jdc -file backup.sql.gz -connect server:8000 -workers 4

# Increase chunk size for high-bandwidth networks
jdc -file backup.sql.gz -connect server:8000 -chunk 16777216
```

**Connection Timeouts**
```bash
# Increase timeout for very large files
jdc -file backup.sql.gz -connect server:8000 -timeout 12h

# Enable adaptive mode for unstable networks
jdc -file backup.sql.gz -connect server:8000 -adaptive
```

**Hash Verification Failures**
```bash
# Check source file integrity first
gzip -t backup.sql.gz

# Retry with different chunk size
jdc -file backup.sql.gz -connect server:8000 -chunk 4194304
```

## 🔐 Security Considerations

### Network Security
```bash
# Use VPN or secure network for sensitive database backups
# Consider SSH tunneling for additional security
ssh -L 8000:backup-server:8000 user@bastion-host

# Then connect to localhost
jdc -file backup.sql.gz -connect localhost:8000
```

### Access Control
```bash
# Restrict server access by IP
iptables -A INPUT -p tcp --dport 8000 -s 192.168.1.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 8000 -j DROP
```

## 📅 Automation Integration

### Cron Job Setup
```bash
# Add to crontab for nightly transfers
# Run at 2 AM daily
0 2 * * * /opt/scripts/database-backup-transfer.sh >> /var/log/cron-backup.log 2>&1
```

### Systemd Service (Server)
```ini
# /etc/systemd/system/jdc-backup-server.service
[Unit]
Description=JDC Backup Server
After=network.target

[Service]
Type=simple
User=backup
Group=backup
ExecStart=/usr/local/bin/backup-server-start.sh
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

This configuration provides enterprise-grade database backup transfer with optimal performance, reliability, and monitoring capabilities.
