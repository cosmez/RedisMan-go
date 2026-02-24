package serializer

import (
	"encoding/base64"
)

// base64Serializer implements the Serializer interface using standard base64 encoding.
type base64Serializer struct{}

func (s base64Serializer) Serialize(data []byte) ([]byte, error) {
	// EncodeToString returns a string, we cast it back to []byte
	encoded := base64.StdEncoding.EncodeToString(data)
	return []byte(encoded), nil
}

func (s base64Serializer) Deserialize(data []byte) ([]byte, error) {
	// DecodeString takes a string and returns []byte and an error
	return base64.StdEncoding.DecodeString(string(data))
}
