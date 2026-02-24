package tui

import (
	"testing"

	"github.com/cosmez/redisman-go/internal/command"
)

// TestAppScaffold verifies that the TUI application can be constructed
// without panicking. It does NOT call app.Run() since that requires
// a real terminal.
func TestAppScaffold(t *testing.T) {
	reg, err := command.NewRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Build the app with a nil connection (we won't execute commands).
	app := newApp(nil, reg)
	if app == nil {
		t.Fatal("newApp returned nil")
	}
	if app.keyList == nil {
		t.Error("keyList is nil")
	}
	if app.outputView == nil {
		t.Error("outputView is nil")
	}
	if app.contentPages == nil {
		t.Error("contentPages is nil")
	}
	if app.cmdInput == nil {
		t.Error("cmdInput is nil")
	}
	if app.filterInput == nil {
		t.Error("filterInput is nil")
	}
	if app.ansiWriter == nil {
		t.Error("ansiWriter is nil")
	}
	if app.tableView == nil {
		t.Error("tableView is nil")
	}
	if app.stringView == nil {
		t.Error("stringView is nil")
	}
	if app.activeContent == nil {
		t.Error("activeContent is nil")
	}
	if app.actionBar == nil {
		t.Error("actionBar is nil")
	}
	if len(app.focusOrder) != 4 {
		t.Errorf("Expected 4 focus targets, got %d", len(app.focusOrder))
	}
}
