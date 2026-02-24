package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/cosmez/redisman-go/internal/command"
	"github.com/cosmez/redisman-go/internal/conn"
	"github.com/cosmez/redisman-go/internal/output"
	"github.com/cosmez/redisman-go/internal/serializer"
	"github.com/fatih/color"
)

func handleCommand(rl *readline.Instance, c *conn.Connection, reg *command.Registry, parsed *command.ParsedCommand) {
	switch parsed.Name {
	case "EXIT":
		os.Exit(0)
	case "CLEAR":
		fmt.Print("\033[2J\033[H")
	case "HELP":
		handleHelp(reg, parsed)
	case "CONNECT":
		handleConnect(rl, c, reg, parsed)
	case "SAFEKEYS":
		handleSafeKeys(c, parsed)
	case "VIEW":
		handleView(c, parsed)
	case "EXPORT":
		handleExport(c, reg, parsed)
	case "SUBSCRIBE":
		handleSubscribe(rl, c, parsed)
	default:
		handleStandardCommand(rl, c, reg, parsed)
	}
}

func handleHelp(reg *command.Registry, parsed *command.ParsedCommand) {
	if len(parsed.Args) == 0 {
		color.Yellow("Usage: HELP <command>")
		return
	}
	cmdName := strings.ToUpper(parsed.Args[0])
	doc := reg.Get(cmdName)
	if doc == nil {
		color.Red("Unknown command: %s", cmdName)
		return
	}
	color.Cyan("%s %s", doc.Command, doc.Arguments)
	fmt.Println(doc.Summary)
	if doc.Since != "" {
		color.Blue("Since: %s", doc.Since)
	}
}

func handleConnect(rl *readline.Instance, c *conn.Connection, reg *command.Registry, parsed *command.ParsedCommand) {
	if len(parsed.Args) < 2 {
		color.Red("Usage: CONNECT <host> <port> [user] [pass]")
		return
	}

	newHost := parsed.Args[0]
	newPort := parsed.Args[1]
	newUser := ""
	newPass := ""

	if len(parsed.Args) == 3 {
		newPass = parsed.Args[2]
	} else if len(parsed.Args) >= 4 {
		newUser = parsed.Args[2]
		newPass = parsed.Args[3]
	}

	newConn, err := conn.Connect(newHost, newPort, newUser, newPass)
	if err != nil {
		color.Red("Connection failed: %v", err)
		return
	}

	c.Close()
	*c = *newConn // Update the connection in place
	host = newHost
	port = newPort
	username = newUser
	password = newPass

	mergeServerCommands(c, reg)
	rl.SetPrompt(fmt.Sprintf("%s:%s> ", host, port))
	printConnectionInfo(c)
}

func handleSafeKeys(c *conn.Connection, parsed *command.ParsedCommand) {
	pattern := "*"
	if len(parsed.Args) > 0 {
		pattern = parsed.Args[0]
	}

	seq := c.SafeKeys(pattern)
	opts := output.PrintOpts{Color: true, Newline: true}
	output.PrintRedisValues(os.Stdout, os.Stdin, seq, opts, 100)
}

func handleView(c *conn.Connection, parsed *command.ParsedCommand) {
	if len(parsed.Args) == 0 {
		color.Red("Usage: VIEW <key>")
		return
	}

	key := parsed.Args[0]
	typeName, single, collection, err := c.GetKeyValue(key)

	if err != nil {
		color.Red("Error: %v", err)
		return
	}

	if typeName == "none" {
		color.Yellow("Key not found")
		return
	}

	opts := output.PrintOpts{Color: true, Newline: true}
	if parsed.Modifier != "" {
		ser, err := serializer.Get(parsed.Modifier)
		if err != nil {
			color.Red("Serializer error: %v", err)
			return
		}
		opts.Serializer = ser
	}

	if single != nil {
		output.PrintRedisValue(os.Stdout, single, opts)
	} else if collection != nil {
		opts.TypeHint = typeName
		output.PrintRedisValues(os.Stdout, os.Stdin, collection, opts, 100)
	}
}

