package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func buildCommand() *cli.Command {
	return &cli.Command{
		Name:  "amem",
		Usage: "A command-line tool that gives an LLM agent memory",
		Commands: []*cli.Command{
			{
				Name:  "help",
				Usage: "Show instructions on using the tool",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return fmt.Errorf("not yet implemented")
				},
			},
			{
				Name:  "agent-docs",
				Usage: "Show documentation to put in, e.g., AGENTS.md",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return fmt.Errorf("not yet implemented")
				},
			},
			{
				Name:  "init",
				Usage: "Start or use a memory database",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "db-path",
						Usage: "Path to the database file",
					},
					&cli.StringFlag{
						Name:  "encryption-key",
						Usage: "Encryption key for the database",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return fmt.Errorf("not yet implemented")
				},
			},
			{
				Name:  "check",
				Usage: "Check the status of the database and its encryption",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return fmt.Errorf("not yet implemented")
				},
			},
			{
				Name:  "add",
				Usage: "Add entities, observations, or relationships",
				Commands: []*cli.Command{
					{
						Name:      "entity",
						Usage:     "Add one or more entities to the database",
						ArgsUsage: "[entity names...]",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
					{
						Name:  "observation",
						Usage: "Add an observation",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "entity",
								Usage:    "Entity the observation is about",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "text",
								Usage:    "Observation text",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
					{
						Name:  "relationship",
						Usage: "Add a relationship",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "from",
								Usage:    "Source entity",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "to",
								Usage:    "Target entity",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "type",
								Usage:    "Relationship type",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
				},
			},
			{
				Name:  "search",
				Usage: "Search for mentions of keywords",
				Commands: []*cli.Command{
					{
						Name:      "entities",
						Usage:     "Search only entities",
						ArgsUsage: "[keywords...]",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
					{
						Name:  "observations",
						Usage: "Search observations",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "about",
								Usage: "Search for observations about an entity",
							},
						},
						ArgsUsage: "[keywords...]",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
					{
						Name:  "relationships",
						Usage: "Search relationships",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "to",
								Usage: "Search for relationships to an entity",
							},
							&cli.StringFlag{
								Name:  "from",
								Usage: "Search for relationships from an entity",
							},
							&cli.StringFlag{
								Name:  "type",
								Usage: "Search for relationships of a specific type",
							},
						},
						ArgsUsage: "[keywords...]",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "with-ids",
						Usage: "Show database IDs with results",
					},
				},
				ArgsUsage: "[keywords...]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return fmt.Errorf("not yet implemented")
				},
			},
			{
				Name:  "delete",
				Usage: "Delete entities, observations, or relationships",
				Commands: []*cli.Command{
					{
						Name:      "entity",
						Usage:     "Delete an entity",
						ArgsUsage: "[entity name]",
						Flags: []cli.Flag{
							&cli.IntSliceFlag{
								Name:  "ids",
								Usage: "Delete by IDs",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
					{
						Name:  "observation",
						Usage: "Delete an observation",
						Flags: []cli.Flag{
							&cli.IntSliceFlag{
								Name:     "ids",
								Usage:    "Delete by IDs",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
					{
						Name:  "relationship",
						Usage: "Delete a relationship",
						Flags: []cli.Flag{
							&cli.IntSliceFlag{
								Name:     "ids",
								Usage:    "Delete by IDs",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
				},
			},
			{
				Name:  "edit",
				Usage: "Edit entities or observations",
				Commands: []*cli.Command{
					{
						Name:      "entity",
						Usage:     "Change an entity's name",
						ArgsUsage: "[entity name]",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "new-name",
								Usage:    "New name for the entity",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
					{
						Name:  "observation",
						Usage: "Change an observation's text",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:     "id",
								Usage:    "Observation ID",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "new-text",
								Usage:    "New text for the observation",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return fmt.Errorf("not yet implemented")
						},
					},
				},
			},
		},
	}
}

func main() {
	cmd := buildCommand()

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
