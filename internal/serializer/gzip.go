package serializer

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// gzipSerializer implements the Serializer interface using gzip compression.
type gzipSerializer struct{}

func (s gzipSerializer) Serialize(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)

	// C#:
	// using (var ms = new MemoryStream())
	// using (var gz = new GZipStream(ms, CompressionMode.Compress)) { ... }
	//
	// Go:
	// We don't use defer w.Close() here because we need to ensure the writer
	// is closed (flushing the gzip footer) *before* we read from the buffer.
	// If we deferred it, it would close after the function returns.
	if _, err := w.Write(data); err != nil {
		w.Close() // Clean up on error
		return nil, fmt.Errorf("gzip write failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("gzip close failed: %w", err)
	}

	return buf.Bytes(), nil
}

func (s gzipSerializer) Deserialize(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip reader init failed: %w", err)
	}
	// defer ensures the reader is closed when the function exits,
	// similar to a finally block in C#.
	defer r.Close()

	uncompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("gzip read failed: %w", err)
	}

	return uncompressed, nil
}
