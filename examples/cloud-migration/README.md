# Moving Data to the Cloud

This example shows you how to efficiently move large amounts of data from your on-premises servers to cloud providers like AWS, Azure, or Google Cloud.

## 📊 What We're Doing

**Situation**: You're moving your company's data from your own servers to the cloud
- **Data Types**: Virtual machine files, databases, file shares, application data
- **File Sizes**: Usually 500GB to 10TB per batch of files
- **Network**: Internet connection to cloud provider, speed varies
- **Goals**: Move data cost-effectively with minimal downtime and perfect accuracy

## 🎯 Best Settings for Cloud Transfers

### Cloud Server Setup (Your VM in the Cloud)
```cmd
rem Set up a receiving server in your cloud environment
jdc.exe -server ^
    -listen 0.0.0.0:8000 ^
    -output "D:\Migration_Staging" ^
    -workers 8 ^
    -buffer 2097152 ^
    -timeout 24h ^
    -retries 20 ^
    -log-level info
```

### On-Premises Setup (Your Local Server)
```cmd
rem Send data from your office to the cloud
jdc.exe -file "D:\VMs\production-vm-001.vmdk" ^
    -connect your-cloud-server.amazonaws.com:8000 ^
    -chunk 4194304 ^
    -buffer 1048576 ^
    -workers 6 ^
    -adaptive ^
    -compress=false ^
    -verify=true ^
    -timeout 24h ^
    -retries 25 ^
    -min-delay 10ms ^
    -max-delay 2000ms
```

## 🔧 Smart Migration Strategy

### Do It in Phases
1. **Test Phase**: Try with small files first to make sure everything works
2. **Pilot Migration**: Move a few important files to test the process
3. **Bulk Migration**: Move the big stuff during off-peak hours (weekends/nights)
4. **Verification**: Double-check everything transferred correctly

### Why These Settings for Cloud
- **Chunk Size**: `4MB` - Good balance for internet-to-cloud transfers
- **Buffer Size**: `1MB` - Handles cloud connection well
- **Workers**: `6` - Uses your internet bandwidth efficiently without overwhelming it
- **Adaptive Mode**: Must have! Adjusts to changing internet conditions

## 📋 Complete Cloud Migration Script

### Pre-Migration Test Script
```batch
@echo off
setlocal enabledelayedexpansion

rem cloud-migration-test.bat
rem Test your connection and speed before the big migration

set CLOUD_SERVER=your-cloud-server.amazonaws.com
set CLOUD_PORT=8000
set TEST_FILE_SIZE=100
set LOG_FILE=C:\Logs\migration-test.log

rem Function to write log messages
:log
echo [%date% %time%] %~1 >> "%LOG_FILE%"
echo [%date% %time%] %~1
goto :eof

call :log "=== Cloud Migration Test Started ==="

rem Test if we can reach the cloud server
call :log "Testing connection to cloud server..."
ping -n 5 %CLOUD_SERVER% >nul 2>&1
if !errorlevel! neq 0 (
    call :log "ERROR: Cannot reach cloud server"
    pause
    exit /b 1
)
call :log "Connection test: PASSED"

rem Create a test file
call :log "Creating test file (%TEST_FILE_SIZE%MB)..."
fsutil file createnew "%TEMP%\migration-test.dat" %TEST_FILE_SIZE%000000 >nul 2>&1

rem Test transfer speed
call :log "Testing transfer speed..."
set start_time=%time%

jdc.exe -file "%TEMP%\migration-test.dat" ^
        -connect %CLOUD_SERVER%:%CLOUD_PORT% ^
        -chunk 4194304 ^
        -workers 4 ^
        -timeout 30m ^
        -log-level info

if !errorlevel! equ 0 (
    set end_time=%time%
    call :log "Transfer test: SUCCESS"
    rem Calculate rough speed (this is simplified)
    call :log "Test file transferred successfully"
) else (
    call :log "Transfer test: FAILED"
)

rem Clean up test file
del "%TEMP%\migration-test.dat" 2>nul

call :log "=== Test Complete ==="
pause
```

