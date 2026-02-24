package serializer

import (
	"github.com/golang/snappy"
)

// snappySerializer implements the Serializer interface using snappy compression.
type snappySerializer struct{}

func (s snappySerializer) Serialize(data []byte) ([]byte, error) {
	// snappy.Encode appends the encoded data to the first argument (dst).
	// Passing nil allocates a new slice of the correct size.
	return snappy.Encode(nil, data), nil
}

func (s snappySerializer) Deserialize(data []byte) ([]byte, error) {
	// snappy.Decode appends the decoded data to the first argument (dst).
	// Passing nil allocates a new slice of the correct size.
	return snappy.Decode(nil, data)
}
