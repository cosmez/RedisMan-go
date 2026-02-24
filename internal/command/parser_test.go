package command

import (
	"bytes"
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple",
			input:    "GET mykey",
			expected: []string{"GET", "mykey"},
		},
		{
			name:     "Quoted String",
			input:    `SET key "hello world"`,
			expected: []string{"SET", "key", "hello world"},
		},
		{
			name:     "Escaped Quotes",
			input:    `SET key "hello \"world\""`,
			expected: []string{"SET", "key", `hello "world"`},
		},
		{
			name:     "Unclosed Quotes",
			input:    `SET key "hello`,
			expected: []string{"SET", "key", "hello"},
		},
		{
			name:     "Multiple Spaces",
			input:    "  GET   mykey  ",
			expected: []string{"GET", "mykey"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("tokenize() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParse(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	tests := []struct {
		name         string
		input        string
		expectedName string
		expectedArgs []string
		expectedMod  string
		expectedPipe string
		expectedRESP []byte
		wantErr      bool
	}{
		{
			name:         "Simple Command",
			input:        "GET mykey",
			expectedName: "GET",
			expectedArgs: []string{"mykey"},
			expectedRESP: []byte("*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n"),
		},
		{
			name:         "With Codec",
			input:        "GET mykey#:gzip",
			expectedName: "GET",
			expectedArgs: []string{"mykey"},
			expectedMod:  "gzip",
			expectedRESP: []byte("*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n"),
		},
		{
			name:         "With Pipe",
			input:        "GET mykey | jq .",
			expectedName: "GET",
			expectedArgs: []string{"mykey"},
			expectedPipe: "jq .",
			expectedRESP: []byte("*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n"),
		},
		{
			name:         "With Codec and Pipe",
			input:        "GET mykey#:gzip | jq .",
			expectedName: "GET",
			expectedArgs: []string{"mykey"},
			expectedMod:  "gzip",
			expectedPipe: "jq .",
			expectedRESP: []byte("*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n"),
		},
		{
			name:         "SET with Codec",
			input:        "SET key value#:base64",
			expectedName: "SET",
			expectedArgs: []string{"key", "value"},
			expectedMod:  "base64",
			// "value" in base64 is "dmFsdWU=" (8 bytes)
			expectedRESP: []byte("*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$8\r\ndmFsdWU=\r\n"),
		},
		{
			name:    "Unknown Codec",
			input:   "SET key value#:unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, reg)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got.Name != tt.expectedName {
				t.Errorf("Parse() Name = %v, want %v", got.Name, tt.expectedName)
			}
			if !reflect.DeepEqual(got.Args, tt.expectedArgs) {
				t.Errorf("Parse() Args = %v, want %v", got.Args, tt.expectedArgs)
			}
			if got.Modifier != tt.expectedMod {
				t.Errorf("Parse() Modifier = %v, want %v", got.Modifier, tt.expectedMod)
			}
			if got.Pipe != tt.expectedPipe {
				t.Errorf("Parse() Pipe = %v, want %v", got.Pipe, tt.expectedPipe)
			}
			if !bytes.Equal(got.CommandBytes, tt.expectedRESP) {
				t.Errorf("Parse() CommandBytes = %q, want %q", got.CommandBytes, tt.expectedRESP)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	t.Run("Get Exact", func(t *testing.T) {
		doc := reg.Get("GET")
		if doc == nil || doc.Command != "GET" {
			t.Errorf("Expected GET doc, got %v", doc)
		}
	})

	t.Run("Get Compound", func(t *testing.T) {
		doc := reg.Get("CLIENT INFO")
		if doc == nil || doc.Command != "CLIENT INFO" {
			t.Errorf("Expected CLIENT INFO doc, got %v", doc)
		}
	})

	t.Run("Get Application Command", func(t *testing.T) {
		doc := reg.Get("EXIT")
		if doc == nil || doc.Group != "application" {
			t.Errorf("Expected EXIT app doc, got %v", doc)
		}
	})

	t.Run("IsDangerous", func(t *testing.T) {
		if !reg.IsDangerous("FLUSHDB") {
			t.Error("Expected FLUSHDB to be dangerous")
		}
		if reg.IsDangerous("GET") {
			t.Error("Expected GET to not be dangerous")
		}
	})

	t.Run("GetCommands Prefix", func(t *testing.T) {
		cmds := reg.GetCommands("CLI")
		found := false
		for _, cmd := range cmds {
			if cmd == "CLIENT INFO" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected CLIENT INFO in prefix search for CLI, got %v", cmds)
		}
	})
}
