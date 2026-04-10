package v2

import (
	"github.com/eknowles/gedcom/v39"
)

// GraphMapper orchestrates entity and relationship extraction
type GraphMapper struct {
	// Entity extractors
	Entities []EntityExtractor

	// Relationship extractors
	Relationships []RelationshipExtractor

	// Event strategy
	EventStrategy EventStrategy
}

// NewGraphMapper creates a new graph mapper with default configuration
func NewGraphMapper() *GraphMapper {
	return &GraphMapper{
		Entities: []EntityExtractor{
			PersonEntityExtractor(),
			FamilyEntityExtractor(),
			NoteEntityExtractor(),
			MediaObjectEntityExtractor(),
		},
		Relationships:  AllRelationshipExtractors(),
		EventStrategy: DefaultEventStrategy(),
	}
}

// ExtractAll extracts all entities and relationships from a document
func (gm *GraphMapper) ExtractAll(doc *gedcom.Document) ([]GraphNode, []GraphEdge, error) {
	entities := gm.ExtractEntities(doc)
	relationships := gm.ExtractRelationships(doc)

	return entities, relationships, nil
}

// ExtractEntities extracts all entity nodes from a document
func (gm *GraphMapper) ExtractEntities(doc *gedcom.Document) []GraphNode {
	var allNodes []GraphNode

	for _, extractor := range gm.Entities {
		nodes := extractor.Extract(doc)
		allNodes = append(allNodes, nodes...)
	}

	return allNodes
}

// ExtractRelationships extracts all relationship edges from a document
func (gm *GraphMapper) ExtractRelationships(doc *gedcom.Document) []GraphEdge {
	var allEdges []GraphEdge

	for _, extractor := range gm.Relationships {
		edges := extractor.Extract(doc)
		allEdges = append(allEdges, edges...)
	}

	return allEdges
}

