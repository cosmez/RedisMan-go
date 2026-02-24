package output

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cosmez/redisman-go/internal/resp"
)

func TestPrintRedisValue(t *testing.T) {
	tests := []struct {
		name     string
		value    resp.RedisValue
		opts     PrintOpts
		expected string
	}{
		{
			name:     "String",
			value:    resp.RedisString{Value: "OK"},
			opts:     PrintOpts{Newline: true},
			expected: "OK\n",
		},
		{
			name:     "Integer",
			value:    resp.RedisInteger{IntValue: 42},
			opts:     PrintOpts{Newline: true},
			expected: "(integer) 42\n",
		},
		{
			name:     "Null",
			value:    resp.RedisNull{},
			opts:     PrintOpts{Newline: true},
			expected: "(nil)\n",
		},
		{
			name:     "BulkString",
			value:    resp.RedisBulkString{Value: "hello", Length: 5},
			opts:     PrintOpts{Newline: true},
			expected: "\"hello\"\n",
		},
		{
			name:     "Null BulkString",
			value:    resp.RedisBulkString{Length: -1},
			opts:     PrintOpts{Newline: true},
			expected: "(nil)\n",
		},
		{
			name:     "Error",
			value:    resp.RedisError{Value: "ERR unknown command"},
			opts:     PrintOpts{Newline: true},
			expected: "ERR unknown command\n",
		},
		{
			name: "Array",
			value: resp.RedisArray{Values: []resp.RedisValue{
				resp.RedisString{Value: "one"},
				resp.RedisString{Value: "two"},
			}},
			opts:     PrintOpts{Newline: true},
			expected: "1) one\n2) two\n",
		},
		{
			name: "Hash Array",
			value: resp.RedisArray{Values: []resp.RedisValue{
				resp.RedisString{Value: "field1"},
				resp.RedisString{Value: "val1"},
				resp.RedisString{Value: "field2"},
				resp.RedisString{Value: "val2"},
			}},
			opts:     PrintOpts{TypeHint: "hash", Newline: true},
			expected: "#field1=val1\n#field2=val2\n",
		},
		{
			name: "Stream Array",
			value: resp.RedisArray{Values: []resp.RedisValue{
				resp.RedisString{Value: "field1"},
				resp.RedisString{Value: "val1"},
			}},
			opts:     PrintOpts{TypeHint: "stream", Newline: true},
			expected: "@field1=val1\n",
		},
		{
			name: "Nested Array",
			value: resp.RedisArray{Values: []resp.RedisValue{
				resp.RedisString{Value: "one"},
				resp.RedisArray{Values: []resp.RedisValue{
					resp.RedisString{Value: "two"},
				}},
			}},
			opts:     PrintOpts{Newline: true},
			expected: "1) one\n2) 1) two\n",
		},
		{
			name:     "Empty Array",
			value:    resp.RedisArray{Values: nil},
			opts:     PrintOpts{Newline: true},
			expected: "(empty array)\n",
		},
		{
			name: "Array with typed values",
			value: resp.RedisArray{Values: []resp.RedisValue{
				resp.RedisBulkString{Value: "hello", Length: 5},
				resp.RedisInteger{IntValue: 42},
			}},
			opts:     PrintOpts{Newline: true},
			expected: "1) \"hello\"\n2) (integer) 42\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PrintRedisValue(&buf, tt.value, tt.opts)
			if got := buf.String(); got != tt.expected {
				t.Errorf("PrintRedisValue() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPrintRedisValues(t *testing.T) {
	values := []resp.RedisValue{
		resp.RedisString{Value: "one"},
		resp.RedisString{Value: "two"},
	}

	seq := func(yield func(resp.RedisValue) bool) {
		for _, v := range values {
			if !yield(v) {
				return
			}
		}
	}

	var buf bytes.Buffer
	var in bytes.Buffer // empty input, won't pause
	PrintRedisValues(&buf, &in, seq, PrintOpts{Newline: true}, 100)

	expected := "1) one\n2) two\n"
	if got := buf.String(); got != expected {
		t.Errorf("PrintRedisValues() = %q, want %q", got, expected)
	}
}

func TestPrintRedisValues_Pagination(t *testing.T) {
	values := []resp.RedisValue{
		resp.RedisString{Value: "one"},
		resp.RedisString{Value: "two"},
		resp.RedisString{Value: "three"},
	}

	seq := func(yield func(resp.RedisValue) bool) {
		for _, v := range values {
			if !yield(v) {
				return
			}
		}
	}

	var buf bytes.Buffer
	in := strings.NewReader("Y\n") // answer yes to continue
	PrintRedisValues(&buf, in, seq, PrintOpts{Newline: true}, 2)

	expected := "1) one\n2) two\nContinue Listing? (Y/N) 3) three\n"
	if got := buf.String(); got != expected {
		t.Errorf("PrintRedisValues() = %q, want %q", got, expected)
	}
}

func TestPrintRedisValues_PaginationStop(t *testing.T) {
	values := []resp.RedisValue{
		resp.RedisString{Value: "one"},
		resp.RedisString{Value: "two"},
		resp.RedisString{Value: "three"},
	}

	seq := func(yield func(resp.RedisValue) bool) {
		for _, v := range values {
			if !yield(v) {
				return
			}
		}
	}

	var buf bytes.Buffer
	in := strings.NewReader("N\n") // answer no to stop
	PrintRedisValues(&buf, in, seq, PrintOpts{Newline: true}, 2)

	expected := "1) one\n2) two\nContinue Listing? (Y/N) "
	if got := buf.String(); got != expected {
		t.Errorf("PrintRedisValues() = %q, want %q", got, expected)
	}
}

func TestExportAsync(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "export.txt")

	val := resp.RedisArray{Values: []resp.RedisValue{
		resp.RedisString{Value: "one"},
		resp.RedisString{Value: "two"},
	}}

	err := ExportAsync(file, val, nil, "")
	if err != nil {
		t.Fatalf("ExportAsync failed: %v", err)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("Failed to read exported file: %v", err)
	}

	expected := "one\ntwo\n"
	if string(content) != expected {
		t.Errorf("ExportAsync() = %q, want %q", string(content), expected)
	}
}

func TestExportAsync_Hash(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "export_hash.txt")

	val := resp.RedisArray{Values: []resp.RedisValue{
		resp.RedisString{Value: "field1"},
		resp.RedisString{Value: "val1"},
		resp.RedisString{Value: "field2"},
		resp.RedisString{Value: "val2"},
	}}

	err := ExportAsync(file, val, nil, "hash")
	if err != nil {
		t.Fatalf("ExportAsync failed: %v", err)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("Failed to read exported file: %v", err)
	}

	expected := "field1=val1\nfield2=val2\n"
	if string(content) != expected {
		t.Errorf("ExportAsync() = %q, want %q", string(content), expected)
	}
}
