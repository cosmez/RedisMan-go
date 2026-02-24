package tui

import (
	"fmt"
	"time"

	"github.com/cosmez/redisman-go/internal/resp"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// --- Helpers ---

// updateActionBar rebuilds the action bar buttons based on the current key type.
func (a *App) updateActionBar() {
	a.actionBar.Clear()

	if a.currentKey == "" {
		// No key selected — show nothing (or a faint hint).
		label := tview.NewTextView().SetText(" No key selected")
		label.SetBackgroundColor(tcell.ColorDarkSlateGray)
		label.SetTextColor(tcell.ColorGray)
		a.actionBar.AddItem(label, 0, 1, false)
		a.actionBar.AddItem(a.statusLabel, 0, 1, false)
		return
	}

	// Shortcut key is shown underlined in yellow via ANSI codes written through
	// tview.ANSIWriter. Plain tview color-tag escaping ("[[]") doesn't work
	// reliably in single-line TextViews, so we use real ANSI sequences instead.
	switch a.currentType {
	case "string":
		a.addActionButton("E", "dit", func() { a.editString() })
		a.addActionButton("R", "efresh", func() { a.refreshCurrentKey() })
		a.addActionButton("X", " Del Key", func() { a.deleteKey() })
	case "list":
		a.addActionButton("E", "dit", func() { a.dispatchEdit() })
		a.addActionButton("A", "dd", func() { a.addListItem() })
		a.addActionButton("D", "elete", func() { a.dispatchDelete() })
		a.addActionButton("R", "efresh", func() { a.refreshCurrentKey() })
		a.addActionButton("X", " Del Key", func() { a.deleteKey() })
	case "set":
		a.addActionButton("A", "dd", func() { a.addSetMember() })
		a.addActionButton("D", "elete", func() { a.dispatchDelete() })
		a.addActionButton("R", "efresh", func() { a.refreshCurrentKey() })
		a.addActionButton("X", " Del Key", func() { a.deleteKey() })
	case "hash":
		a.addActionButton("E", "dit", func() { a.dispatchEdit() })
		a.addActionButton("A", "dd", func() { a.addHashField() })
		a.addActionButton("D", "elete", func() { a.dispatchDelete() })
		a.addActionButton("R", "efresh", func() { a.refreshCurrentKey() })
		a.addActionButton("X", " Del Key", func() { a.deleteKey() })
	case "zset":
		a.addActionButton("E", "dit", func() { a.dispatchEdit() })
		a.addActionButton("A", "dd", func() { a.addZSetMember() })
		a.addActionButton("D", "elete", func() { a.dispatchDelete() })
		a.addActionButton("R", "efresh", func() { a.refreshCurrentKey() })
		a.addActionButton("X", " Del Key", func() { a.deleteKey() })
	case "stream":
		a.addActionButton("A", "dd", func() { a.addStreamEntry() })
		a.addActionButton("D", "elete", func() { a.dispatchDelete() })
		a.addActionButton("R", "efresh", func() { a.refreshCurrentKey() })
		a.addActionButton("X", " Del Key", func() { a.deleteKey() })
	}

	// Status label fills remaining space on the right.
	a.actionBar.AddItem(a.statusLabel, 0, 1, false)
}

// addActionButton adds a clickable button to the action bar.
// shortcut is the highlighted key letter, rest is the remaining label text.
// Uses tview.Button (native mouse/keyboard support) with tview color tags:
// the shortcut letter is rendered in yellow+bold+underline via [yellow::bu].
func (a *App) addActionButton(shortcut string, rest string, action func()) {
	label := fmt.Sprintf("[yellow::bu]%s[-::-]%s", shortcut, rest)
	btn := tview.NewButton(label).SetSelectedFunc(action)
	btn.SetBackgroundColor(tcell.ColorDarkSlateGray)
	btn.SetLabelColor(tcell.ColorWhite)
	btn.SetBackgroundColorActivated(tcell.ColorDarkCyan)
	btn.SetLabelColorActivated(tcell.ColorWhite)

	// visible width: shortcut letter + rest + padding
	width := len(shortcut) + len(rest) + 4
	a.actionBar.AddItem(btn, width, 0, false)
}

// showEditModal displays a form dialog as a modal overlay.
// formSetup configures the form fields; onSubmit is called when the user saves.
// Keyboard shortcuts: Ctrl+S saves, Escape cancels.
func (a *App) showEditModal(title string, formSetup func(form *tview.Form), onSubmit func(form *tview.Form)) {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" " + title + " (Ctrl+S save, Esc cancel) ")
	form.SetButtonsAlign(tview.AlignCenter)

	formSetup(form)

	form.AddButton("Save (Ctrl+S)", func() {
		onSubmit(form)
	})
	form.AddButton("Cancel (Esc)", func() {
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
	})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlS {
			onSubmit(form)
			return nil
		}
		if event.Key() == tcell.KeyEscape {
			a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
			return nil
		}
		return event
	})

	// Center the modal — same approach as confirmDangerous.
	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 60, 0, true).
			AddItem(nil, 0, 1, false),
			0, 2, true).
		AddItem(nil, 0, 1, false)

	a.app.SetRoot(modal, true).SetFocus(form)
}

