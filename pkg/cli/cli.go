package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

// Run executes the CLI application.
func Run(args []string) error {
	app := &cli.Command{
		Name:  "tamamo",
		Usage: "LLM-driven web honeypot generator and server",
		Commands: []*cli.Command{
			newGenerateCommand(),
			newServeCommand(),
			newValidateCommand(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Fprintln(os.Stderr, "Use 'tamamo generate', 'tamamo serve', or 'tamamo validate'")
			return nil
		},
	}

	return app.Run(context.Background(), args)
}
