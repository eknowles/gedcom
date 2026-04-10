//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/eknowles/gedcom/v39"
	v2 "github.com/eknowles/gedcom/v39/exporters/surrealdb/v2"
)

// Example demonstrating that media objects are correctly connected
// in the SurrealDB export, including those attached to events
func main() {
	doc := gedcom.NewDocument()

	// Create a media object
	photo := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "photo.jpg", ""),
		gedcom.NewNode(gedcom.TagTitle, "Birth photo", ""),
	)
	doc.AddNode(photo)

	// Create a person with a birth event that has the media attached
	person := doc.AddIndividual("I1",
		gedcom.NewNameNode("John /Doe/"),
	)

	birth := gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1900"),
	)
	// Attach media to birth event
	birth.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))
	person.AddNode(birth)

	// Export to SurrealDB
	exporter := v2.NewExporter(os.Stdout, "example", "example")
	if err := exporter.Export(doc); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

