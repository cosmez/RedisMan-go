package resp

import "strconv"

// ValueType represents the type of a RESP value.
//
// C#:
// public enum ValueKind { None, String, Integer, BulkString, Array, Null, Error }
//
// Go:
// We use a custom type based on int and the `iota` identifier to create an
// auto-incrementing enumeration.
type ValueType int

const (
	TypeNone ValueType = iota
	TypeString
	TypeInteger
	TypeBulkString
	TypeArray
	TypeNull
	TypeError
)

// RedisValue is the interface that all RESP value types must implement.
//
// C#:
// public interface IRedisValue { ValueKind Kind { get; } string Value { get; } }
//
// Go:
// Interfaces in Go are satisfied implicitly. Any struct that has the methods
// Type() ValueType and StringValue() string automatically implements RedisValue.
// There is no "implements" keyword.
type RedisValue interface {
	Type() ValueType
	StringValue() string
}

// RedisString represents a RESP Simple String (starts with +).
type RedisString struct {
	Value string
}

func (s RedisString) Type() ValueType     { return TypeString }
func (s RedisString) StringValue() string { return s.Value }

// RedisBulkString represents a RESP Bulk String (starts with $).
type RedisBulkString struct {
	Value  string
	Length int
}

func (b RedisBulkString) Type() ValueType     { return TypeBulkString }
func (b RedisBulkString) StringValue() string { return b.Value }

// RedisInteger represents a RESP Integer (starts with :).
type RedisInteger struct {
	IntValue int64
}

func (i RedisInteger) Type() ValueType { return TypeInteger }
func (i RedisInteger) StringValue() string {
	return strconv.FormatInt(i.IntValue, 10)
}

// RedisArray represents a RESP Array (starts with *).
type RedisArray struct {
	Values []RedisValue
}

func (a RedisArray) Type() ValueType { return TypeArray }
func (a RedisArray) StringValue() string {
	// As per roadmap, we return "" and let the output formatter handle display.
	// In C#, this returned the array length.
	return ""
}

// RedisError represents a RESP Error (starts with -).
type RedisError struct {
	Value string
}

func (e RedisError) Type() ValueType     { return TypeError }
func (e RedisError) StringValue() string { return e.Value }

// RedisNull represents a RESP Null Bulk String ($-1) or Null Array (*-1).
type RedisNull struct{}

func (n RedisNull) Type() ValueType { return TypeNull }
func (n RedisNull) StringValue() string {
	// As per roadmap, we return "" and let the output formatter handle display.
	// In C#, this returned "Null".
	return ""
}
