# Moving Database Backups Between Servers

This example shows you how to efficiently move large database backup files between Windows servers in your office or data center.

## 📊 What We're Doing

**Situation**: You need to copy nightly SQL Server database backups from your main database server to your backup server
- **File Sizes**: Usually 100GB to 2TB (really big database backup files)
- **Network**: Fast office network (like 1 Gigabit connection) with good speed
- **How Often**: Every night automatically
- **Must Have**: Files must transfer correctly and be verified

## 🎯 Best Settings for This Job

### Setup the Backup Server (Where Files Go)
```cmd
rem Start the server to receive big database backup files
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
rem Send the SQL Server database backup file
jdc.exe -file "C:\DatabaseBackups\prod_db_20250705.bak" ^
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
- **Compression**: `false` - SQL Server backup files are already compressed

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
for %%f in ("%DB_BACKUP_DIR%\*%TODAY%*.bak") do (
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
rem (This checks if the SQL Server backup file isn't corrupted)
sqlcmd -S localhost -E -Q "RESTORE VERIFYONLY FROM DISK = 'C:\DatabaseBackups\prod_db_20250705.bak'"
```

### After Transfer Completes
```cmd
rem Compare file sizes to make sure they match
for %%f in ("C:\DatabaseBackups\prod_db_20250705.bak") do set original_size=%%~zf
for %%f in ("D:\Database_Backups\prod_db_20250705.bak") do set transferred_size=%%~zf

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
rem Check if your source backup file is corrupted first
sqlcmd -S localhost -E -Q "RESTORE VERIFYONLY FROM DISK = 'C:\DatabaseBackups\backup.bak'"

rem Try with different chunk size
jdc.exe -file backup.bak -connect server:8000 -chunk 4194304
```

## 🔐 Keeping Things Secure

### Network Security
```cmd
rem If you're sending sensitive SQL Server database backups, consider:
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

$backupFiles = Get-ChildItem -Path $BackupDir -Filter "*$Today*.bak"

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

## 📋 SQL Server Integration Script

### Automated SQL Server Backup and Transfer
```batch
@echo off
setlocal enabledelayedexpansion

rem sql-backup-and-transfer.bat
rem This script creates SQL Server backup and transfers it

rem Configuration
set DB_NAME=ProductionDB
set BACKUP_DIR=C:\DatabaseBackups
set BACKUP_SERVER=backup-server.company.local:8000
set LOG_FILE=C:\Logs\sql-backup-transfer.log
set TODAY=%date:~10,4%%date:~4,2%%date:~7,2%
set BACKUP_FILE=%BACKUP_DIR%\%DB_NAME%_%TODAY%.bak

rem Function to write log messages
:log
echo [%date% %time%] %~1 >> "%LOG_FILE%"
echo [%date% %time%] %~1
goto :eof

call :log "=== Starting SQL Server Backup and Transfer ==="

rem Create backup directory if it doesn't exist
if not exist "%BACKUP_DIR%" mkdir "%BACKUP_DIR%"

rem Step 1: Create SQL Server backup
call :log "Creating SQL Server backup for database: %DB_NAME%"
sqlcmd -S localhost -E -Q "BACKUP DATABASE [%DB_NAME%] TO DISK = '%BACKUP_FILE%' WITH COMPRESSION, CHECKSUM, INIT"

if !errorlevel! neq 0 (
    call :log "ERROR: SQL Server backup failed"
    exit /b 1
)

call :log "SUCCESS: SQL Server backup created at %BACKUP_FILE%"

rem Step 2: Verify backup integrity
call :log "Verifying backup integrity..."
sqlcmd -S localhost -E -Q "RESTORE VERIFYONLY FROM DISK = '%BACKUP_FILE%'"

if !errorlevel! neq 0 (
    call :log "ERROR: Backup verification failed"
    exit /b 1
)

call :log "SUCCESS: Backup verification passed"

rem Step 3: Transfer backup file
call :log "Starting transfer to backup server"

jdc.exe -file "%BACKUP_FILE%" ^
        -connect %BACKUP_SERVER% ^
        -chunk 8388608 ^
        -buffer 1048576 ^
        -workers 8 ^
        -verify=true ^
        -timeout 6h ^
        -retries 5 ^
        -log-level info

if !errorlevel! equ 0 (
    call :log "SUCCESS: Transfer completed successfully"
    
    rem Optional: Delete local backup after successful transfer
    rem del "%BACKUP_FILE%"
    rem call :log "Local backup file deleted"
) else (
    call :log "ERROR: Transfer failed"
    exit /b 1
)

