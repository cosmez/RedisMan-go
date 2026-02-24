package output

import (
	"fmt"
	"io"
	"iter"
	"os"
	"os/exec"
	"strings"

	"github.com/cosmez/redisman-go/internal/resp"
	"github.com/cosmez/redisman-go/internal/serializer"
	"github.com/fatih/color"
)

// PrintOpts configures how a RedisValue is printed.
type PrintOpts struct {
	Color      bool
	Serializer serializer.Serializer
	Padding    string
	TypeHint   string // e.g., "hash", "stream"
	Newline    bool
}

var (
	colorString  = color.New(color.FgHiBlue)
	colorInteger = color.New(color.FgHiGreen)
	colorError   = color.New(color.FgRed, color.Bold)
	colorNull    = color.New(color.FgHiBlack)
	colorArray   = color.New(color.FgHiYellow)
	colorIndex   = color.New(color.FgHiBlack)
)

// digitWidth returns the number of digits in n.
func digitWidth(n int) int {
	if n <= 0 {
		return 1
	}
	w := 0
	for n > 0 {
		w++
		n /= 10
	}
	return w
}

// printIndex writes an index string (e.g. " 1) "), optionally colored.
func printIndex(w io.Writer, idx string, useColor bool) {
	if useColor {
		colorIndex.Fprint(w, idx)
	} else {
		fmt.Fprint(w, idx)
	}
}

// PrintRedisValues prints an iterator of RedisValues with pagination.
//
// C# equivalent:
// public static async Task PrintRedisValues(IEnumerable<RedisValue> values, int warningAt = 100, string type = "")
func PrintRedisValues(w io.Writer, r io.Reader, values iter.Seq[resp.RedisValue], opts PrintOpts, warningAt int) {
	i := 0

	for value := range values {
		i++

		switch opts.TypeHint {
		case "stream":
			if array, ok := value.(resp.RedisArray); ok && len(array.Values) >= 2 {
				// Print ID
				idOpts := opts
				idOpts.TypeHint = ""
				idOpts.Newline = false
				PrintRedisValue(w, array.Values[0], idOpts)

				// Print fields
				fieldsOpts := opts
				fieldsOpts.Padding = " "
				fieldsOpts.Newline = false
				PrintRedisValue(w, array.Values[1], fieldsOpts)
			}
		case "hash":
			// SafeHash yields pairs as RedisArray
			hashOpts := opts
			hashOpts.Newline = false
			PrintRedisValue(w, value, hashOpts)
		default:
			printIndex(w, fmt.Sprintf("%d) ", i), opts.Color)
			PrintRedisValue(w, value, opts)
		}

		if warningAt > 0 && i%warningAt == 0 {
			fmt.Fprint(w, "Continue Listing? ")
			colorArray.Fprint(w, "(Y/N) ")

			// Read unbuffered to avoid stealing from REPL
			var line []byte
			buf := make([]byte, 1)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					line = append(line, buf[0])
					if buf[0] == '\n' {
						break
					}
				}
				if err != nil {
					break
				}
			}

			ans := strings.TrimSpace(string(line))
			if len(ans) == 0 || (ans[0] != 'Y' && ans[0] != 'y') {
				break
			}
		}
	}
}

