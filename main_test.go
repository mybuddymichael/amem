package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestTopLevelCommands(t *testing.T) {
	cmd := buildCommand()

	expectedCommands := []string{
		"help", "agent-docs", "init", "check", "add", "search", "delete", "edit",
	}

	if len(cmd.Commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedCommands), len(cmd.Commands))
	}

	for _, expected := range expectedCommands {
		found := false
		for _, c := range cmd.Commands {
			if c.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command %q not found", expected)
		}
	}
}

func TestHelpCommand(t *testing.T) {
	cmd := buildCommand()
	helpCmd := findCommand(cmd.Commands, "help")
	if helpCmd == nil {
		t.Fatal("help command not found")
	}

	if helpCmd.Usage != "Show instructions on using the tool" {
		t.Errorf("Unexpected help usage: %s", helpCmd.Usage)
	}
}

func TestAgentDocsCommand(t *testing.T) {
	cmd := buildCommand()
	agentDocsCmd := findCommand(cmd.Commands, "agent-docs")
	if agentDocsCmd == nil {
		t.Fatal("agent-docs command not found")
	}

	if agentDocsCmd.Usage != "Show documentation to put in, e.g., AGENTS.md" {
		t.Errorf("Unexpected agent-docs usage: %s", agentDocsCmd.Usage)
	}
}

func TestInitCommand(t *testing.T) {
	cmd := buildCommand()
	initCmd := findCommand(cmd.Commands, "init")
	if initCmd == nil {
		t.Fatal("init command not found")
	}

	if initCmd.Usage != "Start or use a memory database" {
		t.Errorf("Unexpected init usage: %s", initCmd.Usage)
	}

	// Check flags
	expectedStringFlags := []string{"db-path", "encryption-key"}
	expectedBoolFlags := []string{"global", "local"}

	if len(initCmd.Flags) != len(expectedStringFlags)+len(expectedBoolFlags) {
		t.Errorf("Expected %d flags, got %d", len(expectedStringFlags)+len(expectedBoolFlags), len(initCmd.Flags))
	}

	for _, name := range expectedStringFlags {
		flag := findFlag(initCmd.Flags, name)
		if flag == nil {
			t.Errorf("Flag %q not found", name)
			continue
		}
		if _, ok := flag.(*cli.StringFlag); !ok {
			t.Errorf("Flag %q is not a StringFlag", name)
		}
	}

	for _, name := range expectedBoolFlags {
		flag := findFlag(initCmd.Flags, name)
		if flag == nil {
			t.Errorf("Flag %q not found", name)
			continue
		}
		if _, ok := flag.(*cli.BoolFlag); !ok {
			t.Errorf("Flag %q is not a BoolFlag", name)
		}
	}
}

func TestCheckCommand(t *testing.T) {
	cmd := buildCommand()
	checkCmd := findCommand(cmd.Commands, "check")
	if checkCmd == nil {
		t.Fatal("check command not found")
	}

	if checkCmd.Usage != "Check the status of the database and its encryption" {
		t.Errorf("Unexpected check usage: %s", checkCmd.Usage)
	}
}

func TestAddCommand(t *testing.T) {
	cmd := buildCommand()
	addCmd := findCommand(cmd.Commands, "add")
	if addCmd == nil {
		t.Fatal("add command not found")
	}

	if addCmd.Usage != "Add entities, observations, or relationships" {
		t.Errorf("Unexpected add usage: %s", addCmd.Usage)
	}

	// Check subcommands
	expectedSubcommands := []string{"entity", "observation", "relationship"}
	if len(addCmd.Commands) != len(expectedSubcommands) {
		t.Errorf("Expected %d subcommands, got %d", len(expectedSubcommands), len(addCmd.Commands))
	}

	for _, expected := range expectedSubcommands {
		if findCommand(addCmd.Commands, expected) == nil {
			t.Errorf("Expected subcommand %q not found", expected)
		}
	}
}

func TestAddEntitySubcommand(t *testing.T) {
	cmd := buildCommand()
	addCmd := findCommand(cmd.Commands, "add")
	entityCmd := findCommand(addCmd.Commands, "entity")
	if entityCmd == nil {
		t.Fatal("add entity subcommand not found")
	}

	if entityCmd.Usage != "Add one or more entities to the database" {
		t.Errorf("Unexpected usage: %s", entityCmd.Usage)
	}

	if entityCmd.ArgsUsage != "[entity names...]" {
		t.Errorf("Unexpected args usage: %s", entityCmd.ArgsUsage)
	}
}

