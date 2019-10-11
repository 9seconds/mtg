package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func Fatal(arg interface{}) {
	if value, ok := arg.(error); ok {
		arg = fmt.Errorf("fatal error: %+v", value)
	}

	PrintStderr(arg)
	os.Exit(1)
}

func PrintStderr(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
}

func PrintStdout(args ...interface{}) {
	fmt.Println(args...)
}

func PrintJSONStderr(data interface{}) {
	printJSON(os.Stderr, data)
}

func PrintJSONStdout(data interface{}) {
	printJSON(os.Stdout, data)
}

func printJSON(writer io.Writer, data interface{}) {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		panic(err)
	}
}
