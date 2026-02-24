# RedisMan

A cross-platform Redis client with REPL, TUI, and one-shot modes. Built in Go.

## Features

- **Three modes**: interactive REPL (default), full-screen TUI (`--tui`), and one-shot (`-c`)
- **Tab completion** on command names with inline documentation hints
- **Built-in command docs** from an embedded registry, merged with live server commands on connect
- **Codec modifiers** (`#:gzip`, `#:base64`, `#:snappy`) — decode values on GET, encode on SET
- **Pipe to shell** (`GET key | jq .`) — stream command output to any subprocess
- **SAFEKEYS** — paginated key listing via SCAN (safe for production, unlike `KEYS *`)
- **VIEW** — type-aware key inspector dispatches to the correct read command per type
- **EXPORT** — write command output to a file without ANSI codes
- **Dangerous command guard** — prompts for Y/N confirmation on FLUSHDB, DEL, KEYS, etc.
- **TUI CRUD editing** — edit key values inline in the TUI with type-aware forms

## Installation

### Binary download

Download a prebuilt binary from [Releases](https://github.com/cosmez/redisman-go/releases).

### From source

```sh
go install github.com/cosmez/redisman-go/cmd/redisman@latest
```

Requires Go 1.25+.

## Usage

### REPL mode (default)

```sh
redisman --host localhost --port 6379
```

### TUI mode

```sh
redisman --tui
```

### One-shot mode

```sh
redisman -c "GET mykey"
redisman -c "SET mykey myvalue"
```

### CLI flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--host` | `-H` | `localhost` | Redis server host |
| `--port` | `-p` | `6379` | Redis server port |
| `--username` | `-u` | | Redis ACL username |
| `--password` | | | Redis password |
| `--command` | `-c` | | Execute a single command and exit |
| `--tui` | | `false` | Launch TUI mode |
| `--version` | `-v` | | Print version and exit |

### REPL built-in commands

| Command | Description |
|---------|-------------|
| `CONNECT host port` | Reconnect to a different server |
| `SAFEKEYS [pattern]` | Paginated key listing via SCAN |
| `VIEW key` | Display key content (type-aware) |
| `EXPORT file cmd...` | Write command output to a file |
| `HELP [command]` | Show command documentation |
| `CLEAR` | Clear screen |
| `EXIT` | Quit |

### Codec modifiers

Append `#:codec` to decode a value on read or encode on write:

```
GET session:abc123 #:gzip
GET config:blob #:base64
SET mykey myvalue #:snappy
```

### Pipe to shell

Pipe Redis output to any command:

```
GET user:1 | jq .
LRANGE queue:jobs 0 -1 | sort
SMEMBERS tags:blog | wc -l
```

## Development

```sh
go build ./...                                        # build all packages
go test ./...                                         # run all tests
go run ./cmd/redisman -- --host localhost --port 6379  # run REPL
go run ./cmd/redisman -- --tui                         # run TUI
```

### Test data

Seed a local Redis instance with sample data for testing:

```sh
./scripts/seed-redis.sh [host] [port]
```

## License

BSD 3-Clause. See [LICENSE](LICENSE).
