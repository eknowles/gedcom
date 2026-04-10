package v1

import (
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

// Test extracting person entities
func TestGraphExtractor_ExtractPersonEntities(t *testing.T) {
	tests := []struct {
		name               string
		setupDoc           func() *gedcom.Document
		expectedCount      int
		expectedProperties map[string]interface{}
	}{
		{
			name: "Extract person with name and sex",
			setupDoc: func() *gedcom.Document {
				doc := gedcom.NewDocument()
				person := doc.AddIndividual("I1")
				person.AddNode(gedcom.NewNameNode("John /Smith/"))
				person.AddNode(gedcom.NewSexNode("M"))
				return doc
			},
			expectedCount: 1,
			expectedProperties: map[string]interface{}{
				"pointer":    "I1",
				"name":       "John Smith",
				"given_name": "John",
				"surname":    "Smith",
				"sex":        "Male",
			},
		},
		{
			name: "Extract person with inline birth event",
			setupDoc: func() *gedcom.Document {
				doc := gedcom.NewDocument()
				person := doc.AddIndividual("I1")
				person.AddNode(gedcom.NewNameNode("Jane /Doe/"))
				person.AddNode(gedcom.NewBirthNode("",
					gedcom.NewDateNode("1 Jan 1900"),
					gedcom.NewPlaceNode("London, England"),
				))
				return doc
			},
			expectedCount: 1,
			expectedProperties: map[string]interface{}{
				"pointer":    "I1",
				"name":       "Jane Doe",
				"given_name": "Jane",
				"surname":    "Doe",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.setupDoc()
			extractor := NewGraphExtractor(doc)

			entities := extractor.extractPersonEntities()

			assert.Len(t, entities, tt.expectedCount)
			if len(entities) > 0 {
				entity := entities[0]
				assert.Equal(t, "person", entity.Type)

				for key, expectedValue := range tt.expectedProperties {
					assert.Equal(t, expectedValue, entity.Properties[key],
						"Mismatch for property: %s", key)
				}
			}
		})
	}
}

// Test extracting family entities
func TestGraphExtractor_ExtractFamilyEntities(t *testing.T) {
	doc := gedcom.NewDocument()

	husband := doc.AddIndividual("I1")
	husband.AddNode(gedcom.NewNameNode("John /Smith/"))

	wife := doc.AddIndividual("I2")
	wife.AddNode(gedcom.NewNameNode("Jane /Doe/"))

	doc.AddFamilyWithHusbandAndWife("F1", husband, wife)

	extractor := NewGraphExtractor(doc)
	entities := extractor.extractFamilyEntities()

	assert.Len(t, entities, 1)
	assert.Equal(t, "family", entities[0].Type)
	assert.Equal(t, "f1", entities[0].ID)
	assert.Equal(t, "F1", entities[0].Properties["pointer"])
	assert.Equal(t, "person:i1", entities[0].Properties["husband_id"])
	assert.Equal(t, "person:i2", entities[0].Properties["wife_id"])
}

// Test extracting note entities
func TestGraphExtractor_ExtractNoteEntities(t *testing.T) {
	doc := gedcom.NewDocument()

	// Note with pointer
	note := gedcom.NewNode(gedcom.TagNote, "This is a note about the family.", "N1",
		gedcom.NewNode(gedcom.TagContinued, "This is a continuation.", ""),
		gedcom.NewNode(gedcom.TagConcatenation, " Extra text.", ""),
	)
	doc.AddNode(note)

	extractor := NewGraphExtractor(doc)
	entities := extractor.extractNoteEntities()

	assert.Len(t, entities, 1)
	assert.Equal(t, "note", entities[0].Type)
	assert.Equal(t, "n1", entities[0].ID)
	assert.Equal(t, "N1", entities[0].Properties["pointer"])
	assert.Contains(t, entities[0].Properties["text"], "This is a note")
	assert.Contains(t, entities[0].Properties["text"], "This is a continuation")
	assert.Contains(t, entities[0].Properties["text"], "Extra text")
}

// Test extracting media object entities
func TestGraphExtractor_ExtractMediaObjectEntities(t *testing.T) {
	doc := gedcom.NewDocument()

	media := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "photo1.jpg", ""),
		gedcom.NewNode(gedcom.TagFile, "photo2.png", ""),
		gedcom.NewNode(gedcom.TagFormat, "jpeg", ""),
		gedcom.NewNode(gedcom.TagTitle, "Family Photo", ""),
	)
	doc.AddNode(media)

	extractor := NewGraphExtractor(doc)
	entities := extractor.extractMediaObjectEntities()

	assert.Len(t, entities, 1)
	assert.Equal(t, "media_object", entities[0].Type)
	assert.Equal(t, "m1", entities[0].ID)
	assert.Equal(t, "M1", entities[0].Properties["pointer"])
	assert.Equal(t, "jpeg", entities[0].Properties["format"])
	assert.Equal(t, "Family Photo", entities[0].Properties["title"])

	files := entities[0].Properties["files"].([]string)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "photo1.jpg")
	assert.Contains(t, files, "photo2.png")
}

