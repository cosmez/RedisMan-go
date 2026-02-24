package command

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed simple_commands.json
var commandsJSON []byte

// Registry holds the documentation for all known Redis commands.
//
// C#:
//
//	public static class Documentation {
//	    private static List<CommandDoc> _commands;
//	    private static HashSet<string> _dangerousCommands;
//	}
type Registry struct {
	docs      []CommandDoc
	index     map[string]int // command name → index in docs slice
	dangerous map[string]bool
}

// NewRegistry initializes and returns a new command documentation registry.
func NewRegistry() (*Registry, error) {
	var docs []CommandDoc
	if err := json.Unmarshal(commandsJSON, &docs); err != nil {
		return nil, fmt.Errorf("failed to parse embedded commands JSON: %w", err)
	}

	// Append hard-coded application commands
	appCommands := []CommandDoc{
		{Command: "EXIT", Summary: "Exit the application", Group: "application"},
		{Command: "CONNECT", Summary: "Connect to a Redis server", Arguments: "[host] [port] [user] [pass]", Group: "application"},
		{Command: "HELP", Summary: "Show help for a command", Arguments: "[command]", Group: "application"},
		{Command: "CLEAR", Summary: "Clear the screen", Group: "application"},
		{Command: "SAFEKEYS", Summary: "Safely iterate over keys using SCAN", Arguments: "[pattern]", Group: "application"},
		{Command: "VIEW", Summary: "View the contents of a key", Arguments: "key", Group: "application"},
		{Command: "EXPORT", Summary: "Export the result of a command to a file", Arguments: "file command [args...]", Group: "application"},
	}
	docs = append(docs, appCommands...)

	// Initialize dangerous commands map for O(1) lookup
	dangerousList := []string{
		"FLUSHDB", "FLUSHALL", "KEYS", "PEXPIRE", "DEL", "CONFIG",
		"SHUTDOWN", "BGREWRITEAOF", "BGSAVE", "SAVE", "SPOP", "SREM",
		"RENAME", "DEBUG",
	}
	dangerousMap := make(map[string]bool, len(dangerousList))
	for _, cmd := range dangerousList {
		dangerousMap[cmd] = true
	}

	idx := make(map[string]int, len(docs))
	for i, doc := range docs {
		idx[doc.Command] = i
	}

	return &Registry{
		docs:      docs,
		index:     idx,
		dangerous: dangerousMap,
	}, nil
}

// Get returns the documentation for a specific command, or nil if not found.
// It handles compound commands like "CLIENT INFO".
func (r *Registry) Get(cmd string) *CommandDoc {
	cmd = strings.ToUpper(cmd)
	if i, ok := r.index[cmd]; ok {
		return &r.docs[i]
	}
	return nil
}

// GetCommands returns a list of command names that start with the given prefix.
// Used for tab completion.
func (r *Registry) GetCommands(prefix string) []string {
	prefix = strings.ToUpper(prefix)
	var matches []string
	for _, doc := range r.docs {
		if strings.HasPrefix(doc.Command, prefix) {
			matches = append(matches, doc.Command)
		}
	}
	return matches
}

// Search returns a list of CommandDocs whose names start with the given prefix.
func (r *Registry) Search(prefix string) []CommandDoc {
	prefix = strings.ToUpper(prefix)
	var matches []CommandDoc
	for _, doc := range r.docs {
		if strings.HasPrefix(doc.Command, prefix) {
			matches = append(matches, doc)
		}
	}
	return matches
}

// IsDangerous returns true if the command is considered dangerous and requires confirmation.
func (r *Registry) IsDangerous(cmd string) bool {
	return r.dangerous[strings.ToUpper(cmd)]
}

// MergeServerCommands incorporates commands discovered from the live Redis
// server into the registry. Commands that already exist keep their built-in
// docs. New commands get a minimal entry for autocomplete.
func (r *Registry) MergeServerCommands(cmds []ServerCommand) {
	for _, sc := range cmds {
		r.mergeOne(sc)
		for _, sub := range sc.Subcommands {
			r.mergeOne(sub)
		}
	}
}

func (r *Registry) mergeOne(sc ServerCommand) {
	if _, exists := r.index[sc.Name]; exists {
		return // keep built-in docs
	}
	doc := CommandDoc{
		Command:   sc.Name,
		Arguments: arityHint(sc.Arity),
		Group:     primaryACLGroup(sc.ACLCats),
	}
	r.index[sc.Name] = len(r.docs)
	r.docs = append(r.docs, doc)
}

// arityHint generates a basic argument hint string from the COMMAND arity.
// Arity includes the command name itself, so actual args = |arity| - 1.
func arityHint(arity int64) string {
	if arity == 0 || arity == 1 {
		return ""
	}
	if arity > 1 {
		n := int(arity) - 1
		parts := make([]string, n)
		for i := range parts {
			parts[i] = fmt.Sprintf("arg%d", i+1)
		}
		return strings.Join(parts, " ")
	}
	// Negative arity: at least |arity| - 1 args
	minArgs := int(-arity) - 1
	if minArgs == 0 {
		return "[arg ...]"
	}
	parts := make([]string, minArgs)
	for i := range parts {
		parts[i] = fmt.Sprintf("arg%d", i+1)
	}
	return strings.Join(parts, " ") + " [arg ...]"
}

// primaryACLGroup picks a human-friendly group name from ACL categories.
// It skips meta-categories and returns the first domain category.
func primaryACLGroup(cats []string) string {
	skip := map[string]bool{
		"@read": true, "@write": true, "@fast": true, "@slow": true,
		"@admin": true, "@dangerous": true, "@keyspace": true,
	}
	for _, cat := range cats {
		if !skip[cat] && strings.HasPrefix(cat, "@") {
			return cat[1:]
		}
	}
	// Fallback to meta categories that make reasonable group names
	for _, cat := range cats {
		if cat == "@connection" || cat == "@pubsub" || cat == "@admin" {
			return cat[1:]
		}
	}
	return ""
}
