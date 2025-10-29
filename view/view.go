package view

import (
	"fmt"

	"amem/db"
)

// FormatEntities prints a formatted list of entities with a header.
func FormatEntities(entities []db.Entity, withIDs bool) {
	if len(entities) == 0 {
		fmt.Println("No entities found")
		return
	}
	fmt.Printf("Found %d entities:\n", len(entities))
	for _, e := range entities {
		fmt.Println(e.Format(withIDs))
	}
}

// FormatObservations prints a formatted list of observations with a header.
func FormatObservations(observations []db.Observation, withIDs bool) {
	if len(observations) == 0 {
		fmt.Println("No observations found")
		return
	}
	fmt.Printf("Found %d observations:\n", len(observations))
	for _, o := range observations {
		fmt.Println(o.Format(withIDs))
	}
}

// FormatRelationships prints a formatted list of relationships with a header.
func FormatRelationships(relationships []db.Relationship, withIDs bool) {
	if len(relationships) == 0 {
		fmt.Println("No relationships found")
		return
	}
	fmt.Printf("Found %d relationships:\n", len(relationships))
	for _, r := range relationships {
		fmt.Println(r.Format(withIDs))
	}
}

// FormatAll prints all search results with section headers.
func FormatAll(entities []db.Entity, observations []db.Observation, relationships []db.Relationship, withIDs bool) {
	totalResults := len(entities) + len(observations) + len(relationships)
	if totalResults == 0 {
		fmt.Println("No results found")
		return
	}

	if len(entities) > 0 {
		fmt.Printf("\nEntities (%d):\n", len(entities))
		for _, e := range entities {
			fmt.Println(e.Format(withIDs))
		}
	}

	if len(observations) > 0 {
		fmt.Printf("\nObservations (%d):\n", len(observations))
		for _, o := range observations {
			fmt.Println(o.Format(withIDs))
		}
	}

	if len(relationships) > 0 {
		fmt.Printf("\nRelationships (%d):\n", len(relationships))
		for _, r := range relationships {
			fmt.Println(r.Format(withIDs))
		}
	}
}