// Test extracting spouse relationships
func TestGraphExtractor_ExtractSpouseRelationships(t *testing.T) {
	doc := gedcom.NewDocument()

	husband := doc.AddIndividual("I1")
	wife := doc.AddIndividual("I2")
	doc.AddFamilyWithHusbandAndWife("F1", husband, wife)

	extractor := NewGraphExtractor(doc)
	relationships := extractor.extractFamilyRelationships()

	// Should have 2 spouse relationships (bidirectional)
	spouseRels := filterRelationshipsByType(relationships, "spouse_of")
	assert.Len(t, spouseRels, 2)

	// Check husband -> wife
	husbandToWife := findRelationship(spouseRels, "i1", "i2")
	assert.NotNil(t, husbandToWife)
	assert.Equal(t, "spouse_of", husbandToWife.Type)
	assert.Equal(t, "family:f1", husbandToWife.Properties["family_id"])

	// Check wife -> husband
	wifeToHusband := findRelationship(spouseRels, "i2", "i1")
	assert.NotNil(t, wifeToHusband)
	assert.Equal(t, "spouse_of", wifeToHusband.Type)
}

// Test extracting parent-child relationships
func TestGraphExtractor_ExtractParentChildRelationships(t *testing.T) {
	doc := gedcom.NewDocument()

	father := doc.AddIndividual("I1")
	mother := doc.AddIndividual("I2")
	child := doc.AddIndividual("I3")

	family := doc.AddFamilyWithHusbandAndWife("F1", father, mother)
	family.AddChild(child)

	extractor := NewGraphExtractor(doc)
	relationships := extractor.extractFamilyRelationships()

	// Should have parent_of and child_of relationships
	parentOfRels := filterRelationshipsByType(relationships, "parent_of")
	childOfRels := filterRelationshipsByType(relationships, "child_of")

	// 2 parent_of (father -> child, mother -> child)
	assert.Len(t, parentOfRels, 2)

	// 2 child_of (child -> father, child -> mother)
	assert.Len(t, childOfRels, 2)

	// Verify father -> child
	fatherToChild := findRelationship(parentOfRels, "i1", "i3")
	assert.NotNil(t, fatherToChild)
	assert.Equal(t, "family:f1", fatherToChild.Properties["family_id"])

	// Verify child -> father
	childToFather := findRelationship(childOfRels, "i3", "i1")
	assert.NotNil(t, childToFather)
}

// Test extracting note relationships
func TestGraphExtractor_ExtractNoteRelationships(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create a note
	note := gedcom.NewNode(gedcom.TagNote, "Family history note", "N1")
	doc.AddNode(note)

	// Create a person with reference to the note
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))

	extractor := NewGraphExtractor(doc)
	relationships := extractor.extractNoteRelationships()

	hasNoteRels := filterRelationshipsByType(relationships, "has_note")
	assert.Len(t, hasNoteRels, 1)

	rel := hasNoteRels[0]
	assert.Equal(t, "i1", rel.FromID)
	assert.Equal(t, "n1", rel.ToID)
}

// Test note relationship with event context
func TestGraphExtractor_ExtractNoteRelationshipsWithEventContext(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create a note
	note := gedcom.NewNode(gedcom.TagNote, "Birth note", "N1")
	doc.AddNode(note)

	// Create a person with birth event that references the note
	person := doc.AddIndividual("I1")
	birth := gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1900"),
	)
	birth.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))
	person.AddNode(birth)

	extractor := NewGraphExtractor(doc)
	relationships := extractor.extractNoteRelationships()

	hasNoteRels := filterRelationshipsByType(relationships, "has_note")
	assert.Len(t, hasNoteRels, 1)

	rel := hasNoteRels[0]
	assert.Equal(t, "i1", rel.FromID)
	assert.Equal(t, "n1", rel.ToID)
	assert.Equal(t, "Birth", rel.Properties["context"])
}

// Test extracting media relationships
func TestGraphExtractor_ExtractMediaRelationships(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create a media object
	media := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "photo.jpg", ""),
	)
	doc.AddNode(media)

	// Create a person with reference to the media
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))

	extractor := NewGraphExtractor(doc)
	relationships := extractor.extractMediaRelationships()

	hasMediaRels := filterRelationshipsByType(relationships, "has_media")
	assert.Len(t, hasMediaRels, 1)

	rel := hasMediaRels[0]
	assert.Equal(t, "i1", rel.FromID)
	assert.Equal(t, "m1", rel.ToID)
}

