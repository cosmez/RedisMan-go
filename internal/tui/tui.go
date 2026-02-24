package tui

import (
	"fmt"
	"io"
	"sync"

	"github.com/cosmez/redisman-go/internal/command"
	"github.com/cosmez/redisman-go/internal/conn"
	"github.com/fatih/color"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// App holds all TUI state.
//
// C#: Roughly equivalent to a WPF Window class with bound properties.
// Go: A plain struct — no inheritance, no data binding.
type App struct {
	conn     *conn.Connection
	registry *command.Registry

	app          *tview.Application
	layout       *tview.Flex   // root layout (restored after modals)
	contentPages *tview.Pages  // swaps between outputView and future type-specific views
	outputView   *tview.TextView
	keyList      *tview.List
	cmdInput     *tview.InputField
	filterInput  *tview.InputField
	ansiWriter   io.Writer // tview.ANSIWriter(outputView) — translates ANSI escapes to tview color tags
	leftPane     *tview.Flex // for updating key list title with scroll position

	// Type-specific key views
	tableView     *tview.Table    // shared table for list/set/hash/zset/stream
	stringView    *tview.TextView // dedicated view for string key values
	activeContent tview.Primitive // currently visible content widget (for focus cycling)

	// Action bar and CRUD state
	actionBar   *tview.Flex     // contextual edit buttons between content and command input
	statusLabel *tview.TextView // transient status feedback (right side of action bar)
	currentKey  string          // name of currently viewed key (empty when on output page)
	currentType string          // Redis type of currently viewed key

	bottomPane *tview.Flex // command input container (for border highlighting)

	focusOrder []tview.Primitive
	focusIndex int

	keys   []string   // current key names (parallel to keyList items)
	connMu sync.Mutex // serializes all connection operations
}

// newApp creates and initializes the TUI application with all widgets.
// Separated from Run() for testability (smoke tests can build the app without
// calling Run, which takes over the terminal).
func newApp(c *conn.Connection, reg *command.Registry) *App {
	a := &App{
		conn:     c,
		registry: reg,
		app:      tview.NewApplication(),
	}

	// --- Left pane: filter + key list ---
	a.filterInput = tview.NewInputField().
		SetLabel("Filter: ").
		SetFieldBackgroundColor(tcell.ColorBlack)

	a.keyList = tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true)

	a.leftPane = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.filterInput, 1, 0, false).
		AddItem(a.keyList, 0, 1, true)
	a.leftPane.SetBorder(true).SetTitle(" Keys ")

	// --- Right pane: content pages (outputView is the default page) ---
	a.outputView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	a.contentPages = tview.NewPages().
		AddPage("output", a.outputView, true, true)

	a.contentPages.SetBorder(true).SetTitle(contentTitle("Output"))

	// --- Type-specific key views ---
	a.tableView = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false). // row-selectable, not cell-selectable
		SetFixed(1, 0)              // freeze header row

	a.stringView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	a.contentPages.
		AddPage("string-view", a.stringView, true, false).
		AddPage("table-view", a.tableView, true, false)

	a.activeContent = a.outputView

	// Escape on type-specific views returns to output page and focuses key list.
	a.tableView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.switchContent("output", "Output")
			a.focusIndex = 1 // keyList
			a.app.SetFocus(a.keyList)
			a.highlightFocusedPane()
			return nil
		}
		return event
	})
	a.stringView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.switchContent("output", "Output")
			a.focusIndex = 1 // keyList
			a.app.SetFocus(a.keyList)
			a.highlightFocusedPane()
			return nil
		}
		return event
	})

	// --- Action bar: contextual edit buttons + status label ---
	a.actionBar = tview.NewFlex().SetDirection(tview.FlexColumn)
	a.actionBar.SetBackgroundColor(tcell.ColorDarkSlateGray)

	a.statusLabel = tview.NewTextView().SetDynamicColors(true)
	a.statusLabel.SetBackgroundColor(tcell.ColorDarkSlateGray)
	a.statusLabel.SetTextAlign(tview.AlignRight)

	// --- Bottom pane: command input ---
	a.cmdInput = tview.NewInputField().
		SetLabel("> ").
		SetFieldBackgroundColor(tcell.ColorBlack)

	a.bottomPane = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.cmdInput, 1, 0, true)
	a.bottomPane.SetBorder(true).SetTitle(" Command ")

	// --- Compose layout ---
	rightSide := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.contentPages, 0, 1, false).
		AddItem(a.actionBar, 1, 0, false).
		AddItem(a.bottomPane, 3, 0, false)

	a.layout = tview.NewFlex().
		AddItem(a.leftPane, 0, 3, false).
		AddItem(rightSide, 0, 7, false)

	// --- ANSI writer for output.PrintRedisValue ---
	a.ansiWriter = tview.ANSIWriter(a.outputView)

	// --- Scroll position indicators in pane titles ---
	a.keyList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		total := a.keyList.GetItemCount()
		a.leftPane.SetTitle(fmt.Sprintf(" Keys [%d/%d] ", index+1, total))
	})
	a.outputView.SetChangedFunc(func() {
		if a.activeContent == a.outputView {
			row, _ := a.outputView.GetScrollOffset()
			a.contentPages.SetTitle(contentTitle(fmt.Sprintf("Output [line %d]", row+1)))
		}
	})

	// --- Forward typing on keyList to filter input ---
	// When the key list is focused and the user types a printable character,
	// move focus to the filter input and append the character. This lets the
	// user start filtering just by typing without having to Tab to the filter.
	a.keyList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		ch := event.Rune()
		if ch != 0 && event.Key() == tcell.KeyRune {
			a.filterInput.SetText(a.filterInput.GetText() + string(ch))
			a.app.SetFocus(a.filterInput)
			a.focusIndex = 0 // filterInput is index 0
			a.highlightFocusedPane()
			return nil
		}
		return event
	})

	// --- Focus cycling ---
	a.focusOrder = []tview.Primitive{a.filterInput, a.keyList, a.outputView, a.cmdInput}
	a.focusIndex = 3 // start on cmdInput

	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			a.focusIndex = (a.focusIndex + 1) % len(a.focusOrder)
			a.app.SetFocus(a.focusOrder[a.focusIndex])
			a.highlightFocusedPane()
			return nil
		case tcell.KeyBacktab:
			a.focusIndex = (a.focusIndex - 1 + len(a.focusOrder)) % len(a.focusOrder)
			a.app.SetFocus(a.focusOrder[a.focusIndex])
			a.highlightFocusedPane()
			return nil
		}
		return event
	})

	// --- Wire event handlers ---
	a.setupKeyHandlers()
	a.setupCommandInput()
	a.setupEditHandlers()

	// Set initial border highlight (cmdInput is focused).
	a.highlightFocusedPane()

	return a
}

// Run creates and starts the TUI application. This is the public entry point
// called from main.go when --tui is passed.
func Run(c *conn.Connection, registry *command.Registry) error {
	// Force color output — fatih/color auto-detects no-terminal and disables
	// colors, but tview.ANSIWriter needs ANSI codes to translate into tview
	// color tags.
	color.NoColor = false

	a := newApp(c, registry)

	// Load keys synchronously before the event loop starts (no concurrency concerns).
	if c != nil {
		a.loadKeysSync("*")
	}

	return a.app.EnableMouse(true).SetRoot(a.layout, true).SetFocus(a.cmdInput).Run()
}