// showTextAreaModal displays a TextArea for editing strings with visible
// Save/Cancel buttons below. Ctrl+S saves, Escape cancels.
func (a *App) showTextAreaModal(title string, initialValue string, onSave func(value string)) {
	textArea := tview.NewTextArea()
	textArea.SetText(initialValue, false)
	textArea.SetBorder(true).SetTitle(" " + title + " ")

	cancel := func() {
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
	}
	save := func() {
		onSave(textArea.GetText())
	}

	textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlS {
			save()
			return nil
		}
		if event.Key() == tcell.KeyEscape {
			cancel()
			return nil
		}
		return event
	})

	// Button row below the text area.
	buttons := tview.NewForm().
		AddButton("Save (Ctrl+S)", save).
		AddButton("Cancel (Esc)", cancel)
	buttons.SetButtonsAlign(tview.AlignCenter)
	buttons.SetBackgroundColor(tcell.ColorDefault)

	// Compose: text area + button row, centered on screen.
	inner := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(textArea, 0, 1, true).
		AddItem(buttons, 3, 0, false)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(inner, 0, 4, true).
			AddItem(nil, 0, 1, false),
			0, 4, true).
		AddItem(nil, 0, 1, false)

	a.app.SetRoot(modal, true).SetFocus(textArea)
}

// sendEditCommand sends a raw Redis command, checks for errors, and refreshes.
// Must NOT be called while holding connMu.
func (a *App) sendEditCommand(args ...string) {
	a.connMu.Lock()
	err := a.conn.SendRaw(args...)
	if err != nil {
		a.connMu.Unlock()
		a.showError("Send error: " + err.Error())
		return
	}

	val, err := a.conn.Receive(5 * time.Second)
	a.connMu.Unlock()

	if err != nil {
		a.showError("Receive error: " + err.Error())
		return
	}
	if errResp, ok := val.(resp.RedisError); ok {
		a.showError("Redis error: " + errResp.Value)
		return
	}

	a.refreshCurrentKey()
	a.showStatus("[green]Saved")
}

// showError writes an error message to the output view and flashes "Error" in the action bar.
func (a *App) showError(msg string) {
	a.switchContent("output", "Output")
	fmt.Fprintf(a.ansiWriter, "[red]%s[white]\n", msg)
	a.showStatus("[red]Error")
}

// showStatus displays a transient message in the action bar status label.
// The message auto-clears after 2 seconds. Uses tview dynamic color tags
// (e.g. "[green]Saved", "[red]Error").
func (a *App) showStatus(msg string) {
	a.statusLabel.SetText(msg + " ")
	go func() {
		time.Sleep(2 * time.Second)
		a.app.QueueUpdateDraw(func() {
			a.statusLabel.SetText("")
		})
	}()
}

