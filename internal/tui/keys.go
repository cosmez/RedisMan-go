package tui

import (
	"fmt"
	"time"

	"github.com/cosmez/redisman-go/internal/output"
	"github.com/cosmez/redisman-go/internal/resp"
	"github.com/rivo/tview"
)

// filterDebounce is the delay before a filter keystroke triggers a key reload.
const filterDebounce = 300 * time.Millisecond

// setupKeyHandlers wires the key list selection and filter input handlers.
func (a *App) setupKeyHandlers() {
	// Key selection — fires when the user presses Enter on a key in the list.
	a.keyList.SetSelectedFunc(a.selectKey)

	// Filter input — debounced reload on each keystroke.
	//
	// C# equivalent: DispatcherTimer with 300ms interval, restarted on each keystroke.
	// Go: time.AfterFunc returns a Timer that can be stopped and restarted.
	var debounceTimer *time.Timer

	a.filterInput.SetChangedFunc(func(text string) {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(filterDebounce, func() {
			pattern := text + "*"
			if text == "" {
				pattern = "*"
			}
			go a.loadKeys(pattern)
		})
	})
}

// loadKeysSync populates the key list synchronously (used before app.Run()).
// No mutex needed — called from the main goroutine before the event loop starts.
func (a *App) loadKeysSync(pattern string) {
	a.keyList.Clear()
	a.keys = a.keys[:0]

	for val := range a.conn.SafeKeys(pattern) {
		if _, ok := val.(resp.RedisError); ok {
			break
		}
		name := val.StringValue()
		a.keys = append(a.keys, name)
		a.keyList.AddItem(name, "", 0, nil)
	}
	a.leftPane.SetTitle(fmt.Sprintf(" Keys [%d] ", len(a.keys)))
}

// loadKeys populates the key list from a background goroutine.
// Uses QueueUpdateDraw for thread-safe UI updates and connMu for connection safety.
//
// C# equivalent: Task.Run(() => { foreach (var key in SafeKeys(pattern)) Dispatcher.Invoke(() => list.Add(key)); })
func (a *App) loadKeys(pattern string) {
	a.connMu.Lock()
	defer a.connMu.Unlock()

	// Clear the list on the UI thread first.
	a.app.QueueUpdateDraw(func() {
		a.keyList.Clear()
		a.keys = a.keys[:0]
	})

	for val := range a.conn.SafeKeys(pattern) {
		if _, ok := val.(resp.RedisError); ok {
			break
		}
		name := val.StringValue()
		a.app.QueueUpdateDraw(func() {
			a.keys = append(a.keys, name)
			a.keyList.AddItem(name, "", 0, nil)
			a.leftPane.SetTitle(fmt.Sprintf(" Keys [%d] ", len(a.keys)))
		})
	}
}

// selectKey is called when the user selects a key in the list.
// It fetches the key's value and displays it in a type-specific view.
//
// C#: Like a DataTemplate selector that picks a different UI for each type.
// Go: Manual switch on typeName → populate appropriate widget → switch page.
func (a *App) selectKey(index int, name string, secondaryText string, shortcut rune) {
	if index < 0 || index >= len(a.keys) {
		return
	}

	a.connMu.Lock()
	typeName, single, collection, err := a.conn.GetKeyValue(name)

	// Consume collection iterators while holding the lock (they lazily call Send/Receive).
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
		a.switchContent("output", "Output")
		a.outputView.Clear()
		fmt.Fprintf(a.ansiWriter, "[red]Error: %v[white]\n", err)
		return
	}

	title := fmt.Sprintf("%s (%s)", name, typeName)

	// Track which key/type is displayed (for CRUD operations).
	a.currentKey = name
	a.currentType = typeName

	// String type: show in dedicated string view.
	if single != nil {
		a.stringView.Clear()
		stringWriter := tview.ANSIWriter(a.stringView)
		opts := output.PrintOpts{Color: true, Newline: true}
		output.PrintRedisValue(stringWriter, single, opts)
		a.stringView.ScrollToBeginning()
		a.switchContent("string-view", title)
		a.focusContent()
		return
	}

	// Collection types: show in shared table view.
	if rows != nil {
		a.populateTable(headers, rows)
		a.switchContent("table-view", title)
		a.focusContent()
		return
	}
}
