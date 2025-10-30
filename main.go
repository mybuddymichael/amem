package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"amem/config"
	"amem/db"
	"amem/keyring"
	"amem/view"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

var version = "dev"

// stdinReader is a global buffered reader for stdin, reused across calls to avoid buffering issues
var stdinReader *bufio.Reader

// resetStdinReader resets the global stdin reader (used in tests)
func resetStdinReader() {
	stdinReader = nil
}

// agentDocsContent is the documentation shown by 'amem agent-docs'
const agentDocsContent = `<memory>
	- If the user instructs you to use your memory, use the 'amem' utility.
	- Run 'amem help' to see the available commands.
	- Use 'amem search' to find relevant memories based on the conversation and user's request.
	- As the conversation progresses, use 'amem add' to add new memories.
	- Be judicious with the memories you add, making sure each is likely to have long-term value.
	- Prefer proper relationships over relational observations.
</memory>`

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

func prompt(message string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", message, defaultValue)
	} else {
		fmt.Printf("%s: ", message)
	}

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("no input provided")
	}

	value := strings.TrimSpace(scanner.Text())
	if value == "" && defaultValue != "" {
		return defaultValue, nil
	}
	return value, nil
}

func securePrompt(message string) (string, error) {
	fmt.Printf("%s: ", message)

	var password []byte
	var err error

	// Check if stdin is a terminal
	if term.IsTerminal(int(os.Stdin.Fd())) {
		// Use secure password reading for terminals
		password, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
	} else {
		// Fall back to regular reading for non-terminals (e.g., in tests or pipes)
		// Use a global reader to avoid buffering issues when called multiple times
		if stdinReader == nil {
			stdinReader = bufio.NewReader(os.Stdin)
		}
		line, err := stdinReader.ReadString('\n')
		if err != nil {
			return "", err
		}
		// Remove the trailing newline
		password = []byte(strings.TrimSuffix(line, "\n"))
		fmt.Println()
	}

	if err != nil {
		return "", err
	}

	value := strings.TrimSpace(string(password))
	if value == "" {
		return "", fmt.Errorf("no input provided")
	}

	return value, nil
}

