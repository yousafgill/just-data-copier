package compression

import (
	"bytes"
	"compress/gzip"
	"io"
	"log/slog"
	"path/filepath"

	"justdatacopier/internal/errors"
	"justdatacopier/internal/filesystem"
)

// CompressData compresses data using gzip with appropriate compression level
func CompressData(data []byte, filename string) ([]byte, error) {
	var buf bytes.Buffer

	// Choose compression level based on file type
	level := gzip.BestSpeed // Default to speed
	if filesystem.GetCompressibleExtensions()[filepath.Ext(filename)] {
		level = gzip.DefaultCompression // Better compression for text files
	}

	writer, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, errors.NewCompressionError("create_writer", err)
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, errors.NewCompressionError("write_data", err)
	}

	if err := writer.Close(); err != nil {
		return nil, errors.NewCompressionError("close_writer", err)
	}

	compressed := buf.Bytes()
	ratio := float64(len(data)) / float64(len(compressed))

	slog.Debug("Data compressed",
		"original_size", len(data),
		"compressed_size", len(compressed),
		"ratio", ratio)

	return compressed, nil
}

// DecompressData decompresses gzip-compressed data
func DecompressData(compressedData []byte, expectedSize int) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, errors.NewCompressionError("create_reader", err)
	}
	defer reader.Close()

	// Pre-allocate buffer with expected size
	buffer := make([]byte, expectedSize)

	n, err := io.ReadFull(reader, buffer)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, errors.NewCompressionError("read_data", err)
	}

	slog.Debug("Data decompressed",
		"compressed_size", len(compressedData),
		"decompressed_size", n)

	return buffer[:n], nil
}

// ShouldCompressFile determines if a file should be compressed based on its name
func ShouldCompressFile(filename string) bool {
	return filesystem.ShouldCompress(filename)
}

// GetCompressionRatio calculates the compression ratio
func GetCompressionRatio(originalSize, compressedSize int) float64 {
	if compressedSize == 0 {
		return 0
	}
	return float64(originalSize) / float64(compressedSize)
}
