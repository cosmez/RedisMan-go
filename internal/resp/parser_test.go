package resp

import (
	"bufio"
	"reflect"
	"strings"
	"testing"
)

func TestParseValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected RedisValue
		wantErr  bool
	}{
		{
			name:     "Simple String",
			input:    "+OK\r\n",
			expected: RedisString{Value: "OK"},
		},
		{
			name:     "Error",
			input:    "-ERR unknown command\r\n",
			expected: RedisError{Value: "ERR unknown command"},
		},
		{
			name:     "Integer",
			input:    ":42\r\n",
			expected: RedisInteger{IntValue: 42},
		},
		{
			name:     "Bulk String",
			input:    "$6\r\nfoobar\r\n",
			expected: RedisBulkString{Value: "foobar", Length: 6},
		},
		{
			name:     "Null Bulk String",
			input:    "$-1\r\n",
			expected: RedisNull{},
		},
		{
			name:     "Empty Bulk String",
			input:    "$0\r\n\r\n",
			expected: RedisBulkString{Value: "", Length: 0},
		},
		{
			name:     "Binary Bulk String",
			input:    "$4\r\n\x00\x01\x02\x03\r\n",
			expected: RedisBulkString{Value: "\x00\x01\x02\x03", Length: 4},
		},
		{
			name:  "Array",
			input: "*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
			expected: RedisArray{
				Values: []RedisValue{
					RedisBulkString{Value: "foo", Length: 3},
					RedisBulkString{Value: "bar", Length: 3},
				},
			},
		},
		{
			name:     "Null Array",
			input:    "*-1\r\n",
			expected: RedisNull{},
		},
		{
			name:     "Empty Array",
			input:    "*0\r\n",
			expected: RedisArray{Values: []RedisValue{}},
		},
		{
			name:  "Nested Array",
			input: "*2\r\n*1\r\n:1\r\n*1\r\n:2\r\n",
			expected: RedisArray{
				Values: []RedisValue{
					RedisArray{Values: []RedisValue{RedisInteger{IntValue: 1}}},
					RedisArray{Values: []RedisValue{RedisInteger{IntValue: 2}}},
				},
			},
		},
		{
			name:    "Invalid Type",
			input:   "?OK\r\n",
			wantErr: true,
		},
		{
			name:    "Invalid Integer",
			input:   ":abc\r\n",
			wantErr: true,
		},
		{
			name:    "Invalid Bulk String Length",
			input:   "$abc\r\n",
			wantErr: true,
		},
		{
			name:    "Invalid Array Count",
			input:   "*abc\r\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := ParseValue(r)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParseValue() got = %v, want %v", got, tt.expected)
			}
		})
	}
}
