package filesystem

import (
	"os"
	"testing"

	"justdatacopier/internal/protocol"
)

func TestSelectHashAlgorithm(t *testing.T) {
	tests := []struct {
		name         string
		fileSize     int64
		expectedAlgo protocol.HashAlgorithm
	}{
		{
			name:         "Small file (1 MB)",
			fileSize:     1 * 1024 * 1024,
			expectedAlgo: protocol.HashMD5,
		},
		{
			name:         "Medium file (1 GB)",
			fileSize:     1 * 1024 * 1024 * 1024,
			expectedAlgo: protocol.HashMD5,
		},
		{
			name:         "Large file (49 GB)",
			fileSize:     49 * 1024 * 1024 * 1024,
			expectedAlgo: protocol.HashMD5,
		},
		{
			name:         "Large file (50 GB exactly)",
			fileSize:     50 * 1024 * 1024 * 1024,
			expectedAlgo: protocol.HashBLAKE2b,
		},
		{
			name:         "Very large file (100 GB)",
			fileSize:     100 * 1024 * 1024 * 1024,
			expectedAlgo: protocol.HashBLAKE2b,
		},
		{
			name:         "Huge file (2 TB)",
			fileSize:     2 * 1024 * 1024 * 1024 * 1024,
			expectedAlgo: protocol.HashBLAKE2b,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectHashAlgorithm(tt.fileSize)
			if result != tt.expectedAlgo {
				t.Errorf("SelectHashAlgorithm(%d) = %v, want %v", tt.fileSize, result, tt.expectedAlgo)
			}
		})
	}
}

func TestCalculateFileHashWithAlgorithm(t *testing.T) {
	// Create a temporary file with known content
	tmpFile, err := os.CreateTemp("", "hash_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	testContent := []byte("Hello, World! This is a test file for hash verification.")
	if _, err := tmpFile.Write(testContent); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}

	tests := []struct {
		name      string
		algorithm protocol.HashAlgorithm
		wantLen   int // Expected hash length in hex characters
	}{
		{
			name:      "MD5 hash",
			algorithm: protocol.HashMD5,
			wantLen:   32, // MD5 produces 16 bytes = 32 hex chars
		},
		{
			name:      "SHA256 hash",
			algorithm: protocol.HashSHA256,
			wantLen:   64, // SHA256 produces 32 bytes = 64 hex chars
		},
		{
			name:      "BLAKE2b hash",
			algorithm: protocol.HashBLAKE2b,
			wantLen:   64, // BLAKE2b-256 produces 32 bytes = 64 hex chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := CalculateFileHashWithAlgorithm(tmpFile, tt.algorithm)
			if err != nil {
				t.Errorf("CalculateFileHashWithAlgorithm() error = %v", err)
				return
			}

			if len(hash) != tt.wantLen {
				t.Errorf("CalculateFileHashWithAlgorithm() hash length = %d, want %d", len(hash), tt.wantLen)
			}

			// Verify hash is hexadecimal
			for _, char := range hash {
				if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
					t.Errorf("Hash contains non-hex character: %c", char)
					break
				}
			}

			t.Logf("%s hash: %s", tt.algorithm, hash)
		})
	}
}

func TestCalculateFileHashConsistency(t *testing.T) {
	// Create a temporary file with known content
	tmpFile, err := os.CreateTemp("", "hash_consistency_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	testContent := []byte("Consistency test content for hash verification.")
	if _, err := tmpFile.Write(testContent); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}

	// Calculate hash multiple times to ensure consistency
	hash1, err := CalculateFileHashWithAlgorithm(tmpFile, protocol.HashMD5)
	if err != nil {
		t.Fatalf("First hash calculation failed: %v", err)
	}

	hash2, err := CalculateFileHashWithAlgorithm(tmpFile, protocol.HashMD5)
	if err != nil {
		t.Fatalf("Second hash calculation failed: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash calculations are inconsistent: %s != %s", hash1, hash2)
	}
}

func TestUnsupportedHashAlgorithm(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "unsupported_hash_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write([]byte("test")); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}

	_, err = CalculateFileHashWithAlgorithm(tmpFile, "unsupported")
	if err == nil {
		t.Error("Expected error for unsupported hash algorithm, got nil")
	}
}