// confirmAndExecute shows a confirmation modal and runs onConfirm if accepted.
func (a *App) confirmAndExecute(message string, onConfirm func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
			if buttonLabel == "Yes" {
				onConfirm()
			}
		})
	a.app.SetRoot(modal, true)
}

// refreshCurrentKey re-fetches and re-displays the current key.
func (a *App) refreshCurrentKey() {
	if a.currentKey == "" {
		return
	}
	// Find the key index in the list and re-select it.
	for i, k := range a.keys {
		if k == a.currentKey {
			a.selectKey(i, a.currentKey, "", 0)
			return
		}
	}
}

// getSelectedRow returns the data-row index (0-based, excluding header),
// cell texts, and ok=true if a valid data row is selected.
func (a *App) getSelectedRow() (dataIndex int, cells []string, ok bool) {
	row, _ := a.tableView.GetSelection()
	if row < 1 { // header row or nothing selected
		return 0, nil, false
	}
	colCount := a.tableView.GetColumnCount()
	for c := 0; c < colCount; c++ {
		cell := a.tableView.GetCell(row, c)
		if cell != nil {
			cells = append(cells, cell.Text)
		}
	}
	return row - 1, cells, true
}

// --- Keyboard shortcut wiring ---

// setupEditHandlers wraps InputCapture on tableView and stringView to add
// edit shortcut keys (e/a/d/r/x). These don't conflict with normal Table
// or read-only TextView navigation because those widgets don't handle rune keys.
func (a *App) setupEditHandlers() {
	// Wrap tableView's existing InputCapture (which handles Escape).
	origTable := a.tableView.GetInputCapture()
	a.tableView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'e':
				a.dispatchEdit()
				return nil
			case 'a':
				a.dispatchAdd()
				return nil
			case 'd':
				a.dispatchDelete()
				return nil
			case 'r':
				a.refreshCurrentKey()
				return nil
			case 'x':
				a.deleteKey()
				return nil
			}
		}
		if origTable != nil {
			return origTable(event)
		}
		return event
	})

	// Enter on a table row triggers edit (same as 'e' key).
	// No-op for types without edit (set, stream) since dispatchEdit has no case for them.
	a.tableView.SetSelectedFunc(func(row, column int) {
		a.dispatchEdit()
	})

	// Wrap stringView's existing InputCapture (which handles Escape).
	origString := a.stringView.GetInputCapture()
	a.stringView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'e':
				a.editString()
				return nil
			case 'r':
				a.refreshCurrentKey()
				return nil
			case 'x':
				a.deleteKey()
				return nil
			}
		}
		if origString != nil {
			return origString(event)
		}
		return event
	})
}

// --- Dispatch ---

func (a *App) dispatchEdit() {
	switch a.currentType {
	case "list":
		a.editListItem()
	case "hash":
		a.editHashField()
	case "zset":
		a.editZSetScore()
	case "string":
		a.editString()
	}
}

func (a *App) dispatchAdd() {
	switch a.currentType {
	case "list":
		a.addListItem()
	case "set":
		a.addSetMember()
	case "hash":
		a.addHashField()
	case "zset":
		a.addZSetMember()
	case "stream":
		a.addStreamEntry()
	}
}

func (a *App) dispatchDelete() {
	switch a.currentType {
	case "list":
		a.deleteListItem()
	case "set":
		a.deleteSetMember()
	case "hash":
		a.deleteHashField()
	case "zset":
		a.deleteZSetMember()
	case "stream":
		a.deleteStreamEntry()
	}
}

// --- Per-type handlers ---

