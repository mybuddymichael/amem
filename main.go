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

func printEntities(entities []db.Entity, withIDs bool) {
	for _, e := range entities {
		if withIDs {
			fmt.Printf("[%d] %s\n", e.ID, e.Text)
		} else {
			fmt.Printf("%s\n", e.Text)
		}
	}
}

func printObservations(observations []db.Observation, withIDs bool) {
	for _, o := range observations {
		if withIDs {
			fmt.Printf("[%d] %s: %s (%s)\n", o.ID, o.EntityText, o.Text, o.Timestamp)
		} else {
			fmt.Printf("%s: %s (%s)\n", o.EntityText, o.Text, o.Timestamp)
		}
	}
}

func printRelationships(relationships []db.Relationship, withIDs bool) {
	for _, r := range relationships {
		if withIDs {
			fmt.Printf("[%d] %s -[%s]-> %s (%s)\n", r.ID, r.FromText, r.Type, r.ToText, r.Timestamp)
		} else {
			fmt.Printf("%s -[%s]-> %s (%s)\n", r.FromText, r.Type, r.ToText, r.Timestamp)
		}
	}
}

func formatEntities(entities []db.Entity, withIDs bool) {
	if len(entities) == 0 {
		fmt.Println("No entities found")
		return
	}
	fmt.Printf("Found %d entities:\n", len(entities))
	printEntities(entities, withIDs)
}

func formatObservations(observations []db.Observation, withIDs bool) {
	if len(observations) == 0 {
		fmt.Println("No observations found")
		return
	}
	fmt.Printf("Found %d observations:\n", len(observations))
	printObservations(observations, withIDs)
}

func formatRelationships(relationships []db.Relationship, withIDs bool) {
	if len(relationships) == 0 {
		fmt.Println("No relationships found")
		return
	}
	fmt.Printf("Found %d relationships:\n", len(relationships))
	printRelationships(relationships, withIDs)
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
					// Discover config
					cwd, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("failed to get current directory: %w", err)
					}

					var configPath string
					var configType string
					var cfg *config.LoadedConfig

					// Try local config first
					localPath, err := config.FindLocal(cwd)
					if err == nil {
						configPath = localPath
						configType = "local"
					} else {
						// Try global config
						globalPath, err := config.GlobalPath()
						if err != nil {
							return fmt.Errorf("failed to get global config path: %w", err)
						}
						configPath = globalPath
						configType = "global"
					}

					// Load config
					cfg, err = config.Load()
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}

					fmt.Printf("✓ Config loaded (%s): %s\n", configType, configPath)
					fmt.Printf("✓ Database path: %s\n", cfg.DBPath)

					// Check if database file exists
					if _, err := os.Stat(cfg.DBPath); err != nil {
						if os.IsNotExist(err) {
							return fmt.Errorf("✗ Database file not found at %s", cfg.DBPath)
						}
						return fmt.Errorf("failed to check database file: %w", err)
					}
					fmt.Printf("✓ Database file exists\n")

					// Try to open database (validates encryption key)
					database, err := db.Open(cfg.DBPath, cfg.EncryptionKey)
					if err != nil {
						return fmt.Errorf("✗ Failed to open database: %w", err)
					}
					defer func() {
						if err := database.Close(); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to close database: %v\n", err)
						}
					}()

					fmt.Printf("✓ Encryption key valid\n")

					// Get counts
					entityCount, err := database.CountEntities()
					if err != nil {
						return fmt.Errorf("failed to count entities: %w", err)
					}

					obsCount, err := database.CountObservations()
					if err != nil {
						return fmt.Errorf("failed to count observations: %w", err)
					}

					relCount, err := database.CountRelationships()
					if err != nil {
						return fmt.Errorf("failed to count relationships: %w", err)
					}

					fmt.Printf("\nDatabase contents:\n")
					fmt.Printf("  Entities: %d\n", entityCount)
					fmt.Printf("  Observations: %d\n", obsCount)
					fmt.Printf("  Relationships: %d\n", relCount)

					return nil
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
							keywords := cmd.Args().Slice()
							withIDs := cmd.Bool("with-ids")

							return withDB(func(database *db.DB) error {
								results, err := database.SearchEntities(keywords)
								if err != nil {
									return err
								}
								formatEntities(results, withIDs)
								return nil
							})
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
							keywords := cmd.Args().Slice()
							entityText := cmd.String("about")
							withIDs := cmd.Bool("with-ids")

							return withDB(func(database *db.DB) error {
								results, err := database.SearchObservations(entityText, keywords)
								if err != nil {
									return err
								}
								formatObservations(results, withIDs)
								return nil
							})
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
							keywords := cmd.Args().Slice()
							fromText := cmd.String("from")
							toText := cmd.String("to")
							relType := cmd.String("type")
							withIDs := cmd.Bool("with-ids")

							return withDB(func(database *db.DB) error {
								results, err := database.SearchRelationships(fromText, toText, relType, keywords)
								if err != nil {
									return err
								}
								formatRelationships(results, withIDs)
								return nil
							})
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
					keywords := cmd.Args().Slice()
					withIDs := cmd.Bool("with-ids")

					return withDB(func(database *db.DB) error {
						entities, observations, relationships, err := database.SearchAll(keywords)
						if err != nil {
							return err
						}

						totalResults := len(entities) + len(observations) + len(relationships)
						if totalResults == 0 {
							fmt.Println("No results found")
							return nil
						}

						if len(entities) > 0 {
							fmt.Printf("\nEntities (%d):\n", len(entities))
							printEntities(entities, withIDs)
						}

						if len(observations) > 0 {
							fmt.Printf("\nObservations (%d):\n", len(observations))
							printObservations(observations, withIDs)
						}

						if len(relationships) > 0 {
							fmt.Printf("\nRelationships (%d):\n", len(relationships))
							printRelationships(relationships, withIDs)
						}

						return nil
					})
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
							entityName := cmd.Args().First()
							ids := cmd.IntSlice("ids")

							// Validate that only one method is provided
							if entityName != "" && len(ids) > 0 {
								return fmt.Errorf("cannot specify both entity name and --ids")
							}
							if entityName == "" && len(ids) == 0 {
								return fmt.Errorf("must specify either entity name or --ids")
							}

							return withDB(func(database *db.DB) error {
								if entityName != "" {
									// Delete by name
									if err := database.DeleteEntityByText(entityName); err != nil {
										return err
									}
									fmt.Printf("Deleted entity: %s\n", entityName)
								} else {
									// Delete by IDs
									for _, id := range ids {
										if err := database.DeleteEntity(int64(id)); err != nil {
											return fmt.Errorf("failed to delete entity ID %d: %w", id, err)
										}
										fmt.Printf("Deleted entity ID %d\n", id)
									}
								}
								return nil
							})
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
							ids := cmd.IntSlice("ids")

							return withDB(func(database *db.DB) error {
								for _, id := range ids {
									if err := database.DeleteObservation(int64(id)); err != nil {
										return fmt.Errorf("failed to delete observation ID %d: %w", id, err)
									}
									fmt.Printf("Deleted observation ID %d\n", id)
								}
								return nil
							})
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
							ids := cmd.IntSlice("ids")

							return withDB(func(database *db.DB) error {
								for _, id := range ids {
									if err := database.DeleteRelationship(int64(id)); err != nil {
										return fmt.Errorf("failed to delete relationship ID %d: %w", id, err)
									}
									fmt.Printf("Deleted relationship ID %d\n", id)
								}
								return nil
							})
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
