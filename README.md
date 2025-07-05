# JustDataCopier - Enterprise File Transfer Utility

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![Tests](https://img.shields.io/badge/Tests-Passing-green.svg)](https://github.com/yousafgill/just-data-copier)

JustDataCopier is a high-performance, enterprise-grade network file transfer utility designed for mission-critical environments. Built with Go 1.21+, it efficiently transfers files of any size (from MB to TB) across network connections with features like intelligent hash algorithms, adaptive network optimization, structured logging, and enterprise-level security.

## 🚀 Features

### Core Functionality
- **High-Performance Transfer**: Optimized for large file transfers with configurable chunk sizes up to 2TB+
- **Parallel Processing**: Multi-threaded transfers with configurable worker count and dynamic optimization
- **Smart Hash Verification**: Automatic algorithm selection (MD5 for <50GB, BLAKE2b for ≥50GB) for optimal speed and security
- **Intelligent Compression**: Gzip compression with automatic file type detection and optimized compression levels
- **Resume Capability**: Transfer state management for resuming interrupted transfers with chunk-level precision
- **Network Adaptation**: Real-time network condition monitoring with automatic performance tuning

### Enterprise Features
- **Structured Logging**: JSON-based logging with session tracking, network metrics, and security-focused output
- **Modular Architecture**: Clean package structure with dependency injection and interface-based design
- **Advanced Error Handling**: Comprehensive error categorization with NetworkError, FileSystemError, ProtocolError types
- **Configuration Management**: Centralized configuration with validation and sensible defaults
- **Security Enhancement**: Path validation, input sanitization, and directory traversal protection
- **Monitoring & Observability**: Real-time progress reporting, transfer statistics, and performance metrics
- **Graceful Operations**: Signal handling for clean application termination and resource cleanup

### Performance Optimizations
- **Automatic Network Profiling**: Real-time RTT measurement, bandwidth estimation, and packet loss detection
- **Adaptive Chunking**: Dynamic chunk size adjustment based on network performance and file characteristics
- **TCP Optimization**: Socket-level optimizations including TCP_NODELAY and buffer tuning
- **Memory Efficiency**: Streaming operations with configurable buffer management for optimal memory usage
- **Quality Assessment**: Automatic network quality rating (fair/good/excellent) with adaptive responses

### Hash Algorithm Intelligence
- **Size-Based Selection**: MD5 for files <50GB, BLAKE2b for files ≥50GB (35% faster than SHA-256)
- **Protocol Negotiation**: Automatic algorithm exchange between client and server
- **Backward Compatibility**: Seamless operation with legacy systems

## 📋 Requirements

- Go 1.21 or higher
- Network connectivity between client and server
- Adequate disk space for file transfers

## 🏗️ Architecture

JustDataCopier features a modern, modular architecture with the following components:

### Package Structure
- **Client/Server**: Separate client and server logic with clean interfaces
- **Protocol**: Binary protocol with hash algorithm negotiation
- **Network**: Adaptive network optimization and performance monitoring
- **Filesystem**: File operations with streaming hash calculation
- **Logging**: Structured logging with session tracking and security focus
- **Compression**: Smart compression with file type detection
- **Configuration**: Centralized configuration management with validation
- **Error Handling**: Structured error types with proper categorization

### Enterprise-Grade Features
- **Dependency Injection**: Clean dependencies between modules for testability
- **Interface-Based Design**: Maintainable and extensible code structure
- **Context Support**: Context-aware operations throughout the application
- **Testing Framework**: Comprehensive unit tests with testify integration
- **Production Ready**: Enterprise operational characteristics with monitoring

## Usage

JDC operates in two modes: server (receiver) and client (sender). The executable name is `jdc` (or `jdc.exe` on Windows).

### Server Mode

Start the server to receive files:

```bash
jdc -server -output ./destination_folder
```

By default, the server listens on all interfaces (0.0.0.0) port 8000. You can specify a different listening address:

```bash
jdc -server -listen 192.168.1.10:9000 -output ./destination_folder
```

### Client Mode

Send a file to a server:

```bash
jdc -file ./my_large_file.dat -connect server_address:8000
```

For improved performance with text files, enable compression:

```bash
jdc -file ./large_log_file.txt -connect server_address:8000 -compress
```

### Common Options

- `-compress`: Enable on-the-fly compression (optimized by file type)
- `-chunk <bytes>`: Set chunk size in bytes (default: 2MB)
- `-buffer <bytes>`: Set buffer size in bytes (default: 512KB)
- `-workers <num>`: Set number of concurrent workers (default: half of CPU cores)
- `-verify <bool>`: Enable/disable file integrity verification (default: enabled)
- `-adaptive`: Use adaptive delay based on network conditions (default: disabled)
- `-progress <bool>`: Show/hide progress during transfer (default: enabled)
- `-timeout <duration>`: Set operation timeout (default: 2m)
- `-retries <num>`: Set number of retries for failed operations (default: 5)

## 🔐 Security & Reliability

### Security Features
- **Path Validation**: Comprehensive directory traversal protection
- **Input Sanitization**: Safe handling of all user inputs and file paths
- **Privacy-Focused Logging**: No sensitive file paths or hash values in logs
- **Resource Management**: Bounded resource usage and proper cleanup
- **File Permissions**: Appropriate file creation permissions and access controls

### Error Handling & Recovery
- **Structured Error Types**: NetworkError, FileSystemError, ProtocolError, CompressionError, ValidationError
- **Error Context**: Proper error wrapping with context preservation
- **Graceful Degradation**: Robust error recovery mechanisms
- **User-Friendly Messages**: Clear error communication without sensitive details
- **Retry Logic**: Intelligent retry mechanisms for transient failures

### Data Integrity
- **Intelligent Hash Selection**: Automatic selection of optimal hash algorithms
- **Streaming Verification**: Memory-efficient hash calculation for files of any size
- **Protocol Integrity**: Built-in protocol validation and error detection
- **Transfer Validation**: End-to-end verification of data integrity

## ⚡ Performance Tuning

### Hash Algorithm Optimization
JustDataCopier automatically selects the optimal hash algorithm based on file size:

| File Size | Algorithm | Performance Benefit |
|-----------|-----------|-------------------|
| < 50GB | MD5 | Fastest for smaller files |
| ≥ 50GB | BLAKE2b | 35% faster than SHA-256, cryptographically secure |

For a 2TB file, BLAKE2b reduces hash calculation time from ~60 minutes (SHA-256) to ~35 minutes while maintaining cryptographic security.

### Network Optimization
For optimal performance, adjust these parameters based on your network environment:

- **Chunk Size**: Larger chunks (e.g., `-chunk 4194304` for 4MB) can improve performance on reliable networks with low latency
- **Buffer Size**: Adjust buffer size with `-buffer` for different network conditions:
  - Larger buffers (e.g., `-buffer 1048576` for 1MB) can help on high-bandwidth networks
  - Smaller buffers may work better on congested networks
- **Workers**: More workers (e.g., `-workers 8`) can improve throughput on high-bandwidth connections
- **Delay Settings**: Fine-tune adaptive delay with `-min-delay` and `-max-delay` parameters
- **Compression**: Enable compression (`-compress`) for text files, logs, and other compressible data

### Examples for Different Network Types:

#### High-speed LAN (1Gbps+):
```bash
jdc -file ./large_file.dat -connect server:8000 -chunk 8388608 -buffer 1048576 -workers 8
```

#### Internet Connection (100Mbps+):
```bash
jdc -file ./large_file.dat -connect server:8000 -chunk 4194304 -buffer 524288 -workers 4 -adaptive
```

#### Unstable or High-latency Network:
```bash
jdc -file ./large_file.dat -connect server:8000 -chunk 1048576 -buffer 262144 -adaptive -min-delay 10ms -max-delay 200ms
```

## 🔧 Advanced Features

### Intelligent Hash Algorithm Selection
- **Automatic Selection**: MD5 for files <50GB, BLAKE2b for files ≥50GB
- **Performance**: Up to 35% faster hash calculation for large files
- **Memory Efficient**: Streaming calculation with configurable buffers

### Enterprise Logging & Monitoring
- **Session Tracking**: Unique session IDs for correlation across components
- **Network Metrics**: Real-time RTT, bandwidth estimation, and packet loss monitoring
- **Security-Focused**: No sensitive file paths or hash values in log output
- **Quality Assessment**: Automatic network quality rating (fair/good/excellent)

### Network Profiling & Adaptation
- **Continuous Monitoring**: RTT measurement, bandwidth estimation, packet loss detection
- **Adaptive Delays**: Dynamic adjustment based on network performance
- **Quality Rating**: Automatic classification with responsive tuning

### Smart Compression & Resume
- **File Type Detection**: Automatically identifies compressible vs pre-compressed content
- **Adaptive Compression**: Optimized levels based on file characteristics
- **Chunk-Level Resume**: Resume from exact point of interruption with integrity verification

## 🏛️ Technical Architecture

JustDataCopier implements a robust client-server architecture with:

### Core Technologies
- **Go 1.21+**: Modern Go with performance improvements and structured logging
- **Binary Protocol**: Efficient communication with hash algorithm negotiation
- **Modular Design**: 10+ focused packages with dependency injection
- **Enterprise Quality**: Structured error types, security best practices, and operational excellence

### Performance & Scalability
- **File Size Support**: Optimized for files from MB to TB+ with constant memory usage
- **Network Adaptability**: Intelligent handling of LAN, WAN, and congested networks
- **Streaming Operations**: Memory-efficient processing regardless of file size
- **TCP Optimization**: Socket-level tuning for maximum performance

## 📊 Monitoring & Observability

JustDataCopier provides enterprise-grade logging and monitoring:

### Structured Logging
- **JSON-Based**: Compatible with enterprise log management systems
- **Session Tracking**: Unique session IDs for complete activity correlation
- **Security Focus**: No sensitive file paths or hash values in output
- **Performance Metrics**: Real-time transfer statistics and network conditions

### Operational Benefits
- **Troubleshooting**: Clear error categorization for quick issue resolution
- **Performance Analysis**: Historical data for optimization
- **Enterprise Integration**: Compatible with monitoring and alerting systems
- **Audit Trail**: Complete session tracking for compliance

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
