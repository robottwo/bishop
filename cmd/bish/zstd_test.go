package main

import (
	"bytes"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestIsValidZstdFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(string) error
		expected bool
	}{
		{
			name: "Non-existent file returns false",
			setup: func(path string) error {
				return nil
			},
			expected: false,
		},
		{
			name: "Empty file returns false",
			setup: func(path string) error {
				return os.WriteFile(path, []byte{}, 0644)
			},
			expected: false,
		},
		{
			name: "Valid zstd file returns true",
			setup: func(path string) error {
				file, err := os.Create(path)
				if err != nil {
					return err
				}
				defer func() {
					_ = file.Close()
				}()

				encoder, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedDefault))
				if err != nil {
					return err
				}
				defer func() {
					_ = encoder.Close()
				}()

				_, err = encoder.Write([]byte("test log entry"))
				return err
			},
			expected: true,
		},
		{
			name: "Invalid zstd header returns false",
			setup: func(path string) error {
				return os.WriteFile(path, []byte{0x00, 0x00, 0x00, 0x00}, 0644)
			},
			expected: false,
		},
		{
			name: "Plain text file returns false",
			setup: func(path string) error {
				return os.WriteFile(path, []byte("plain text log"), 0644)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.log")

			err := tt.setup(testFile)
			require.NoError(t, err)

			result := isValidZstdFile(testFile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewCompressedSink(t *testing.T) {
	tests := []struct {
		name           string
		existingFile   bool
		fileContent    []byte
		expectWrite    bool
		expectTruncate bool
	}{
		{
			name:           "Non-existent file creates new file",
			existingFile:   false,
			fileContent:    nil,
			expectWrite:    true,
			expectTruncate: false,
		},
		{
			name:           "Existing valid zstd file appends",
			existingFile:   true,
			fileContent:    createValidZstdFrame(t, "initial log"),
			expectWrite:    true,
			expectTruncate: false,
		},
		{
			name:           "Existing invalid file is truncated",
			existingFile:   true,
			fileContent:    []byte("corrupted data"),
			expectWrite:    true,
			expectTruncate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.log")

			if tt.existingFile {
				err := os.WriteFile(testFile, tt.fileContent, 0644)
				require.NoError(t, err)
			}

			fileURL, err := url.Parse("zstd://" + filepath.ToSlash(testFile))
			require.NoError(t, err)

			sink, err := newCompressedSink(fileURL)
			require.NoError(t, err)
			require.NotNil(t, sink)

			_, err = sink.Write([]byte("test log entry"))
			assert.NoError(t, err)

			err = sink.Sync()
			assert.NoError(t, err)

			// Close the sink to finalize the zstd frame before reading
			err = sink.Close()
			assert.NoError(t, err)

			if tt.expectTruncate {
				content, err := os.ReadFile(testFile)
				require.NoError(t, err)

				dec, err := zstd.NewReader(bytes.NewReader(content))
				require.NoError(t, err)
				defer dec.Close()

				result, err := io.ReadAll(dec)
				assert.NoError(t, err)
				assert.Contains(t, string(result), "test log entry")
				assert.NotContains(t, string(result), "corrupted data")
			}
		})
	}
}

func TestCompressedSinkWrite(t *testing.T) {
	t.Run("Write and read back", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.log")

		fileURL, err := url.Parse("zstd://" + filepath.ToSlash(testFile))
		require.NoError(t, err)

		sink, err := newCompressedSink(fileURL)
		require.NoError(t, err)
		defer func() {
			_ = sink.Close()
		}()

		testData := []byte("test log message")
		n, err := sink.Write(testData)
		assert.NoError(t, err)
		assert.Equal(t, len(testData), n)

		err = sink.Close()
		assert.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		dec, err := zstd.NewReader(bytes.NewReader(content))
		require.NoError(t, err)
		defer dec.Close()

		result, err := io.ReadAll(dec)
		assert.NoError(t, err)
		assert.Equal(t, testData, result)
	})

	t.Run("Write returns input byte count (io.Writer contract)", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.log")

		fileURL, err := url.Parse("zstd://" + filepath.ToSlash(testFile))
		require.NoError(t, err)

		sink, err := newCompressedSink(fileURL)
		require.NoError(t, err)
		defer func() {
			_ = sink.Close()
		}()

		testData := []byte("test message that will be compressed")
		n, err := sink.Write(testData)
		assert.NoError(t, err)

		// io.Writer contract: return number of input bytes written,
		// NOT compressed bytes (which would be different)
		assert.Equal(t, len(testData), n, "Write should return len(p), not compressed byte count")
	})
}

