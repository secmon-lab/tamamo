package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/secmon-lab/tamamo/pkg/cli"
)

const maxErrorMessageLen = 200

func main() {
	if err := cli.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "Error: %s\n", rootMessage(err))
		for curr := errors.Unwrap(err); curr != nil; curr = errors.Unwrap(curr) {
			fmt.Fprintf(os.Stderr, "  caused by: %s\n", rootMessage(curr))
		}
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}
}

// rootMessage returns the error's own message without the wrapped cause chain,
// truncated and sanitized for terminal display.
func rootMessage(err error) string {
	msg := err.Error()
	inner := errors.Unwrap(err)
	if inner != nil {
		innerMsg := inner.Error()
		suffix := ": " + innerMsg
		if len(msg) > len(suffix) && msg[len(msg)-len(suffix):] == suffix {
			return msg[:len(msg)-len(suffix)]
		}
	}

	return sanitizeMessage(msg)
}

// sanitizeMessage cleans up an error message for terminal display.
func sanitizeMessage(msg string) string {
	// Strip HTML tags
	if strings.Contains(msg, "<") {
		var b strings.Builder
		inTag := false
		for _, r := range msg {
			if r == '<' {
				inTag = true
				continue
			}
			if r == '>' {
				inTag = false
				continue
			}
			if !inTag {
				b.WriteRune(r)
			}
		}
		msg = b.String()
	}

	// Collapse whitespace
	fields := strings.Fields(msg)
	msg = strings.Join(fields, " ")

	// Truncate
	if len(msg) > maxErrorMessageLen {
		msg = msg[:maxErrorMessageLen] + "..."
	}

	return msg
}
