# amem (agent memory)

`amem` is a command-line tool that gives an LLM and agent a memory.

With this, your LLM agent can save information between sessions, and all of it stored and encrypted locally.

Easily track entities (people, places, things, etc.), observations about those entities, and relationships between entities. 

Specific directories can have their own separate memories, even nested within each other. Amem will search up parent directories until it finds a config file, and if not it will use the global config, if available.

## Install and run

If on macOS, you can easily install this with Homebrew:

```bash
brew tap mybuddymichael/tap
brew install amem
```

Or, if you have a version of Go installed, you can clone this repo and run `go install`.

Then, initialize a new database with `amem init`.

## How it works

A memory file is just small, encrypted sqlite database with the following tables:

- Entities (people, places, things, etc.)
- Observations (notes about those entities)
- Relationships (connections between entities)

When initializing the memory file, you tell `amem` whether the database is specific to a directory or a global database, where the file should be stored, and the encryption key.

Specific directories can have their own separate memory databases, even nested within each other. Amem will search up parent directories looking for a local config file, and if it doesn't find oneit will use the global config, if available.

## Usage examples

### Getting started

| Command | Description |
|---------|-------------|
| `amem help` | Show instructions on using amem. |
| `amem agent-docs` | Show documentation to put in, e.g., AGENTS.md or CLAUDE.md. |
| `amem init --db-path ~/.amem.db --encryption-key=L9XlJvCKeifThcHz0FQsf` | Start or use a memory database. |
| `amem check` | Check the status of the database and its encryption. |
| `amem add -h` | Get help about a command. |

### Adding things

| Command | Description |
|---------|-------------|
| `amem add entity "Michael" "GitHub"` | Add one or more entities to the database. |
| `amem add observation --entity "Michael" --text "Working on an agent memory project"` | Add an observation. |
| `amem add relationship --from "Michael" --to "GitHub" --type "uses"` | Add a relationship. |

### Searching

| Command | Description |
|---------|-------------|
| `amem search "Michael" "GitHub" "uses" "tools"` | Search for any mentions of key words. |
| `amem search --all "Michael" "GitHub" "uses" "tools"` | Search for any mentions of key words. |
| `amem search entities "Michael" "tools"` | Search only entities. |
| `amem search observations --about "GitHub"` | Search for observations about an entity. |
| `amem search observations --about "GitHub" -- "tools" "AI" "LLM"` | Search for observations about an entity with specific phrases. |
| `amem search relationships "Michael"` | Search only relationships. |
| `amem search relationships --to "GitHub"` | Search for relationships where an entity is involved. |
| `amem search --type "uses" --from "Michael"` | Search for relationships by type or entity. |
| `amem search --with-ids` | Show database IDs with results. |

### Editing

| Command | Description |
|---------|-------------|
| `amem edit entity "Michael" --new-name "Michael Hanson"` | Change an entity's name. |
| `amem edit observation --id 1 --new-text "Working on a new agent memory project"` | Change an observation's text. |

### Deleting things

| Command | Description |
|---------|-------------|
| `amem delete entity "GitHub"` | Delete an entity |
| `amem delete observation --ids 1` | Delete an observation with an ID. |
| `amem delete relationship --ids 14` | Delete a relationship with an ID. |
| `amem delete entity --ids 14 15 12 9 1 5` | Delete multiple entities by ID. |

### Configuration

| Command | Description |
|---------|-------------|
| `amem change-encryption-key --old-key=lXnJE --new-key=L9XlJvCKeifThcHz0FQsf` | Change the encryption key. |

## Configuration

Each directory can have its own config file, which is used to specify the database path.

Config is stored as JSON and specifies the database path. amem discovers config in this order:

1. **Local config** (project-specific): `.amem/config.json` – searched by walking up the directory tree from the current directory
2. **Global config** (user-wide): `~/.config/amem/config.json`

The first config found is used. Once located, amem reads the database path from `db_path` in the config and loads the encrypted database from that location. The encryption key is retrieved from the OS keychain (stored under service `amem`), or falls back to the `AMEM_ENCRYPTION_KEY` environment variable if the keyring is unavailable.

Use `amem init` to create a config file.

## Stack

- Go
- sqlite
- [urfave/cli/v3](https://github.com/urfave/cli) for the CLI
- [go-sqlcipher](https://github.com/mutecomm/go-sqlcipher) for encrypting the database
- [go-keyring](https://github.com/zalando/go-keyring) for OS keychain integration

## Database schema

The database is just an sqlite database with the following tables:

| Table | Columns |
|-------|---------|
| entities | id, text |
| observations | id, entity_id, text, timestamp |
| relationships | id, from_id, to_id, type, timestamp |

## Encryption

The database is always fully encrypted using [go-sqlcipher](https://github.com/mutecomm/go-sqlcipher). The encryption key is stored in the OS keychain. An existing key can be replaced with a new key using `amem change-encryption-key`.
