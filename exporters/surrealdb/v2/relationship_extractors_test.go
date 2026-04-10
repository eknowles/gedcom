package v2

import (
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

func TestParentOfRelationship(t *testing.T) {
	doc := gedcom.NewDocument()

	father := doc.AddIndividual("I1")
	mother := doc.AddIndividual("I2")
	child := doc.AddIndividual("I3")

	family := doc.AddFamilyWithHusbandAndWife("F1", father, mother)
	family.AddChild(child)

	extractor := ParentOfRelationship()
	edges := extractor.Extract(doc)

	// Should have 2 parent_of edges (father -> child, mother -> child)
	assert.Len(t, edges, 2)

	// Find father -> child edge
	var fatherToChild *GraphEdge
	for i := range edges {
		if edges[i].FromID == "i1" && edges[i].ToID == "i3" {
			fatherToChild = &edges[i]
			break
		}
	}

	assert.NotNil(t, fatherToChild)
	assert.Equal(t, "parent_of", fatherToChild.Type)
}

func TestChildOfRelationship(t *testing.T) {
	doc := gedcom.NewDocument()

	father := doc.AddIndividual("I1")
	mother := doc.AddIndividual("I2")
	child := doc.AddIndividual("I3")

	family := doc.AddFamilyWithHusbandAndWife("F1", father, mother)
	family.AddChild(child)

	extractor := ChildOfRelationship()
	edges := extractor.Extract(doc)

	// Should have 2 child_of edges (child -> father, child -> mother)
	assert.Len(t, edges, 2)

	// Find child -> father edge
	var childToFather *GraphEdge
	for i := range edges {
		if edges[i].FromID == "i3" && edges[i].ToID == "i1" {
			childToFather = &edges[i]
			break
		}
	}

	assert.NotNil(t, childToFather)
	assert.Equal(t, "child_of", childToFather.Type)
}

func TestSpouseOfRelationship(t *testing.T) {
	doc := gedcom.NewDocument()

	husband := doc.AddIndividual("I1")
	wife := doc.AddIndividual("I2")
	doc.AddFamilyWithHusbandAndWife("F1", husband, wife)

	extractor := SpouseOfRelationship()
	edges := extractor.Extract(doc)

	// Should have 2 spouse_of edges (bidirectional)
	assert.Len(t, edges, 2)

	// Find husband -> wife edge
	var husbandToWife *GraphEdge
	var wifeToHusband *GraphEdge
	for i := range edges {
		if edges[i].FromID == "i1" && edges[i].ToID == "i2" {
			husbandToWife = &edges[i]
		}
		if edges[i].FromID == "i2" && edges[i].ToID == "i1" {
			wifeToHusband = &edges[i]
		}
	}

	assert.NotNil(t, husbandToWife)
	assert.NotNil(t, wifeToHusband)
	assert.Equal(t, "spouse_of", husbandToWife.Type)
	assert.Equal(t, "spouse_of", wifeToHusband.Type)
}

func TestHasNoteRelationship(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create a note with pointer
	note := gedcom.NewNode(gedcom.TagNote, "Family history", "N1")
	doc.AddNode(note)

	// Create person with reference to note
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))

	extractor := HasNoteRelationship()
	edges := extractor.Extract(doc)

	assert.Len(t, edges, 1)
	assert.Equal(t, "has_note", edges[0].Type)
	assert.Equal(t, "i1", edges[0].FromID)
	assert.Equal(t, "n1", edges[0].ToID)
}

func TestHasNoteRelationship_WithEventContext(t *testing.T) {
	doc := gedcom.NewDocument()
	
	// Create a note with pointer
	note := gedcom.NewNode(gedcom.TagNote, "Birth note", "N1")
	doc.AddNode(note)

	// Create person with birth event that has a note
	person := doc.AddIndividual("I1")
	birth := gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1900"),
	)
	birth.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))
	person.AddNode(birth)

	extractor := HasNoteRelationship()
	edges := extractor.Extract(doc)

	assert.Len(t, edges, 1)
	assert.Equal(t, "has_note", edges[0].Type)
	assert.Equal(t, "i1", edges[0].FromID)
	assert.Equal(t, "n1", edges[0].ToID)
	// Note: context extraction is complex and may need refinement
}

