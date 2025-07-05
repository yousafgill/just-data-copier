# Branch Office Backup

This example shows how to efficiently transfer daily backups from branch offices over WAN links with varying quality.

## 📊 Scenario Overview

**Use Case**: Daily backup transfer from remote branch offices to central data center
- **File Sizes**: 10GB - 200GB compressed backups
- **Network**: Internet connection, 50-200Mbps, variable latency (20-100ms)
- **Challenges**: Network congestion, variable bandwidth, potential interruptions
- **Requirements**: Reliable transfer with resume capability, adaptive performance

## 🎯 Optimal Configuration

### Server Setup (Data Center)
```bash
# Central backup server with adaptive settings
jdc -server \
    -listen 0.0.0.0:8000 \
    -output /datacenter/branch-backups \
    -workers 4 \
    -buffer 524288 \
    -timeout 8h \
    -retries 10 \
    -log-level info
```

### Client Setup (Branch Office)
```bash
# Branch office backup transfer with adaptive optimization
jdc -file /backup/branch_backup_20250705.tar.gz \
    -connect datacenter.company.com:8000 \
    -chunk 2097152 \
    -buffer 524288 \
    -workers 3 \
    -adaptive \
    -compress=false \
    -verify=true \
    -timeout 8h \
    -retries 15 \
    -min-delay 5ms \
    -max-delay 500ms
```

## 🔧 Configuration Breakdown

### Network Adaptation
- **Chunk Size**: `2MB` - Balanced for variable bandwidth
- **Buffer Size**: `512KB` - Moderate size for internet connections
- **Workers**: `3` - Conservative to avoid overwhelming connection
- **Adaptive Mode**: `enabled` - Essential for variable WAN conditions

### Reliability Settings
- **Min/Max Delay**: `5ms-500ms` - Wide range for network adaptation
- **Retries**: `15` - Higher retry count for unstable connections
- **Timeout**: `8h` - Long timeout for overnight transfers

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
