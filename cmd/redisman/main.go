package main

import (
	"fmt"
	"os"

	"github.com/cosmez/redisman-go/internal/command"
	"github.com/cosmez/redisman-go/internal/conn"
	"github.com/cosmez/redisman-go/internal/output"
	"github.com/cosmez/redisman-go/internal/tui"
	"github.com/spf13/cobra"
)

var (
	version = "dev" // set at build time via -ldflags "-X main.version=..."

	host     string
	port     string
	username string
	password string
	cmdStr  string
	tuiMode bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "redisman",
		Short:   "A cross-platform Redis client",
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			if tuiMode {
				runTUI()
				return
			}

			if cmdStr != "" {
				runOneShot()
			} else {
				runRepl()
			}
		},
	}

	rootCmd.Flags().StringVarP(&host, "host", "H", "localhost", "Redis server host")
	rootCmd.Flags().StringVarP(&port, "port", "p", "6379", "Redis server port")
	rootCmd.Flags().StringVarP(&username, "username", "u", "", "Redis ACL username")
	rootCmd.Flags().StringVar(&password, "password", "", "Redis password")
	rootCmd.Flags().StringVarP(&cmdStr, "command", "c", "", "Execute a single command and exit")
	rootCmd.Flags().BoolVar(&tuiMode, "tui", false, "Launch TUI mode")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runTUI() {
	reg, err := command.NewRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load commands: %v\n", err)
		os.Exit(1)
	}

	c, err := conn.Connect(host, port, username, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	mergeServerCommands(c, reg)

	if err := tui.Run(c, reg); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

func runOneShot() {
	c, err := conn.Connect(host, port, username, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	reg, err := command.NewRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load commands: %v\n", err)
		os.Exit(1)
	}

	parsed, err := command.Parse(cmdStr, reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	if err := c.Send(parsed); err != nil {
		fmt.Fprintf(os.Stderr, "Send error: %v\n", err)
		os.Exit(1)
	}

	val, err := c.Receive(0) // No timeout for one-shot
	if err != nil {
		fmt.Fprintf(os.Stderr, "Receive error: %v\n", err)
		os.Exit(1)
	}

	opts := output.PrintOpts{
		Color:   false, // Usually no color for one-shot scripts
		Newline: true,
	}

	if parsed.Pipe != "" {
		if err := output.PipeRedisValue(os.Stdout, val, parsed.Pipe); err != nil {
			fmt.Fprintf(os.Stderr, "Pipe error: %v\n", err)
			os.Exit(1)
		}
	} else {
		output.PrintRedisValue(os.Stdout, val, opts)
	}
}
