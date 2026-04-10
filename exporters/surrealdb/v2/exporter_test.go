package v2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

func TestExporter_Export(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create a simple person
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNameNode("John /Smith/"))
	person.AddNode(gedcom.NewSexNode("M"))
	person.AddNode(gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1900"),
		gedcom.NewPlaceNode("London"),
	))

	// Export
	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "gedcom")
	err := exporter.Export(doc)

	assert.NoError(t, err)

	output := buf.String()

	// Verify header
	assert.Contains(t, output, "-- SurrealDB Export for GEDCOM (v2)")
	assert.Contains(t, output, "USE NS test DB gedcom")

	// Verify schema
	assert.Contains(t, output, "DEFINE TABLE person SCHEMAFULL")
	assert.Contains(t, output, "DEFINE TABLE family SCHEMAFULL")

	// Verify person record
	assert.Contains(t, output, "CREATE person:i1")
	assert.Contains(t, output, `"pointer": "I1"`)
	// Verify name object structure
	assert.Contains(t, output, `"name":`)
	assert.Contains(t, output, `"full":"John Smith"`)
	assert.Contains(t, output, `"given":"John"`)
	assert.Contains(t, output, `"surname":"Smith"`)
	assert.Contains(t, output, `"sex": "M"`)

	// Verify birth object (nested inside birth)
	assert.Contains(t, output, `"birth":`)
	assert.Contains(t, output, `"date":"1 Jan 1900"`) // Note: no space after colon in JSON
	assert.Contains(t, output, `"place":"London"`)
}

func TestExporter_ExportFamily(t *testing.T) {
	doc := gedcom.NewDocument()

	husband := doc.AddIndividual("I1")
	husband.AddNode(gedcom.NewNameNode("John /Smith/"))

	wife := doc.AddIndividual("I2")
	wife.AddNode(gedcom.NewNameNode("Jane /Doe/"))

	child := doc.AddIndividual("I3")
	child.AddNode(gedcom.NewNameNode("Bob /Smith/"))

	family := doc.AddFamilyWithHusbandAndWife("F1", husband, wife)
	family.AddChild(child)

	// Export
	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "gedcom")
	err := exporter.Export(doc)

	assert.NoError(t, err)

	output := buf.String()

	// Verify family record
	assert.Contains(t, output, "CREATE family:f1")
	assert.Contains(t, output, `"husband_id": person:i1`)
	assert.Contains(t, output, `"wife_id": person:i2`)

	// Verify spouse relationships
	assert.Contains(t, output, "RELATE person:i1->spouse_of->person:i2")
	assert.Contains(t, output, "RELATE person:i2->spouse_of->person:i1")

	// Verify parent-child relationships
	assert.Contains(t, output, "RELATE person:i1->parent_of->person:i3")
	assert.Contains(t, output, "RELATE person:i2->parent_of->person:i3")
	assert.Contains(t, output, "RELATE person:i3->child_of->person:i1")
	assert.Contains(t, output, "RELATE person:i3->child_of->person:i2")
}

func TestExporter_ExportNoteRelationships(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create note
	note := gedcom.NewNode(gedcom.TagNote, "Family history", "N1")
	doc.AddNode(note)

	// Create person with note reference
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNameNode("John /Smith/"))
	person.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))

	// Export
	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "gedcom")
	err := exporter.Export(doc)

	assert.NoError(t, err)

	output := buf.String()

	// Verify note record
	assert.Contains(t, output, "CREATE note:n1")
	assert.Contains(t, output, `"text": "Family history"`)

	// Verify has_note relationship
	assert.Contains(t, output, "RELATE person:i1->has_note->note:n1")
}

func TestExporter_ExportMediaRelationships(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create media object
	media := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "photo.jpg", ""),
		gedcom.NewNode(gedcom.TagTitle, "Family Photo", ""),
	)
	doc.AddNode(media)

	// Create person with media reference
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNameNode("John /Smith/"))
	person.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))

	// Export
	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "gedcom")
	err := exporter.Export(doc)

	assert.NoError(t, err)

	output := buf.String()

	// Verify media record
	assert.Contains(t, output, "CREATE media_object:m1")
	assert.Contains(t, output, `"files": ["photo.jpg"]`)
	assert.Contains(t, output, `"title": "Family Photo"`)

	// Verify has_media relationship
	assert.Contains(t, output, "RELATE person:i1->has_media->media_object:m1")
}

func TestExporter_JSONFormat(t *testing.T) {
	doc := gedcom.NewDocument()
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNameNode("John /Smith/"))

	// Export as JSON
	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "gedcom")
	exporter.SetUseJSON(true)
	err := exporter.Export(doc)

	assert.NoError(t, err)

	output := buf.String()

	// Should not contain SQL header
	assert.NotContains(t, output, "DEFINE TABLE")

	// Should contain JSON
	assert.Contains(t, output, `"pointer"`)
	assert.Contains(t, output, `"name"`)
}

func TestExporter_CustomMapper(t *testing.T) {
	doc := gedcom.NewDocument()
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNameNode("John /Smith/"))

	// Create custom mapper with only person extractor
	customMapper := &GraphMapper{
		Entities: []EntityExtractor{
			PersonEntityExtractor(),
		},
		Relationships: []RelationshipExtractor{},
		EventStrategy: DefaultEventStrategy(),
	}

	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "gedcom")
	exporter.SetMapper(customMapper)
	err := exporter.Export(doc)

	assert.NoError(t, err)

	output := buf.String()

	// Should have person
	assert.Contains(t, output, "CREATE person:i1")

	// Should not have any relationships (since we didn't include them)
	assert.NotContains(t, output, "RELATE")
}

func TestCleanProperties(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected int // expected number of properties after cleaning
	}{
		{
			name: "Remove nil values",
			input: map[string]interface{}{
				"name": "John",
				"age":  nil,
			},
			expected: 1,
		},
		{
			name: "Remove empty strings",
			input: map[string]interface{}{
				"name":  "John",
				"email": "",
			},
			expected: 1,
		},
		{
			name: "Remove empty maps",
			input: map[string]interface{}{
				"name":  "John",
				"birth": map[string]interface{}{},
			},
			expected: 1,
		},
		{
			name: "Keep valid values",
			input: map[string]interface{}{
				"name": "John",
				"age":  30,
				"birth": map[string]interface{}{
					"date": "1 Jan 1900",
				},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanProperties(tt.input)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestFormatContent(t *testing.T) {
	props := map[string]interface{}{
		"name": "John Smith",
		"age":  30,
		"birth": map[string]interface{}{
			"date": "1 Jan 1900",
		},
	}

	result := formatContent(props)

	// Should be valid JSON-like object
	assert.Contains(t, result, `"name": "John Smith"`)
	assert.Contains(t, result, `"age": 30`)
	assert.Contains(t, result, `"birth":`)
	assert.True(t, strings.HasPrefix(result, "{"))
	assert.True(t, strings.HasSuffix(result, "}"))
}

func TestExtractTableFromID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"person:i1", "person"},
		{"family:f1", "family"},
		{"note:n1", "note"},
		{"i1", "person"},
		{"f1", "family"},
		{"n1", "note"},
		{"m1", "media_object"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractTableFromID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractIDFromTableID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"person:i1", "i1"},
		{"family:f1", "f1"},
		{"i1", "i1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractIDFromTableID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}