### Full Migration Script
```batch
@echo off
setlocal enabledelayedexpansion

rem cloud-bulk-migration.bat
rem Migrate large amounts of data to the cloud

set MIGRATION_SOURCE=D:\Migration_Data
set CLOUD_SERVER=your-cloud-server.amazonaws.com:8000
set LOG_FILE=C:\Logs\cloud-migration.log
set MIGRATION_ID=MIGRATION_%date:~10,4%%date:~4,2%%date:~7,2%_%time:~0,2%%time:~3,2%%time:~6,2%
set MAX_PARALLEL=3

rem Files to track progress
set MIGRATION_LOG=C:\Logs\migration-progress-%MIGRATION_ID%.txt
set FAILED_FILES=C:\Logs\migration-failures-%MIGRATION_ID%.txt

:log
echo [%date% %time%] [%MIGRATION_ID%] %~1 >> "%LOG_FILE%"
echo [%date% %time%] [%MIGRATION_ID%] %~1
goto :eof

rem Function to transfer a single file
:transfer_file
set file_path=%~1
set file_name=%~nx1
set attempt=1
set max_attempts=3

:retry_file
call :log "Starting transfer: %file_name% (attempt %attempt%)"

jdc.exe -file "%file_path%" ^
        -connect %CLOUD_SERVER% ^
        -chunk 4194304 ^
        -buffer 1048576 ^
        -workers 6 ^
        -adaptive ^
        -verify=true ^
        -timeout 24h ^
        -retries 25 ^
        -min-delay 10ms ^
        -max-delay 2000ms ^
        -log-level info

if !errorlevel! equ 0 (
    call :log "SUCCESS: %file_name% (attempt %attempt%)"
    echo %file_name%:SUCCESS:%date%:%time% >> "%MIGRATION_LOG%"
    exit /b 0
) else (
    call :log "FAILED: %file_name% (attempt %attempt%)"
    set /a attempt+=1
    
    if !attempt! leq %max_attempts% (
        set /a wait_time=!attempt! * 300
        call :log "Retrying %file_name% in !wait_time! seconds..."
        timeout /t !wait_time! /nobreak >nul
        goto :retry_file
    )
)

call :log "FINAL FAILURE: %file_name% after %max_attempts% attempts"
echo %file_name%:FAILED:%date%:%time% >> "%FAILED_FILES%"
exit /b 1

rem Main migration process
call :log "=== Cloud Migration Started ==="
call :log "Migration ID: %MIGRATION_ID%"
call :log "Source: %MIGRATION_SOURCE%"
call :log "Destination: %CLOUD_SERVER%"

rem Check if source folder exists
if not exist "%MIGRATION_SOURCE%" (
    call :log "ERROR: Source folder not found: %MIGRATION_SOURCE%"
    pause
    exit /b 1
)

rem Count files and calculate total size
call :log "Scanning files to migrate..."
set file_count=0
set total_size=0

for /r "%MIGRATION_SOURCE%" %%f in (*.*) do (
    rem Only include files larger than 1MB
    for %%s in ("%%f") do if %%~zs gtr 1048576 (
        set /a file_count+=1
        set /a total_size+=%%~zs
    )
)

if !file_count! equ 0 (
    call :log "No files found for migration"
    pause
    exit /b 0
)

set /a total_size_gb=!total_size! / 1073741824
call :log "Found !file_count! files for migration"
call :log "Total size: approximately !total_size_gb! GB"

rem Start migration
set start_time=%time%
set success_count=0
set failure_count=0

for /r "%MIGRATION_SOURCE%" %%f in (*.*) do (
    rem Only migrate files larger than 1MB
    for %%s in ("%%f") do if %%~zs gtr 1048576 (
        call :transfer_file "%%f"
        if !errorlevel! equ 0 (
            set /a success_count+=1
        ) else (
            set /a failure_count+=1
        )
    )
)

set end_time=%time%

rem Summary
call :log "=== Migration Summary ==="
call :log "Total files: !file_count!"
call :log "Successful: !success_count!"
call :log "Failed: !failure_count!"

if !failure_count! equ 0 (
    call :log "Migration completed successfully!"
    exit /b 0
) else (
    call :log "Migration completed with failures. Check: %FAILED_FILES%"
    exit /b 1
)
```

## 🔍 Cloud Provider Specific Tips

### For Amazon AWS
```cmd
rem If using AWS, you might transfer to an EC2 instance first
jdc.exe -file large-dataset.tar ^
        -connect ec2-xx-xxx-xxx-xx.us-east-1.compute.amazonaws.com:8000 ^
        -chunk 8388608 ^
        -workers 8 ^
        -adaptive ^
        -timeout 48h
```

### For Microsoft Azure
```cmd
rem For Azure, transfer to your Azure VM
jdc.exe -file vm-image.vhd ^
        -connect your-vm.eastus.cloudapp.azure.com:8000 ^
        -chunk 4194304 ^
        -workers 6 ^
        -adaptive ^
        -timeout 36h
```

