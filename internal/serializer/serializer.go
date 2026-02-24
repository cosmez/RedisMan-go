package serializer

import (
	"fmt"
	"strings"
)

// Serializer defines the interface for codec plugins.
//
// C#:
// public interface ISerializer { byte[] Serialize(byte[] data); byte[] Deserialize(byte[] data); }
//
// Go:
// Interfaces are implicitly satisfied. We return (result, error) instead of
// throwing exceptions on failure.
type Serializer interface {
	Serialize([]byte) ([]byte, error)
	Deserialize([]byte) ([]byte, error)
}

// Get returns a Serializer instance by name.
//
// C#:
// public static ISerializer Get(string name) { ... return null; }
//
// Go:
// We return an error if the name is unknown, rather than returning nil.
// This forces the caller to handle the missing codec explicitly.
func Get(name string) (Serializer, error) {
	switch strings.ToLower(name) {
	case "base64":
		return base64Serializer{}, nil
	case "gzip":
		return gzipSerializer{}, nil
	case "snappy":
		return snappySerializer{}, nil
	default:
		return nil, fmt.Errorf("unknown serializer: %q", name)
	}
}
