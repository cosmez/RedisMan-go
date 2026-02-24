package conn

import (
	"bufio"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/cosmez/redisman-go/internal/command"
	"github.com/cosmez/redisman-go/internal/resp"
)

// setupMockConnection creates a Connection using net.Pipe for testing without a real Redis server.
func setupMockConnection() (*Connection, net.Conn) {
	clientConn, serverConn := net.Pipe()
	c := &Connection{
		Host:   "localhost",
		Port:   "6379",
		conn:   clientConn,
		reader: bufio.NewReader(clientConn),
	}
	return c, serverConn
}

func TestSendReceive(t *testing.T) {
	c, serverConn := setupMockConnection()
	defer c.Close()
	defer serverConn.Close()

	// Test Send
	go func() {
		cmd, _ := command.Parse("PING", nil)
		err := c.Send(cmd)
		if err != nil {
			t.Errorf("Send failed: %v", err)
		}
	}()

	buf := make([]byte, 1024)
	n, err := serverConn.Read(buf)
	if err != nil {
		t.Fatalf("Server read failed: %v", err)
	}
	expectedReq := "*1\r\n$4\r\nPING\r\n"
	if string(buf[:n]) != expectedReq {
		t.Errorf("Expected %q, got %q", expectedReq, string(buf[:n]))
	}

	// Test Receive
	go func() {
		_, err := serverConn.Write([]byte("+PONG\r\n"))
		if err != nil {
			t.Errorf("Server write failed: %v", err)
		}
	}()

	response, err := c.Receive(1 * time.Second)
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	strResp, ok := response.(resp.RedisString)
	if !ok || strResp.Value != "PONG" {
		t.Errorf("Expected PONG, got %v", response)
	}
}

func TestGetServerInfo(t *testing.T) {
	c, serverConn := setupMockConnection()
	defer c.Close()
	defer serverConn.Close()

	go func() {
		// Read INFO command
		buf := make([]byte, 1024)
		serverConn.Read(buf)

		// Send INFO response
		infoResp := "$43\r\n# Server\r\nredis_version:7.0.0\r\nos:Linux\r\n\r\n\r\n"
		serverConn.Write([]byte(infoResp))
	}()

	err := c.getServerInfo()
	if err != nil {
		t.Fatalf("getServerInfo failed: %v", err)
	}

	if c.ServerInfo["redis_version"] != "7.0.0" {
		t.Errorf("Expected redis_version 7.0.0, got %v", c.ServerInfo["redis_version"])
	}
	if c.ServerInfo["os"] != "Linux" {
		t.Errorf("Expected os Linux, got %v", c.ServerInfo["os"])
	}
}

func TestSafeKeys(t *testing.T) {
	c, serverConn := setupMockConnection()
	defer c.Close()
	defer serverConn.Close()

	go func() {
		// Read SCAN 0
		buf := make([]byte, 1024)
		serverConn.Read(buf)

		// Send first batch (cursor "10", keys "key1", "key2")
		resp1 := "*2\r\n$2\r\n10\r\n*2\r\n$4\r\nkey1\r\n$4\r\nkey2\r\n"
		serverConn.Write([]byte(resp1))

		// Read SCAN 10
		serverConn.Read(buf)

		// Send second batch (cursor "0", key "key3")
		resp2 := "*2\r\n$1\r\n0\r\n*1\r\n$4\r\nkey3\r\n"
		serverConn.Write([]byte(resp2))
	}()

	var keys []string
	for val := range c.SafeKeys("*") {
		if errResp, ok := val.(resp.RedisError); ok {
			t.Fatalf("Iterator returned error: %v", errResp.Value)
		}
		keys = append(keys, val.StringValue())
	}

	expected := []string{"key1", "key2", "key3"}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Expected keys %v, got %v", expected, keys)
	}
}

func TestSendRaw(t *testing.T) {
	c, serverConn := setupMockConnection()
	defer c.Close()
	defer serverConn.Close()

	go func() {
		err := c.SendRaw("SET", "my key", "hello world\nline2")
		if err != nil {
			t.Errorf("SendRaw failed: %v", err)
		}
	}()

	buf := make([]byte, 1024)
	n, err := serverConn.Read(buf)
	if err != nil {
		t.Fatalf("Server read failed: %v", err)
	}

	expected := "*3\r\n$3\r\nSET\r\n$6\r\nmy key\r\n$17\r\nhello world\nline2\r\n"
	if string(buf[:n]) != expected {
		t.Errorf("Expected %q, got %q", expected, string(buf[:n]))
	}
}

func TestGetKeyValue_String(t *testing.T) {
	c, serverConn := setupMockConnection()
	defer c.Close()
	defer serverConn.Close()

	go func() {
		// Read TYPE
		buf := make([]byte, 1024)
		serverConn.Read(buf)
		// Send TYPE response
		serverConn.Write([]byte("+string\r\n"))

		// Read GET
		serverConn.Read(buf)
		// Send GET response
		serverConn.Write([]byte("$5\r\nvalue\r\n"))
	}()

	typeName, single, collection, err := c.GetKeyValue("mykey")
	if err != nil {
		t.Fatalf("GetKeyValue failed: %v", err)
	}

	if typeName != "string" {
		t.Errorf("Expected type string, got %v", typeName)
	}
	if single.StringValue() != "value" {
		t.Errorf("Expected single value 'value', got %v", single.StringValue())
	}
	if collection != nil {
		t.Error("Expected nil collection for string type")
	}
}
