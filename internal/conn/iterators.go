package conn

import (
	"fmt"
	"iter"
	"time"

	"github.com/cosmez/redisman-go/internal/command"
	"github.com/cosmez/redisman-go/internal/resp"
)

// SafeKeys iterates over all keys matching a pattern using the SCAN command.
//
// C#:
//
//	public IEnumerable<RedisValue> SafeKeys(string pattern) {
//	    string cursor = "0";
//	    while (true) { ... yield return key; ... }
//	}
//
// Go:
// We use Go 1.23's iter.Seq. If an error occurs, we yield a RedisError and stop.
func (c *Connection) SafeKeys(pattern string) iter.Seq[resp.RedisValue] {
	return func(yield func(resp.RedisValue) bool) {
		cursor := "0"
		for {
			cmdStr := fmt.Sprintf("SCAN %s MATCH %s COUNT 100", cursor, pattern)
			cmd, _ := command.Parse(cmdStr, nil)

			if err := c.Send(cmd); err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("SCAN send failed: %v", err)})
				return
			}

			response, err := c.Receive(10 * time.Second)
			if err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("SCAN receive failed: %v", err)})
				return
			}

			if errResp, ok := response.(resp.RedisError); ok {
				yield(errResp)
				return
			}

			array, ok := response.(resp.RedisArray)
			if !ok || len(array.Values) < 2 {
				yield(resp.RedisError{Value: "unexpected SCAN response format"})
				return
			}

			// Update cursor
			cursor = array.Values[0].StringValue()

			// Iterate over keys
			keysArray, ok := array.Values[1].(resp.RedisArray)
			if !ok {
				yield(resp.RedisError{Value: "unexpected SCAN keys array format"})
				return
			}

			for _, key := range keysArray.Values {
				if !yield(key) {
					return // Consumer stopped iterating
				}
			}

			if cursor == "0" {
				break
			}
		}
	}
}

// SafeSets iterates over all members of a Set using the SSCAN command.
func (c *Connection) SafeSets(key string) iter.Seq[resp.RedisValue] {
	return func(yield func(resp.RedisValue) bool) {
		cursor := "0"
		for {
			cmdStr := fmt.Sprintf("SSCAN %s %s COUNT 100", key, cursor)
			cmd, _ := command.Parse(cmdStr, nil)

			if err := c.Send(cmd); err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("SSCAN send failed: %v", err)})
				return
			}

			response, err := c.Receive(10 * time.Second)
			if err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("SSCAN receive failed: %v", err)})
				return
			}

			if errResp, ok := response.(resp.RedisError); ok {
				yield(errResp)
				return
			}

			array, ok := response.(resp.RedisArray)
			if !ok || len(array.Values) < 2 {
				yield(resp.RedisError{Value: "unexpected SSCAN response format"})
				return
			}

			cursor = array.Values[0].StringValue()

			membersArray, ok := array.Values[1].(resp.RedisArray)
			if !ok {
				yield(resp.RedisError{Value: "unexpected SSCAN members array format"})
				return
			}

			for _, member := range membersArray.Values {
				if !yield(member) {
					return
				}
			}

			if cursor == "0" {
				break
			}
		}
	}
}

// SafeHash iterates over all fields and values of a Hash using the HSCAN command.
//
// C# Bug Fix:
// The C# version iterated over the top-level array, yielding the cursor string
// and the sub-array. We correctly iterate over the sub-array (Values[1]).
func (c *Connection) SafeHash(key string) iter.Seq[resp.RedisValue] {
	return func(yield func(resp.RedisValue) bool) {
		cursor := "0"
		for {
			cmdStr := fmt.Sprintf("HSCAN %s %s COUNT 100", key, cursor)
			cmd, _ := command.Parse(cmdStr, nil)

			if err := c.Send(cmd); err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("HSCAN send failed: %v", err)})
				return
			}

			response, err := c.Receive(10 * time.Second)
			if err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("HSCAN receive failed: %v", err)})
				return
			}

			if errResp, ok := response.(resp.RedisError); ok {
				yield(errResp)
				return
			}

			array, ok := response.(resp.RedisArray)
			if !ok || len(array.Values) < 2 {
				yield(resp.RedisError{Value: "unexpected HSCAN response format"})
				return
			}

			cursor = array.Values[0].StringValue()

			fieldsArray, ok := array.Values[1].(resp.RedisArray)
			if !ok {
				yield(resp.RedisError{Value: "unexpected HSCAN fields array format"})
				return
			}

			for i := 0; i < len(fieldsArray.Values); i += 2 {
				if i+1 < len(fieldsArray.Values) {
					pair := resp.RedisArray{Values: []resp.RedisValue{fieldsArray.Values[i], fieldsArray.Values[i+1]}}
					if !yield(pair) {
						return
					}
				}
			}

			if cursor == "0" {
				break
			}
		}
	}
}

