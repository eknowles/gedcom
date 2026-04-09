package surrealdb

import (
	"bytes"
	"strings"
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeID_NumericIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Pure numeric ID gets i prefix",
			input:    "66679759",
			expected: "i66679759",
		},
		{
			name:     "ID starting with letter stays as is (lowercased)",
			input:    "I1",
			expected: "i1",
		},
		{
			name:     "ID with @ symbols removed",
			input:    "@I123@",
			expected: "i123",
		},
		{
			name:     "Numeric ID with @ symbols",
			input:    "@66679759@",
			expected: "i66679759",
		},
		{
			name:     "ID with spaces replaced",
			input:    "ID 123",
			expected: "id_123",
		},
		{
			name:     "ID with dashes replaced",
			input:    "ID-123",
			expected: "id_123",
		},
		{
			name:     "Numeric ID with special chars",
			input:    "@123-456@",
			expected: "i123_456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple document with one individual
			doc := gedcom.NewDocument()
			individual := doc.AddIndividual(tt.input)
			individual.AddName("Test /Name/")

			// Export to SurrealDB
			var buf bytes.Buffer
			exporter := NewExporter(&buf, "test", "test")
			err := exporter.Export(doc)
			assert.NoError(t, err)

			// Check that the output contains the expected ID format
			output := buf.String()
			expectedIDFormat := "person:" + tt.expected
			assert.Contains(t, output, expectedIDFormat, "Output should contain %s", expectedIDFormat)
		})
	}
}

func TestSurrealDBExport_IDFormat(t *testing.T) {
	doc := gedcom.NewDocument()

	// Test with traditional GEDCOM ID (I1)
	person1 := doc.AddIndividual("I1")
	person1.AddName("John /Smith/")

	// Test with numeric-only ID (like from some GEDCOM files)
	person2 := doc.AddIndividual("66679759")
	person2.AddName("Jane /Doe/")

	// Export to SurrealDB
	var buf bytes.Buffer
	exporter := NewExporter(&buf, "gedcom", "family_tree")
	err := exporter.Export(doc)
	assert.NoError(t, err)

	output := buf.String()

	// Check that I1 becomes person:i1 (letter-prefixed stays as is, just lowercased)
	assert.Contains(t, output, "person:i1", "Traditional GEDCOM ID I1 should become person:i1")

	// Check that numeric ID gets prefixed with 'i'
	assert.Contains(t, output, "person:i66679759", "Numeric ID should get 'i' prefix")

	// Check that IDs don't have double prefix
	assert.NotContains(t, output, "person:ii1", "Should not have double prefix")
}

func TestSurrealDBExport_FamilyIDFormat(t *testing.T) {
	doc := gedcom.NewDocument()

	husband := doc.AddIndividual("I1")
	husband.AddName("John /Smith/")

	wife := doc.AddIndividual("I2")
	wife.AddName("Jane /Smith/")

	// Test with numeric family ID
	doc.AddFamilyWithHusbandAndWife("123456", husband, wife)

	var buf bytes.Buffer
	exporter := NewExporter(&buf, "gedcom", "family_tree")
	err := exporter.Export(doc)
	assert.NoError(t, err)

	output := buf.String()

	// Check family ID gets prefixed if numeric
	assert.Contains(t, output, "family:i123456", "Numeric family ID should get 'i' prefix")

	// Check relationships reference correct person IDs
	assert.Contains(t, output, "person:i1", "Should reference person:i1")
	assert.Contains(t, output, "person:i2", "Should reference person:i2")
}

func TestSurrealDBExport_RelationshipsWithNumericIDs(t *testing.T) {
	doc := gedcom.NewDocument()

	father := doc.AddIndividual("1001")
	father.AddName("Father /Smith/")

	mother := doc.AddIndividual("1002")
	mother.AddName("Mother /Smith/")

	child := doc.AddIndividual("1003")
	child.AddName("Child /Smith/")

	family := doc.AddFamilyWithHusbandAndWife("2001", father, mother)
	family.AddChild(child)

	var buf bytes.Buffer
	exporter := NewExporter(&buf, "gedcom", "family_tree")
	err := exporter.Export(doc)
	assert.NoError(t, err)

	output := buf.String()

	// Verify all IDs are properly prefixed
	assert.Contains(t, output, "person:i1001", "Father ID should be prefixed")
	assert.Contains(t, output, "person:i1002", "Mother ID should be prefixed")
	assert.Contains(t, output, "person:i1003", "Child ID should be prefixed")
	assert.Contains(t, output, "family:i2001", "Family ID should be prefixed")

	// Verify relationships use correct IDs
	assert.Contains(t, output, "RELATE person:i1001->spouse_of->person:i1002",
		"Spouse relationship should use prefixed IDs")
	assert.Contains(t, output, "RELATE person:i1001->parent_of->person:i1003",
		"Parent-child relationship should use prefixed IDs")
}