### For Google Cloud
```cmd
rem For Google Cloud Platform
jdc.exe -file database-backup.sql.gz ^
        -connect your-instance.us-central1-a.googlecloud.com:8000 ^
        -chunk 4194304 ^
        -workers 6 ^
        -adaptive ^
        -timeout 24h
```

## 📊 Saving Money on Cloud Transfers

### Transfer During Off-Peak Hours
```batch
rem Schedule transfers for off-peak hours to save money
rem Check what time it is and decide whether to transfer now

set current_hour=%time:~0,2%
set /a current_hour_num=1%current_hour% - 100

rem Transfer between 2 AM and 6 AM for better rates
if %current_hour_num% geq 2 if %current_hour_num% lss 6 (
    echo Off-peak hours detected, starting migration...
    call cloud-bulk-migration.bat
) else (
    echo Peak hours, will schedule for later...
    rem You could set up a scheduled task here
)
```

### Control Your Internet Usage
```cmd
rem Limit transfer speed to avoid overwhelming your internet
jdc.exe -file large-file.dat ^
        -connect cloud-server:8000 ^
        -chunk 2097152 ^
        -workers 2 ^
        -adaptive ^
        -max-delay 1000ms
```

## 🔐 Security for Cloud Migration

### Extra Security for Sensitive Data
```batch
rem For highly sensitive data, you might want to:
rem 1. Encrypt files before transferring
rem 2. Use VPN connection to cloud
rem 3. Transfer to a secure staging area first

rem Example: Check if VPN is active
ping your-vpn-gateway.company.com >nul 2>&1
if errorlevel 1 (
    echo WARNING: VPN not detected. Connect VPN for sensitive data.
    pause
)

rem Then transfer to internal cloud address
jdc.exe -file sensitive-data.zip ^
        -connect internal-cloud-server.local:8000 ^
        -verify=true
```

### Different Handling for Different File Types
```batch
rem Handle different types of files appropriately
for %%f in (D:\Migration_Data\*.*) do (
    echo %%f | find /i "confidential" >nul
    if not errorlevel 1 (
        echo Transferring sensitive file with extra security: %%~nxf
        jdc.exe -file "%%f" -connect secure-cloud:8443 -verify=true
    ) else (
        echo Transferring regular file: %%~nxf
        jdc.exe -file "%%f" -connect regular-cloud:8000 -verify=true
    )
)
```

## 📈 Monitor Your Migration Progress

### Simple Progress Dashboard
```batch
@echo off
rem migration-dashboard.bat
rem Simple way to see how your migration is going

:loop
cls
echo === Cloud Migration Dashboard ===
echo.
echo Current time: %date% %time%
echo.

echo Current transfers running:
tasklist /fi "imagename eq jdc.exe" | find /c "jdc.exe"

echo.
echo Recent transfer results:
if exist "C:\Logs\migration-progress-*.txt" (
    for /f %%f in ('dir /b /o-d "C:\Logs\migration-progress-*.txt" ^| head -1') do (
        echo Successful transfers today:
        find /c ":SUCCESS:" "C:\Logs\%%f"
        echo Failed transfers today:
        find /c ":FAILED:" "C:\Logs\migration-failures-*.txt" 2>nul
    )
)

echo.
echo Disk space on migration drive:
dir D:\ | find "bytes free"

echo.
echo Press Ctrl+C to stop monitoring...
timeout /t 30 /nobreak >nul
goto loop
```

## 🚨 What to Do When Things Go Wrong

### If Migration Gets Stuck
```batch
rem Check what's happening
tasklist | find "jdc.exe"

rem If needed, stop all transfers and restart
taskkill /f /im jdc.exe

rem Then restart with more conservative settings
jdc.exe -file stuck-file.dat ^
        -connect cloud:8000 ^
        -chunk 1048576 ^
        -workers 2 ^
        -timeout 48h
```

### If You Need to Undo Migration
```batch
rem migration-rollback.bat
rem Use this if you need to reverse a migration

set MIGRATION_ID=%1
set MIGRATION_LOG=C:\Logs\migration-progress-%MIGRATION_ID%.txt

if not exist "%MIGRATION_LOG%" (
    echo Migration log not found: %MIGRATION_LOG%
    pause
    exit /b 1
)

echo Files that were successfully migrated:
for /f "tokens=1,2 delims=:" %%a in ('find ":SUCCESS:" "%MIGRATION_LOG%"') do (
    echo Would need to remove from cloud: %%a
)

echo.
echo This is just a preview. Add actual rollback commands as needed.
pause
```