// Test inline event properties extraction
func TestGraphExtractor_ExtractEventProperties(t *testing.T) {
	doc := gedcom.NewDocument()
	extractor := NewGraphExtractor(doc)

	birth := gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1900"),
		gedcom.NewPlaceNode("London, England"),
	)

	props := extractor.extractEventProperties(birth)

	assert.Equal(t, "1 Jan 1900", props["date"])
	assert.Equal(t, "London, England", props["place"])
	assert.NotNil(t, props["date_range_start"])
	assert.NotNil(t, props["date_range_end"])
}

// Test complex event detection
func TestGraphExtractor_IsComplexEvent(t *testing.T) {
	doc := gedcom.NewDocument()
	extractor := NewGraphExtractor(doc)

	tests := []struct {
		name        string
		setupEvent  func() gedcom.Node
		isComplex   bool
	}{
		{
			name: "Simple birth event",
			setupEvent: func() gedcom.Node {
				return gedcom.NewBirthNode("",
					gedcom.NewDateNode("1 Jan 1900"),
					gedcom.NewPlaceNode("London"),
				)
			},
			isComplex: false,
		},
		{
			name: "Birth event with note",
			setupEvent: func() gedcom.Node {
				birth := gedcom.NewBirthNode("",
					gedcom.NewDateNode("1 Jan 1900"),
				)
				birth.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))
				return birth
			},
			isComplex: true,
		},
		{
			name: "Birth event with source",
			setupEvent: func() gedcom.Node {
				birth := gedcom.NewBirthNode("",
					gedcom.NewDateNode("1 Jan 1900"),
				)
				birth.AddNode(gedcom.NewNode(gedcom.TagSource, "@S1@", ""))
				return birth
			},
			isComplex: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tt.setupEvent()
			result := extractor.isComplexEvent(event)
			assert.Equal(t, tt.isComplex, result)
		})
	}
}

// Test complete extraction
func TestGraphExtractor_ExtractAll(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create people
	father := doc.AddIndividual("I1")
	father.AddNode(gedcom.NewNameNode("John /Smith/"))
	father.AddNode(gedcom.NewSexNode("M"))

	mother := doc.AddIndividual("I2")
	mother.AddNode(gedcom.NewNameNode("Jane /Doe/"))
	mother.AddNode(gedcom.NewSexNode("F"))

	child := doc.AddIndividual("I3")
	child.AddNode(gedcom.NewNameNode("Bob /Smith/"))

	// Create family
	family := doc.AddFamilyWithHusbandAndWife("F1", father, mother)
	family.AddChild(child)

	// Create note
	note := gedcom.NewNode(gedcom.TagNote, "Family note", "N1")
	doc.AddNode(note)
	father.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))

	extractor := NewGraphExtractor(doc)
	entities, relationships, err := extractor.ExtractAll()

	assert.NoError(t, err)

	// Check entities
	personEntities := filterEntitiesByType(entities, "person")
	familyEntities := filterEntitiesByType(entities, "family")
	noteEntities := filterEntitiesByType(entities, "note")

	assert.Len(t, personEntities, 3, "Should have 3 people")
	assert.Len(t, familyEntities, 1, "Should have 1 family")
	assert.Len(t, noteEntities, 1, "Should have 1 note")

	// Check relationships
	spouseRels := filterRelationshipsByType(relationships, "spouse_of")
	parentRels := filterRelationshipsByType(relationships, "parent_of")
	childRels := filterRelationshipsByType(relationships, "child_of")
	noteRels := filterRelationshipsByType(relationships, "has_note")

	assert.Len(t, spouseRels, 2, "Should have 2 spouse relationships (bidirectional)")
	assert.Len(t, parentRels, 2, "Should have 2 parent_of relationships")
	assert.Len(t, childRels, 2, "Should have 2 child_of relationships")
	assert.Len(t, noteRels, 1, "Should have 1 has_note relationship")
}

// Helper functions for tests

func filterRelationshipsByType(rels []GraphRelationship, relType string) []GraphRelationship {
	var filtered []GraphRelationship
	for _, rel := range rels {
		if rel.Type == relType {
			filtered = append(filtered, rel)
		}
	}
	return filtered
}

func filterEntitiesByType(entities []GraphEntity, entityType string) []GraphEntity {
	var filtered []GraphEntity
	for _, entity := range entities {
		if entity.Type == entityType {
			filtered = append(filtered, entity)
		}
	}
	return filtered
}

func findRelationship(rels []GraphRelationship, fromID, toID string) *GraphRelationship {
	for _, rel := range rels {
		if rel.FromID == fromID && rel.ToID == toID {
			return &rel
		}
	}
	return nil
}

