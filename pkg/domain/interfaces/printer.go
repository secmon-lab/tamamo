package interfaces

// Printer displays scenario generation progress on the CLI.
type Printer interface {
	ToolStart(toolName string, args map[string]any)
	ToolEnd(toolName string, result map[string]any, err error)
	Message(text string)
}
