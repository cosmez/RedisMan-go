package tui

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/cosmez/redisman-go/internal/command"
	"github.com/cosmez/redisman-go/internal/conn"
	"github.com/cosmez/redisman-go/internal/output"
	"github.com/cosmez/redisman-go/internal/serializer"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// setupCommandInput wires the command input field with Enter handling
// and tab completion.
func (a *App) setupCommandInput() {
	// Tab completion — tview renders a navigable dropdown automatically.
	//
	// C# equivalent: TextBox with an AutoComplete popup (like WPF's AutoCompleteBox).
	// Go/tview: SetAutocompleteFunc returns candidate strings; tview draws the dropdown.
	a.cmdInput.SetAutocompleteFunc(func(currentText string) []string {
		if currentText == "" || strings.Contains(currentText, " ") {
			return nil
		}
		return a.registry.GetCommands(strings.ToUpper(currentText))
	})

	// Enter handler — parse and execute the command.
	a.cmdInput.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		text := strings.TrimSpace(a.cmdInput.GetText())
		if text == "" {
			return
		}
		a.cmdInput.SetText("")
		a.executeCommand(text)
	})
}

// executeCommand parses and routes a command, writing results to the output view.
func (a *App) executeCommand(input string) {
	// Switch to output page for command results.
	a.switchContent("output", "Output")

	// Echo the command.
	fmt.Fprintf(a.ansiWriter, "\n[green]> %s[white]\n", input)

	parsed, err := command.Parse(input, a.registry)
	if err != nil {
		fmt.Fprintf(a.ansiWriter, "[red]Parse error: %v[white]\n", err)
		a.outputView.ScrollToEnd()
		return
	}

	switch parsed.Name {
	case "EXIT":
		a.app.Stop()
	case "CLEAR":
		a.outputView.Clear()
	case "HELP":
		a.handleHelp(parsed)
	case "CONNECT":
		a.handleConnect(parsed)
	case "SAFEKEYS":
		a.handleSafeKeys(parsed)
	case "VIEW":
		a.handleView(parsed)
	case "EXPORT":
		a.handleExport(parsed)
	case "SUBSCRIBE":
		fmt.Fprintf(a.ansiWriter, "[yellow]SUBSCRIBE is not supported in TUI mode. Use REPL mode instead.[white]\n")
	default:
		a.handleStandardCommand(parsed)
	}

	a.outputView.ScrollToEnd()
}

func (a *App) handleHelp(parsed *command.ParsedCommand) {
	if len(parsed.Args) == 0 {
		fmt.Fprintf(a.ansiWriter, "[yellow]Usage: HELP <command>[white]\n")
		return
	}
	cmdName := strings.ToUpper(parsed.Args[0])
	doc := a.registry.Get(cmdName)
	if doc == nil {
		fmt.Fprintf(a.ansiWriter, "[red]Unknown command: %s[white]\n", cmdName)
		return
	}
	fmt.Fprintf(a.ansiWriter, "[cyan]%s %s[white]\n", doc.Command, doc.Arguments)
	fmt.Fprintf(a.ansiWriter, "%s\n", doc.Summary)
	if doc.Since != "" {
		fmt.Fprintf(a.ansiWriter, "[blue]Since: %s[white]\n", doc.Since)
	}
}