func TestAddObservationSubcommand(t *testing.T) {
	cmd := buildCommand()
	addCmd := findCommand(cmd.Commands, "add")
	obsCmd := findCommand(addCmd.Commands, "observation")
	if obsCmd == nil {
		t.Fatal("add observation subcommand not found")
	}

	if obsCmd.Usage != "Add an observation" {
		t.Errorf("Unexpected usage: %s", obsCmd.Usage)
	}

	// Check required flags
	expectedFlags := map[string]struct {
		usage    string
		required bool
	}{
		"entity": {"Entity the observation is about", true},
		"text":   {"Observation text", true},
	}

	for name, expected := range expectedFlags {
		flag := findFlag(obsCmd.Flags, name)
		if flag == nil {
			t.Errorf("Flag %q not found", name)
			continue
		}
		strFlag, ok := flag.(*cli.StringFlag)
		if !ok {
			t.Errorf("Flag %q is not a StringFlag", name)
			continue
		}
		if strFlag.Required != expected.required {
			t.Errorf("Flag %q required mismatch: expected %v, got %v", name, expected.required, strFlag.Required)
		}
	}
}

func TestAddRelationshipSubcommand(t *testing.T) {
	cmd := buildCommand()
	addCmd := findCommand(cmd.Commands, "add")
	relCmd := findCommand(addCmd.Commands, "relationship")
	if relCmd == nil {
		t.Fatal("add relationship subcommand not found")
	}

	if relCmd.Usage != "Add a relationship" {
		t.Errorf("Unexpected usage: %s", relCmd.Usage)
	}

	// Check required flags
	expectedFlags := map[string]struct {
		usage    string
		required bool
	}{
		"from": {"Source entity", true},
		"to":   {"Target entity", true},
		"type": {"Relationship type", true},
	}

	for name, expected := range expectedFlags {
		flag := findFlag(relCmd.Flags, name)
		if flag == nil {
			t.Errorf("Flag %q not found", name)
			continue
		}
		strFlag, ok := flag.(*cli.StringFlag)
		if !ok {
			t.Errorf("Flag %q is not a StringFlag", name)
			continue
		}
		if strFlag.Required != expected.required {
			t.Errorf("Flag %q required mismatch: expected %v, got %v", name, expected.required, strFlag.Required)
		}
	}
}

func TestSearchCommand(t *testing.T) {
	cmd := buildCommand()
	searchCmd := findCommand(cmd.Commands, "search")
	if searchCmd == nil {
		t.Fatal("search command not found")
	}

	if searchCmd.Usage != "Search for mentions of keywords" {
		t.Errorf("Unexpected search usage: %s", searchCmd.Usage)
	}

	// Check subcommands
	expectedSubcommands := []string{"entities", "observations", "relationships"}
	if len(searchCmd.Commands) != len(expectedSubcommands) {
		t.Errorf("Expected %d subcommands, got %d", len(expectedSubcommands), len(searchCmd.Commands))
	}

	// Check --with-ids flag
	flag := findFlag(searchCmd.Flags, "with-ids")
	if flag == nil {
		t.Error("with-ids flag not found")
	} else if _, ok := flag.(*cli.BoolFlag); !ok {
		t.Error("with-ids is not a BoolFlag")
	}
}

func TestSearchEntitiesSubcommand(t *testing.T) {
	cmd := buildCommand()
	searchCmd := findCommand(cmd.Commands, "search")
	entitiesCmd := findCommand(searchCmd.Commands, "entities")
	if entitiesCmd == nil {
		t.Fatal("search entities subcommand not found")
	}

	if entitiesCmd.Usage != "Search only entities" {
		t.Errorf("Unexpected usage: %s", entitiesCmd.Usage)
	}

	if entitiesCmd.ArgsUsage != "[keywords...]" {
		t.Errorf("Unexpected args usage: %s", entitiesCmd.ArgsUsage)
	}
}

func TestSearchObservationsSubcommand(t *testing.T) {
	cmd := buildCommand()
	searchCmd := findCommand(cmd.Commands, "search")
	obsCmd := findCommand(searchCmd.Commands, "observations")
	if obsCmd == nil {
		t.Fatal("search observations subcommand not found")
	}

	if obsCmd.Usage != "Search observations" {
		t.Errorf("Unexpected usage: %s", obsCmd.Usage)
	}

	// Check --about flag
	flag := findFlag(obsCmd.Flags, "about")
	if flag == nil {
		t.Error("about flag not found")
	} else if _, ok := flag.(*cli.StringFlag); !ok {
		t.Error("about is not a StringFlag")
	}
}

