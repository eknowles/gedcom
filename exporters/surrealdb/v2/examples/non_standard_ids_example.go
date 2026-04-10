//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/eknowles/gedcom/v39"
	v2 "github.com/eknowles/gedcom/v39/exporters/surrealdb/v2"
)

// Example demonstrating the fix for notes and media with non-standard IDs
// This reproduces the issue where notes/media with "I" prefixed pointers
// were incorrectly linked as person->person instead of person->note/media
func main() {
	doc := gedcom.NewDocument()

	// Create note with I-prefix (like @I4829149@ from the bug report)
	note := gedcom.NewNode(gedcom.TagNote, "Family history note", "I4829149")
	doc.AddNode(note)

	// Create media with I-prefix (like @I91157422@ from the bug report)
	media := gedcom.NewNode(gedcom.TagObject, "", "I91157422",
		gedcom.NewNode(gedcom.TagFile, "family_photo.jpg", ""),
	)
	doc.AddNode(media)

	// Create person that references both
	person := doc.AddIndividual("I90588064",
		gedcom.NewNameNode("John /Doe/"),
	)
	person.AddNode(gedcom.NewNode(gedcom.TagNote, "@I4829149@", ""))
	person.AddNode(gedcom.NewNode(gedcom.TagObject, "@I91157422@", ""))

	// Another person with I-prefix
	person2 := doc.AddIndividual("I66679759",
		gedcom.NewNameNode("Jane /Smith/"),
	)
	person2.AddNode(gedcom.NewNode(gedcom.TagObject, "@I91157422@", ""))

	// Export to SurrealDB
	exporter := v2.NewExporter(os.Stdout, "example", "example")
	if err := exporter.Export(doc); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "\n\n✅ FIXED: The output should now show:")
	fmt.Fprintln(os.Stderr, "  - RELATE person:i90588064->has_note->note:i4829149")
	fmt.Fprintln(os.Stderr, "  - RELATE person:i66679759->has_media->media_object:i91157422")
	fmt.Fprintln(os.Stderr, "\n❌ NOT (the previous bug):")
	fmt.Fprintln(os.Stderr, "  - RELATE person:i90588064->has_note->person:i4829149")
	fmt.Fprintln(os.Stderr, "  - RELATE person:i66679759->has_media->person:i91157422")
}