call :log "=== Backup and Transfer Process Completed ==="
```

## 🔍 SQL Server Backup Validation

### Pre-Transfer Validation
```cmd
rem Check disk space
dir "D:\Database_Backups"

rem Test connectivity to backup server
ping backup-server.company.local
telnet backup-server.company.local 8000

rem Verify SQL Server backup file integrity
sqlcmd -S localhost -E -Q "RESTORE VERIFYONLY FROM DISK = 'C:\DatabaseBackups\prod_db_20250705.bak'"
```

### Post-Transfer Validation
```cmd
rem Compare file sizes
for %%f in ("C:\DatabaseBackups\prod_db_20250705.bak") do set original_size=%%~zf
for %%f in ("D:\Database_Backups\prod_db_20250705.bak") do set transferred_size=%%~zf

if %original_size% equ %transferred_size% (
    echo File size verification: PASSED
) else (
    echo File size verification: FAILED
    exit /b 1
)

rem Verify transferred backup file integrity (on backup server)
sqlcmd -S backup-server -E -Q "RESTORE VERIFYONLY FROM DISK = 'D:\Database_Backups\prod_db_20250705.bak'"
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

### Slow Transfer Speed**
```cmd
# Reduce workers if CPU-bound
jdc.exe -file backup.bak -connect server:8000 -workers 4

# Increase chunk size for high-bandwidth networks
jdc.exe -file backup.bak -connect server:8000 -chunk 16777216
```

**Connection Timeouts**
```cmd
# Increase timeout for very large SQL Server backup files
jdc.exe -file backup.bak -connect server:8000 -timeout 12h

# Enable adaptive mode for unstable networks
jdc.exe -file backup.bak -connect server:8000 -adaptive
```

**Hash Verification Failures**
```cmd
# Check source backup file integrity first
sqlcmd -S localhost -E -Q "RESTORE VERIFYONLY FROM DISK = 'backup.bak'"

# Retry with different chunk size
jdc.exe -file backup.bak -connect server:8000 -chunk 4194304
```

## 🔐 Security Considerations

### Network Security
```cmd
rem Use VPN or secure network for sensitive SQL Server database backups
rem Consider Windows Authentication and encrypted connections

rem Example: Connect through VPN tunnel
rem Set up site-to-site VPN or use Windows built-in VPN client

rem Restrict server access by IP using Windows Firewall
netsh advfirewall firewall add rule name="JDC Backup Server" dir=in action=allow protocol=TCP localport=8000 remoteip=192.168.1.0/24
netsh advfirewall firewall add rule name="JDC Backup Block" dir=in action=block protocol=TCP localport=8000
```

### Access Control
```cmd
rem Use Windows Authentication for SQL Server
rem Ensure JDC service runs under appropriate service account
rem Set up proper NTFS permissions on backup directories

rem Example: Set backup directory permissions
icacls "D:\Database_Backups" /grant "DOMAIN\BackupService:(OI)(CI)F" /T
```

## 📅 Windows Task Scheduler Integration

### Automated Daily Backups
1. **Open Task Scheduler** (taskschd.msc)
2. **Create Basic Task**
   - Name: "SQL Server Backup Transfer"
   - Description: "Daily backup and transfer of production database"
3. **Set Trigger**
   - Daily at 2:00 AM
   - Start date: Today
4. **Set Action**
   - Program: `C:\Scripts\sql-backup-and-transfer.bat`
   - Arguments: (optional parameters)
5. **Configure Settings**
   - Run whether user is logged on or not
   - Run with highest privileges
   - Configure for Windows Server 2019/2022

### PowerShell Scheduled Job
```powershell
# Create scheduled job for database backup transfer
$Trigger = New-JobTrigger -Daily -At "2:00 AM"
$Option = New-ScheduledJobOption -RunElevated -RequireNetwork

Register-ScheduledJob -Name "DatabaseBackupTransfer" `
                     -FilePath "C:\Scripts\Backup-Database.ps1" `
                     -Trigger $Trigger `
                     -ScheduledJobOption $Option
```

### Windows Service Option
```cmd
rem Install as Windows Service using NSSM (Non-Sucking Service Manager)
rem Download NSSM from https://nssm.cc/

rem Install service
nssm install "JDC-BackupServer" "C:\Tools\jdc.exe"
nssm set "JDC-BackupServer" Parameters "-server -listen 0.0.0.0:8000 -output D:\Database_Backups"
nssm set "JDC-BackupServer" DisplayName "JDC Database Backup Server"
nssm set "JDC-BackupServer" Description "JustDataCopier server for receiving SQL Server backups"

rem Start the service
net start "JDC-BackupServer"
```

This configuration provides enterprise-grade SQL Server backup transfer with optimal performance, reliability, and Windows integration capabilities.
