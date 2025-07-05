# Database Backup Transfer

This example demonstrates optimal configuration for transferring large database backups in enterprise environments.

## 📊 Scenario Overview

**Use Case**: Nightly database backup transfer from production server to backup facility
- **File Sizes**: 100GB - 2TB database dumps
- **Network**: Dedicated 1Gbps link with 5-10ms latency
- **Frequency**: Daily automated transfers
- **Requirements**: High reliability, integrity verification, monitoring

## 🎯 Optimal Configuration

### Server Setup (Backup Facility)
```bash
# Start server with optimal settings for large files
jdc -server \
    -listen 0.0.0.0:8000 \
    -output /backup/databases \
    -workers 8 \
    -buffer 1048576 \
    -timeout 6h \
    -log-level info
```

### Client Setup (Production Server)
```bash
# Transfer large database backup
jdc -file /var/backups/prod_db_20250705.sql.gz \
    -connect backup-server:8000 \
    -chunk 8388608 \
    -buffer 1048576 \
    -workers 8 \
    -compress=false \
    -verify=true \
    -timeout 6h \
    -retries 5
```

## 🔧 Configuration Breakdown

### Network Optimization
- **Chunk Size**: `8MB` - Optimal for high-bandwidth, low-latency networks
- **Buffer Size**: `1MB` - Large buffers for sustained throughput
- **Workers**: `8` - Maximizes parallelism on gigabit connection
- **Compression**: `false` - Database dumps are usually pre-compressed

### Reliability Settings
- **Verify**: `true` - Critical for database integrity
- **Timeout**: `6h` - Generous timeout for 2TB files
- **Retries**: `5` - Automatic retry for transient failures

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
