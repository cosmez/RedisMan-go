package conn

import (
	"fmt"
	"strings"
	"time"

	"github.com/cosmez/redisman-go/internal/command"
	"github.com/cosmez/redisman-go/internal/resp"
)

// getServerInfo sends the INFO command and parses the result into the ServerInfo map.
//
// C#:
//
//	private void GetServerInfo() {
//	    Send("INFO");
//	    var response = Receive();
//	    // ... parse string into Dictionary
//	}
// FetchServerCommands sends the COMMAND command to the Redis server and
// parses the response into a slice of ServerCommand for registry merging.
// Returns nil, nil if the server does not support COMMAND.
func (c *Connection) FetchServerCommands() ([]command.ServerCommand, error) {
	cmdParsed, _ := command.Parse("COMMAND", nil)
	if err := c.Send(cmdParsed); err != nil {
		return nil, fmt.Errorf("failed to send COMMAND: %w", err)
	}

	response, err := c.Receive(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to receive COMMAND response: %w", err)
	}

	// Server returned an error (old Redis, restricted ACL) — degrade gracefully.
	if _, ok := response.(resp.RedisError); ok {
		return nil, nil
	}

	array, ok := response.(resp.RedisArray)
	if !ok {
		return nil, fmt.Errorf("expected array for COMMAND, got %T", response)
	}

	var cmds []command.ServerCommand
	for _, entry := range array.Values {
		sc, err := parseCommandEntry(entry)
		if err != nil {
			continue // skip malformed entries
		}
		cmds = append(cmds, sc)
	}

	return cmds, nil
}

// parseCommandEntry converts a single COMMAND response entry (a RedisArray)
// into a ServerCommand. The entry has up to 10 elements; older Redis versions
// may return fewer.
func parseCommandEntry(v resp.RedisValue) (command.ServerCommand, error) {
	arr, ok := v.(resp.RedisArray)
	if !ok || len(arr.Values) < 2 {
		return command.ServerCommand{}, fmt.Errorf("expected array with >= 2 elements")
	}

	// [0] Name: lowercase with "|" separating subcommands
	name := strings.ToUpper(strings.ReplaceAll(arr.Values[0].StringValue(), "|", " "))

	// [1] Arity
	var arity int64
	if intVal, ok := arr.Values[1].(resp.RedisInteger); ok {
		arity = intVal.IntValue
	}

	// [6] ACL categories (Redis 7.0+)
	var aclCats []string
	if len(arr.Values) > 6 {
		aclCats = extractStringArray(arr.Values[6])
	}

	// [9] Subcommands (Redis 7.0+)
	var subcommands []command.ServerCommand
	if len(arr.Values) > 9 {
		if subArr, ok := arr.Values[9].(resp.RedisArray); ok {
			for _, subEntry := range subArr.Values {
				sub, err := parseCommandEntry(subEntry)
				if err != nil {
					continue
				}
				subcommands = append(subcommands, sub)
			}
		}
	}

	return command.ServerCommand{
		Name:        name,
		Arity:       arity,
		ACLCats:     aclCats,
		Subcommands: subcommands,
	}, nil
}

// extractStringArray pulls string values out of a RedisArray.
func extractStringArray(v resp.RedisValue) []string {
	arr, ok := v.(resp.RedisArray)
	if !ok {
		return nil
	}
	strs := make([]string, 0, len(arr.Values))
	for _, elem := range arr.Values {
		strs = append(strs, elem.StringValue())
	}
	return strs
}

func (c *Connection) getServerInfo() error {
	infoCmd, _ := command.Parse("INFO", nil)
	if err := c.Send(infoCmd); err != nil {
		return fmt.Errorf("failed to send INFO command: %w", err)
	}

	response, err := c.Receive(5 * time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive INFO response: %w", err)
	}

	bulkStr, ok := response.(resp.RedisBulkString)
	if !ok {
		return fmt.Errorf("expected bulk string for INFO, got %T", response)
	}

	c.ServerInfo = make(map[string]string)
	lines := strings.Split(bulkStr.Value, "\r\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			c.ServerInfo[parts[0]] = parts[1]
		}
	}

	return nil
}
