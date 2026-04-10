package v2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

// TestEntityTypesWithNonStandardIDs tests that entity types are correctly
// identified even when IDs don't follow the standard naming convention
// (e.g., notes with @I...@ pointers instead of @N...@)
func TestEntityTypesWithNonStandardIDs(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create note with "I" prefix (non-standard but valid)
	note := gedcom.NewNode(gedcom.TagNote, "This is a note", "I4829149")
	doc.AddNode(note)

	// Create media with "I" prefix (non-standard but valid)
	media := gedcom.NewNode(gedcom.TagObject, "", "I91157422",
		gedcom.NewNode(gedcom.TagFile, "photo.jpg", ""),
	)
	doc.AddNode(media)

	// Create person with references to the note and media
	person := doc.AddIndividual("I90588064",
		gedcom.NewNameNode("John /Doe/"),
	)
	person.AddNode(gedcom.NewNode(gedcom.TagNote, "@I4829149@", ""))
	person.AddNode(gedcom.NewNode(gedcom.TagObject, "@I91157422@", ""))

	// Export
	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "test")
	err := exporter.Export(doc)

	assert.NoError(t, err)
	output := buf.String()

	// Verify note entity is created with correct type
	assert.Contains(t, output, "CREATE note:i4829149",
		"Note should be created as 'note' type even with 'I' prefix")

	// Verify media entity is created with correct type
	assert.Contains(t, output, "CREATE media_object:i91157422",
		"Media object should be created as 'media_object' type even with 'I' prefix")

	// Verify note relationship uses correct entity type
	assert.Contains(t, output, "RELATE person:i90588064->has_note->note:i4829149",
		"Note relationship should point to 'note' table, not 'person' table")

	// Verify media relationship uses correct entity type
	assert.Contains(t, output, "RELATE person:i90588064->has_media->media_object:i91157422",
		"Media relationship should point to 'media_object' table, not 'person' table")

	// Verify it does NOT create incorrect person relationships
	assert.NotContains(t, output, "RELATE person:i90588064->has_note->person:i4829149",
		"Should NOT create note relationship to 'person' table")
	assert.NotContains(t, output, "RELATE person:i90588064->has_media->person:i91157422",
		"Should NOT create media relationship to 'person' table")
}

// TestEntityTypesInExport verifies that all relationship types use correct entity tables
func TestEntityTypesInExport(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create entities with standard IDs
	noteStd := gedcom.NewNode(gedcom.TagNote, "Standard note", "N1")
	doc.AddNode(noteStd)

	mediaStd := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "file.jpg", ""),
	)
	doc.AddNode(mediaStd)

	// Create entities with non-standard IDs
	noteNonStd := gedcom.NewNode(gedcom.TagNote, "Non-standard note", "I999")
	doc.AddNode(noteNonStd)

	mediaNonStd := gedcom.NewNode(gedcom.TagObject, "", "I888",
		gedcom.NewNode(gedcom.TagFile, "file2.jpg", ""),
	)
	doc.AddNode(mediaNonStd)

	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))
	person.AddNode(gedcom.NewNode(gedcom.TagNote, "@I999@", ""))
	person.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))
	person.AddNode(gedcom.NewNode(gedcom.TagObject, "@I888@", ""))

	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "test")
	err := exporter.Export(doc)

	assert.NoError(t, err)
	output := buf.String()

	// Count relationships - should have exactly 4
	hasNoteCount := strings.Count(output, "->has_note->note:")
	hasMediaCount := strings.Count(output, "->has_media->media_object:")

	assert.Equal(t, 2, hasNoteCount, "Should have 2 has_note relationships")
	assert.Equal(t, 2, hasMediaCount, "Should have 2 has_media relationships")

	// Ensure NO incorrect relationships to person table
	incorrectNoteRel := strings.Count(output, "->has_note->person:")
	incorrectMediaRel := strings.Count(output, "->has_media->person:")

	assert.Equal(t, 0, incorrectNoteRel, "Should have NO has_note relationships to person table")
	assert.Equal(t, 0, incorrectMediaRel, "Should have NO has_media relationships to person table")
}
