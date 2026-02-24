package command

import (
	"strings"
	"unicode"
)

// tokenize splits the input string into tokens, respecting double quotes and escape sequences.
//
// C#:
// private static List<string> ParseToken(string input)
//
// Go:
// We iterate rune by rune. We explicitly unescape `\"` to `"` when building the token,
// unlike the C# version which left the backslash in the token.
func tokenize(input string) []string {
	var tokens []string
	var currentToken strings.Builder
	inQuotes := false
	escaped := false

	for _, r := range input {
		if escaped {
			// If the previous character was a backslash, append this character literally
			currentToken.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			// Start an escape sequence
			escaped = true
			continue
		}

		if r == '"' {
			// Toggle quote state
			inQuotes = !inQuotes
			continue
		}

		if unicode.IsSpace(r) && !inQuotes {
			// If we hit a space outside of quotes, finish the current token
			if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
			continue
		}

		// Otherwise, append the character to the current token
		currentToken.WriteRune(r)
	}

	// Append the last token if there is one
	if currentToken.Len() > 0 {
		tokens = append(tokens, currentToken.String())
	}

	return tokens
}
