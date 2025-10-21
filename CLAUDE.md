# amem (agent memory)

## What this is

A command-line tool that gives an LLM agent memory.

## Stack

- Go
- sqlite
- [urfave/cli/v3](https://github.com/urfave/cli) for the CLI
- [go-sqlcipher](https://github.com/mutecomm/go-sqlcipher) for encrypting the database

## Examples

`amem onboard`
`amem init --db-path ~/.amem.db --encryption-key=L9XlJvCKeifThcHz0FQsf`
`amem check`
`amem add -h`
`amem add entity "Michael"`
`amem add observation --entity "Michael" --text "Working on his new agent memory project"`
`amem add relationship --from "Michael" --to "GitHub" --type "uses"`
`amem search "Michael" "GitHub" "uses" "tools"`
`amem search entities "Michael"`
`amem search relationships "Michael"`
`amem search relationships --to "GitHub"`
`amem search --type "uses" --from "Michael"`
`amem delete entity "GitHub"`
`amem search --with-ids`
`amem delete observation --id 1`
`amem delete relationship --id 14`

