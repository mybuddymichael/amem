package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"amem/config"
	"amem/db"
	"amem/keyring"
	"github.com/urfave/cli/v3"
)

// withDB loads config, opens database, executes fn, and handles cleanup
func withDB(fn func(*db.DB) error) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.DBPath, cfg.EncryptionKey)
	if err != nil {
		return err
	}
	defer func() {
		if err := database.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close database: %v\n", err)
		}
	}()

	return fn(database)
}

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
					&cli.BoolFlag{
						Name:  "global",
						Usage: "Force global config",
					},
					&cli.BoolFlag{
						Name:  "local",
						Usage: "Use local config (.amem/config.json)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					dbPath := cmd.String("db-path")
					encryptionKey := cmd.String("encryption-key")
					useGlobal := cmd.Bool("global")
					useLocal := cmd.Bool("local")

					if dbPath == "" {
						return fmt.Errorf("--db-path is required")
					}
					if encryptionKey == "" {
						return fmt.Errorf("--encryption-key is required")
					}
					if useGlobal && useLocal {
						return fmt.Errorf("cannot specify both --global and --local")
					}

					// Determine config path
					var configPath string
					var keyringAccount string
					if useLocal {
						cwd, err := os.Getwd()
						if err != nil {
							return fmt.Errorf("failed to get current directory: %w", err)
						}
						configPath = config.LocalPath(cwd)
						keyringAccount = "local:" + cwd
					} else {
						var err error
						configPath, err = config.GlobalPath()
						if err != nil {
							return fmt.Errorf("failed to get global config path: %w", err)
						}
						keyringAccount = "global"
					}

					// Check if config exists and warn
					if _, err := os.Stat(configPath); err == nil {
						fmt.Fprintf(os.Stderr, "Warning: overwriting existing config at %s\n", configPath)
					}

					// Convert db path to absolute
					absDBPath, err := filepath.Abs(dbPath)
					if err != nil {
						return fmt.Errorf("failed to resolve absolute path: %w", err)
					}

					// If path is an existing directory, append amem.db to it
					if info, err := os.Stat(absDBPath); err == nil && info.IsDir() {
						absDBPath = filepath.Join(absDBPath, "amem.db")
					}

					// Ensure parent directory exists
					dir := filepath.Dir(absDBPath)
					if err := os.MkdirAll(dir, 0o755); err != nil {
						return fmt.Errorf("failed to create directory: %w", err)
					}

					// Check if database file already exists
					if _, err := os.Stat(absDBPath); err == nil {
						return fmt.Errorf("database already exists at %s (will not overwrite)", absDBPath)
					}

					// Initialize database
					database, err := db.Init(absDBPath, encryptionKey)
					if err != nil {
						return fmt.Errorf("failed to initialize database: %w", err)
					}
					defer func() {
						if err := database.Close(); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to close database: %v\n", err)
						}
					}()

					// Save config
					cfg := &config.Config{
						DBPath: absDBPath,
					}
					if err := config.Write(configPath, cfg); err != nil {
						return fmt.Errorf("failed to write config: %w", err)
					}

					// Save encryption key to keychain
					if err := keyring.Set(keyringAccount, encryptionKey); err != nil {
						return fmt.Errorf("failed to save encryption key: %w", err)
					}

					fmt.Printf("Database initialized at %s\n", absDBPath)
					fmt.Printf("Config saved to %s\n", configPath)
					return nil
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
							entities := cmd.Args().Slice()
							if len(entities) == 0 {
								return fmt.Errorf("at least one entity name is required")
							}

							return withDB(func(database *db.DB) error {
								for _, entity := range entities {
									_, err := database.AddEntity(entity)
									if err != nil {
										return fmt.Errorf("failed to add entity '%s': %w", entity, err)
									}
									fmt.Printf("Added entity: %s\n", entity)
								}
								return nil
							})
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
							entity := cmd.String("entity")
							text := cmd.String("text")

							return withDB(func(database *db.DB) error {
								_, err := database.AddObservation(entity, text)
								if err != nil {
									return err
								}

								fmt.Printf("Added observation about '%s'\n", entity)
								return nil
							})
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
							from := cmd.String("from")
							to := cmd.String("to")
							relType := cmd.String("type")

							return withDB(func(database *db.DB) error {
								_, err := database.AddRelationship(from, to, relType)
								if err != nil {
									return err
								}

								fmt.Printf("Added relationship: %s -[%s]-> %s\n", from, relType, to)
								return nil
							})
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
