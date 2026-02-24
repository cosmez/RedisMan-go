package command

// ParsedCommand represents a fully parsed and encoded Redis command.
//
// C#:
// public class ParsedCommand {
//     public string Text { get; set; }
//     public string Name { get; set; }
//     public string[] Args { get; set; }
//     public byte[] CommandBytes { get; set; }
//     public string Modifier { get; set; }
//     public string Pipe { get; set; }
//     public CommandDoc Doc { get; set; }
// }
type ParsedCommand struct {
	Text         string      // original input text
	Name         string      // command name, empty if none
	Args         []string    // command arguments, empty if none
	CommandBytes []byte      // RESP-encoded command ready to send to Redis
	Modifier     string      // codec name e.g. "gzip", empty if none
	Pipe         string      // shell command after "|", empty if none
	Doc          *CommandDoc // documentation, nil if not found
}

// CommandDoc represents the documentation for a single Redis command.
//
// C#:
// public class CommandDoc {
//     public string Command { get; set; }
//     public string Summary { get; set; }
//     public string Arguments { get; set; }
//     public string Since { get; set; }
//     public string Group { get; set; }
// }
type CommandDoc struct {
	Command   string `json:"command"`
	Summary   string `json:"summary"`
	Arguments string `json:"arguments"`
	Since     string `json:"since"`
	Group     string `json:"group"`
}

// ServerCommand represents a command discovered from the Redis COMMAND response.
// Defined here (not in conn) so conn can produce these and command can consume
// them without a circular import.
type ServerCommand struct {
	Name        string          // e.g. "CONFIG SET" (uppercased, pipe replaced with space)
	Arity       int64           // positive = exact arg count, negative = minimum
	ACLCats     []string        // e.g. ["@string", "@read", "@fast"]
	Subcommands []ServerCommand // recursive subcommands
}
