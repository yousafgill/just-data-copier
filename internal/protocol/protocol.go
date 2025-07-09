package protocol

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"justdatacopier/internal/errors"
)

// Protocol version for future compatibility
const (
	ProtocolVersion = 1
)

// Command operation codes
const (
	CmdInit      = 1  // Initialize transfer
	CmdRequest   = 2  // Request chunk
	CmdData      = 3  // Send chunk data
	CmdComplete  = 4  // Transfer complete
	CmdError     = 5  // Error occurred
	CmdHash      = 6  // File hash for verification
	CmdHashAlgo  = 7  // Hash algorithm exchange
	CmdPing      = 8  // Ping for network profiling
	CmdPong      = 9  // Pong response to ping
	CmdVersion   = 10 // Protocol version negotiation
	CmdResume    = 11 // Resume information
	CmdResumeAck = 12 // Resume acknowledgment
)

// Hash algorithm types
type HashAlgorithm string

const (
	HashMD5     HashAlgorithm = "md5"
	HashSHA256  HashAlgorithm = "sha256"
	HashBLAKE2b HashAlgorithm = "blake2b"
)

// Size thresholds for hash algorithm selection
const (
	LargeFileSizeThreshold = 50 * 1024 * 1024 * 1024 // 50GB in bytes
)

// Message represents a protocol message
type Message struct {
	Command byte
	Data    []byte
}

// ReadCommand reads a command byte from the reader with context support
func ReadCommand(ctx context.Context, reader *bufio.Reader) (byte, error) {
	return readByteWithContext(ctx, reader)
}

// SendCommand sends a command byte to the writer
func SendCommand(writer *bufio.Writer, cmd byte) error {
	if err := writer.WriteByte(cmd); err != nil {
		return errors.NewProtocolError("send_command", fmt.Sprintf("failed to send command %d", cmd), err)
	}
	return nil
}

// ReadString reads a newline-terminated string with context support
func ReadString(ctx context.Context, reader *bufio.Reader) (string, error) {
	str, err := readStringWithContext(ctx, reader, '\n')
	if err != nil {
		return "", errors.NewProtocolError("read_string", "failed to read string", err)
	}
	// Remove newline
	if len(str) > 0 && str[len(str)-1] == '\n' {
		str = str[:len(str)-1]
	}
	return strings.TrimSpace(str), nil
}

// SendString sends a newline-terminated string
func SendString(writer *bufio.Writer, str string) error {
	if _, err := writer.WriteString(str + "\n"); err != nil {
		return errors.NewProtocolError("send_string", "failed to send string", err)
	}
	return nil
}

// ReadInt64 reads an int64 value as a string
func ReadInt64(ctx context.Context, reader *bufio.Reader) (int64, error) {
	str, err := ReadString(ctx, reader)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, errors.NewProtocolError("read_int64", fmt.Sprintf("invalid integer: %s", str), err)
	}
	return val, nil
}

// SendInt64 sends an int64 value as a string
func SendInt64(writer *bufio.Writer, val int64) error {
	return SendString(writer, strconv.FormatInt(val, 10))
}

// SendError sends an error message to the peer
func SendError(writer *bufio.Writer, message string) error {
	if err := SendCommand(writer, CmdError); err != nil {
		return err
	}
	if err := SendString(writer, message); err != nil {
		return err
	}
	return writer.Flush()
}

// FlushWriter flushes the writer buffer
func FlushWriter(writer *bufio.Writer) error {
	if err := writer.Flush(); err != nil {
		return errors.NewProtocolError("flush", "failed to flush writer", err)
	}
	return nil
}

// ResumeInfo represents resume information exchanged between client and server
type ResumeInfo struct {
	CanResume       bool   `json:"can_resume"`
	ResumeOffset    int64  `json:"resume_offset"`
	CompletedChunks []bool `json:"completed_chunks"`
	TotalChunks     int64  `json:"total_chunks"`
}

// SendResumeInfo sends resume information to the peer
func SendResumeInfo(writer *bufio.Writer, resumeInfo *ResumeInfo) error {
	// Send resume command
	if err := SendCommand(writer, CmdResume); err != nil {
		return err
	}

	// Send can_resume flag
	if resumeInfo.CanResume {
		if err := writer.WriteByte(1); err != nil {
			return errors.NewProtocolError("send_resume_info", "failed to send can_resume flag", err)
		}

		// Send resume offset
		if err := SendInt64(writer, resumeInfo.ResumeOffset); err != nil {
			return err
		}

		// Send total chunks
		if err := SendInt64(writer, resumeInfo.TotalChunks); err != nil {
			return err
		}

		// Send completed chunks bitmap (simplified as comma-separated)
		chunkList := ""
		for i, completed := range resumeInfo.CompletedChunks {
			if completed {
				if chunkList != "" {
					chunkList += ","
				}
				chunkList += fmt.Sprintf("%d", i)
			}
		}
		if err := SendString(writer, chunkList); err != nil {
			return err
		}
	} else {
		if err := writer.WriteByte(0); err != nil {
			return errors.NewProtocolError("send_resume_info", "failed to send can_resume flag", err)
		}
	}

	return FlushWriter(writer)
}

