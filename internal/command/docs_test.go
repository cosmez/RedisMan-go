package command

import "testing"

func TestRegistryGet_UsesIndex(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}

	// Known command should be found
	doc := reg.Get("GET")
	if doc == nil {
		t.Fatal("GET should exist in registry")
	}
	if doc.Command != "GET" {
		t.Errorf("Expected command GET, got %s", doc.Command)
	}

	// Case insensitive
	doc = reg.Get("get")
	if doc == nil {
		t.Fatal("get (lowercase) should resolve to GET")
	}

	// Unknown command returns nil
	if reg.Get("NONEXISTENT_CMD_XYZ") != nil {
		t.Error("Unknown command should return nil")
	}
}

func TestMergeServerCommands_NewCommandAdded(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}

	if reg.Get("NEWCMD") != nil {
		t.Fatal("NEWCMD should not exist before merge")
	}

	cmds := []ServerCommand{
		{Name: "NEWCMD", Arity: -2, ACLCats: []string{"@string", "@read"}},
	}
	reg.MergeServerCommands(cmds)

	doc := reg.Get("NEWCMD")
	if doc == nil {
		t.Fatal("NEWCMD should exist after merge")
	}
	if doc.Arguments != "arg1 [arg ...]" {
		t.Errorf("Expected arity hint %q, got %q", "arg1 [arg ...]", doc.Arguments)
	}
	if doc.Group != "string" {
		t.Errorf("Expected group %q, got %q", "string", doc.Group)
	}
}

func TestMergeServerCommands_ExistingCommandPreserved(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}

	original := reg.Get("GET")
	if original == nil {
		t.Fatal("GET should exist in built-in docs")
	}
	origSummary := original.Summary
	origArgs := original.Arguments

	cmds := []ServerCommand{
		{Name: "GET", Arity: 2, ACLCats: []string{"@string", "@read", "@fast"}},
	}
	reg.MergeServerCommands(cmds)

	doc := reg.Get("GET")
	if doc.Summary != origSummary {
		t.Errorf("Expected original summary preserved, got %q", doc.Summary)
	}
	if doc.Arguments != origArgs {
		t.Errorf("Expected original arguments preserved, got %q", doc.Arguments)
	}
}

func TestMergeServerCommands_SubcommandsMerged(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}

	cmds := []ServerCommand{
		{
			Name:  "NEWPARENT",
			Arity: -1,
			Subcommands: []ServerCommand{
				{Name: "NEWPARENT CHILD", Arity: 3},
			},
		},
	}
	reg.MergeServerCommands(cmds)

	if reg.Get("NEWPARENT") == nil {
		t.Error("NEWPARENT should exist after merge")
	}
	if reg.Get("NEWPARENT CHILD") == nil {
		t.Error("NEWPARENT CHILD should exist after merge")
	}

	child := reg.Get("NEWPARENT CHILD")
	if child.Arguments != "arg1 arg2" {
		t.Errorf("Expected arity hint %q, got %q", "arg1 arg2", child.Arguments)
	}
}

func TestMergeServerCommands_AppearsInAutocomplete(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}

	cmds := []ServerCommand{
		{Name: "ZZZTESTCMD", Arity: 2},
	}
	reg.MergeServerCommands(cmds)

	matches := reg.GetCommands("ZZZTEST")
	found := false
	for _, m := range matches {
		if m == "ZZZTESTCMD" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Merged command should appear in GetCommands autocomplete")
	}
}

func TestArityHint(t *testing.T) {
	tests := []struct {
		arity int64
		want  string
	}{
		{0, ""},
		{1, ""},
		{2, "arg1"},
		{3, "arg1 arg2"},
		{5, "arg1 arg2 arg3 arg4"},
		{-1, "[arg ...]"},
		{-2, "arg1 [arg ...]"},
		{-3, "arg1 arg2 [arg ...]"},
	}
	for _, tc := range tests {
		got := arityHint(tc.arity)
		if got != tc.want {
			t.Errorf("arityHint(%d) = %q, want %q", tc.arity, got, tc.want)
		}
	}
}

func TestPrimaryACLGroup(t *testing.T) {
	tests := []struct {
		cats []string
		want string
	}{
		{[]string{"@read", "@string", "@fast"}, "string"},
		{[]string{"@write", "@hash", "@slow"}, "hash"},
		{[]string{"@read", "@fast"}, ""},
		{[]string{"@admin", "@slow", "@dangerous"}, "admin"},
		{[]string{"@connection"}, "connection"},
		{[]string{"@pubsub", "@slow"}, "pubsub"},
		{nil, ""},
		{[]string{}, ""},
	}
	for _, tc := range tests {
		got := primaryACLGroup(tc.cats)
		if got != tc.want {
			t.Errorf("primaryACLGroup(%v) = %q, want %q", tc.cats, got, tc.want)
		}
	}
}