func (a *App) handleConnect(parsed *command.ParsedCommand) {
	if len(parsed.Args) < 2 {
		fmt.Fprintf(a.ansiWriter, "[red]Usage: CONNECT <host> <port> [user] [pass][white]\n")
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

	a.connMu.Lock()
	newConn, err := conn.Connect(newHost, newPort, newUser, newPass)
	if err != nil {
		a.connMu.Unlock()
		fmt.Fprintf(a.ansiWriter, "[red]Connection failed: %v[white]\n", err)
		return
	}

	a.conn.Close()
	*a.conn = *newConn
	a.connMu.Unlock()

	// Merge server commands for autocomplete.
	cmds, fetchErr := a.conn.FetchServerCommands()
	if fetchErr == nil && cmds != nil {
		a.registry.MergeServerCommands(cmds)
	}

	fmt.Fprintf(a.ansiWriter, "[green]Connected to %s:%s[white]\n", newHost, newPort)

	// Reload keys in background.
	go a.loadKeys("*")
}

func (a *App) handleSafeKeys(parsed *command.ParsedCommand) {
	pattern := "*"
	if len(parsed.Args) > 0 {
		pattern = parsed.Args[0]
	}

	a.connMu.Lock()
	i := 0
	opts := output.PrintOpts{Color: true, Newline: true}
	for val := range a.conn.SafeKeys(pattern) {
		i++
		fmt.Fprintf(a.ansiWriter, "%d) ", i)
		output.PrintRedisValue(a.ansiWriter, val, opts)
	}
	a.connMu.Unlock()
}

func (a *App) handleView(parsed *command.ParsedCommand) {
	if len(parsed.Args) == 0 {
		fmt.Fprintf(a.ansiWriter, "[red]Usage: VIEW <key>[white]\n")
		return
	}

	key := parsed.Args[0]

	a.connMu.Lock()
	typeName, single, collection, err := a.conn.GetKeyValue(key)

	// Consume collection iterators while holding the lock.
	var headers []string
	var rows [][]string
	if err == nil && collection != nil {
		switch typeName {
		case "list":
			headers, rows = consumeList(collection)
		case "set":
			headers, rows = consumeSet(collection)
		case "hash":
			headers, rows = consumeHash(collection)
		case "zset":
			headers, rows = consumeSortedSet(collection)
		case "stream":
			headers, rows = consumeStream(collection)
		}
	}
	a.connMu.Unlock()

	if err != nil {
		fmt.Fprintf(a.ansiWriter, "[red]Error: %v[white]\n", err)
		return
	}

	if typeName == "none" {
		fmt.Fprintf(a.ansiWriter, "[yellow]Key not found[white]\n")
		return
	}

	title := fmt.Sprintf("%s (%s)", key, typeName)

	// String type: show in dedicated string view.
	if single != nil {
		a.stringView.Clear()
		stringWriter := tview.ANSIWriter(a.stringView)
		opts := output.PrintOpts{Color: true, Newline: true}
		if parsed.Modifier != "" {
			if ser, serErr := serializer.Get(parsed.Modifier); serErr == nil {
				opts.Serializer = ser
			}
		}
		output.PrintRedisValue(stringWriter, single, opts)
		a.stringView.ScrollToBeginning()
		a.switchContent("string-view", title)
		return
	}

	// Collection types: show in shared table view.
	if rows != nil {
		a.populateTable(headers, rows)
		a.switchContent("table-view", title)
		return
	}
}

func (a *App) handleExport(parsed *command.ParsedCommand) {
	if len(parsed.Args) < 2 {
		fmt.Fprintf(a.ansiWriter, "[red]Usage: EXPORT <filename> <command> [args...][white]\n")
		return
	}

	filename := parsed.Args[0]
	subCmdStr := strings.Join(parsed.Args[1:], " ")

	subParsed, err := command.Parse(subCmdStr, a.registry)
	if err != nil {
		fmt.Fprintf(a.ansiWriter, "[red]Parse error: %v[white]\n", err)
		return
	}

	if subParsed.Name == "VIEW" {
		if len(subParsed.Args) == 0 {
			fmt.Fprintf(a.ansiWriter, "[red]Usage: EXPORT <filename> VIEW <key>[white]\n")
			return
		}
		key := subParsed.Args[0]

		a.connMu.Lock()
		typeName, single, collection, kvErr := a.conn.GetKeyValue(key)
		if kvErr != nil {
			a.connMu.Unlock()
			fmt.Fprintf(a.ansiWriter, "[red]Error: %v[white]\n", kvErr)
			return
		}
		if typeName == "none" {
			a.connMu.Unlock()
			fmt.Fprintf(a.ansiWriter, "[yellow]Key not found[white]\n")
			return
		}
		if exportErr := output.ExportAsync(filename, single, collection, typeName); exportErr != nil {
			a.connMu.Unlock()
			fmt.Fprintf(a.ansiWriter, "[red]Export failed: %v[white]\n", exportErr)
			return
		}
		a.connMu.Unlock()
		fmt.Fprintf(a.ansiWriter, "[green]Exported to %s[white]\n", filename)
		return
	}

	a.connMu.Lock()
	if sendErr := a.conn.Send(subParsed); sendErr != nil {
		a.connMu.Unlock()
		fmt.Fprintf(a.ansiWriter, "[red]Send error: %v[white]\n", sendErr)
		return
	}
	val, recvErr := a.conn.Receive(0)
	a.connMu.Unlock()

	if recvErr != nil {
		fmt.Fprintf(a.ansiWriter, "[red]Receive error: %v[white]\n", recvErr)
		return
	}

	if exportErr := output.ExportAsync(filename, val, nil, ""); exportErr != nil {
		fmt.Fprintf(a.ansiWriter, "[red]Export failed: %v[white]\n", exportErr)
	} else {
		fmt.Fprintf(a.ansiWriter, "[green]Exported to %s[white]\n", filename)
	}
}

// handleStandardCommand sends a Redis command and displays the result.
// Dangerous commands show a confirmation modal first.
func (a *App) handleStandardCommand(parsed *command.ParsedCommand) {
	if a.registry.IsDangerous(parsed.Name) {
		a.confirmDangerous(parsed, func() {
			a.sendAndDisplay(parsed)
		})
		return
	}
	a.sendAndDisplay(parsed)
}

// sendAndDisplay sends a parsed command to Redis and writes the result to the output view.
func (a *App) sendAndDisplay(parsed *command.ParsedCommand) {
	a.connMu.Lock()
	if err := a.conn.Send(parsed); err != nil {
		a.connMu.Unlock()
		fmt.Fprintf(a.ansiWriter, "[red]Send error: %v[white]\n", err)
		return
	}

	// Match REPL timeout behavior: 5s default, 0 for blocking commands.
	timeout := 5 * time.Second
	blockingCmds := map[string]bool{
		"BLPOP": true, "BRPOP": true, "XREAD": true, "BZPOPMIN": true, "BZPOPMAX": true,
	}
	if blockingCmds[parsed.Name] {
		timeout = 0
	}

	val, err := a.conn.Receive(timeout)
	a.connMu.Unlock()

	if err != nil {
		fmt.Fprintf(a.ansiWriter, "[red]Receive error: %v[white]\n", err)
		return
	}

	opts := output.PrintOpts{Color: true, Newline: true}
	if parsed.Modifier != "" {
		if ser, serErr := serializer.Get(parsed.Modifier); serErr == nil {
			opts.Serializer = ser
		}
	}

	if parsed.Pipe != "" {
		// Pipe output to a shell command, capture into buffer, then write to outputView.
		var buf bytes.Buffer
		if pipeErr := output.PipeRedisValue(&buf, val, parsed.Pipe); pipeErr != nil {
			fmt.Fprintf(a.ansiWriter, "[red]Pipe error: %v[white]\n", pipeErr)
		} else {
			fmt.Fprint(a.ansiWriter, buf.String())
		}
	} else {
		output.PrintRedisValue(a.ansiWriter, val, opts)
	}

	a.outputView.ScrollToEnd()
}

// confirmDangerous shows a modal dialog for dangerous command confirmation.
//
// C# equivalent: MessageBox.Show("Are you sure?", ..., MessageBoxButton.YesNo)
// Go/tview: tview.Modal temporarily replaces the root; restored on button press.
func (a *App) confirmDangerous(parsed *command.ParsedCommand, onConfirm func()) {
	hint := ""
	if parsed.Name == "KEYS" {
		hint = "\nHint: You can use SAFEKEYS or SCAN instead."
	}

	modal := tview.NewModal().
		SetText(fmt.Sprintf("The command %s is considered dangerous.\nExecute anyway?%s", parsed.Name, hint)).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			// Restore the normal layout.
			a.app.SetRoot(a.layout, true).SetFocus(a.cmdInput)
			if buttonLabel == "Yes" {
				onConfirm()
			} else {
				fmt.Fprintf(a.ansiWriter, "[yellow]Aborted.[white]\n")
			}
		})

	a.app.SetRoot(modal, true)
}
