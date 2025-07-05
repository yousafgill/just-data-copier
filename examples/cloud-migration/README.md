# Cloud Migration Data Transfer

This example demonstrates efficient data migration from on-premises infrastructure to cloud environments using JustDataCopier.

## 📊 Scenario Overview

**Use Case**: Large-scale data migration from on-premises datacenter to cloud (AWS/Azure/GCP)
- **Data Types**: VM images, databases, file shares, application data
- **File Sizes**: 500GB - 10TB per migration batch
- **Network**: Internet connection with cloud provider, variable bandwidth
- **Requirements**: Cost-effective transfer, minimal downtime, data integrity

## 🎯 Optimal Configuration

### Cloud Receiver Setup (Cloud Instance)
```bash
# High-performance cloud instance setup
jdc -server \
    -listen 0.0.0.0:8000 \
    -output /mnt/migration-staging \
    -workers 8 \
    -buffer 2097152 \
    -timeout 24h \
    -retries 20 \
    -log-level info
```

### On-Premises Sender Setup
```bash
# On-premises to cloud transfer with optimization
jdc -file /data/vm-images/production-vm-001.vmdk \
    -connect cloud-migration.region.provider.com:8000 \
    -chunk 4194304 \
    -buffer 1048576 \
    -workers 6 \
    -adaptive \
    -compress=false \
    -verify=true \
    -timeout 24h \
    -retries 25 \
    -min-delay 10ms \
    -max-delay 2000ms
```

## 🔧 Migration Strategy

### Phased Migration Approach
1. **Assessment Phase**: Network testing and capacity planning
2. **Pilot Migration**: Small dataset to validate configuration
3. **Bulk Migration**: Large datasets during off-peak hours
4. **Validation Phase**: Comprehensive data integrity verification

### Network Optimization for Cloud
- **Chunk Size**: `4MB` - Optimal for internet to cloud transfers
- **Buffer Size**: `1MB` - Large buffers for sustained cloud connectivity
- **Workers**: `6` - Balanced for internet bandwidth utilization
- **Adaptive Mode**: Essential for variable internet performance

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
