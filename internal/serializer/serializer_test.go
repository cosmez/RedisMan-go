package serializer

import (
	"bytes"
	"testing"
)

func TestSerializerRoundTrip(t *testing.T) {
	codecs := []string{"base64", "gzip", "snappy"}

	testCases := []struct {
		name  string
		input []byte
	}{
		{
			name:  "Plain ASCII",
			input: []byte("Hello, World! This is a test string."),
		},
		{
			name:  "Binary Bytes",
			input: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x00},
		},
		{
			name:  "Empty Slice",
			input: []byte{},
		},
	}

	for _, codecName := range codecs {
		t.Run(codecName, func(t *testing.T) {
			codec, err := Get(codecName)
			if err != nil {
				t.Fatalf("Get(%q) failed: %v", codecName, err)
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Serialize
					serialized, err := codec.Serialize(tc.input)
					if err != nil {
						t.Fatalf("Serialize failed: %v", err)
					}

					// Deserialize
					deserialized, err := codec.Deserialize(serialized)
					if err != nil {
						t.Fatalf("Deserialize failed: %v", err)
					}

					// Compare
					if !bytes.Equal(tc.input, deserialized) {
						t.Errorf("Round-trip failed.\nExpected: %v\nGot:      %v", tc.input, deserialized)
					}
				})
			}
		})
	}
}

func TestGetUnknownSerializer(t *testing.T) {
	codec, err := Get("unknown")
	if err == nil {
		t.Error("Expected error for unknown serializer, got nil")
	}
	if codec != nil {
		t.Errorf("Expected nil codec for unknown serializer, got %T", codec)
	}
}
