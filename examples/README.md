# JustDataCopier - Real-World Usage Examples

This directory contains practical examples and optimal configurations for various real-world scenarios using JustDataCopier.

## 📁 Example Scenarios

### 🏢 **Enterprise Data Center**
- [Database Backup Transfer](./database-backup/README.md) - Large database backups (100GB-2TB)
- [VM Image Migration](./vm-migration/README.md) - Virtual machine images and snapshots
- [Log Aggregation](./log-aggregation/README.md) - Collecting logs from multiple servers

### 🌐 **Remote Office Sync**
- [Branch Office Backup](./branch-office/README.md) - Daily backups over WAN links
- [Development Asset Sync](./dev-assets/README.md) - Syncing development resources
- [Media File Distribution](./media-distribution/README.md) - Large media files to remote locations

### ☁️ **Cloud & Hybrid**
- [Cloud Migration](./cloud-migration/README.md) - On-premises to cloud data transfer
- [Disaster Recovery](./disaster-recovery/README.md) - Emergency data replication
- [Cross-Region Sync](./cross-region/README.md) - Multi-region data synchronization

### 🔬 **Research & Development**
- [Scientific Data Transfer](./scientific-data/README.md) - Large datasets and research files
- [Build Artifact Distribution](./build-artifacts/README.md) - CI/CD pipeline integration
- [Model Training Data](./ml-datasets/README.md) - Machine learning datasets

## 🚀 Quick Configuration Guide

### Network Type Identification
```bash
# Test your network conditions first
ping -c 4 destination_server
iperf3 -c destination_server -t 10  # If available
```

### Optimal Settings by Network Type

| Network Type | Chunk Size | Buffer Size | Workers | Additional Flags |
|--------------|------------|-------------|---------|------------------|
| **Gigabit LAN** | 8MB | 1MB | 8 | `-verify=true` |
| **Fast Internet** | 4MB | 512KB | 4 | `-adaptive` |
| **Slow/Unstable** | 1MB | 256KB | 2 | `-adaptive -retries 10` |
| **High Latency** | 2MB | 512KB | 2 | `-adaptive -timeout 5m` |

## 📊 Performance Benchmarks

### Hash Algorithm Performance (Estimated)
| File Size | MD5 Time | BLAKE2b Time | SHA-256 Time | JDC Choice |
|-----------|----------|--------------|--------------|------------|
| 1GB | 30s | 45s | 60s | **MD5** ✓ |
| 50GB | 25min | 37min | 50min | **MD5** ✓ |
| 100GB | 50min | 74min | 100min | **BLAKE2b** ✓ |
| 500GB | 4.2h | 6.2h | 8.3h | **BLAKE2b** ✓ |
| 2TB | 16.7h | 24.8h | 33.3h | **BLAKE2b** ✓ |

## 🛠️ Common Troubleshooting

### Performance Issues
```bash
# Enable debug logging
jdc -file large.dat -connect server:8000 -log-level debug

# Network optimization
jdc -file large.dat -connect server:8000 -adaptive -workers 4

# For compression-friendly files
jdc -file logs.txt -connect server:8000 -compress
```

### Connection Problems
```bash
# Increase timeout for unstable connections
jdc -file large.dat -connect server:8000 -timeout 10m -retries 15

# Reduce chunk size for problematic networks
jdc -file large.dat -connect server:8000 -chunk 524288 -adaptive
```

### Memory Constraints
```bash
# Reduce buffer size for memory-limited systems
jdc -file large.dat -connect server:8000 -buffer 131072 -workers 2
```

## 📖 Best Practices

### Before Starting Large Transfers
1. **Test Connection**: Start with a small test file
2. **Check Space**: Ensure adequate disk space on both sides
3. **Network Stability**: Test during off-peak hours for critical transfers
4. **Backup Strategy**: Always have backups before large data operations

### Security Considerations
1. **Network Security**: Use VPN or secure networks for sensitive data
2. **Access Control**: Restrict server access to authorized clients only
3. **Monitoring**: Enable logging for audit trails
4. **Validation**: Always verify transfers with hash checking enabled

### Operational Excellence
1. **Monitoring**: Use structured logs for transfer tracking
2. **Automation**: Script repetitive transfers with error handling
3. **Documentation**: Record optimal settings for your environment
4. **Testing**: Validate configurations in non-production first

## 🔗 External Resources

- [Network Performance Testing Tools](./tools/network-testing.md)
- [Automation Scripts](./scripts/README.md)
- [Monitoring Setup](./monitoring/README.md)
- [Security Configuration](./security/README.md)

---

**Note**: All examples include sample data generation scripts and validation procedures. Adjust parameters based on your specific network conditions and requirements.
