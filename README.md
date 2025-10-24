# amem (agent memory)

## What this is

A command-line tool that gives an LLM agent memory.

## Stack

- Go
- sqlite
- [urfave/cli/v3](https://github.com/urfave/cli) for the CLI
- [go-sqlcipher](https://github.com/mutecomm/go-sqlcipher) for encrypting the database
- [go-keyring](https://github.com/zalando/go-keyring) for OS keychain integration

## Configuration

Config is stored as JSON and specifies the database path. amem discovers config in this order:

1. **Local config** (project-specific): `.amem/config.json` – searched by walking up the directory tree from the current directory
2. **Global config** (user-wide): `~/.config/amem/config.json`

The first config found is used. Once located, amem reads the database path from `db_path` in the config and loads the encrypted database from that location. The encryption key is retrieved from the OS keychain (stored under service `amem`), or falls back to the `AMEM_ENCRYPTION_KEY` environment variable if the keyring is unavailable.

Use `amem init` to create a config file.

## Examples

| Command | Description |
|---------|-------------|
| `amem help` | Show instructions on using the tool. |
| `amen agent-docs` | Show documentation to put in, e.g., AGENTS.md. |
| `amem init --db-path ~/.amem.db --encryption-key=L9XlJvCKeifThcHz0FQsf` | Start or use a memory database. |
| `amem check` | Check the status of the database and its encryption. |
| `amem add -h` | Get help about a command. |
| `amem add entity "Michael" "GitHub"` | Add one or more entities to the database. |
| `amem add observation --entity "Michael" --text "Working on his new agent memory project"` | Add an observation. |
| `amem add relationship --from "Michael" --to "GitHub" --type "uses"` | Add a relationship. |
| `amem search "Michael" "GitHub" "uses" "tools"` | Search for any mentions of key words. |
| `amem search entities "Michael" "tools"` | Search only entities. |
| `amem search observations --about "GitHub"` | Search for observations about an entity. |
| `amem search observations --about "GitHub" -- "tools" "AI" "LLM"` | Search for observations about an entity with specific phrases. |
| `amem search relationships "Michael"` | Search only relationships. |
| `amem search relationships --to "GitHub"` | Search for relationships where an entity is involved. |
| `amem search --type "uses" --from "Michael"` | Search for relationships by type or entity. |
| `amem search --with-ids` | Show database IDs with results. |
| `amem delete entity "GitHub"` | Delete an entity |
| `amem delete observation --ids 1` | Delete an observation with an ID. |
| `amem delete relationship --ids 14` | Delete a relationship with an ID. |
| `amem delete entity --ids 14 15 12 9 1 5` | Delete multiple entities by ID. |
| `amem edit entity "Michael" --new-name "Michael Hanson"` | Change an entity's name. |
| `amem edit observation --id 1 --new-text "Working on a new agent memory project"` | Change an observation's text. |
| `amem change-encryption-key --new-key=L9XlJvCKeifThcHz0FQsf` | Change the encryption key. |

## Database schema

| Table | Columns |
|-------|---------|
| entities | id, text |
| observations | id, entity_id, text, timestamp |
| relationships | id, from_id, to_id, type, timestamp |


## Encryption

The database is always fully encrypted using [go-sqlcipher](https://github.com/mutecomm/go-sqlcipher).
