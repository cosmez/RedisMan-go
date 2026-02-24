package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ParseValue reads a single RESP value from the provided reader.
//
// C#:
// public static IRedisValue Parse(StreamReader reader)
//
// Go:
// We use bufio.Reader instead of StreamReader. Errors are returned as values
// rather than thrown as exceptions.
func ParseValue(r *bufio.Reader) (RedisValue, error) {
	// Read the first byte to determine the type
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch b {
	case '+':
		return parseSimpleString(r)
	case '-':
		return parseError(r)
	case ':':
		return parseInteger(r)
	case '$':
		return parseBulkString(r)
	case '*':
		return parseArray(r)
	default:
		return nil, fmt.Errorf("unknown RESP type byte: %q", b)
	}
}

// readLine reads until \n and strips the trailing \r\n.
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	// Strip \r\n
	return strings.TrimSuffix(line, "\r\n"), nil
}

func parseSimpleString(r *bufio.Reader) (RedisValue, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	return RedisString{Value: line}, nil
}

func parseError(r *bufio.Reader) (RedisValue, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	return RedisError{Value: line}, nil
}

func parseInteger(r *bufio.Reader) (RedisValue, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	val, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid integer format: %w", err)
	}
	return RedisInteger{IntValue: val}, nil
}

func parseBulkString(r *bufio.Reader) (RedisValue, error) {
	// Read the length line first
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}

	length, err := strconv.Atoi(line)
	if err != nil {
		return nil, fmt.Errorf("invalid bulk string length: %w", err)
	}

	// A length of -1 indicates a Null Bulk String
	if length == -1 {
		return RedisNull{}, nil
	}

	// Negative lengths other than -1 are invalid
	if length < -1 {
		return nil, fmt.Errorf("invalid bulk string length: %d", length)
	}

	// Read exact byte count to be binary-safe
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("failed to read bulk string payload: %w", err)
	}

	// Consume the trailing \r\n
	crlf := make([]byte, 2)
	if _, err := io.ReadFull(r, crlf); err != nil {
		return nil, fmt.Errorf("failed to read bulk string trailing CRLF: %w", err)
	}
	if crlf[0] != '\r' || crlf[1] != '\n' {
		return nil, fmt.Errorf("expected CRLF after bulk string payload, got %q", crlf)
	}

	return RedisBulkString{Value: string(buf), Length: length}, nil
}

func parseArray(r *bufio.Reader) (RedisValue, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}

	// A count of -1 indicates a Null Array
	count, err := strconv.Atoi(line)
	if err != nil {
		return nil, fmt.Errorf("invalid array count: %w", err)
	}

	// Negative counts other than -1 are invalid
	if count == -1 {
		return RedisNull{}, nil
	}

	// Negative counts other than -1 are invalid
	if count < -1 {
		return nil, fmt.Errorf("invalid array count: %d", count)
	}

	// Parse each element
	values := make([]RedisValue, count)
	for i := 0; i < count; i++ {
		val, err := ParseValue(r)
		if err != nil {
			return nil, fmt.Errorf("failed to parse array element %d: %w", i, err)
		}
		values[i] = val
	}

	return RedisArray{Values: values}, nil
}