func TestSearchRelationshipsSubcommand(t *testing.T) {
	cmd := buildCommand()
	searchCmd := findCommand(cmd.Commands, "search")
	relCmd := findCommand(searchCmd.Commands, "relationships")
	if relCmd == nil {
		t.Fatal("search relationships subcommand not found")
	}

	if relCmd.Usage != "Search relationships" {
		t.Errorf("Unexpected usage: %s", relCmd.Usage)
	}

	// Check flags
	expectedFlags := []string{"to", "from", "type"}
	for _, name := range expectedFlags {
		flag := findFlag(relCmd.Flags, name)
		if flag == nil {
			t.Errorf("Flag %q not found", name)
		} else if _, ok := flag.(*cli.StringFlag); !ok {
			t.Errorf("Flag %q is not a StringFlag", name)
		}
	}
}

func TestDeleteCommand(t *testing.T) {
	cmd := buildCommand()
	deleteCmd := findCommand(cmd.Commands, "delete")
	if deleteCmd == nil {
		t.Fatal("delete command not found")
	}

	if deleteCmd.Usage != "Delete entities, observations, or relationships" {
		t.Errorf("Unexpected delete usage: %s", deleteCmd.Usage)
	}

	// Check subcommands
	expectedSubcommands := []string{"entity", "observation", "relationship"}
	if len(deleteCmd.Commands) != len(expectedSubcommands) {
		t.Errorf("Expected %d subcommands, got %d", len(expectedSubcommands), len(deleteCmd.Commands))
	}
}

func TestDeleteEntitySubcommand(t *testing.T) {
	cmd := buildCommand()
	deleteCmd := findCommand(cmd.Commands, "delete")
	entityCmd := findCommand(deleteCmd.Commands, "entity")
	if entityCmd == nil {
		t.Fatal("delete entity subcommand not found")
	}

	if entityCmd.Usage != "Delete an entity" {
		t.Errorf("Unexpected usage: %s", entityCmd.Usage)
	}

	// Check --ids flag (optional for entity)
	flag := findFlag(entityCmd.Flags, "ids")
	if flag == nil {
		t.Error("ids flag not found")
	} else if _, ok := flag.(*cli.IntSliceFlag); !ok {
		t.Error("ids is not an IntSliceFlag")
	}
}

func TestDeleteObservationSubcommand(t *testing.T) {
	cmd := buildCommand()
	deleteCmd := findCommand(cmd.Commands, "delete")
	obsCmd := findCommand(deleteCmd.Commands, "observation")
	if obsCmd == nil {
		t.Fatal("delete observation subcommand not found")
	}

	if obsCmd.Usage != "Delete an observation" {
		t.Errorf("Unexpected usage: %s", obsCmd.Usage)
	}

	// Check --ids flag (required)
	flag := findFlag(obsCmd.Flags, "ids")
	if flag == nil {
		t.Fatal("ids flag not found")
	}
	intSliceFlag, ok := flag.(*cli.IntSliceFlag)
	if !ok {
		t.Fatal("ids is not an IntSliceFlag")
	}
	if !intSliceFlag.Required {
		t.Error("ids flag should be required")
	}
}

func TestDeleteRelationshipSubcommand(t *testing.T) {
	cmd := buildCommand()
	deleteCmd := findCommand(cmd.Commands, "delete")
	relCmd := findCommand(deleteCmd.Commands, "relationship")
	if relCmd == nil {
		t.Fatal("delete relationship subcommand not found")
	}

	if relCmd.Usage != "Delete a relationship" {
		t.Errorf("Unexpected usage: %s", relCmd.Usage)
	}

	// Check --ids flag (required)
	flag := findFlag(relCmd.Flags, "ids")
	if flag == nil {
		t.Fatal("ids flag not found")
	}
	intSliceFlag, ok := flag.(*cli.IntSliceFlag)
	if !ok {
		t.Fatal("ids is not an IntSliceFlag")
	}
	if !intSliceFlag.Required {
		t.Error("ids flag should be required")
	}
}

