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
	CmdInit     = 1 // Initialize transfer
	CmdRequest  = 2 // Request chunk
	CmdData     = 3 // Send chunk data
	CmdComplete = 4 // Transfer complete
	CmdError    = 5 // Error occurred
	CmdHash     = 6 // File hash for verification
	CmdPing     = 7 // Ping for network profiling
	CmdPong     = 8 // Pong response to ping
	CmdVersion  = 9 // Protocol version negotiation
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