// deleteKey deletes the entire key (all types).
func (a *App) deleteKey() {
	if a.currentKey == "" {
		return
	}
	a.confirmAndExecute(
		fmt.Sprintf("Delete entire key %q?", a.currentKey),
		func() {
			key := a.currentKey
			a.connMu.Lock()
			err := a.conn.SendRaw("DEL", key)
			if err != nil {
				a.connMu.Unlock()
				a.showError("Send error: " + err.Error())
				return
			}
			_, err = a.conn.Receive(5 * time.Second)
			a.connMu.Unlock()
			if err != nil {
				a.showError("Receive error: " + err.Error())
				return
			}
			a.currentKey = ""
			a.currentType = ""
			a.switchContent("output", "Output")
			fmt.Fprintf(a.ansiWriter, "[green]Deleted key %q[white]\n", key)
			a.showStatus("[green]Deleted")
			// Refresh key list.
			pattern := a.filterInput.GetText() + "*"
			if a.filterInput.GetText() == "" {
				pattern = "*"
			}
			go a.loadKeys(pattern)
		},
	)
}

// --- String ---

func (a *App) editString() {
	if a.currentKey == "" {
		return
	}
	// Fetch current value.
	a.connMu.Lock()
	err := a.conn.SendRaw("GET", a.currentKey)
	if err != nil {
		a.connMu.Unlock()
		a.showError("Send error: " + err.Error())
		return
	}
	val, err := a.conn.Receive(5 * time.Second)
	a.connMu.Unlock()
	if err != nil {
		a.showError("Receive error: " + err.Error())
		return
	}

	current := val.StringValue()
	key := a.currentKey

	a.showTextAreaModal("Edit String: "+key, current, func(newValue string) {
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
		a.sendEditCommand("SET", key, newValue)
	})
}

// --- List ---

func (a *App) editListItem() {
	idx, cells, ok := a.getSelectedRow()
	if !ok || len(cells) < 2 {
		return
	}
	key := a.currentKey
	currentValue := cells[1]

	a.showEditModal("Edit List Item", func(form *tview.Form) {
		form.AddInputField("Value", currentValue, 50, nil, nil)
	}, func(form *tview.Form) {
		newValue := form.GetFormItemByLabel("Value").(*tview.InputField).GetText()
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
		a.sendEditCommand("LSET", key, fmt.Sprintf("%d", idx), newValue)
	})
}

func (a *App) addListItem() {
	if a.currentKey == "" {
		return
	}
	key := a.currentKey

	a.showEditModal("Add List Item", func(form *tview.Form) {
		form.AddDropDown("Position", []string{"Head (LPUSH)", "Tail (RPUSH)"}, 1, nil)
		form.AddInputField("Value", "", 50, nil, nil)
	}, func(form *tview.Form) {
		_, position := form.GetFormItemByLabel("Position").(*tview.DropDown).GetCurrentOption()
		value := form.GetFormItemByLabel("Value").(*tview.InputField).GetText()
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
		cmd := "RPUSH"
		if position == "Head (LPUSH)" {
			cmd = "LPUSH"
		}
		a.sendEditCommand(cmd, key, value)
	})
}

func (a *App) deleteListItem() {
	_, cells, ok := a.getSelectedRow()
	if !ok || len(cells) < 2 {
		return
	}
	key := a.currentKey
	value := cells[1]

	a.confirmAndExecute(
		fmt.Sprintf("Delete list item %q?", value),
		func() {
			a.sendEditCommand("LREM", key, "1", value)
		},
	)
}

// --- Set ---

func (a *App) addSetMember() {
	if a.currentKey == "" {
		return
	}
	key := a.currentKey

	a.showEditModal("Add Set Member", func(form *tview.Form) {
		form.AddInputField("Member", "", 50, nil, nil)
	}, func(form *tview.Form) {
		member := form.GetFormItemByLabel("Member").(*tview.InputField).GetText()
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
		a.sendEditCommand("SADD", key, member)
	})
}

func (a *App) deleteSetMember() {
	_, cells, ok := a.getSelectedRow()
	if !ok || len(cells) < 1 {
		return
	}
	key := a.currentKey
	member := cells[0]

	a.confirmAndExecute(
		fmt.Sprintf("Remove set member %q?", member),
		func() {
			a.sendEditCommand("SREM", key, member)
		},
	)
}

// --- Hash ---

