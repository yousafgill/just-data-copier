# JustDataCopier - Real-World Usage Examples for Windows

This folder contains practical examples and best configurations for various real-world scenarios using JustDataCopier on Windows systems.

## 📁 Common Scenarios

### 🏢 **Business & Enterprise**
- [Database Backup Transfer](./database-backup/README.md) - Moving large database backups (100GB-2TB)


### 🌐 **Remote Office & Branches**
- [Branch Office Backup](./branch-office/README.md) - Daily backups over internet connections


## 🚀 Quick Setup Guide

### Find Out Your Network Speed
```cmd
# Test your internet connection first
ping google.com
# For detailed speed test, use online tools like speedtest.net
```

### Best Settings for Different Connections

| Connection Type | Chunk Size | Buffer Size | Workers | Extra Options |
|----------------|------------|-------------|---------|---------------|
| **Office LAN (Fast)** | 8MB | 1MB | 8 | `-verify` |
| **Good Internet** | 4MB | 512KB | 4 | `-adaptive` |
| **Slow Internet** | 1MB | 256KB | 2 | `-adaptive -retries 10` |
| **Unstable Connection** | 2MB | 512KB | 2 | `-adaptive -timeout 5m` |

**Note**: `-verify` enables hash verification for file integrity (both client and server must enable it).

## 📊 How Fast Will It Be?

### File Transfer Speed Examples (Rough Estimates)
| File Size | Over Fast LAN | Over Good Internet | Over Slow Internet |
|-----------|---------------|-------------------|-------------------|
| 1GB | ~1-2 minutes | ~5-10 minutes | ~15-30 minutes |
| 10GB | ~10-20 minutes | ~45-90 minutes | ~2-5 hours |
| 100GB | ~2-3 hours | ~8-15 hours | ~1-2 days |
| 1TB | ~1-2 days | ~3-7 days | ~1-2 weeks |

*These are rough estimates - your actual speed depends on your specific network*

## 🛠️ Common Problems & Solutions

### Transfer Going Slow?
```cmd
# Try reducing the number of workers
jdc.exe -file large.dat -connect server:8000 -workers 2

# Or use adaptive mode
jdc.exe -file large.dat -connect server:8000 -adaptive
```

### Connection Keeps Dropping?
```cmd
# Increase timeout and retries
jdc.exe -file large.dat -connect server:8000 -timeout 10m -retries 15

# Use smaller chunks for bad connections
jdc.exe -file large.dat -connect server:8000 -chunk 524288 -adaptive
```

### Computer Running Out of Memory?
```cmd
# Use smaller buffers
jdc.exe -file large.dat -connect server:8000 -buffer 131072 -workers 2
```

## 📖 Things to Remember

### Before Starting Big Transfers
1. **Test First**: Always try a small file first to make sure everything works
2. **Check Space**: Make sure you have enough disk space on both computers
3. **Good Timing**: Start big transfers when the network isn't busy (like at night)
4. **Have Backups**: Always keep backups of important files before moving them

### Keep Your Data Safe
1. **Secure Networks**: Use VPN or secure networks for sensitive files
2. **Limit Access**: Only let authorized people connect to your transfer server
3. **Watch the Logs**: Keep an eye on transfer logs to spot problems
4. **Test Everything**: Try your setup with test files before using it for real work

### Make It Work Better
1. **Monitor Transfers**: Keep track of how transfers are going
2. **Script Repetitive Tasks**: Use batch files for transfers you do often
3. **Document Settings**: Write down what settings work best for your network
4. **Test Changes**: Always test new settings in a safe environment first


**Note**: All examples work on Windows and include sample batch files and validation steps. Adjust the settings based on your actual network speed and conditions.

## 📖 Quick Command Reference

### Basic Commands
```cmd
# Server Mode (receive files)
jdc.exe -server -output "D:\ReceivedFiles"

# Client Mode (send files)
jdc.exe -file "myfile.dat" -connect server-ip:8000

# With hash verification (both sides must enable)
# Server:
jdc.exe -server -output "D:\ReceivedFiles" -verify

# Client:
jdc.exe -file "myfile.dat" -connect server-ip:8000 -verify
```

### Most Common Options
```cmd
-verify                    # Enable hash verification (recommended for important files)
-chunk <size>              # Chunk size: 1048576 (1MB), 2097152 (2MB), 4194304 (4MB), 8388608 (8MB)
-workers <number>          # Number of parallel transfers: 2, 4, 8
-buffer <size>             # Buffer size: 262144 (256KB), 524288 (512KB), 1048576 (1MB)
-adaptive                  # Automatically adjust to network conditions (good for internet)
-compress                  # Enable compression (good for text files, bad for ZIP/videos)
-timeout <duration>        # How long to wait: 2m, 5m, 1h, 6h
-retries <number>          # How many times to retry: 5, 10, 15
-progress                  # Show progress bar (default: on)
```

### Size Reference
```cmd
# Chunk sizes:
1048576    = 1MB    (slow/unstable internet)
2097152    = 2MB    (good internet, default)
4194304    = 4MB    (fast internet)
8388608    = 8MB    (LAN/very fast connections)

# Buffer sizes:
262144     = 256KB  (slow connections)
524288     = 512KB  (normal connections, default)
1048576    = 1MB    (fast connections)
```
