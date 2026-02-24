package conn

import (
	"fmt"
	"iter"
	"time"

	"github.com/cosmez/redisman-go/internal/command"
	"github.com/cosmez/redisman-go/internal/resp"
)

// GetKeyValue determines the type of a key and returns either its single value
// or an iterator for its collection.
//
// C#:
// public (string typeName, IRedisValue single, IEnumerable<IRedisValue> collection) GetKeyValue(string key)
func (c *Connection) GetKeyValue(key string) (typeName string, single resp.RedisValue, collection iter.Seq[resp.RedisValue], err error) {
	typeCmd, _ := command.Parse(fmt.Sprintf("TYPE %s", key), nil)
	if err := c.Send(typeCmd); err != nil {
		return "", nil, nil, fmt.Errorf("failed to send TYPE command: %w", err)
	}

	response, err := c.Receive(5 * time.Second)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to receive TYPE response: %w", err)
	}

	if errResp, ok := response.(resp.RedisError); ok {
		return "", nil, nil, fmt.Errorf("TYPE command failed: %s", errResp.Value)
	}

	strResp, ok := response.(resp.RedisString)
	if !ok {
		return "", nil, nil, fmt.Errorf("expected simple string for TYPE, got %T", response)
	}

	typeName = strResp.Value

	switch typeName {
	case "string":
		getCmd, _ := command.Parse(fmt.Sprintf("GET %s", key), nil)
		if err := c.Send(getCmd); err != nil {
			return typeName, nil, nil, fmt.Errorf("failed to send GET command: %w", err)
		}
		single, err = c.Receive(5 * time.Second)
		if err != nil {
			return typeName, nil, nil, fmt.Errorf("failed to receive GET response: %w", err)
		}
		return typeName, single, nil, nil

	case "list":
		return typeName, nil, c.SafeList(key), nil
	case "set":
		return typeName, nil, c.SafeSets(key), nil
	case "zset":
		return typeName, nil, c.SafeSortedSets(key), nil
	case "hash":
		return typeName, nil, c.SafeHash(key), nil
	case "stream":
		return typeName, nil, c.SafeStream(key), nil
	case "none":
		return typeName, nil, nil, fmt.Errorf("key does not exist")
	default:
		return typeName, nil, nil, fmt.Errorf("unsupported key type: %s", typeName)
	}
}