func TestHasMediaRelationship(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create a media object with pointer
	media := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "photo.jpg", ""),
	)
	doc.AddNode(media)

	// Create person with reference to media
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))

	extractor := HasMediaRelationship()
	edges := extractor.Extract(doc)

	assert.Len(t, edges, 1)
	assert.Equal(t, "has_media", edges[0].Type)
	assert.Equal(t, "i1", edges[0].FromID)
	assert.Equal(t, "m1", edges[0].ToID)
}

func TestHasMediaRelationship_WithEventContext(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create a media object with pointer
	media := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "birth_photo.jpg", ""),
	)
	doc.AddNode(media)

	// Create person with birth event that has a media object
	person := doc.AddIndividual("I1")
	birth := gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1900"),
	)
	birth.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))
	person.AddNode(birth)

	extractor := HasMediaRelationship()
	edges := extractor.Extract(doc)

	assert.Len(t, edges, 1, "Should find media attached to birth event")
	assert.Equal(t, "has_media", edges[0].Type)
	assert.Equal(t, "i1", edges[0].FromID)
	assert.Equal(t, "m1", edges[0].ToID)
}

func TestFamilyRelationships(t *testing.T) {
	doc := gedcom.NewDocument()
	
	// Create note and media
	note := gedcom.NewNode(gedcom.TagNote, "Family note", "N1")
	doc.AddNode(note)

	media := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "family.jpg", ""),
	)
	doc.AddNode(media)

	// Create family with note and media
	husband := doc.AddIndividual("I1")
	wife := doc.AddIndividual("I2")
	family := doc.AddFamilyWithHusbandAndWife("F1", husband, wife)
	family.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))
	family.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))

	// Extract family note relationships
	noteExtractor := FamilyHasNoteRelationship()
	noteEdges := noteExtractor.Extract(doc)

	assert.Len(t, noteEdges, 1)
	assert.Equal(t, "has_note", noteEdges[0].Type)
	assert.Equal(t, "f1", noteEdges[0].FromID)
	assert.Equal(t, "n1", noteEdges[0].ToID)

	// Extract family media relationships
	mediaExtractor := FamilyHasMediaRelationship()
	mediaEdges := mediaExtractor.Extract(doc)

	assert.Len(t, mediaEdges, 1)
	assert.Equal(t, "has_media", mediaEdges[0].Type)
	assert.Equal(t, "f1", mediaEdges[0].FromID)
	assert.Equal(t, "m1", mediaEdges[0].ToID)
}

func TestChildOfRelationship_NoDuplicateParents(t *testing.T) {
	// Test case: A child should only have one relationship to each unique parent,
	// even if they appear in multiple families (malformed data scenario)
	doc := gedcom.NewDocument()

	father := doc.AddIndividual("I1")
	mother := doc.AddIndividual("I2")
	child := doc.AddIndividual("I3")

	// Create first family with both parents
	family1 := doc.AddFamilyWithHusbandAndWife("F1", father, mother)
	family1.AddChild(child)

	// Create second family with same parents (this shouldn't happen but can in malformed data)
	family2 := doc.AddFamilyWithHusbandAndWife("F2", father, mother)
	family2.AddChild(child)

	extractor := ChildOfRelationship()
	edges := extractor.Extract(doc)

	// Count relationships from child to each parent
	childToFatherCount := 0
	childToMotherCount := 0
	for _, edge := range edges {
		if edge.FromID == "i3" && edge.ToID == "i1" {
			childToFatherCount++
		}
		if edge.FromID == "i3" && edge.ToID == "i2" {
			childToMotherCount++
		}
	}

	// Should have exactly 1 relationship to father and 1 to mother
	// NOT 2 relationships to each (one per family)
	assert.Equal(t, 1, childToFatherCount, "Should have exactly one child_of relationship to father")
	assert.Equal(t, 1, childToMotherCount, "Should have exactly one child_of relationship to mother")
	assert.Equal(t, 2, len(edges), "Should have exactly 2 total relationships (1 to father, 1 to mother)")
}

