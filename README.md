# JustDataCopier - Enterprise File Transfer Utility

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![Tests](https://img.shields.io/badge/Tests-Passing-green.svg)](https://github.com/yousafgill/just-data-copier)

JustDataCopier is a high-performance, enterprise-grade network file transfer utility designed for mission-critical environments. Built with Go 1.21+, it efficiently transfers files from MB to TB with intelligent hash algorithms, adaptive network optimization, and enterprise-level security.

## 🚀 Key Features

- **High-Performance Transfer**: Optimized parallel processing with configurable chunks up to 2TB+
- **Smart Hash Verification**: Auto-selects MD5 (<50GB) or BLAKE2b (≥50GB) for 35% faster processing
- **Intelligent Compression**: Automatic file type detection with optimized compression levels
- **Resume Capability**: Chunk-level precision resume with integrity verification
- **Network Adaptation**: Real-time profiling with automatic performance tuning
- **Enterprise Security**: Path validation, input sanitization, and structured error handling
- **Structured Logging**: JSON-based logging with session tracking and privacy focus
- **Modular Architecture**: Clean interfaces with dependency injection and comprehensive testing

## 📋 Requirements & Architecture

### System Requirements
- Go 1.21+ for building from source
- Network connectivity between client and server
- Adequate disk space for file transfers

### Architecture Overview
- **Modular Design**: Clean package structure with dependency injection
- **Binary Protocol**: Efficient communication with hash algorithm negotiation
- **Enterprise Quality**: Structured error types, security best practices, and operational excellence
- **Scalability**: Optimized for files from MB to TB+ with optimized memory usage

## 🚀 Quick Start

### Server Mode (Receiver)
```bash
jdc -server -output ./destination_folder
jdc -server -listen 192.168.1.10:9000 -output ./destination_folder
```

### Client Mode (Sender)
```bash
jdc -file ./my_large_file.dat -connect server_address:8000
jdc -file ./large_log_file.txt -connect server_address:8000 -compress
```

### Common Options
- `-compress`: Enable compression (optimized by file type)
- `-chunk <bytes>`: Chunk size (default: 2MB)
- `-buffer <bytes>`: Buffer size (default: 512KB)
- `-workers <num>`: Concurrent workers (default: half CPU cores)
- `-adaptive`: Enable adaptive network optimization
- `-timeout <duration>`: Operation timeout (default: 2m)

## 🔐 Security & Performance

### Security Features
- **Path Validation**: Directory traversal protection and input sanitization
- **Privacy Logging**: No sensitive file paths or hash values in logs
- **Structured Errors**: Categorized error types without sensitive details
- **Resource Management**: Bounded usage and proper cleanup

### Performance Optimization

#### Hash Algorithm Intelligence
| File Size | Algorithm | Performance Benefit |
|-----------|-----------|-------------------|
| < 50GB | MD5 | Fastest for smaller files |
| ≥ 50GB | BLAKE2b | 35% faster than SHA-256, secure |

#### Network Tuning Examples
```bash
# High-speed LAN (1Gbps+)
jdc -file ./file.dat -connect server:8000 -chunk 8388608 -workers 8

# Internet (100Mbps+)  
jdc -file ./file.dat -connect server:8000 -chunk 4194304 -workers 4 -adaptive

# High-latency Network
jdc -file ./file.dat -connect server:8000 -chunk 1048576 -adaptive
```

## 🔧 Advanced Capabilities

### Intelligent Features
- **Hash Selection**: Automatic MD5/BLAKE2b selection based on file size
- **Network Adaptation**: Real-time RTT, bandwidth, and packet loss monitoring
- **Smart Compression**: File type detection with adaptive compression levels
- **Resume Support**: Chunk-level precision resume with integrity verification

### Enterprise Monitoring
- **Structured Logging**: JSON-based with session tracking and security focus
- **Performance Metrics**: Real-time transfer statistics and network conditions
- **Error Categorization**: Clear classification without sensitive information
- **Operational Integration**: Compatible with enterprise monitoring systems

## 🏛️ Technical Implementation

### Core Technologies
- **Go 1.21+**: Modern performance with structured logging and modular design
- **Binary Protocol**: Efficient communication with hash algorithm negotiation
- **TCP Optimization**: Socket-level tuning for maximum throughput
- **Streaming Operations**: Memory-efficient processing for any file size

### Enterprise Quality
- **Error Handling**: Structured types (NetworkError, FileSystemError, etc.)
- **Security**: Input validation, path sanitization, and access control
- **Monitoring**: Comprehensive metrics and observability features
- **Production Ready**: Graceful shutdown, signal handling, and resource management

## 📚 Examples & Use Cases

For detailed real-world usage examples and optimal configurations for Windows environments, see the [examples directory](./examples/README.md):

- **[Database Backup Transfer](./examples/database-backup/README.md)** - Moving large database backups (100GB-2TB) between Windows servers
- **[Branch Office Backup](./examples/branch-office/README.md)** - Daily backups over internet connections with Windows batch scripts  
- **[Cloud Migration](./examples/cloud-migration/README.md)** - Large-scale data migration from Windows servers to cloud providers

Each example includes complete Windows batch files, PowerShell scripts, optimal configurations, monitoring setup, and troubleshooting guides written in easy-to-understand language.

## 📊 Monitoring

### Logging Features
- **JSON Format**: Enterprise-compatible structured logging
- **Session Tracking**: Unique IDs for complete activity correlation
- **Privacy Focus**: No sensitive paths or hash values in output
- **Performance Data**: Real-time metrics and network conditions

## 📄 License and Disclaimer

JustDataCopier is provided as free software under the MIT License, designed to help you achieve reliable and efficient file transfers.

### 🎯 **Positive Use Statement**
This enterprise-grade utility is built with care to provide robust, secure, and high-performance file transfer capabilities for your critical data operations. We're committed to delivering software that enhances your productivity and data management workflows.

### ⚠️ **Important Disclaimers**
While we've designed JustDataCopier with enterprise-grade reliability and extensive testing, please note:

- **Data Responsibility**: Users are solely responsible for backing up their data before transfers. Always maintain proper backups of critical files.
- **Usage Responsibility**: This software is provided "as is" without warranty. Users assume full responsibility for its proper configuration and use.
- **No Liability**: The author and contributors are not liable for any data loss, corruption, or damages resulting from the use or misuse of this software.
- **Testing Recommended**: Always test transfers with non-critical data first to ensure compatibility with your environment.
- **Network Security**: Users are responsible for ensuring secure network configurations and appropriate access controls.

### 🤝 **Professional Use Guidelines**
- **Attribution Required**: For integration into commercial products or services, proper attribution to the original author is required
- **Community Spirit**: This software embodies the open-source philosophy of shared knowledge and collaborative improvement
- **Enterprise Ready**: Includes comprehensive features suitable for mission-critical operations when properly configured and tested

### 📝 **Attribution Requirements**
- **Author**: Yousaf Gill <yousafgill@gmail.com>
- **Repository**: https://github.com/yousafgill/just-data-copier
- **Copyright**: © 2025 Yousaf Gill. All rights reserved.

*By using this software, you acknowledge that you have read, understood, and agree to these terms while appreciating the effort put into creating a powerful tool for your file transfer needs.*

## 🤝 Contributing

JustDataCopier welcomes contributions from the community to enhance its enterprise capabilities:

### Development Standards
- **Code Quality**: Follow Go best practices and maintain the modular architecture
- **Testing**: Include comprehensive unit tests for new features
- **Documentation**: Update documentation for any feature additions or changes
- **Performance**: Ensure changes maintain or improve transfer performance

### Contribution Areas
- **Performance Optimization**: Network algorithms, hash performance, compression improvements
- **Enterprise Features**: Enhanced monitoring, security features, operational capabilities
- **Platform Support**: Cross-platform compatibility and optimization
- **Protocol Enhancements**: Communication protocol improvements and new features

Please submit pull requests, create issues, or suggest improvements through the GitHub repository. All contributions help make JustDataCopier a better enterprise file transfer solution.