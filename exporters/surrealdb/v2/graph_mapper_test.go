package v2

import (
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

func TestGraphMapper_ExtractAll(t *testing.T) {
	// Create a comprehensive test document
	doc := gedcom.NewDocument()

	// Create people
	father := doc.AddIndividual("I1")
	father.AddNode(gedcom.NewNameNode("John /Smith/"))
	father.AddNode(gedcom.NewSexNode("M"))
	father.AddNode(gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1900"),
		gedcom.NewPlaceNode("London"),
	))

	mother := doc.AddIndividual("I2")
	mother.AddNode(gedcom.NewNameNode("Jane /Doe/"))
	mother.AddNode(gedcom.NewSexNode("F"))

	child := doc.AddIndividual("I3")
	child.AddNode(gedcom.NewNameNode("Bob /Smith/"))

	// Create family
	family := doc.AddFamilyWithHusbandAndWife("F1", father, mother)
	family.AddChild(child)

	// Create note
	note := gedcom.NewNode(gedcom.TagNote, "Family history note", "N1")
	doc.AddNode(note)
	father.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))

	// Create media
	media := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "family.jpg", ""),
	)
	doc.AddNode(media)
	family.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))

	// Extract using mapper
	mapper := NewGraphMapper()
	entities, relationships, err := mapper.ExtractAll(doc)

	assert.NoError(t, err)

	// Verify entities
	personEntities := filterNodesByType(entities, EntityPerson)
	familyEntities := filterNodesByType(entities, EntityFamily)
	noteEntities := filterNodesByType(entities, EntityNote)
	mediaEntities := filterNodesByType(entities, EntityMediaObject)

	assert.Len(t, personEntities, 3, "Should have 3 people")
	assert.Len(t, familyEntities, 1, "Should have 1 family")
	assert.Len(t, noteEntities, 1, "Should have 1 note")
	assert.Len(t, mediaEntities, 1, "Should have 1 media object")

	// Verify person properties
	johnNode := findNodeByID(personEntities, "i1")
	assert.NotNil(t, johnNode)

	// Check name object
	name := johnNode.Properties["name"].(map[string]interface{})
	assert.NotNil(t, name)
	assert.Equal(t, "John Smith", name["full"])

	assert.Equal(t, "M", johnNode.Properties["sex"])

	// Verify birth event inline
	birth := johnNode.Properties["birth"].(map[string]interface{})
	assert.NotNil(t, birth)
	assert.Equal(t, "1 Jan 1900", birth["date"])
	assert.Equal(t, "London", birth["place"])

	// Verify relationships
	spouseRels := filterEdgesByType(relationships, "spouse_of")
	parentRels := filterEdgesByType(relationships, "parent_of")
	childRels := filterEdgesByType(relationships, "child_of")
	noteRels := filterEdgesByType(relationships, "has_note")
	mediaRels := filterEdgesByType(relationships, "has_media")

	assert.Len(t, spouseRels, 2, "Should have 2 spouse relationships (bidirectional)")
	assert.Len(t, parentRels, 2, "Should have 2 parent_of relationships")
	assert.Len(t, childRels, 2, "Should have 2 child_of relationships")
	assert.Len(t, noteRels, 1, "Should have 1 has_note relationship")
	assert.Len(t, mediaRels, 1, "Should have 1 has_media relationship")
}

func TestGraphMapper_ExtractEntities(t *testing.T) {
	doc := gedcom.NewDocument()
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNameNode("John /Smith/"))

	mapper := NewGraphMapper()
	entities := mapper.ExtractEntities(doc)

	assert.GreaterOrEqual(t, len(entities), 1, "Should have at least 1 entity")

	personEntities := filterNodesByType(entities, EntityPerson)
	assert.Len(t, personEntities, 1)
	assert.Equal(t, "i1", personEntities[0].ID)
}

func TestGraphMapper_ExtractRelationships(t *testing.T) {
	doc := gedcom.NewDocument()

	husband := doc.AddIndividual("I1")
	wife := doc.AddIndividual("I2")
	doc.AddFamilyWithHusbandAndWife("F1", husband, wife)

	mapper := NewGraphMapper()
	edges := mapper.ExtractRelationships(doc)

	assert.GreaterOrEqual(t, len(edges), 2, "Should have at least 2 relationships")

	spouseRels := filterEdgesByType(edges, "spouse_of")
	assert.Len(t, spouseRels, 2, "Should have 2 spouse relationships")
}

func TestGraphMapper_CustomConfiguration(t *testing.T) {
	doc := gedcom.NewDocument()
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNameNode("John /Smith/"))

	// Create mapper with only person extractor
	mapper := &GraphMapper{
		Entities: []EntityExtractor{
			PersonEntityExtractor(),
		},
		Relationships: []RelationshipExtractor{},
		EventStrategy: DefaultEventStrategy(),
	}

	entities, relationships, err := mapper.ExtractAll(doc)

	assert.NoError(t, err)
	assert.Len(t, entities, 1)
	assert.Len(t, relationships, 0)
}

// Helper functions

func filterNodesByType(nodes []GraphNode, entityType EntityType) []GraphNode {
	var filtered []GraphNode
	for _, node := range nodes {
		if node.Type == entityType {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

func filterEdgesByType(edges []GraphEdge, relType string) []GraphEdge {
	var filtered []GraphEdge
	for _, edge := range edges {
		if edge.Type == relType {
			filtered = append(filtered, edge)
		}
	}
	return filtered
}

func findNodeByID(nodes []GraphNode, id string) *GraphNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}