func securePromptWithConfirmation(message string) (string, error) {
	key, err := securePrompt(message)
	if err != nil {
		return "", err
	}

	confirmation, err := securePrompt(message + " (confirm)")
	if err != nil {
		return "", err
	}

	if key != confirmation {
		return "", fmt.Errorf("keys do not match")
	}

	return key, nil
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
					return cli.ShowAppHelp(cmd)
				},
			},
			{
				Name:  "agent-docs",
				Usage: "Show documentation to put in, e.g., AGENTS.md",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Print(agentDocsContent)
					return nil
				},
			},
			{
				Name:  "version",
				Usage: "Display the version",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println(version)
					return nil
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

					if useGlobal && useLocal {
						return fmt.Errorf("cannot specify both --global and --local")
					}

					// Prompt for db-path if not provided
					if dbPath == "" {
						var defaultPath string
						if useLocal {
							cwd, err := os.Getwd()
							if err != nil {
								return fmt.Errorf("failed to get current directory: %w", err)
							}
							defaultPath = filepath.Join(cwd, "amem.db")
						} else {
							homeDir, err := os.UserHomeDir()
							if err != nil {
								return fmt.Errorf("failed to get home directory: %w", err)
							}
							defaultPath = filepath.Join(homeDir, "amem.db")
						}
						var err error
						dbPath, err = prompt("Database path", defaultPath)
						if err != nil {
							return fmt.Errorf("failed to read db-path: %w", err)
						}
						if dbPath == "" {
							return fmt.Errorf("db-path is required")
						}
					}

					// Prompt for encryption-key if not provided
					if encryptionKey == "" {
						var err error
						encryptionKey, err = securePromptWithConfirmation("Encryption key")
						if err != nil {
							return fmt.Errorf("failed to read encryption-key: %w", err)
						}
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
				Name:  "change-encryption-key",
				Usage: "Change the encryption key for the database",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "new-key",
						Usage:    "New encryption key for the database",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					newKey := cmd.String("new-key")

					// Load config to get current key
					cfg, err := config.Load()
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}

					// Determine keyring account
					cwd, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("failed to get current directory: %w", err)
					}

					var keyringAccount string
					localPath, err := config.FindLocal(cwd)
					if err == nil {
						// Using local config
						keyringAccount = "local:" + cwd
						_ = localPath // avoid unused variable warning
					} else {
						// Using global config
						keyringAccount = "global"
					}

					// Check if database exists
					if _, err := os.Stat(cfg.DBPath); err != nil {
						if os.IsNotExist(err) {
							return fmt.Errorf("database file not found at %s", cfg.DBPath)
						}
						return fmt.Errorf("failed to check database file: %w", err)
					}

					// Prompt for confirmation
					fmt.Println("WARNING: This will re-encrypt the entire database with a new key.")
					fmt.Println("Make sure you have a backup before proceeding.")
					confirmation, err := prompt("Continue? Type 'yes' to confirm", "")
					if err != nil {
						return fmt.Errorf("failed to read confirmation: %w", err)
					}
					if confirmation != "yes" {
						fmt.Println("Operation cancelled.")
						return nil
					}

					// Open database with current key
					database, err := db.Open(cfg.DBPath, cfg.EncryptionKey)
					if err != nil {
						return fmt.Errorf("failed to open database with current key: %w", err)
					}
					defer func() {
						if err := database.Close(); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to close database: %v\n", err)
						}
					}()

					// Rekey the database
					if err := database.Rekey(newKey); err != nil {
						return fmt.Errorf("failed to rekey database: %w", err)
					}

					// Update keyring with new key
					if err := keyring.Set(keyringAccount, newKey); err != nil {
						// Database is already rekeyed, so we can't fail here
						// Print the new key so user doesn't lose it
						fmt.Fprintf(os.Stderr, "WARNING: Failed to save new key to keyring: %v\n", err)
						fmt.Fprintf(os.Stderr, "Your database has been rekeyed successfully, but the key was not saved to the keyring.\n")
						fmt.Fprintf(os.Stderr, "Please save this key manually:\n")
						fmt.Printf("\nNew encryption key: %s\n\n", newKey)
						fmt.Fprintf(os.Stderr, "You can set it using the AMEM_ENCRYPTION_KEY environment variable.\n")
						return nil
					}

					// Verify by reopening with new key
					_ = database.Close()
					verifyDB, err := db.Open(cfg.DBPath, newKey)
					if err != nil {
						return fmt.Errorf("rekeying succeeded but verification failed: %w", err)
					}
					defer func() {
						if err := verifyDB.Close(); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to close verification database: %v\n", err)
						}
					}()

					fmt.Println("✓ Database encryption key changed successfully")
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
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "any",
								Usage: "Match any keyword (OR logic, default)",
							},
							&cli.BoolFlag{
								Name:  "all",
								Usage: "Match all keywords (AND logic)",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							keywords := cmd.Args().Slice()
							withIDs := cmd.Bool("with-ids")
							useAny := cmd.Bool("any")
							useAll := cmd.Bool("all")

							if useAny && useAll {
								return fmt.Errorf("cannot specify both --any and --all")
							}

							// Default to union (any)
							useUnion := !useAll

							return withDB(func(database *db.DB) error {
								results, err := database.SearchEntities(keywords, useUnion)
								if err != nil {
									return err
								}
								view.FormatEntities(results, withIDs)
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
							&cli.BoolFlag{
								Name:  "any",
								Usage: "Match any keyword (OR logic, default)",
							},
							&cli.BoolFlag{
								Name:  "all",
								Usage: "Match all keywords (AND logic)",
							},
						},
						ArgsUsage: "[keywords...]",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							keywords := cmd.Args().Slice()
							entityText := cmd.String("about")
							withIDs := cmd.Bool("with-ids")
							useAny := cmd.Bool("any")
							useAll := cmd.Bool("all")

							if useAny && useAll {
								return fmt.Errorf("cannot specify both --any and --all")
							}

							// Default to union (any)
							useUnion := !useAll

							return withDB(func(database *db.DB) error {
								results, err := database.SearchObservations(entityText, keywords, useUnion)
								if err != nil {
									return err
								}
								view.FormatObservations(results, withIDs)
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
							&cli.BoolFlag{
								Name:  "any",
								Usage: "Match any keyword (OR logic, default)",
							},
							&cli.BoolFlag{
								Name:  "all",
								Usage: "Match all keywords (AND logic)",
							},
						},
						ArgsUsage: "[keywords...]",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							keywords := cmd.Args().Slice()
							fromText := cmd.String("from")
							toText := cmd.String("to")
							relType := cmd.String("type")
							withIDs := cmd.Bool("with-ids")
							useAny := cmd.Bool("any")
							useAll := cmd.Bool("all")

							if useAny && useAll {
								return fmt.Errorf("cannot specify both --any and --all")
							}

							// Default to union (any)
							useUnion := !useAll

							return withDB(func(database *db.DB) error {
								results, err := database.SearchRelationships(fromText, toText, relType, keywords, useUnion)
								if err != nil {
									return err
								}
								view.FormatRelationships(results, withIDs)
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
					&cli.BoolFlag{
						Name:  "any",
						Usage: "Match any keyword (OR logic, default)",
					},
					&cli.BoolFlag{
						Name:  "all",
						Usage: "Match all keywords (AND logic)",
					},
				},
				ArgsUsage: "[keywords...]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					keywords := cmd.Args().Slice()
					withIDs := cmd.Bool("with-ids")
					useAny := cmd.Bool("any")
					useAll := cmd.Bool("all")

					if useAny && useAll {
						return fmt.Errorf("cannot specify both --any and --all")
					}

					// Default to union (any)
					useUnion := !useAll

					return withDB(func(database *db.DB) error {
						entities, observations, relationships, err := database.SearchAll(keywords, useUnion)
						if err != nil {
							return err
						}

						view.FormatAll(entities, observations, relationships, withIDs)
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
							entityName := cmd.Args().First()
							newName := cmd.String("new-name")

							if entityName == "" {
								return fmt.Errorf("entity name is required")
							}

							return withDB(func(database *db.DB) error {
								if err := database.UpdateEntity(entityName, newName); err != nil {
									return err
								}
								fmt.Printf("Updated entity '%s' to '%s'\n", entityName, newName)
								return nil
							})
						},
					},
					{
						Name:  "observation",
						Usage: "Change an observation's text or entity",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:     "id",
								Usage:    "Observation ID",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "new-text",
								Usage: "New text for the observation",
							},
							&cli.IntFlag{
								Name:  "new-entity-id",
								Usage: "New entity ID for the observation",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							id := cmd.Int("id")
							newText := cmd.String("new-text")
							newEntityID := cmd.Int("new-entity-id")

							// At least one flag must be provided
							if newText == "" && newEntityID == 0 {
								return fmt.Errorf("at least one of --new-text or --new-entity-id must be provided")
							}

							return withDB(func(database *db.DB) error {
								if newText != "" {
									if err := database.UpdateObservation(int64(id), newText); err != nil {
										return err
									}
								}
								if newEntityID != 0 {
									if err := database.UpdateObservationEntity(int64(id), int64(newEntityID)); err != nil {
										return err
									}
								}
								fmt.Printf("Updated observation ID %d\n", id)
								return nil
							})
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