func TestSurrealDBExport_MixedIDFormats(t *testing.T) {
	doc := gedcom.NewDocument()

	// Mix of traditional and numeric IDs
	person1 := doc.AddIndividual("I1")
	person1.AddName("Traditional /ID/")

	person2 := doc.AddIndividual("999888")
	person2.AddName("Numeric /ID/")

	person3 := doc.AddIndividual("F42")
	person3.AddName("Letter-Numeric /Mix/")

	family := doc.AddFamilyWithHusbandAndWife("FAM1", person1, person2)
	family.AddChild(person3)

	var buf bytes.Buffer
	exporter := NewExporter(&buf, "gedcom", "family_tree")
	err := exporter.Export(doc)
	assert.NoError(t, err)

	output := buf.String()
	lines := strings.Split(output, "\n")

	// Verify each ID format
	foundI1 := false
	foundI999888 := false
	foundF42 := false
	foundFAM1 := false

	for _, line := range lines {
		if strings.Contains(line, "person:i1 ") || strings.Contains(line, "person:i1,") {
			foundI1 = true
		}
		if strings.Contains(line, "person:i999888") {
			foundI999888 = true
		}
		if strings.Contains(line, "person:f42") {
			foundF42 = true
		}
		if strings.Contains(line, "family:fam1") {
			foundFAM1 = true
		}
	}

	assert.True(t, foundI1, "Should find person:i1")
	assert.True(t, foundI999888, "Should find person:i999888 (with i prefix)")
	assert.True(t, foundF42, "Should find person:f42 (lowercase, no extra prefix)")
	assert.True(t, foundFAM1, "Should find family:fam1")
}

func TestSurrealDBExport_NotesAndMediaObjects(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create a person
	person := doc.AddIndividual("I1")
	person.AddName("John /Doe/")

	// Add a note with pointer to the document - create it as a full node
	note := gedcom.NewNode(gedcom.TagNote, "This is a test note about the family.", "N1",
		gedcom.NewNode(gedcom.TagFromString("CONT"), "This is a continuation line.", ""))
	doc.AddNode(note)

	// Add a media object (OBJE) to the document
	mediaObj := gedcom.NewNode(gedcom.TagObject, "", "O1",
		gedcom.NewNode(gedcom.TagFile, "family_photo.jpg", ""),
		gedcom.NewNode(gedcom.TagFormat, "jpeg", ""),
		gedcom.NewNode(gedcom.TagTitle, "Family Portrait", ""),
		gedcom.NewNode(gedcom.TagNote, "Photo taken in 1950", ""),
	)
	doc.AddNode(mediaObj)

	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "test_db")
	err := exporter.Export(doc)
	assert.NoError(t, err)

	output := buf.String()

	// Verify note table is defined
	assert.Contains(t, output, "DEFINE TABLE note SCHEMAFULL")

	// Verify media_object table is defined
	assert.Contains(t, output, "DEFINE TABLE media_object SCHEMAFULL")

	// Verify note record is created
	assert.Contains(t, output, "note:n1")
	assert.Contains(t, output, "This is a test note about the family")
	assert.Contains(t, output, "This is a continuation line")

	// Verify media object record is created
	assert.Contains(t, output, "media_object:o1")
	assert.Contains(t, output, "family_photo.jpg")
	assert.Contains(t, output, "jpeg")
	assert.Contains(t, output, "Family Portrait")
}

func TestSurrealDBExport_MultipleMediaFiles(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create a media object with multiple files
	mediaObj := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "photo1.jpg", ""),
		gedcom.NewNode(gedcom.TagFile, "photo2.png", ""),
		gedcom.NewNode(gedcom.TagFormat, "jpeg", ""),
		gedcom.NewNode(gedcom.TagTitle, "Multiple Photos", ""),
	)
	doc.AddNode(mediaObj)

	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "test_db")
	err := exporter.Export(doc)
	assert.NoError(t, err)

	output := buf.String()

	// Verify both files are in the array
	assert.Contains(t, output, "photo1.jpg")
	assert.Contains(t, output, "photo2.png")
	assert.Contains(t, output, "media_object:m1")
}