// PipeRedisValue pipes a RedisValue to a shell command.
//
// C# equivalent:
// public static void PipeRedisValue(ParsedCommand command, RedisValue value)
func PipeRedisValue(w io.Writer, v resp.RedisValue, shellCmd string) error {
	if shellCmd == "" {
		return nil
	}

	args := strings.Fields(shellCmd)
	if len(args) == 0 {
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = w
	cmd.Stderr = w

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	writeRawValue(stdin, v)
	stdin.Close()

	return cmd.Wait()
}

func writeRawValue(w io.Writer, v resp.RedisValue) {
	if v == nil {
		return
	}
	if array, ok := v.(resp.RedisArray); ok {
		for _, element := range array.Values {
			writeRawValue(w, element)
		}
	} else {
		fmt.Fprintln(w, v.StringValue())
	}
}

// ExportAsync writes a RedisValue or an iterator of RedisValues to a file.
//
// C# equivalent:
// public static async Task ExportAsync(Connection connection, string filename, ParsedCommand command)
func ExportAsync(filename string, v resp.RedisValue, values iter.Seq[resp.RedisValue], typeHint string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if v != nil {
		writeValueAsync(f, v, typeHint)
	}

	if values != nil {
		for value := range values {
			writeValueAsync(f, value, typeHint)
		}
	}

	return nil
}

func writeValueAsync(w io.Writer, v resp.RedisValue, typeHint string) {
	if v == nil {
		return
	}

	if array, ok := v.(resp.RedisArray); ok {
		for i := 0; i < len(array.Values); {
			writeValueAsync(w, array.Values[i], typeHint)
			i++
			if typeHint == "hash" && i < len(array.Values) {
				fmt.Fprint(w, "=")
				writeValueAsync(w, array.Values[i], typeHint)
				i++
			}
			fmt.Fprintln(w)
		}
	} else {
		var outputText string
		switch v.Type() {
		case resp.TypeString, resp.TypeBulkString:
			if v.Type() == resp.TypeBulkString && v.(resp.RedisBulkString).Length == -1 {
				outputText = "(null)"
			} else {
				// The user requested "Print without quotes" for PrintRedisValue,
				// but C# WriteValueAsync used quotes. We'll omit quotes here too for consistency.
				outputText = v.StringValue()
			}
		case resp.TypeNull:
			outputText = "(null)"
		case resp.TypeInteger, resp.TypeError:
			outputText = v.StringValue()
		}
		fmt.Fprint(w, outputText)
	}
}

// PrintRedisValue prints a RedisValue to the given writer with optional ANSI colors.
//
// C# equivalent:
// public static async Task PrintRedisValue(RedisValue value, string padding = "", bool color = true, string type = "", bool newline = true, ISerializer serializer = null)
func PrintRedisValue(w io.Writer, v resp.RedisValue, opts PrintOpts) {
	if v == nil {
		return
	}

	getDeserialized := func(val string) string {
		if opts.Serializer != nil {
			bytes := []byte(val)
			deserialized, err := opts.Serializer.Deserialize(bytes)
			if err == nil {
				return string(deserialized)
			}
		}
		return val
	}

	switch val := v.(type) {
	case resp.RedisArray:
		// Empty array
		if len(val.Values) == 0 {
			if opts.Color {
				colorNull.Fprint(w, "(empty array)")
			} else {
				fmt.Fprint(w, "(empty array)")
			}
			if opts.Newline {
				fmt.Fprintln(w)
			}
			return
		}

		// Hash/stream formatting (unchanged behavior)
		if opts.TypeHint == "hash" || opts.TypeHint == "stream" {
			if opts.Padding != "" {
				fmt.Fprintln(w)
			}
			for i := 0; i < len(val.Values); {
				switch opts.TypeHint {
				case "hash":
					fmt.Fprintf(w, "%s#", opts.Padding)
				case "stream":
					fmt.Fprintf(w, "%s@", opts.Padding)
				}

				childOpts := opts
				childOpts.Padding = opts.Padding + "  "
				childOpts.Newline = false
				childOpts.TypeHint = ""
				PrintRedisValue(w, val.Values[i], childOpts)
				i++

				if i < len(val.Values) {
					fmt.Fprint(w, "=")
					PrintRedisValue(w, val.Values[i], childOpts)
					i++
				}
				fmt.Fprintln(w)
			}
			return
		}

		// Normal array: right-aligned indices, inline nested arrays
		digits := digitWidth(len(val.Values))
		idxWidth := digits + 2 // e.g. " 1) " for digits=1

		visualIdx := 1
		for i := 0; i < len(val.Values); i++ {
			idxStr := fmt.Sprintf("%*d) ", digits, visualIdx)

			if i == 0 && opts.Padding != "" {
				// First child of nested array: print inline (no padding, parent already positioned us)
				printIndex(w, idxStr, opts.Color)
			} else if i > 0 {
				fmt.Fprint(w, opts.Padding)
				printIndex(w, idxStr, opts.Color)
			} else {
				// Top-level first element (no padding)
				printIndex(w, idxStr, opts.Color)
			}

			childOpts := opts
			childOpts.Padding = opts.Padding + strings.Repeat(" ", idxWidth)
			childOpts.Newline = false
			childOpts.TypeHint = ""
			PrintRedisValue(w, val.Values[i], childOpts)

			// Non-empty child arrays already end with \n from their last element
			if childArray, ok := val.Values[i].(resp.RedisArray); ok && len(childArray.Values) > 0 {
				// skip newline — child already printed one
			} else {
				fmt.Fprintln(w)
			}

			visualIdx++
		}

	default:
		var outputText string
		var c *color.Color

		switch val.Type() {
		case resp.TypeString:
			outputText = getDeserialized(val.StringValue())
			c = colorString
		case resp.TypeNull:
			outputText = "(nil)"
			c = colorNull
		case resp.TypeBulkString:
			if val.(resp.RedisBulkString).Length == -1 {
				outputText = "(nil)"
				c = colorNull
			} else {
				outputText = fmt.Sprintf("\"%s\"", getDeserialized(val.StringValue()))
				c = colorString
			}
		case resp.TypeInteger:
			outputText = fmt.Sprintf("(integer) %s", val.StringValue())
			c = colorInteger
		case resp.TypeError:
			outputText = val.StringValue()
			c = colorError
		}

		if opts.Color && c != nil {
			c.Fprint(w, outputText)
		} else {
			fmt.Fprint(w, outputText)
		}

		if opts.Newline {
			fmt.Fprintln(w)
		}
	}
}