func TestParentOfRelationship_NoDuplicateChildren(t *testing.T) {
	// Test case: A parent should only have one relationship to each unique child,
	// even if the child appears in multiple families
	doc := gedcom.NewDocument()

	father := doc.AddIndividual("I1")
	mother := doc.AddIndividual("I2")
	child := doc.AddIndividual("I3")

	// Create first family
	family1 := doc.AddFamilyWithHusbandAndWife("F1", father, mother)
	family1.AddChild(child)

	// Create second family with same parents and child
	family2 := doc.AddFamilyWithHusbandAndWife("F2", father, mother)
	family2.AddChild(child)

	extractor := ParentOfRelationship()
	edges := extractor.Extract(doc)

	// Count relationships from each parent to child
	fatherToChildCount := 0
	motherToChildCount := 0
	for _, edge := range edges {
		if edge.FromID == "i1" && edge.ToID == "i3" {
			fatherToChildCount++
		}
		if edge.FromID == "i2" && edge.ToID == "i3" {
			motherToChildCount++
		}
	}

	// Should have exactly 1 relationship from father and 1 from mother
	assert.Equal(t, 1, fatherToChildCount, "Should have exactly one parent_of relationship from father")
	assert.Equal(t, 1, motherToChildCount, "Should have exactly one parent_of relationship from mother")
	assert.Equal(t, 2, len(edges), "Should have exactly 2 total relationships (1 from father, 1 from mother)")
}

func TestChildOfRelationship_OneParentOnly(t *testing.T) {
	// Test case: Child with only father (mother unknown)
	doc := gedcom.NewDocument()

	father := doc.AddIndividual("I1")
	child := doc.AddIndividual("I2")

	family := doc.AddFamily("F1")
	family.SetHusbandPointer(father.Pointer())
	// No wife - single parent scenario
	family.AddChild(child)

	extractor := ChildOfRelationship()
	edges := extractor.Extract(doc)

	// Should have exactly 1 child_of relationship (child -> father)
	assert.Len(t, edges, 1)
	assert.Equal(t, "child_of", edges[0].Type)
	assert.Equal(t, "i2", edges[0].FromID)
	assert.Equal(t, "i1", edges[0].ToID)
}

func TestChildOfRelationship_MotherOnly(t *testing.T) {
	// Test case: Child with only mother (father unknown)
	doc := gedcom.NewDocument()

	mother := doc.AddIndividual("I1")
	child := doc.AddIndividual("I2")

	family := doc.AddFamily("F1")
	family.SetWifePointer(mother.Pointer())
	// No husband - single parent scenario
	family.AddChild(child)

	extractor := ChildOfRelationship()
	edges := extractor.Extract(doc)

	// Should have exactly 1 child_of relationship (child -> mother)
	assert.Len(t, edges, 1)
	assert.Equal(t, "child_of", edges[0].Type)
	assert.Equal(t, "i2", edges[0].FromID)
	assert.Equal(t, "i1", edges[0].ToID)
}

func TestChildOfRelationship_NoParents(t *testing.T) {
	// Test case: Child with no known parents (orphan/incomplete data)
	doc := gedcom.NewDocument()

	_ = doc.AddIndividual("I1")

	// No family assigned - orphan scenario
	extractor := ChildOfRelationship()
	edges := extractor.Extract(doc)

	// Should have no child_of relationships
	assert.Len(t, edges, 0, "Child with no parents should have no child_of relationships")
}

func TestChildOfRelationship_FamilyWithNoParents(t *testing.T) {
	// Test case: Child in a family but both parents are nil
	doc := gedcom.NewDocument()

	child := doc.AddIndividual("I1")

	family := doc.AddFamily("F1")
	// No husband or wife set
	family.AddChild(child)

	extractor := ChildOfRelationship()
	edges := extractor.Extract(doc)

	// Should have no child_of relationships since both parents are nil
	assert.Len(t, edges, 0, "Child in family with no parents should have no child_of relationships")
}

func TestParentOfRelationship_SingleParent(t *testing.T) {
	// Test case: Single parent with child
	doc := gedcom.NewDocument()

	mother := doc.AddIndividual("I1")
	child := doc.AddIndividual("I2")

	family := doc.AddFamily("F1")
	family.SetWifePointer(mother.Pointer())
	family.AddChild(child)

	extractor := ParentOfRelationship()
	edges := extractor.Extract(doc)

	// Should have exactly 1 parent_of relationship (mother -> child)
	assert.Len(t, edges, 1)
	assert.Equal(t, "parent_of", edges[0].Type)
	assert.Equal(t, "i1", edges[0].FromID)
	assert.Equal(t, "i2", edges[0].ToID)
}

func TestExtractPointerFromValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@N1@", "N1"},
		{"@I123@", "I123"},
		{"@F1@", "F1"},
		{"N1", ""},
		{"@N1", ""},
		{"N1@", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractPointerFromValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}




