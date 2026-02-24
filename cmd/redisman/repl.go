package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/cosmez/redisman-go/internal/command"
	"github.com/cosmez/redisman-go/internal/conn"
	"github.com/fatih/color"
	"golang.org/x/term"
)

// replCompleter implements readline.AutoCompleter for tab completion.
type replCompleter struct {
	reg *command.Registry
}

// Do returns completion candidates based on the current input.
func (c *replCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	text := string(line[:pos])
	// Only complete the first word
	if strings.Contains(text, " ") {
		return nil, 0
	}

	matches := c.reg.GetCommands(text)
	for _, match := range matches {
		// Append the remaining part of the command in uppercase, plus a space
		remaining := strings.ToUpper(match[len(text):])
		newLine = append(newLine, []rune(remaining+" "))
	}
	return newLine, len(text)
}

// replHinter implements readline.Painter and readline.Listener to display
// command hints below the input line. Paint only clears stale hints; the
// actual hint is rendered by OnChange (Listener) which fires after readline
// finishes its display update, writing directly to os.Stdout so that
// readline's cursor math is never affected.
type replHinter struct {
	reg       *command.Registry
	promptLen int
	termWidth int
}

// copyAppend returns a new slice: line + suffix runes (never mutates line's backing array).
func copyAppend(line []rune, suffix string) []rune {
	sfx := []rune(suffix)
	out := make([]rune, len(line)+len(sfx))
	copy(out, line)
	copy(out[len(line):], sfx)
	return out
}

// Paint clears any stale hint below the input line. The actual hint is
// rendered separately by OnChange after readline positions the cursor.
func (h *replHinter) Paint(line []rune, pos int) []rune {
	return copyAppend(line, "\033[J")
}

// OnChange is called by readline after each keystroke. It renders a command
// hint on the line below the input using direct terminal writes with
// save/restore cursor, so readline's internal state is unaffected.
func (h *replHinter) OnChange(line []rune, pos int, key rune) ([]rune, int, bool) {
	if len(line) == 0 {
		return nil, 0, false
	}

	text := string(line)
	parts := strings.SplitN(text, " ", 2)
	cmd := parts[0]

	// Auto-uppercase the command word when it matches a known Redis command.
	// This fixes mixed-case after tab completion (e.g., "hgetaLL" → "HGETALL")
	// and gives visual feedback as the user types.
	if cmd != "" {
		upper := strings.ToUpper(cmd)
		if cmd != upper && h.reg.Get(upper) != nil {
			return []rune(upper + text[len(cmd):]), pos, true
		}
	}

	// Only show hints after a space (command name is complete)
	if len(parts) < 2 || cmd == "" {
		return nil, 0, false
	}

	doc := h.findDoc(text)
	if doc == nil {
		return nil, 0, false
	}

	hint := fmt.Sprintf("%s %s", doc.Command, doc.Arguments)
	col := h.promptLen + pos

	// Calculate how many terminal rows the hint occupies.
	hintWidth := 2 + len(hint) + 3 + len(doc.Summary) // "  <hint> - <summary>"
	hintRows := 1
	if h.termWidth > 0 {
		hintRows = (hintWidth + h.termWidth - 1) / h.termWidth
	}

	// \n\r         — newline (scrolls if on last row) + carriage return
	// \033[K       — clear to end of line
	// hint text with colors
	// \033[<n>A    — move back up by hint row count
	// \r\033[<c>C  — move to cursor column
	fmt.Fprintf(os.Stdout, "\n\r\033[K  \033[36m%s\033[0m\033[34m - %s\033[0m\033[%dA\r\033[%dC",
		hint, doc.Summary, hintRows, col)

	return nil, 0, false
}

// findDoc looks up command documentation, trying compound commands first
// (e.g., "CLIENT INFO") then falling back to the base command (e.g., "CLIENT").
func (h *replHinter) findDoc(text string) *command.CommandDoc {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil
	}

	base := strings.ToUpper(parts[0])

	// Try compound command (e.g., "CLIENT INFO")
	if len(parts) >= 2 {
		compound := base + " " + strings.ToUpper(parts[1])
		if doc := h.reg.Get(compound); doc != nil {
			return doc
		}
	}

	return h.reg.Get(base)
}

func runRepl() {
	reg, err := command.NewRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load commands: %v\n", err)
		os.Exit(1)
	}

	c, err := conn.Connect(host, port, username, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	mergeServerCommands(c, reg)
	printConnectionInfo(c)

	homeDir, _ := os.UserHomeDir()
	historyFile := filepath.Join(homeDir, ".redisman_history")

	prompt := fmt.Sprintf("%s:%s> ", host, port)
	tw, _, _ := term.GetSize(int(os.Stdout.Fd()))
	hinter := &replHinter{reg: reg, promptLen: len(prompt), termWidth: tw}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     historyFile,
		AutoComplete:    &replCompleter{reg: reg},
		Painter:         hinter,
		Listener:        hinter,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize readline: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parsed, err := command.Parse(line, reg)
		if err != nil {
			color.Red("Parse error: %v", err)
			continue
		}

		if parsed.Name == "" {
			continue
		}

		handleCommand(rl, c, reg, parsed)

		// Refresh terminal width in case the window was resized.
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			hinter.termWidth = w
		}
	}
}

func printConnectionInfo(c *conn.Connection) {
	if c.ServerInfo == nil {
		return
	}

	if errStr, ok := c.ServerInfo["error"]; ok {
		color.Yellow("Warning: Could not fetch server info: %s", errStr)
		return
	}

	version := c.ServerInfo["redis_version"]
	mode := c.ServerInfo["redis_mode"]
	if mode == "" {
		mode = "standalone"
	}
	color.Green("Connected to Redis %s %s", version, mode)

	memUsed := c.ServerInfo["used_memory_human"]
	memTotal := c.ServerInfo["total_system_memory_human"]
	if memTotal == "" {
		memTotal = "Unknown"
	}
	color.Cyan("Memory: %s / %s", memUsed, memTotal)

	clients := c.ServerInfo["connected_clients"]
	color.Cyan("Connected Clients: %s", clients)

	// Count databases
	for k, v := range c.ServerInfo {
		if strings.HasPrefix(k, "db") {
			// e.g., db0:keys=150,expires=0,avg_ttl=0
			parts := strings.Split(v, ",")
			if len(parts) > 0 {
				keysPart := strings.Split(parts[0], "=")
				if len(keysPart) == 2 {
					color.Cyan("%s (%s Total Keys)", k, keysPart[1])
				}
			}
		}
	}
	fmt.Println()
}

// mergeServerCommands fetches the COMMAND list from the server and merges
// any new commands into the registry for autocomplete. Failures are non-fatal.
func mergeServerCommands(c *conn.Connection, reg *command.Registry) {
	cmds, err := c.FetchServerCommands()
	if err != nil {
		color.Yellow("Warning: Could not fetch server commands: %v", err)
		return
	}
	if cmds != nil {
		reg.MergeServerCommands(cmds)
	}
}
