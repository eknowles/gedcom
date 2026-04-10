package v2

import (
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

func TestPersonEntityExtractor(t *testing.T) {
	doc := gedcom.NewDocument()
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNameNode("John /Smith/"))
	person.AddNode(gedcom.NewSexNode("M"))
	person.AddNode(gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1900"),
		gedcom.NewPlaceNode("London"),
	))

	extractor := PersonEntityExtractor()
	nodes := extractor.Extract(doc)

	assert.Len(t, nodes, 1)
	node := nodes[0]

	assert.Equal(t, EntityPerson, node.Type)
	assert.Equal(t, "i1", node.ID)
	assert.Equal(t, "I1", node.Properties["pointer"])

	// Check name object
	name := node.Properties["name"].(map[string]interface{})
	assert.NotNil(t, name)
	assert.Equal(t, "John", name["given"])
	assert.Equal(t, "Smith", name["surname"])
	assert.Equal(t, "John Smith", name["full"])

	assert.Equal(t, "M", node.Properties["sex"])
	assert.Equal(t, false, node.Properties["is_living"])

	// Check birth event object
	birth := node.Properties["birth"].(map[string]interface{})
	assert.NotNil(t, birth)
	assert.Equal(t, "1 Jan 1900", birth["date"])
	assert.Equal(t, "London", birth["place"])
}

func TestFamilyEntityExtractor(t *testing.T) {
	doc := gedcom.NewDocument()
	husband := doc.AddIndividual("I1")
	wife := doc.AddIndividual("I2")
	doc.AddFamilyWithHusbandAndWife("F1", husband, wife)

	extractor := FamilyEntityExtractor()
	nodes := extractor.Extract(doc)

	assert.Len(t, nodes, 1)
	node := nodes[0]

	assert.Equal(t, EntityFamily, node.Type)
	assert.Equal(t, "f1", node.ID)
	assert.Equal(t, "F1", node.Properties["pointer"])
	assert.Equal(t, "person:i1", node.Properties["husband_id"])
	assert.Equal(t, "person:i2", node.Properties["wife_id"])
}

func TestNoteEntityExtractor(t *testing.T) {
	tests := []struct {
		name          string
		setupDoc      func() *gedcom.Document
		expectedCount int
	}{
		{
			name: "Extract note with pointer",
			setupDoc: func() *gedcom.Document {
				doc := gedcom.NewDocument()
				note := gedcom.NewNode(gedcom.TagNote, "This is a note", "N1",
					gedcom.NewNode(gedcom.TagContinued, "Continued text", ""),
				)
				doc.AddNode(note)
				return doc
			},
			expectedCount: 1,
		},
		{
			name: "Skip inline note without pointer",
			setupDoc: func() *gedcom.Document {
				doc := gedcom.NewDocument()
				person := doc.AddIndividual("I1")
				person.AddNode(gedcom.NewNode(gedcom.TagNote, "Inline note", ""))
				return doc
			},
			expectedCount: 0, // Should be skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.setupDoc()
			extractor := NoteEntityExtractor()
			nodes := extractor.Extract(doc)

			assert.Len(t, nodes, tt.expectedCount)

			if tt.expectedCount > 0 {
				node := nodes[0]
				assert.Equal(t, EntityNote, node.Type)
				assert.Contains(t, node.Properties["text"], "note")
			}
		})
	}
}

func TestMediaObjectEntityExtractor(t *testing.T) {
	doc := gedcom.NewDocument()
	media := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "photo1.jpg", ""),
		gedcom.NewNode(gedcom.TagFile, "photo2.jpg", ""),
		gedcom.NewNode(gedcom.TagFormat, "jpeg", ""),
		gedcom.NewNode(gedcom.TagTitle, "Family Photo", ""),
	)
	doc.AddNode(media)

	extractor := MediaObjectEntityExtractor()
	nodes := extractor.Extract(doc)

	assert.Len(t, nodes, 1)
	node := nodes[0]

	assert.Equal(t, EntityMediaObject, node.Type)
	assert.Equal(t, "m1", node.ID)
	assert.Equal(t, "M1", node.Properties["pointer"])
	assert.Equal(t, "jpeg", node.Properties["format"])
	assert.Equal(t, "Family Photo", node.Properties["title"])

	files := node.Properties["files"].([]string)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "photo1.jpg")
	assert.Contains(t, files, "photo2.jpg")
}

func TestEventStrategy_ShouldCreateNode(t *testing.T) {
	tests := []struct {
		name           string
		setupEvent     func() gedcom.Node
		strategy       EventStrategy
		expectedCreate bool
	}{
		{
			name: "Simple birth - inline",
			setupEvent: func() gedcom.Node {
				return gedcom.NewBirthNode("",
					gedcom.NewDateNode("1 Jan 1900"),
					gedcom.NewPlaceNode("London"),
				)
			},
			strategy:       DefaultEventStrategy(),
			expectedCreate: false,
		},
		{
			name: "Birth with note - create node",
			setupEvent: func() gedcom.Node {
				birth := gedcom.NewBirthNode("",
					gedcom.NewDateNode("1 Jan 1900"),
				)
				birth.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))
				return birth
			},
			strategy:       DefaultEventStrategy(),
			expectedCreate: true,
		},
		{
			name: "Birth with source - create node",
			setupEvent: func() gedcom.Node {
				birth := gedcom.NewBirthNode("",
					gedcom.NewDateNode("1 Jan 1900"),
				)
				birth.AddNode(gedcom.NewNode(gedcom.TagSource, "@S1@", ""))
				return birth
			},
			strategy:       DefaultEventStrategy(),
			expectedCreate: true,
		},
		{
			name: "Always create nodes strategy",
			setupEvent: func() gedcom.Node {
				return gedcom.NewBirthNode("",
					gedcom.NewDateNode("1 Jan 1900"),
				)
			},
			strategy: EventStrategy{
				InlineIfSimple: false,
				ComplexityTags: []gedcom.Tag{},
			},
			expectedCreate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tt.setupEvent()
			result := tt.strategy.ShouldCreateNode(event)
			assert.Equal(t, tt.expectedCreate, result)
		})
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@I1@", "i1"},
		{"@I123@", "i123"},
		{"@F1@", "f1"},
		{"@N1@", "n1"},
		{"@M-1@", "m_1"},
		{"@P 1@", "p_1"},
		{"@123@", "i123"}, // Starts with digit, prefix with 'i'
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
