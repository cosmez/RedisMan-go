package conn

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/cosmez/redisman-go/internal/command"
	"github.com/cosmez/redisman-go/internal/resp"
)

// Connection represents a TCP connection to a Redis server.
//
// C#:
//
//	public class Connection : IDisposable {
//	    private TcpClient _client;
//	    private StreamReader _reader;
//	    public Dictionary<string, string> ServerInfo { get; private set; }
//	}
type Connection struct {
	Host       string
	Port       string
	reader     *bufio.Reader
	conn       net.Conn
	ServerInfo map[string]string
}

// Connect establishes a TCP connection to Redis and performs authentication if required.
//
// C#:
// public Connection(string host, int port, string password = null)
//
// Go:
// We return (*Connection, error) instead of throwing exceptions in a constructor.
func Connect(host, port, user, pass string) (*Connection, error) {
	address := net.JoinHostPort(host, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	c := &Connection{
		Host:   host,
		Port:   port,
		conn:   conn,
		reader: bufio.NewReader(conn),
	}

	// Handle Authentication
	if pass != "" {
		var authCmd *command.ParsedCommand
		if user == "" {
			// Legacy AUTH
			authCmd, _ = command.Parse(fmt.Sprintf("AUTH %s", pass), nil)
		} else {
			// ACL AUTH (Redis 6+)
			authCmd, _ = command.Parse(fmt.Sprintf("AUTH %s %s", user, pass), nil)
		}

		if err := c.Send(authCmd); err != nil {
			c.Close()
			return nil, fmt.Errorf("failed to send AUTH command: %w", err)
		}

		response, err := c.Receive(5 * time.Second)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("failed to receive AUTH response: %w", err)
		}

		if errResp, ok := response.(resp.RedisError); ok {
			c.Close()
			return nil, fmt.Errorf("authentication failed: %s", errResp.Value)
		}

		if strResp, ok := response.(resp.RedisString); !ok || strResp.Value != "OK" {
			c.Close()
			return nil, fmt.Errorf("unexpected AUTH response: %v", response)
		}
	}

	// Fetch Server Info
	if err := c.getServerInfo(); err != nil {
		// We don't fail the connection if INFO fails, just log/ignore it
		// as some restricted environments might block the INFO command.
		c.ServerInfo = map[string]string{"error": err.Error()}
	}

	return c, nil
}

// Send writes a parsed command to the Redis server.
func (c *Connection) Send(cmd *command.ParsedCommand) error {
	_, err := c.conn.Write(cmd.CommandBytes)
	return err
}

// SendRaw writes a RESP command directly from raw string arguments,
// bypassing command.Parse(). This avoids quoting issues for values
// that contain spaces, quotes, newlines, or binary data — RESP bulk
// strings are length-prefixed, so any byte sequence is safe.
//
// C#: No direct equivalent — the C# version always routed through the parser.
func (c *Connection) SendRaw(args ...string) error {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("*%d\r\n", len(args)))
	for _, arg := range args {
		b := []byte(arg)
		buf.WriteString(fmt.Sprintf("$%d\r\n", len(b)))
		buf.Write(b)
		buf.WriteString("\r\n")
	}
	_, err := c.conn.Write(buf.Bytes())
	return err
}

// Receive reads a single RESP value from the server, optionally with a timeout.
func (c *Connection) Receive(timeout time.Duration) (resp.RedisValue, error) {
	if timeout > 0 {
		if err := c.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return nil, fmt.Errorf("failed to set read deadline: %w", err)
		}
		// Reset deadline after read
		defer c.conn.SetReadDeadline(time.Time{})
	}

	return resp.ParseValue(c.reader)
}

// Close terminates the TCP connection.
func (c *Connection) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