func TestEditCommand(t *testing.T) {
	cmd := buildCommand()
	editCmd := findCommand(cmd.Commands, "edit")
	if editCmd == nil {
		t.Fatal("edit command not found")
	}

	if editCmd.Usage != "Edit entities or observations" {
		t.Errorf("Unexpected edit usage: %s", editCmd.Usage)
	}

	// Check subcommands
	expectedSubcommands := []string{"entity", "observation"}
	if len(editCmd.Commands) != len(expectedSubcommands) {
		t.Errorf("Expected %d subcommands, got %d", len(expectedSubcommands), len(editCmd.Commands))
	}
}

func TestEditEntitySubcommand(t *testing.T) {
	cmd := buildCommand()
	editCmd := findCommand(cmd.Commands, "edit")
	entityCmd := findCommand(editCmd.Commands, "entity")
	if entityCmd == nil {
		t.Fatal("edit entity subcommand not found")
	}

	if entityCmd.Usage != "Change an entity's name" {
		t.Errorf("Unexpected usage: %s", entityCmd.Usage)
	}

	// Check --new-name flag (required)
	flag := findFlag(entityCmd.Flags, "new-name")
	if flag == nil {
		t.Fatal("new-name flag not found")
	}
	strFlag, ok := flag.(*cli.StringFlag)
	if !ok {
		t.Fatal("new-name is not a StringFlag")
	}
	if !strFlag.Required {
		t.Error("new-name flag should be required")
	}
}

func TestEditObservationSubcommand(t *testing.T) {
	cmd := buildCommand()
	editCmd := findCommand(cmd.Commands, "edit")
	obsCmd := findCommand(editCmd.Commands, "observation")
	if obsCmd == nil {
		t.Fatal("edit observation subcommand not found")
	}

	if obsCmd.Usage != "Change an observation's text" {
		t.Errorf("Unexpected usage: %s", obsCmd.Usage)
	}

	// Check flags
	expectedFlags := map[string]struct {
		flagType string
		required bool
	}{
		"id":       {"IntFlag", true},
		"new-text": {"StringFlag", true},
	}

	for name, expected := range expectedFlags {
		flag := findFlag(obsCmd.Flags, name)
		if flag == nil {
			t.Errorf("Flag %q not found", name)
			continue
		}

		var required bool
		switch expected.flagType {
		case "IntFlag":
			intFlag, ok := flag.(*cli.IntFlag)
			if !ok {
				t.Errorf("Flag %q is not an IntFlag", name)
				continue
			}
			required = intFlag.Required
		case "StringFlag":
			strFlag, ok := flag.(*cli.StringFlag)
			if !ok {
				t.Errorf("Flag %q is not a StringFlag", name)
				continue
			}
			required = strFlag.Required
		}

		if required != expected.required {
			t.Errorf("Flag %q required mismatch: expected %v, got %v", name, expected.required, required)
		}
	}
}

func TestPrompt(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		defaultValue string
		input        string
		expected     string
		expectError  bool
	}{
		{
			name:         "user provides value",
			message:      "Enter value",
			defaultValue: "",
			input:        "custom\n",
			expected:     "custom",
			expectError:  false,
		},
		{
			name:         "user accepts default",
			message:      "Enter value",
			defaultValue: "default",
			input:        "\n",
			expected:     "default",
			expectError:  false,
		},
		{
			name:         "user provides value with default available",
			message:      "Enter value",
			defaultValue: "default",
			input:        "custom\n",
			expected:     "custom",
			expectError:  false,
		},
		{
			name:         "user provides value with whitespace",
			message:      "Enter value",
			defaultValue: "",
			input:        "  custom  \n",
			expected:     "custom",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()

			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			os.Stdin = r

			// Write input
			go func() {
				defer func() { _ = w.Close() }()
				_, _ = w.Write([]byte(tt.input))
			}()

			// Capture stdout to verify prompt message
			oldStdout := os.Stdout
			defer func() { os.Stdout = oldStdout }()

			rOut, wOut, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			os.Stdout = wOut

			// Run prompt
			result, err := prompt(tt.message, tt.defaultValue)

			// Close write end and read stdout
			_ = wOut.Close()
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(rOut)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Helper functions
func findCommand(commands []*cli.Command, name string) *cli.Command {
	for _, cmd := range commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

func findFlag(flags []cli.Flag, name string) cli.Flag {
	for _, flag := range flags {
		if flag.Names()[0] == name {
			return flag
		}
	}
	return nil
}