func handleExport(c *conn.Connection, reg *command.Registry, parsed *command.ParsedCommand) {
	if len(parsed.Args) < 2 {
		color.Red("Usage: EXPORT <filename> <command> [args...]")
		return
	}

	filename := parsed.Args[0]
	subCmdStr := strings.Join(parsed.Args[1:], " ")

	subParsed, err := command.Parse(subCmdStr, reg)
	if err != nil {
		color.Red("Parse error: %v", err)
		return
	}

	if subParsed.Name == "VIEW" {
		if len(subParsed.Args) == 0 {
			color.Red("Usage: EXPORT <filename> VIEW <key>")
			return
		}
		key := subParsed.Args[0]
		typeName, single, collection, err := c.GetKeyValue(key)
		if err != nil {
			color.Red("Error: %v", err)
			return
		}
		if typeName == "none" {
			color.Yellow("Key not found")
			return
		}
		if err := output.ExportAsync(filename, single, collection, typeName); err != nil {
			color.Red("Export failed: %v", err)
		} else {
			color.Green("Exported to %s", filename)
		}
		return
	}

	if err := c.Send(subParsed); err != nil {
		color.Red("Send error: %v", err)
		return
	}

	val, err := c.Receive(0)
	if err != nil {
		color.Red("Receive error: %v", err)
		return
	}

	if err := output.ExportAsync(filename, val, nil, ""); err != nil {
		color.Red("Export failed: %v", err)
	} else {
		color.Green("Exported to %s", filename)
	}
}

func handleSubscribe(rl *readline.Instance, c *conn.Connection, parsed *command.ParsedCommand) {
	if err := c.Send(parsed); err != nil {
		color.Red("Send error: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	color.Yellow("Subscribed. Press Ctrl+C to stop.")

	// Run subscription in a goroutine
	go func() {
		seq := c.Subscribe(ctx)
		opts := output.PrintOpts{Color: true, Newline: true}
		for msg := range seq {
			if parsed.Pipe != "" {
				output.PipeRedisValue(os.Stdout, msg, parsed.Pipe)
			} else {
				output.PrintRedisValue(os.Stdout, msg, opts)
			}
		}
	}()

	// Wait for Ctrl+C
	for {
		_, err := rl.Readline()
		if err == readline.ErrInterrupt {
			cancel()
			break
		}
	}
}

func handleStandardCommand(_ *readline.Instance, c *conn.Connection, reg *command.Registry, parsed *command.ParsedCommand) {
	if reg.IsDangerous(parsed.Name) {
		color.Yellow("The command %s is considered dangerous to execute, execute anyway? (Y/N)", parsed.Name)
		if parsed.Name == "KEYS" {
			color.Cyan("Hint: You can execute SAFEKEYS or SCAN instead.")
		}

		// Read single character confirmation
		var ans []byte
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				ans = append(ans, buf[0])
				if buf[0] == '\n' {
					break
				}
			}
			if err != nil {
				break
			}
		}

		ansStr := strings.TrimSpace(string(ans))
		if len(ansStr) == 0 || (ansStr[0] != 'Y' && ansStr[0] != 'y') {
			color.Yellow("Aborted.")
			return
		}
	}

	if err := c.Send(parsed); err != nil {
		color.Red("Send error: %v", err)
		return
	}

	// Check if blocking command
	timeout := 5 * time.Second
	blockingCmds := map[string]bool{
		"BLPOP": true, "BRPOP": true, "XREAD": true, "BZPOPMIN": true, "BZPOPMAX": true,
	}
	if blockingCmds[parsed.Name] {
		timeout = 0
	}

	val, err := c.Receive(timeout)
	if err != nil {
		color.Red("Receive error: %v", err)
		return
	}

	opts := output.PrintOpts{Color: true, Newline: true}
	if parsed.Modifier != "" {
		ser, err := serializer.Get(parsed.Modifier)
		if err != nil {
			color.Red("Serializer error: %v", err)
			return
		}
		opts.Serializer = ser
	}

	if parsed.Pipe != "" {
		if err := output.PipeRedisValue(os.Stdout, val, parsed.Pipe); err != nil {
			color.Red("Pipe error: %v", err)
		}
	} else {
		output.PrintRedisValue(os.Stdout, val, opts)
	}
}
