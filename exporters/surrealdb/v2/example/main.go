package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/eknowles/gedcom/v39"
	v2 "github.com/eknowles/gedcom/v39/exporters/surrealdb/v2"
)

// Example demonstrates using the v2 SurrealDB exporter with the Graph Pattern
func main() {
	// Create a sample GEDCOM document
	doc := gedcom.NewDocument()

	// Add individuals
	john := doc.AddIndividual("I1")
	john.AddNode(gedcom.NewNameNode("John /Smith/"))
	john.AddNode(gedcom.NewSexNode("M"))
	john.AddNode(gedcom.NewBirthNode("",
		gedcom.NewDateNode("15 Mar 1950"),
		gedcom.NewPlaceNode("London, England"),
	))

	jane := doc.AddIndividual("I2")
	jane.AddNode(gedcom.NewNameNode("Jane /Doe/"))
	jane.AddNode(gedcom.NewSexNode("F"))
	jane.AddNode(gedcom.NewBirthNode("",
		gedcom.NewDateNode("22 Jul 1952"),
		gedcom.NewPlaceNode("Manchester, England"),
	))

	// Add child
	bob := doc.AddIndividual("I3")
	bob.AddNode(gedcom.NewNameNode("Bob /Smith/"))
	bob.AddNode(gedcom.NewSexNode("M"))
	bob.AddNode(gedcom.NewBirthNode("",
		gedcom.NewDateNode("10 Jan 1980"),
		gedcom.NewPlaceNode("Birmingham, England"),
	))

	// Create family
	family := doc.AddFamilyWithHusbandAndWife("F1", john, jane)
	family.AddChild(bob)

	// Add marriage event
	marriage := gedcom.NewNode(gedcom.TagMarriage, "", "",
		gedcom.NewDateNode("5 Jun 1975"),
		gedcom.NewPlaceNode("London, England"),
	)
	family.AddNode(marriage)

	// Add a note
	note := gedcom.NewNode(gedcom.TagNote, "Family immigrated from Ireland in 1920s", "N1")
	doc.AddNode(note)
	john.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))

	// Add media
	media := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "family_photo_1975.jpg", ""),
		gedcom.NewNode(gedcom.TagTitle, "Wedding Photo", ""),
	)
	doc.AddNode(media)
	family.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))

	// Export to SurrealDB v2 format
	var buf bytes.Buffer
	exporter := v2.NewExporter(&buf, "genealogy", "smith_family")

	fmt.Println("Exporting GEDCOM to SurrealDB v2 format...")
	fmt.Println("===========================================")
	fmt.Println()

	err := exporter.Export(doc)
	if err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	// Print the output
	output := buf.String()
	fmt.Println(output)

	// Statistics
	fmt.Println("\n===========================================")
	fmt.Println("Export Statistics:")
	fmt.Println("===========================================")
	fmt.Printf("Individuals: %d\n", len(doc.Individuals()))
	fmt.Printf("Families: %d\n", len(doc.Families()))
	fmt.Printf("Total output size: %d bytes\n", len(output))
	fmt.Println("\nTo import into SurrealDB:")
	fmt.Println("  surreal import -e http://localhost:8000 \\")
	fmt.Println("    --user root --pass root \\")
	fmt.Println("    --ns genealogy --db smith_family \\")
	fmt.Println("    output.surql")
}

