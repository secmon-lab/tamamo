package cli

import (
	"fmt"

	"github.com/secmon-lab/tamamo/pkg/domain/interfaces"
)

// cliPrinter implements interfaces.Printer for CLI output.
type cliPrinter struct{}

var _ interfaces.Printer = (*cliPrinter)(nil)

func newCLIPrinter() *cliPrinter {
	return &cliPrinter{}
}

func (p *cliPrinter) ToolStart(toolName string, args map[string]any) {
	if path, ok := args["path"]; ok {
		fmt.Printf("🔧 Writing %s...\n", path)
	} else {
		fmt.Printf("🔧 Calling %s...\n", toolName)
	}
}

func (p *cliPrinter) ToolEnd(toolName string, result map[string]any, err error) {
	if err != nil {
		fmt.Printf("   ✗ %s failed: %s\n", toolName, err.Error())
		return
	}
	if result != nil {
		if path, ok := result["path"]; ok {
			if bytes, ok := result["bytes"]; ok {
				fmt.Printf("   ✓ %s (%v bytes)\n", path, bytes)
				return
			}
		}
	}
	fmt.Printf("   ✓ %s done\n", toolName)
}

func (p *cliPrinter) Message(text string) {
	fmt.Println(text)
}
