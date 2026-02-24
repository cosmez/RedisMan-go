package tui

import (
	"iter"
	"strconv"
	"strings"

	"github.com/cosmez/redisman-go/internal/resp"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// appName is the application name shown in the content pane border title.
const appName = "RedisMan"

// contentTitle formats a content pane title with the app name prefix.
// e.g. contentTitle("Output") → " RedisMan | Output "
//      contentTitle("")       → " RedisMan "
func contentTitle(subtitle string) string {
	if subtitle == "" {
		return " " + appName + " "
	}
	return " " + appName + " | " + subtitle + " "
}

// switchContent switches the visible content page and updates focus cycling.
//
// C#: Like switching a TabControl's selected tab and updating keyboard navigation.
// Go: Manual page switch + focusOrder slot reassignment.
func (a *App) switchContent(pageName string, title string) {
	a.contentPages.SwitchToPage(pageName)
	a.contentPages.SetTitle(contentTitle(title))

	switch pageName {
	case "output":
		a.activeContent = a.outputView
		a.currentKey = ""
		a.currentType = ""
	case "string-view":
		a.activeContent = a.stringView
	case "table-view":
		a.activeContent = a.tableView
	}
	a.focusOrder[2] = a.activeContent
	a.updateActionBar()
}

// focusContent moves focus to the active content view and updates focusIndex
// so Tab/Backtab cycling stays in sync.
func (a *App) focusContent() {
	a.focusIndex = 2 // content view is always index 2 in focusOrder
	a.app.SetFocus(a.activeContent)
	a.highlightFocusedPane()
}

// highlightFocusedPane updates border colors to indicate which pane has focus.
//
// C#: Like setting a WPF Border.BorderBrush on the focused panel and clearing others.
// Go: Manual SetBorderColor on each container based on focusIndex.
func (a *App) highlightFocusedPane() {
	const (
		defaultColor   = tcell.ColorWhite
		highlightColor = tcell.ColorAqua
	)

	a.leftPane.SetBorderColor(defaultColor)
	a.contentPages.SetBorderColor(defaultColor)
	a.bottomPane.SetBorderColor(defaultColor)

	switch {
	case a.focusIndex <= 1: // filterInput or keyList
		a.leftPane.SetBorderColor(highlightColor)
	case a.focusIndex == 2: // content view
		a.contentPages.SetBorderColor(highlightColor)
	case a.focusIndex == 3: // command input
		a.bottomPane.SetBorderColor(highlightColor)
	}
}

// --- Iterator consumers ---
// These drain iter.Seq iterators into [][]string rows.
// Must be called while holding connMu (iterators lazily call Send/Receive).

// consumeList drains a list iterator into table rows: [index, value].
func consumeList(collection iter.Seq[resp.RedisValue]) (headers []string, rows [][]string) {
	headers = []string{"#", "Value"}
	i := 0
	for val := range collection {
		if _, ok := val.(resp.RedisError); ok {
			break
		}
		i++
		rows = append(rows, []string{strconv.Itoa(i), val.StringValue()})
	}
	return
}

// consumeSet drains a set iterator into table rows: [member].
func consumeSet(collection iter.Seq[resp.RedisValue]) (headers []string, rows [][]string) {
	headers = []string{"Member"}
	for val := range collection {
		if _, ok := val.(resp.RedisError); ok {
			break
		}
		rows = append(rows, []string{val.StringValue()})
	}
	return
}

// consumeHash drains a hash iterator into table rows: [field, value].
// SafeHash yields RedisArray{field, value} pairs.
func consumeHash(collection iter.Seq[resp.RedisValue]) (headers []string, rows [][]string) {
	headers = []string{"Field", "Value"}
	for val := range collection {
		if _, ok := val.(resp.RedisError); ok {
			break
		}
		if arr, ok := val.(resp.RedisArray); ok && len(arr.Values) >= 2 {
			rows = append(rows, []string{arr.Values[0].StringValue(), arr.Values[1].StringValue()})
		}
	}
	return
}

// consumeSortedSet drains a zset iterator into table rows: [member, score].
// SafeSortedSets yields alternating member, score values (not paired like SafeHash).
func consumeSortedSet(collection iter.Seq[resp.RedisValue]) (headers []string, rows [][]string) {
	headers = []string{"Member", "Score"}
	var pending string
	hasPending := false
	for val := range collection {
		if _, ok := val.(resp.RedisError); ok {
			break
		}
		if !hasPending {
			pending = val.StringValue()
			hasPending = true
		} else {
			rows = append(rows, []string{pending, val.StringValue()})
			hasPending = false
		}
	}
	return
}

// consumeStream drains a stream iterator into table rows: [id, data].
// SafeStream yields RedisArray{id, RedisArray{field, value, field, value, ...}}.
func consumeStream(collection iter.Seq[resp.RedisValue]) (headers []string, rows [][]string) {
	headers = []string{"ID", "Data"}
	for val := range collection {
		if _, ok := val.(resp.RedisError); ok {
			break
		}
		if arr, ok := val.(resp.RedisArray); ok && len(arr.Values) >= 2 {
			id := arr.Values[0].StringValue()
			data := formatStreamFields(arr.Values[1])
			rows = append(rows, []string{id, data})
		}
	}
	return
}

// formatStreamFields formats stream entry fields as "field1=val1, field2=val2, ...".
func formatStreamFields(v resp.RedisValue) string {
	arr, ok := v.(resp.RedisArray)
	if !ok {
		return v.StringValue()
	}
	var parts []string
	for i := 0; i < len(arr.Values)-1; i += 2 {
		parts = append(parts, arr.Values[i].StringValue()+"="+arr.Values[i+1].StringValue())
	}
	return strings.Join(parts, ", ")
}

// populateTable clears the shared table and fills it with headers and rows.
func (a *App) populateTable(headers []string, rows [][]string) {
	a.tableView.Clear()

	// Header row (fixed, styled).
	for col, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		a.tableView.SetCell(0, col, cell)
	}

	// Data rows.
	for r, row := range rows {
		for c, val := range row {
			cell := tview.NewTableCell(val).
				SetExpansion(1)
			// First column gets a distinct color for visual structure.
			if c == 0 && len(headers) > 1 {
				cell.SetTextColor(tcell.ColorAqua)
			}
			a.tableView.SetCell(r+1, c, cell) // +1 for header row
		}
	}

	a.tableView.ScrollToBeginning()
}
