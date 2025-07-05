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
- **Size-Based Selection**: Automatic algorithm selection for optimal performance
  - Files < 50GB: MD5 (fastest for smaller files)
  - Files ≥ 50GB: BLAKE2b (35% faster than SHA-256, cryptographically secure)
- **Protocol Negotiation**: Automatic algorithm exchange between client and server
- **Backward Compatibility**: Seamless operation with legacy systems
- **Performance Benefits**: Up to 35% faster hash calculation for large files while maintaining security

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

JustDataCopier uses size-based hash algorithm selection to optimize performance:

- **Automatic Selection**: Based on file size for optimal speed/security balance
- **Protocol Negotiation**: Server determines algorithm and negotiates with client  
- **Backward Compatibility**: Works seamlessly with legacy systems
- **Memory Efficient**: Streaming calculation with configurable buffers
- **Performance Optimized**: Up to 35% faster for large files while maintaining security

### Enterprise Logging & Monitoring

#### Structured JSON Logging
- **Session Tracking**: Unique session IDs for correlation across components
- **Network Metrics**: Real-time RTT, bandwidth estimation, and packet loss monitoring
- **Transfer Metrics**: Chunk-level progress, transfer rates, and throughput statistics
- **Security-Focused**: No sensitive file paths or hash values in log output
- **Quality Assessment**: Automatic network quality rating (fair/good/excellent)

#### Log Information Includes
- Session initialization and completion statistics
- Network performance metrics and quality assessment
- Transfer progress with chunk-level detail
- Configuration parameters and optimization settings
- Error categorization without sensitive information

### Modular Architecture Benefits

#### Code Quality
- **Modular Design**: 10+ focused packages with single responsibilities
- **Interface-Based**: Dependency injection for testability and maintainability  
- **Comprehensive Testing**: Unit tests with testify framework
- **Go Best Practices**: Modern Go 1.21+ with proper error handling

#### Operational Excellence
- **Production Ready**: Enterprise-grade operational characteristics
- **Resource Management**: Proper cleanup and graceful shutdown
- **Configuration**: Centralized config with validation and defaults
- **Monitoring**: Full observability with metrics and structured logging

### Network Profiling & Adaptation

JDC automatically profiles network conditions and adapts performance:
- **Round-trip Time (RTT)**: Continuous measurement for latency optimization
- **Bandwidth Estimation**: Real-time throughput calculation and adjustment
- **Packet Loss Detection**: Monitoring with automatic quality assessment
- **Adaptive Delays**: Dynamic adjustment based on network performance
- **Quality Rating**: Automatic classification (fair/good/excellent) with responsive tuning

### Smart Compression System

The compression feature provides intelligent optimization:
- **File Type Detection**: Automatically identifies compressible vs pre-compressed content
- **Adaptive Compression Levels**: Optimized levels based on file characteristics
- **Performance Metrics**: Real-time compression ratio reporting
- **Selective Compression**: Skips compression for already compressed formats (images, videos, archives)
- **Text Optimization**: Enhanced compression for logs, configuration files, and text data

### Adaptive Networking

When enabled with `-adaptive`, monitors network performance and adjusts chunk delays for optimal throughput:
- Automatically reduces send rate when network congestion is detected
- Increases send rate when network conditions improve
- Configurable with `-min-delay` and `-max-delay`
- Provides real-time feedback about network conditions
- Disabled by default for more predictable behavior

### High-Performance Transfer Resume

Advanced resume capability with enterprise features:
- **Chunk-Level Precision**: Resume from exact point of interruption
- **State Persistence**: Secure state file management with automatic cleanup
- **Integrity Verification**: Validates resumed transfers with hash verification
- **Session Recovery**: Maintains session context across resume operations
- **Progress Preservation**: Continues progress tracking from interruption point

## 🏛️ Technical Architecture

### Enterprise Client-Server Design

JustDataCopier implements a robust client-server architecture optimized for enterprise environments:

1. **Server Mode (Receiver)**:
   - Multi-client connection handling with concurrent session management
   - Intelligent hash algorithm selection based on file characteristics
   - Chunk reassembly with integrity verification and progress tracking
   - Security-focused file management with path validation
   - Session-based logging with comprehensive metrics

