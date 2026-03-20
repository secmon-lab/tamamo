package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/secmon-lab/tamamo/pkg/cli/config"
	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
	scenarioSvc "github.com/secmon-lab/tamamo/pkg/service/scenario"
	"github.com/urfave/cli/v3"
)

func newValidateCommand() *cli.Command {
	var (
		logCfg       config.Logger
		scenarioPath string
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "scenario",
			Aliases:     []string{"s"},
			Usage:       "Path to scenario directory or ZIP file",
			Required:    true,
			Destination: &scenarioPath,
		},
	}
	flags = append(flags, logCfg.Flags()...)

	return &cli.Command{
		Name:    "validate",
		Aliases: []string{"v"},
		Usage:   "Validate scenario data integrity",
		Flags:   flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			s, err := scenarioSvc.Load(ctx, scenarioPath)
			if err != nil {
				return err
			}

			if err := s.Validate(); err != nil {
				printValidationErrors(s, err)
				return fmt.Errorf("validation failed with errors")
			}

			printValidationSuccess(s)
			return nil
		},
	}
}

func printValidationErrors(s *scenario.Scenario, err error) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "❌ Scenario validation failed")
	fmt.Fprintln(os.Stderr, "")

	if s.Meta.Name != "" {
		fmt.Fprintf(os.Stderr, "   Scenario: %s\n", s.Meta.Name)
	}

	// Unwrap joined errors
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		for i, e := range joined.Unwrap() {
			fmt.Fprintf(os.Stderr, "   %d. %s\n", i+1, e.Error())
		}
	} else {
		fmt.Fprintf(os.Stderr, "   - %s\n", err.Error())
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "   Total: %d error(s)\n", countErrors(err))
}

func printValidationSuccess(s *scenario.Scenario) {
	fmt.Println("")
	fmt.Println("✅ Scenario validation passed")
	fmt.Println("")
	fmt.Printf("   Scenario: %s\n", s.Meta.Name)
	fmt.Printf("   Pages:    %d\n", len(s.Pages))
	fmt.Printf("   Routes:   %d\n", len(s.Routes))
	if s.Meta.ServerSignature != "" {
		fmt.Printf("   Server:   %s\n", s.Meta.ServerSignature)
	}
}

func countErrors(err error) int {
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		return len(joined.Unwrap())
	}
	return 1
}
