package compression

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressDecompressData(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		filename string
	}{
		{
			name:     "text file",
			data:     "This is a test string that should compress well because it has repetitive patterns. This is a test string that should compress well.",
			filename: "test.txt",
		},
		{
			name:     "binary file",
			data:     "Some binary data: \x00\x01\x02\x03\x04\x05",
			filename: "test.bin",
		},
		{
			name:     "empty data",
			data:     "",
			filename: "empty.txt",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalData := []byte(test.data)

			// Compress the data
			compressed, err := CompressData(originalData, test.filename)
			require.NoError(t, err)
			assert.NotNil(t, compressed)

			// Decompress the data
			decompressed, err := DecompressData(compressed, len(originalData))
			require.NoError(t, err)
			assert.Equal(t, originalData, decompressed)
		})
	}
}

func TestCompressDataError(t *testing.T) {
	// Test with invalid data - this shouldn't actually fail with our implementation
	// but we can test the structure
	data := []byte("test data")
	filename := "test.txt"

	compressed, err := CompressData(data, filename)
	assert.NoError(t, err)
	assert.NotNil(t, compressed)
}

func TestDecompressDataError(t *testing.T) {
	// Test with invalid compressed data
	invalidData := []byte("not compressed data")
	_, err := DecompressData(invalidData, 10)
	assert.Error(t, err)
}

func TestShouldCompressFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"test.txt", true},
		{"test.log", true},
		{"test.json", true},
		{"test.zip", false},
		{"test.jpg", false},
		{"test.mp4", false},
		{"test.xyz", false}, // unknown extension
	}

	for _, test := range tests {
		result := ShouldCompressFile(test.filename)
		assert.Equal(t, test.expected, result, "Filename: %s", test.filename)
	}
}

func TestGetCompressionRatio(t *testing.T) {
	tests := []struct {
		original   int
		compressed int
		expected   float64
	}{
		{100, 50, 2.0},
		{200, 100, 2.0},
		{100, 100, 1.0},
		{50, 100, 0.5},
		{100, 0, 0},
	}

	for _, test := range tests {
		result := GetCompressionRatio(test.original, test.compressed)
		assert.Equal(t, test.expected, result)
	}
}