This setup gives you a solid foundation for moving your data to the cloud reliably and cost-effectively!

## 📋 Complete Migration Script

### Pre-Migration Assessment
```bash
#!/bin/bash
# cloud-migration-assessment.sh

set -euo pipefail

# Configuration
CLOUD_ENDPOINT="cloud-migration.region.provider.com"
CLOUD_PORT="8000"
TEST_FILE_SIZE="100M"  # 100MB test file
LOG_FILE="/var/log/migration-assessment.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Network connectivity test
test_connectivity() {
    log "Testing connectivity to cloud endpoint..."
    
    if ping -c 5 "$CLOUD_ENDPOINT" >/dev/null 2>&1; then
        avg_rtt=$(ping -c 10 "$CLOUD_ENDPOINT" | tail -1 | awk -F '/' '{print $5}')
        log "Connectivity: OK (Average RTT: ${avg_rtt}ms)"
        return 0
    else
        log "Connectivity: FAILED"
        return 1
    fi
}

# Bandwidth test with sample transfer
test_bandwidth() {
    log "Creating test file ($TEST_FILE_SIZE)..."
    dd if=/dev/urandom of="/tmp/migration-test.dat" bs=1M count=100 2>/dev/null
    
    log "Testing transfer speed..."
    start_time=$(date +%s)
    
    if jdc -file "/tmp/migration-test.dat" \
           -connect "$CLOUD_ENDPOINT:$CLOUD_PORT" \
           -chunk 4194304 \
           -workers 4 \
           -timeout 30m; then
        
        end_time=$(date +%s)
        duration=$((end_time - start_time))
        speed=$(echo "scale=2; 100 / $duration" | bc)
        log "Transfer test: SUCCESS (${speed} MB/s)"
        
        # Cleanup test file
        rm -f "/tmp/migration-test.dat"
        return 0
    else
        log "Transfer test: FAILED"
        rm -f "/tmp/migration-test.dat"
        return 1
    fi
}

# Estimate migration time
estimate_migration_time() {
    local total_data_gb="$1"
    local test_speed_mbps="$2"
    
    # Convert and calculate
    local total_data_mb=$((total_data_gb * 1024))
    local estimated_hours=$(echo "scale=1; $total_data_mb / $test_speed_mbps / 3600" | bc)
    
    log "Migration time estimate for ${total_data_gb}GB: ${estimated_hours} hours"
}

# Main assessment
main() {
    log "=== Cloud Migration Assessment Started ==="
    
    if test_connectivity && test_bandwidth; then
        log "Network assessment: PASSED"
        log "Ready for migration"
        
        # Example estimation for 1TB migration
        estimate_migration_time 1024 10  # 1TB at 10 MB/s
    else
        log "Network assessment: FAILED"
        log "Review network configuration before migration"
        exit 1
    fi
    
    log "=== Assessment Complete ==="
}

main "$@"
```