2. **Client Mode (Sender)**:
   - Automatic network profiling and performance optimization
   - Hash algorithm negotiation with server for optimal performance
   - Parallel chunk transmission with adaptive network handling
   - Real-time progress reporting and transfer statistics
   - Resume capability with state persistence and recovery

### Modern Implementation Standards

#### Core Technologies
- **Go 1.21+**: Modern Go with generics and performance improvements
- **Structured Logging**: JSON-based logging with `log/slog` for enterprise monitoring
- **Context-Driven**: Context-aware operations for proper cancellation and timeouts
- **Interface Design**: Clean interfaces for testing, mocking, and extensibility
- **Binary Protocol**: Efficient binary communication with hash algorithm negotiation

#### Performance Engineering
- **Streaming Operations**: Memory-efficient processing for files of any size
- **TCP Optimization**: Socket-level tuning including TCP_NODELAY and buffer optimization
- **Parallel Processing**: Multi-worker architecture with dynamic worker adjustment
- **Buffer Management**: Configurable buffering strategies for different network conditions
- **Resource Efficiency**: Minimal memory footprint with proper resource cleanup

#### Enterprise Quality
- **Error Categorization**: Structured error types (NetworkError, FileSystemError, etc.)
- **Security Best Practices**: Input validation, path sanitization, and access control
- **Monitoring Integration**: Comprehensive metrics and observability features
- **Production Readiness**: Graceful shutdown, signal handling, and operational excellence

### Scalability & Performance

#### File Size Support
- **Small Files (MB)**: Optimized with MD5 for speed
- **Large Files (GB)**: Balanced performance with intelligent chunking
- **Enterprise Files (TB+)**: BLAKE2b optimization for 35% faster processing
- **Memory Efficiency**: Constant memory usage regardless of file size

#### Network Adaptability
- **LAN Optimization**: High-throughput settings for local network transfers
- **WAN Adaptation**: Intelligent handling of latency and bandwidth constraints
- **Quality Assessment**: Real-time network condition monitoring and adaptation
- **Congestion Handling**: Automatic rate adjustment based on network performance

## 📊 Monitoring & Observability

### Structured Logging System

JustDataCopier provides enterprise-grade logging with comprehensive observability:

#### Session-Based Tracking
- **Unique Session IDs**: Correlate activities across client-server communication
- **Session Lifecycle**: Complete tracking from initialization to completion
- **Transfer Metrics**: Real-time progress, throughput, and performance statistics
- **Configuration Logging**: Parameter settings and optimization choices

#### Network Performance Monitoring
- **Real-time Metrics**: Round-trip time, bandwidth estimation, packet loss percentage
- **Quality Assessment**: Automatic network quality rating (fair/good/excellent)
- **Performance Adaptation**: Dynamic adjustment logging based on network conditions
- **Transfer Statistics**: Chunk-level progress with rate and efficiency metrics

#### Security & Privacy Focus
- **No Sensitive Data**: File paths and hash values excluded from logs
- **Error Classification**: Structured error types without sensitive details
- **Clean Output**: Professional log format suitable for enterprise monitoring
- **Session Context**: All activities traceable through session identifiers

### Log Integration

#### Format & Structure
- **JSON-Based**: Structured logging compatible with enterprise log management systems
- **Timestamped Entries**: Precise timing information for performance analysis
- **Categorized Levels**: DEBUG, INFO, WARN, ERROR with appropriate detail levels
- **Metric Rich**: Performance data, network conditions, and transfer progress

#### Operational Benefits
- **Troubleshooting**: Clear error categorization and context for issue resolution
- **Performance Analysis**: Historical data for network and transfer optimization
- **Monitoring Integration**: Compatible with enterprise monitoring and alerting systems
- **Audit Trail**: Complete session tracking for compliance and operational review

## 📄 License and Attribution

JustDataCopier is provided as free software for non-commercial use under the MIT License.

**Professional Use Disclaimer**:
- This enterprise-grade utility is provided "as is" with comprehensive features for production environments
- For integration into commercial products or services, proper attribution to the original author is required
- The software includes enterprise features suitable for mission-critical file transfer operations

**Attribution Requirements**:
- Author: Yousaf Gill <yousafgill@gmail.com>
- Repository: https://github.com/yousafgill/just-data-copier
- Copyright © 2025 Yousaf Gill. All rights reserved.

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
