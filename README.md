# Just Data Copier

A high-performance network file transfer utility written in Go, designed for efficient and reliable transfer of large files across networks.

## Overview

Just Data Copier (JDC) is a command-line utility that specializes in optimizing file transfers between networked systems. It's particularly useful for transferring large datasets, media files, backups, and any scenario where network efficiency and reliability are crucial.

## Features

- **High-Speed File Transfers**: Optimized for transferring large files over networks with maximum efficiency
- **Multi-threaded Design**: Parallel chunk transfers for maximum throughput, using configurable worker threads
- **Adaptive Network Handling**: Dynamically adjusts to network conditions for optimal performance
- **Smart Compression**: Automatically compresses compressible files and skips already compressed formats
- **Resume Support**: Ability to resume interrupted transfers without starting over
- **File Integrity Verification**: Built-in MD5 verification ensures data integrity
- **Configurable Performance**: Fine-tune settings based on your network environment
- **Network Auto-profiling**: Automatically detects network characteristics and optimizes transfer parameters
- **TCP Optimization**: Implements various TCP-level optimizations for increased performance

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

## Performance Tuning

For optimal performance, you can adjust these parameters based on your specific network environment:

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

## Advanced Features

### Automatic Network Profiling

JDC automatically profiles the network at the beginning of a transfer to optimize:
- Round-trip time (RTT) measurement
- Optimal chunk size calculation
- Bandwidth estimation
- Worker thread allocation

### Intelligent Compression

The compression feature intelligently:
- Compresses text files and other highly compressible formats using gzip
- Skips compression for already compressed formats (images, videos, archives)
- Adapts compression levels based on file type (better compression for text files, faster compression for binary)
- Shows compression ratios during transfer

### Adaptive Networking

When enabled with `-adaptive`, monitors network performance and adjusts chunk delays for optimal throughput:
- Automatically reduces send rate when network congestion is detected
- Increases send rate when network conditions improve
- Configurable with `-min-delay` and `-max-delay`
- Provides real-time feedback about network conditions
- Disabled by default for more predictable behavior

### Transfer Resume Capability

If a transfer is interrupted:
- The partial state is saved in a `.justdatacopier.state` file
- When restarting the transfer with the same parameters, only missing chunks are transferred
- State files are automatically cleaned up after successful transfers

## Technical Details

### Architecture

Just Data Copier uses a client-server architecture:

1. **Server Mode**: 
   - Listens for incoming connections
   - Manages output directory and file creation
   - Handles chunk requests and reassembly
   - Verifies file integrity upon completion

2. **Client Mode**:
   - Establishes connection to the server
   - Profiles network conditions
   - Sends file metadata (name, size)
   - Manages chunking and sending of file data
   - Provides hash for verification

### Implementation Details

- **Chunking**: Files are split into configurable-sized chunks for parallel processing
- **TCP Optimizations**: Sets TCP_NODELAY, larger buffers, and keep-alive for optimized transmission
- **Context-based Cancellation**: Uses Go contexts for proper timeout handling and cancellation
- **Buffered I/O**: Employs bufio for efficient reading and writing
- **Error Handling**: Implements retry mechanisms for transient failures
- **Logging**: Provides detailed logs for monitoring and troubleshooting

### Logging

Log files are stored in the `logs/` directory with timestamps:
```
logs/justdatacopier_YYYYMMDD_HHMMSS.log
```

## License and Usage Disclaimer

Just Data Copier is provided as free software for non-commercial use. It is licensed under the MIT License.

**Disclaimer**:
- This utility is provided "as is" without warranty of any kind
- If you use this software in your projects or services, proper attribution to the original author is required
- For commercial use, please contact the author

**Attribution**:
- Author: Yousaf Gill <yousafgill@gmail.com>
- Repository: https://github.com/yousafgill/just-data-copier
- Copyright © 2025 Yousaf Gill. All rights reserved.

## Contributing

Contributions to improve Just Data Copier are welcome. Please feel free to submit pull requests, create issues, or suggest improvements to the GitHub repository.