### Bulk Migration Script
```bash
#!/bin/bash
# cloud-bulk-migration.sh

set -euo pipefail

# Configuration
MIGRATION_SOURCE="/data/migration-batch"
CLOUD_ENDPOINT="cloud-migration.region.provider.com:8000"
LOG_FILE="/var/log/cloud-migration.log"
MIGRATION_ID="MIGRATION_$(date +%Y%m%d_%H%M%S)"
MAX_PARALLEL=3  # Maximum parallel transfers

# Migration tracking
MIGRATION_MANIFEST="/var/log/migration-manifest-${MIGRATION_ID}.txt"
FAILED_TRANSFERS="/var/log/migration-failures-${MIGRATION_ID}.txt"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$MIGRATION_ID] $1" | tee -a "$LOG_FILE"
}

# Function to transfer single file with retry
transfer_file() {
    local file_path="$1"
    local relative_path="${file_path#$MIGRATION_SOURCE/}"
    local attempt=1
    local max_attempts=3
    
    log "Starting transfer: $relative_path"
    
    while [[ $attempt -le $max_attempts ]]; do
        if jdc -file "$file_path" \
               -connect "$CLOUD_ENDPOINT" \
               -chunk 4194304 \
               -buffer 1048576 \
               -workers 6 \
               -adaptive \
               -verify=true \
               -timeout 24h \
               -retries 25 \
               -min-delay 10ms \
               -max-delay 2000ms \
               -log-level info; then
            
            log "SUCCESS: $relative_path (attempt $attempt)"
            echo "$relative_path:SUCCESS:$(date):$(stat -c%s "$file_path")" >> "$MIGRATION_MANIFEST"
            return 0
        else
            log "FAILED: $relative_path (attempt $attempt)"
            ((attempt++))
            
            if [[ $attempt -le $max_attempts ]]; then
                sleep_time=$((attempt * 300))  # Progressive backoff: 5min, 10min
                log "Retrying $relative_path in ${sleep_time}s..."
                sleep $sleep_time
            fi
        fi
    done
    
    log "FINAL FAILURE: $relative_path after $max_attempts attempts"
    echo "$relative_path:FAILED:$(date):$(stat -c%s "$file_path")" >> "$FAILED_TRANSFERS"
    return 1
}

# Function to process files in parallel
process_migration_batch() {
    local file_list=("$@")
    local active_jobs=0
    local pids=()
    
    for file_path in "${file_list[@]}"; do
        # Wait if we've reached max parallel transfers
        while [[ $active_jobs -ge $MAX_PARALLEL ]]; do
            for i in "${!pids[@]}"; do
                if ! kill -0 "${pids[i]}" 2>/dev/null; then
                    wait "${pids[i]}"
                    unset "pids[i]"
                    ((active_jobs--))
                fi
            done
            sleep 5
        done
        
        # Start new transfer in background
        transfer_file "$file_path" &
        pids+=($!)
        ((active_jobs++))
        
        log "Started transfer job for $(basename "$file_path") (PID: $!)"
    done
    
    # Wait for all remaining jobs
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
}

# Function to validate migration
validate_migration() {
    log "Starting migration validation..."
    
    local total_files=0
    local successful_files=0
    local failed_files=0
    
    if [[ -f "$MIGRATION_MANIFEST" ]]; then
        total_files=$(wc -l < "$MIGRATION_MANIFEST")
        successful_files=$(grep ":SUCCESS:" "$MIGRATION_MANIFEST" | wc -l)
    fi
    
    if [[ -f "$FAILED_TRANSFERS" ]]; then
        failed_files=$(wc -l < "$FAILED_TRANSFERS")
    fi
    
    log "=== Migration Summary ==="
    log "Total files processed: $total_files"
    log "Successful transfers: $successful_files"
    log "Failed transfers: $failed_files"
    log "Success rate: $(( successful_files * 100 / (successful_files + failed_files) ))%"
    
    if [[ $failed_files -eq 0 ]]; then
        log "Migration completed successfully!"
        return 0
    else
        log "Migration completed with failures. Check: $FAILED_TRANSFERS"
        return 1
    fi
}

# Main migration execution
main() {
    log "=== Cloud Migration Started ==="
    log "Migration ID: $MIGRATION_ID"
    log "Source: $MIGRATION_SOURCE"
    log "Destination: $CLOUD_ENDPOINT"
    
    # Validate source directory
    if [[ ! -d "$MIGRATION_SOURCE" ]]; then
        log "ERROR: Source directory not found: $MIGRATION_SOURCE"
        exit 1
    fi
    
    # Generate file list
    log "Scanning source directory..."
    mapfile -t files < <(find "$MIGRATION_SOURCE" -type f -size +1M)  # Files larger than 1MB
    
    if [[ ${#files[@]} -eq 0 ]]; then
        log "No files found for migration"
        exit 0
    fi
    
    log "Found ${#files[@]} files for migration"
    
    # Calculate total size
    total_size=0
    for file in "${files[@]}"; do
        size=$(stat -c%s "$file")
        total_size=$((total_size + size))
    done
    total_size_gb=$(echo "scale=2; $total_size / 1024 / 1024 / 1024" | bc)
    log "Total data size: ${total_size_gb}GB"
    
    # Process migration
    start_time=$(date +%s)
    process_migration_batch "${files[@]}"
    end_time=$(date +%s)
    
    duration=$((end_time - start_time))
    duration_hours=$(echo "scale=2; $duration / 3600" | bc)
    log "Migration duration: ${duration_hours} hours"
    
    # Validate results
    validate_migration
    migration_status=$?
    
    log "=== Migration Complete ==="
    exit $migration_status
}

main "$@"
```

## 🔍 Cloud Provider Optimizations