func (a *App) editHashField() {
	_, cells, ok := a.getSelectedRow()
	if !ok || len(cells) < 2 {
		return
	}
	key := a.currentKey
	field := cells[0]
	currentValue := cells[1]

	a.showEditModal("Edit Hash Field: "+field, func(form *tview.Form) {
		form.AddInputField("Value", currentValue, 50, nil, nil)
	}, func(form *tview.Form) {
		newValue := form.GetFormItemByLabel("Value").(*tview.InputField).GetText()
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
		a.sendEditCommand("HSET", key, field, newValue)
	})
}

func (a *App) addHashField() {
	if a.currentKey == "" {
		return
	}
	key := a.currentKey

	a.showEditModal("Add Hash Field", func(form *tview.Form) {
		form.AddInputField("Field", "", 50, nil, nil)
		form.AddInputField("Value", "", 50, nil, nil)
	}, func(form *tview.Form) {
		field := form.GetFormItemByLabel("Field").(*tview.InputField).GetText()
		value := form.GetFormItemByLabel("Value").(*tview.InputField).GetText()
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
		a.sendEditCommand("HSET", key, field, value)
	})
}

func (a *App) deleteHashField() {
	_, cells, ok := a.getSelectedRow()
	if !ok || len(cells) < 2 {
		return
	}
	key := a.currentKey
	field := cells[0]

	a.confirmAndExecute(
		fmt.Sprintf("Delete hash field %q?", field),
		func() {
			a.sendEditCommand("HDEL", key, field)
		},
	)
}

// --- Sorted Set ---

func (a *App) editZSetScore() {
	_, cells, ok := a.getSelectedRow()
	if !ok || len(cells) < 2 {
		return
	}
	key := a.currentKey
	member := cells[0]
	currentScore := cells[1]

	a.showEditModal("Edit Score: "+member, func(form *tview.Form) {
		form.AddInputField("Score", currentScore, 20, nil, nil)
	}, func(form *tview.Form) {
		newScore := form.GetFormItemByLabel("Score").(*tview.InputField).GetText()
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
		a.sendEditCommand("ZADD", key, newScore, member)
	})
}

func (a *App) addZSetMember() {
	if a.currentKey == "" {
		return
	}
	key := a.currentKey

	a.showEditModal("Add Sorted Set Member", func(form *tview.Form) {
		form.AddInputField("Member", "", 50, nil, nil)
		form.AddInputField("Score", "0", 20, nil, nil)
	}, func(form *tview.Form) {
		member := form.GetFormItemByLabel("Member").(*tview.InputField).GetText()
		score := form.GetFormItemByLabel("Score").(*tview.InputField).GetText()
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
		a.sendEditCommand("ZADD", key, score, member)
	})
}

func (a *App) deleteZSetMember() {
	_, cells, ok := a.getSelectedRow()
	if !ok || len(cells) < 2 {
		return
	}
	key := a.currentKey
	member := cells[0]

	a.confirmAndExecute(
		fmt.Sprintf("Remove sorted set member %q?", member),
		func() {
			a.sendEditCommand("ZREM", key, member)
		},
	)
}

// --- Stream ---

func (a *App) addStreamEntry() {
	if a.currentKey == "" {
		return
	}
	key := a.currentKey

	a.showEditModal("Add Stream Entry", func(form *tview.Form) {
		form.AddInputField("Field", "", 50, nil, nil)
		form.AddInputField("Value", "", 50, nil, nil)
	}, func(form *tview.Form) {
		field := form.GetFormItemByLabel("Field").(*tview.InputField).GetText()
		value := form.GetFormItemByLabel("Value").(*tview.InputField).GetText()
		a.app.SetRoot(a.layout, true).SetFocus(a.activeContent)
		a.sendEditCommand("XADD", key, "*", field, value)
	})
}

func (a *App) deleteStreamEntry() {
	_, cells, ok := a.getSelectedRow()
	if !ok || len(cells) < 1 {
		return
	}
	key := a.currentKey
	id := cells[0]

	a.confirmAndExecute(
		fmt.Sprintf("Delete stream entry %q?", id),
		func() {
			a.sendEditCommand("XDEL", key, id)
		},
	)
}
