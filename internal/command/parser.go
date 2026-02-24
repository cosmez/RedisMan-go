package command

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/cosmez/redisman-go/internal/serializer"
)

// Parse takes a raw input string, extracts modifiers, tokenizes it, and builds the RESP bytes.
//
// C#:
// public static ParsedCommand Parse(string input)
//
// Go:
// We return a pointer to ParsedCommand and an error. We also pass the Registry
// explicitly rather than relying on a global static class.
func Parse(input string, reg *Registry) (*ParsedCommand, error) {
	if strings.TrimSpace(input) == "" {
		return &ParsedCommand{}, nil
	}

	parsed := &ParsedCommand{
		Text: input,
	}

	// 1. Detect and strip `| shell cmd` suffix
	// We do this FIRST to avoid extracting `gzip | jq .` as a codec name
	// if the user types `GET key #:gzip | jq .`
	if pipeIdx := strings.Index(input, " | "); pipeIdx != -1 {
		parsed.Pipe = strings.TrimSpace(input[pipeIdx+3:])
		input = input[:pipeIdx]
	}

	// 2. Detect and strip `#:codec` suffix
	if codecIdx := strings.LastIndex(input, "#:"); codecIdx != -1 {
		parsed.Modifier = strings.TrimSpace(input[codecIdx+2:])
		input = input[:codecIdx]
	}

	// 3. Tokenize remaining text
	tokens := tokenize(input)
	if len(tokens) == 0 {
		return parsed, nil
	}

	// 4. Extract Name and Args
	parsed.Name = strings.ToUpper(tokens[0])
	if len(tokens) > 1 {
		parsed.Args = tokens[1:]
	}

	// 5. Look up documentation
	if reg != nil {
		// Try exact match first
		parsed.Doc = reg.Get(parsed.Name)
		// Try compound match (e.g., "CLIENT INFO")
		if len(parsed.Args) > 0 {
			compoundName := parsed.Name + " " + strings.ToUpper(parsed.Args[0])
			if compoundDoc := reg.Get(compoundName); compoundDoc != nil {
				parsed.Doc = compoundDoc
			}
		}
	}

	// 6. Build RESP bytes
	var buf bytes.Buffer
	// Array header: *N\r\n
	buf.WriteString(fmt.Sprintf("*%d\r\n", len(tokens)))

	for i, token := range tokens {
		tokenBytes := []byte(token)

		// Special case: Serialize the value argument of SET if a modifier is present
		if parsed.Name == "SET" && i == 2 && parsed.Modifier != "" {
			codec, err := serializer.Get(parsed.Modifier)
			if err != nil {
				return nil, fmt.Errorf("failed to get serializer %q: %w", parsed.Modifier, err)
			}
			serializedBytes, err := codec.Serialize(tokenBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to serialize value: %w", err)
			}
			tokenBytes = serializedBytes
		}

		// Bulk string header: $len\r\n
		buf.WriteString(fmt.Sprintf("$%d\r\n", len(tokenBytes)))
		// Bulk string payload: bytes\r\n
		buf.Write(tokenBytes)
		buf.WriteString("\r\n")
	}

	parsed.CommandBytes = buf.Bytes()

	return parsed, nil
}