func TestCompressedSinkMultiFrame(t *testing.T) {
	t.Run("Multiple sessions create readable multi-frame file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.log")

		fileURL, err := url.Parse("zstd://" + filepath.ToSlash(testFile))
		require.NoError(t, err)

		firstLog := "first session log entry"
		sink1, err := newCompressedSink(fileURL)
		require.NoError(t, err)
		_, err = sink1.Write([]byte(firstLog))
		assert.NoError(t, err)
		err = sink1.Close()
		assert.NoError(t, err)

		secondLog := "second session log entry"
		sink2, err := newCompressedSink(fileURL)
		require.NoError(t, err)
		_, err = sink2.Write([]byte(secondLog))
		assert.NoError(t, err)
		err = sink2.Close()
		assert.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		dec, err := zstd.NewReader(bytes.NewReader(content))
		require.NoError(t, err)
		defer dec.Close()

		result, err := io.ReadAll(dec)
		assert.NoError(t, err)

		// zstd decoder should handle multiple frames and concatenate them
		assert.Contains(t, string(result), firstLog)
		assert.Contains(t, string(result), secondLog)
	})
}

func TestCompressedSinkSync(t *testing.T) {
	t.Run("Sync flushes data to disk", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.log")

		fileURL, err := url.Parse("zstd://" + filepath.ToSlash(testFile))
		require.NoError(t, err)

		sink, err := newCompressedSink(fileURL)
		require.NoError(t, err)
		defer func() {
			_ = sink.Close()
		}()

		testData := []byte("sync test")
		_, err = sink.Write(testData)
		assert.NoError(t, err)

		err = sink.Sync()
		assert.NoError(t, err)

		// Verify data was written
		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Greater(t, len(content), 0)
	})
}

func TestCompressedSinkClose(t *testing.T) {
	t.Run("Close properly cleans up resources", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.log")

		fileURL, err := url.Parse("zstd://" + filepath.ToSlash(testFile))
		require.NoError(t, err)

		sink, err := newCompressedSink(fileURL)
		require.NoError(t, err)

		// Write some data
		testData := []byte("test data")
		_, err = sink.Write(testData)
		assert.NoError(t, err)

		// Close the sink
		err = sink.Close()
		assert.NoError(t, err)

		// Verify we can reopen the file (file descriptor was properly closed)
		newSink, err := newCompressedSink(fileURL)
		assert.NoError(t, err, "Should be able to create new sink after closing previous one")
		defer func() {
			_ = newSink.Close()
		}()

		// Verify we can write to the reopened file
		_, err = newSink.Write([]byte("more data"))
		assert.NoError(t, err)

		// Verify both sets of data are present
		err = newSink.Close()
		assert.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		dec, err := zstd.NewReader(bytes.NewReader(content))
		require.NoError(t, err)
		defer dec.Close()

		result, err := io.ReadAll(dec)
		assert.NoError(t, err)
		assert.Contains(t, string(result), "test data")
		assert.Contains(t, string(result), "more data")
	})
}

func TestCompressedSinkIntegration(t *testing.T) {
	t.Run("Integration with zap logger", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "bish.log")

		fileURL, err := url.Parse("zstd://" + filepath.ToSlash(logFile))
		require.NoError(t, err)

		sink, err := newCompressedSink(fileURL)
		require.NoError(t, err)

		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = ""
		encoder := zapcore.NewJSONEncoder(encoderConfig)
		core := zapcore.NewCore(encoder, zapcore.AddSync(sink), zap.InfoLevel)
		logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

		logger.Info("test message 1")
		logger.Info("test message 2")

		err = logger.Sync()
		assert.NoError(t, err)

		_, err = os.Stat(logFile)
		assert.NoError(t, err, "Log file should exist")

		assert.True(t, isValidZstdFile(logFile))

		content, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Greater(t, len(content), 0)

		err = sink.Close()
		if err != nil {
			t.Logf("Warning: failed to close sink on cleanup: %v", err)
		}
	})
}

func createValidZstdFrame(t *testing.T, data string) []byte {
	var buf bytes.Buffer
	encoder, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
	require.NoError(t, err)
	_, err = encoder.Write([]byte(data))
	require.NoError(t, err)
	err = encoder.Close()
	require.NoError(t, err)
	return buf.Bytes()
}