// ReadResumeInfo reads resume information from the peer
func ReadResumeInfo(ctx context.Context, reader *bufio.Reader) (*ResumeInfo, error) {
	resumeInfo := &ResumeInfo{}

	// Read can_resume flag
	canResumeByte, err := readByteWithContext(ctx, reader)
	if err != nil {
		return nil, errors.NewProtocolError("read_resume_info", "failed to read can_resume flag", err)
	}
	resumeInfo.CanResume = canResumeByte == 1

	if resumeInfo.CanResume {
		// Read resume offset
		resumeInfo.ResumeOffset, err = ReadInt64(ctx, reader)
		if err != nil {
			return nil, err
		}

		// Read total chunks
		resumeInfo.TotalChunks, err = ReadInt64(ctx, reader)
		if err != nil {
			return nil, err
		}

		// Read completed chunks
		chunkListStr, err := ReadString(ctx, reader)
		if err != nil {
			return nil, err
		}

		// Initialize completed chunks array
		resumeInfo.CompletedChunks = make([]bool, resumeInfo.TotalChunks)

		// Parse completed chunks
		if chunkListStr != "" {
			chunkStrs := strings.Split(chunkListStr, ",")
			for _, chunkStr := range chunkStrs {
				if chunkIndex, parseErr := strconv.ParseInt(strings.TrimSpace(chunkStr), 10, 64); parseErr == nil {
					if chunkIndex >= 0 && chunkIndex < resumeInfo.TotalChunks {
						resumeInfo.CompletedChunks[chunkIndex] = true
					}
				}
			}
		}
	}

	return resumeInfo, nil
}

// SendResumeAck sends resume acknowledgment
func SendResumeAck(writer *bufio.Writer, accepted bool) error {
	if err := SendCommand(writer, CmdResumeAck); err != nil {
		return err
	}

	if accepted {
		if err := writer.WriteByte(1); err != nil {
			return errors.NewProtocolError("send_resume_ack", "failed to send accepted flag", err)
		}
	} else {
		if err := writer.WriteByte(0); err != nil {
			return errors.NewProtocolError("send_resume_ack", "failed to send accepted flag", err)
		}
	}

	return FlushWriter(writer)
}

// ReadResumeAck reads resume acknowledgment
func ReadResumeAck(ctx context.Context, reader *bufio.Reader) (bool, error) {
	ackByte, err := readByteWithContext(ctx, reader)
	if err != nil {
		return false, errors.NewProtocolError("read_resume_ack", "failed to read ack flag", err)
	}
	return ackByte == 1, nil
}

// SendHashAlgorithm sends hash algorithm to writer
func SendHashAlgorithm(writer *bufio.Writer, algorithm HashAlgorithm) error {
	if err := SendCommand(writer, CmdHashAlgo); err != nil {
		return err
	}
	return SendString(writer, string(algorithm))
}

// ReadHashAlgorithm reads hash algorithm from reader
func ReadHashAlgorithm(ctx context.Context, reader *bufio.Reader) (HashAlgorithm, error) {
	algoStr, err := ReadString(ctx, reader)
	if err != nil {
		return "", err
	}

	// Validate algorithm
	switch HashAlgorithm(algoStr) {
	case HashMD5, HashSHA256, HashBLAKE2b:
		return HashAlgorithm(algoStr), nil
	default:
		return "", errors.NewProtocolError("hash_algorithm", "invalid hash algorithm", fmt.Errorf("unsupported algorithm: %s", algoStr))
	}
}

// SendBool sends a boolean value as a string
func SendBool(writer *bufio.Writer, val bool) error {
	if val {
		return SendString(writer, "true")
	}
	return SendString(writer, "false")
}

// ReadBool reads a boolean value as a string
func ReadBool(ctx context.Context, reader *bufio.Reader) (bool, error) {
	str, err := ReadString(ctx, reader)
	if err != nil {
		return false, err
	}

	val, err := strconv.ParseBool(str)
	if err != nil {
		return false, errors.NewProtocolError("read_bool", fmt.Sprintf("invalid boolean: %s", str), err)
	}
	return val, nil
}

// Helper functions for context-aware I/O operations

func readByteWithContext(ctx context.Context, reader *bufio.Reader) (byte, error) {
	type byteResult struct {
		b   byte
		err error
	}

	resultCh := make(chan byteResult, 1)
	readCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-readCtx.Done():
			return
		default:
			b, err := reader.ReadByte()
			select {
			case <-readCtx.Done():
				return
			default:
				resultCh <- byteResult{b, err}
			}
		}
	}()

	select {
	case result := <-resultCh:
		return result.b, result.err
	case <-ctx.Done():
		cancel()
		return 0, ctx.Err()
	}
}

func readStringWithContext(ctx context.Context, reader *bufio.Reader, delim byte) (string, error) {
	type stringResult struct {
		s   string
		err error
	}

	resultCh := make(chan stringResult, 1)
	readCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-readCtx.Done():
			return
		default:
			str, err := reader.ReadString(delim)
			select {
			case <-readCtx.Done():
				return
			default:
				resultCh <- stringResult{str, err}
			}
		}
	}()

	select {
	case result := <-resultCh:
		return result.s, result.err
	case <-ctx.Done():
		cancel()
		return "", ctx.Err()
	}
}

// ReadWithContext reads data from a reader with context cancellation support
func ReadWithContext(ctx context.Context, reader *bufio.Reader, buffer []byte) (int, error) {
	type readResult struct {
		n   int
		err error
	}

	resultCh := make(chan readResult, 1)
	readCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-readCtx.Done():
			return
		default:
			n, err := reader.Read(buffer)
			select {
			case <-readCtx.Done():
				return
			default:
				resultCh <- readResult{n, err}
			}
		}
	}()

	select {
	case result := <-resultCh:
		return result.n, result.err
	case <-ctx.Done():
		cancel()
		return 0, ctx.Err()
	}
}