// SafeSortedSets iterates over all members and scores of a Sorted Set using the ZSCAN command.
func (c *Connection) SafeSortedSets(key string) iter.Seq[resp.RedisValue] {
	return func(yield func(resp.RedisValue) bool) {
		cursor := "0"
		for {
			cmdStr := fmt.Sprintf("ZSCAN %s %s COUNT 100", key, cursor)
			cmd, _ := command.Parse(cmdStr, nil)

			if err := c.Send(cmd); err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("ZSCAN send failed: %v", err)})
				return
			}

			response, err := c.Receive(10 * time.Second)
			if err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("ZSCAN receive failed: %v", err)})
				return
			}

			if errResp, ok := response.(resp.RedisError); ok {
				yield(errResp)
				return
			}

			array, ok := response.(resp.RedisArray)
			if !ok || len(array.Values) < 2 {
				yield(resp.RedisError{Value: "unexpected ZSCAN response format"})
				return
			}

			cursor = array.Values[0].StringValue()

			membersArray, ok := array.Values[1].(resp.RedisArray)
			if !ok {
				yield(resp.RedisError{Value: "unexpected ZSCAN members array format"})
				return
			}

			for _, member := range membersArray.Values {
				if !yield(member) {
					return
				}
			}

			if cursor == "0" {
				break
			}
		}
	}
}

// SafeList iterates over all elements of a List using the LRANGE command.
//
// C# Bug Fix:
// The C# version used overlapping indices (i to i+100) and an unnecessary LLEN call.
// We use start to start+99 and stop when the returned array is empty.
func (c *Connection) SafeList(key string) iter.Seq[resp.RedisValue] {
	return func(yield func(resp.RedisValue) bool) {
		start := 0
		for {
			cmdStr := fmt.Sprintf("LRANGE %s %d %d", key, start, start+99)
			cmd, _ := command.Parse(cmdStr, nil)

			if err := c.Send(cmd); err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("LRANGE send failed: %v", err)})
				return
			}

			response, err := c.Receive(10 * time.Second)
			if err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("LRANGE receive failed: %v", err)})
				return
			}

			if errResp, ok := response.(resp.RedisError); ok {
				yield(errResp)
				return
			}

			array, ok := response.(resp.RedisArray)
			if !ok {
				yield(resp.RedisError{Value: "unexpected LRANGE response format"})
				return
			}

			if len(array.Values) == 0 {
				break // Reached the end of the list
			}

			for _, element := range array.Values {
				if !yield(element) {
					return
				}
			}

			start += 100
		}
	}
}

// SafeStream iterates over all entries of a Stream using the XRANGE command.
//
// C# Bug Fix:
// The C# version used the FIRST element's ID as the next cursor, causing an O(N^2) loop.
// We use the LAST element's ID and prepend `(` to make it exclusive (Redis 6.2+).
func (c *Connection) SafeStream(key string) iter.Seq[resp.RedisValue] {
	return func(yield func(resp.RedisValue) bool) {
		cursor := "-" // Start from the beginning
		for {
			cmdStr := fmt.Sprintf("XRANGE %s %s + COUNT 100", key, cursor)
			cmd, _ := command.Parse(cmdStr, nil)

			if err := c.Send(cmd); err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("XRANGE send failed: %v", err)})
				return
			}

			response, err := c.Receive(10 * time.Second)
			if err != nil {
				yield(resp.RedisError{Value: fmt.Sprintf("XRANGE receive failed: %v", err)})
				return
			}

			if errResp, ok := response.(resp.RedisError); ok {
				yield(errResp)
				return
			}

			array, ok := response.(resp.RedisArray)
			if !ok {
				yield(resp.RedisError{Value: "unexpected XRANGE response format"})
				return
			}

			if len(array.Values) == 0 {
				break // Reached the end of the stream
			}

			for _, entry := range array.Values {
				if !yield(entry) {
					return
				}
			}

			// Get the ID of the last entry to use as the next cursor
			lastEntry, ok := array.Values[len(array.Values)-1].(resp.RedisArray)
			if !ok || len(lastEntry.Values) == 0 {
				yield(resp.RedisError{Value: "unexpected XRANGE entry format"})
				return
			}

			lastID := lastEntry.Values[0].StringValue()
			// Prepend '(' to make the range exclusive for the next call (Redis 6.2+)
			cursor = "(" + lastID
		}
	}
}
