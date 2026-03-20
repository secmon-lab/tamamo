package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/cli"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

const maxErrorMessageLen = 200

func main() {
	if err := cli.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "")

		if goerr.HasTag(err, errutil.TagValidation) {
			printValidationError(err)
		} else {
			printDetailedError(err)
		}

		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}
}

// printValidationError prints a concise validation error with truncated messages.
func printValidationError(err error) {
	fmt.Fprintf(os.Stderr, "Validation Error: %s\n", rootMessage(err))
	for curr := errors.Unwrap(err); curr != nil; curr = errors.Unwrap(curr) {
		fmt.Fprintf(os.Stderr, "  caused by: %s\n", rootMessage(curr))
	}
	printGoErrDetails(err)
}

// printDetailedError prints a full error chain without truncation for debugging.
func printDetailedError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", rootMessage(err))
	for curr := errors.Unwrap(err); curr != nil; curr = errors.Unwrap(curr) {
		msg := curr.Error()
		// For the leaf error (no further wrapping), print the full message
		if errors.Unwrap(curr) == nil {
			fmt.Fprintf(os.Stderr, "  caused by: %s\n", msg)
		} else {
			fmt.Fprintf(os.Stderr, "  caused by: %s\n", rootMessage(curr))
		}
	}
	printGoErrDetails(err)
}

// printGoErrDetails prints goerr values and tags attached to the error.
func printGoErrDetails(err error) {
	if values := goerr.Values(err); len(values) > 0 {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  Details:")
		keys := make([]string, 0, len(values))
		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(os.Stderr, "    %s: %v\n", k, values[k])
		}
	}

	if tags := goerr.Tags(err); len(tags) > 0 {
		fmt.Fprintf(os.Stderr, "  Tags: %s\n", strings.Join(tags, ", "))
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
