package generator

import "github.com/m-mizutani/gollem"

// NewWriteFileToolSetForTest exposes writeFileToolSet for testing.
func NewWriteFileToolSetForTest(outputDir string) gollem.ToolSet {
	return newWriteFileToolSet(outputDir)
}