### AWS Specific Configuration
```bash
# Optimized for AWS Transfer
# Use AWS Transfer Family or Direct Connect when available
jdc -file large-dataset.tar \
    -connect aws-transfer-endpoint.us-east-1.amazonaws.com:8000 \
    -chunk 8388608 \
    -workers 8 \
    -adaptive \
    -timeout 48h
```

### Azure Specific Configuration
```bash
# Optimized for Azure
# Consider Azure ExpressRoute for large migrations
jdc -file vm-image.vhd \
    -connect azure-migration.eastus.cloudapp.azure.com:8000 \
    -chunk 4194304 \
    -workers 6 \
    -adaptive \
    -compress=false \
    -timeout 36h
```

### Google Cloud Configuration
```bash
# Optimized for Google Cloud
# Use Cloud Storage Transfer Service for very large datasets
jdc -file database-backup.sql.gz \
    -connect gcp-migration.us-central1.googleusercontent.com:8000 \
    -chunk 4194304 \
    -workers 6 \
    -adaptive \
    -timeout 24h
```

## 📊 Cost Optimization Strategies

### Off-Peak Transfer Scheduling
```bash
#!/bin/bash
# Schedule transfers during off-peak hours for cost savings

# Check if we're in off-peak hours (example: 2 AM - 6 AM local time)
current_hour=$(date +%H)
if [[ $current_hour -ge 2 && $current_hour -lt 6 ]]; then
    echo "Off-peak hours detected, starting migration..."
    ./cloud-bulk-migration.sh
else
    echo "Peak hours, scheduling for off-peak..."
    echo "0 2 * * * /opt/migration/cloud-bulk-migration.sh" | crontab -
fi
```

### Bandwidth Throttling for Cost Control
```bash
# Limit bandwidth to control data transfer costs
jdc -file large-file.dat \
    -connect cloud-endpoint:8000 \
    -chunk 2097152 \
    -workers 2 \
    -adaptive \
    -max-delay 1000ms  # Throttle to reduce bandwidth usage
```

## 🔐 Security for Cloud Migration

### Encrypted Transfer Setup
```bash
# Example using SSH tunnel for additional encryption
ssh -L 8000:internal-migration-server:8000 cloud-gateway.provider.com &
SSH_PID=$!

# Transfer through encrypted tunnel
jdc -file sensitive-data.tar.gz.gpg \
    -connect localhost:8000 \
    -verify=true

# Cleanup
kill $SSH_PID
```

### Data Classification and Handling
```bash
# Different handling based on data sensitivity
case "$file_type" in
    "pii"|"financial")
        # High security: encrypted, verified, logged
        jdc -file "$file" -connect secure-endpoint:8443 \
            -verify=true -encrypt=true -log-level debug
        ;;
    "public"|"cached")
        # Standard transfer
        jdc -file "$file" -connect standard-endpoint:8000 \
            -verify=true
        ;;
esac
```

## 📈 Performance Monitoring

### Real-time Migration Dashboard
```bash
#!/bin/bash
# migration-dashboard.sh - Simple console dashboard

watch -n 30 '
echo "=== Cloud Migration Dashboard ==="
echo "Current transfers:"
ps aux | grep jdc | grep -v grep | wc -l

echo -e "\nTransfer progress:"
tail -5 /var/log/cloud-migration.log | grep -E "(SUCCESS|FAILED|Starting)"

echo -e "\nNetwork utilization:"
iftop -t -s 5 2>/dev/null | grep Total || echo "Install iftop for network stats"

echo -e "\nDisk space:"
df -h /data/migration-batch | tail -1

echo -e "\nMemory usage:"
free -h | grep Mem
'
```

## 🚨 Disaster Recovery

### Migration Rollback Plan
```bash
#!/bin/bash
# migration-rollback.sh

MIGRATION_ID="$1"
MANIFEST="/var/log/migration-manifest-${MIGRATION_ID}.txt"

if [[ -f "$MANIFEST" ]]; then
    echo "Rolling back migration: $MIGRATION_ID"
    
    # List successful transfers for rollback
    grep ":SUCCESS:" "$MANIFEST" | while IFS=: read -r file status timestamp size; do
        echo "Would rollback: $file (Size: $size bytes)"
        # Add actual rollback logic here
    done
else
    echo "Migration manifest not found: $MANIFEST"
    exit 1
fi
```

This comprehensive cloud migration example provides enterprise-grade data transfer capabilities with proper error handling, monitoring, and security considerations for large-scale cloud migrations.
