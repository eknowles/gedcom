package v2

import (
	"github.com/eknowles/gedcom/v39"
)

// RelationshipExtractor extracts relationships (edges) from GEDCOM data
type RelationshipExtractor struct {
	// Type of relationship (spouse_of, parent_of, has_note, etc.)
	RelationType string

	// Direction of relationship
	Direction RelationshipDirection

	// Source and target entity types
	FromEntity EntityType
	ToEntity   EntityType

	// Tags that trigger this relationship extraction
	Tags []gedcom.Tag

	// Function to resolve target entity IDs from a source node
	ResolveTarget func(node gedcom.Node, doc *gedcom.Document) []string

	// Properties for the relationship edge
	Properties []PropertyExtractor
}

// Extract extracts relationships from a document
func (re *RelationshipExtractor) Extract(doc *gedcom.Document) []GraphEdge {
	var edges []GraphEdge

	// Track bidirectional pairs to avoid duplicates
	seenPairs := make(map[string]bool)

	// Find source nodes based on FromEntity type
	sourceNodes := findEntitiesByType(doc, re.FromEntity)

	for _, sourceNode := range sourceNodes {
		sourceID := sanitizeID(sourceNode.Pointer())
		if sourceID == "" {
			continue
		}

		// Resolve target IDs
		targetIDs := re.ResolveTarget(sourceNode, doc)

		for _, targetID := range targetIDs {
			if targetID == "" {
				continue
			}

			// For bidirectional relationships, check if we've already processed this pair
			if re.Direction == DirectionBidirectional {
				// Create a canonical pair key (sorted order)
				var pairKey string
				if sourceID < targetID {
					pairKey = sourceID + "|" + targetID
				} else {
					pairKey = targetID + "|" + sourceID
				}

				if seenPairs[pairKey] {
					continue // Skip, already processed
				}
				seenPairs[pairKey] = true
			}

			// Extract edge properties
			properties := ExtractProperties(sourceNode, re.Properties)

			// Create edge
			edge := GraphEdge{
				FromID:     sourceID,
				ToID:       targetID,
				FromType:   re.FromEntity,
				ToType:     re.ToEntity,
				Type:       re.RelationType,
				Properties: properties,
			}

			edges = append(edges, edge)

			// Create reverse edge if bidirectional
			if re.Direction == DirectionBidirectional {
				reverseEdge := GraphEdge{
					FromID:     targetID,
					ToID:       sourceID,
					FromType:   re.ToEntity,
					ToType:     re.FromEntity,
					Type:       re.RelationType,
					Properties: properties,
				}
				edges = append(edges, reverseEdge)
			}
		}
	}

	return edges
}

// findEntitiesByType finds all nodes of a specific entity type in the document
func findEntitiesByType(doc *gedcom.Document, entityType EntityType) []gedcom.Node {
	var nodes []gedcom.Node

	switch entityType {
	case EntityPerson:
		for _, ind := range doc.Individuals() {
			nodes = append(nodes, ind)
		}
	case EntityFamily:
		for _, fam := range doc.Families() {
			nodes = append(nodes, fam)
		}
	case EntityNote:
		for _, rootNode := range doc.Nodes() {
			if noteNode, ok := rootNode.(*gedcom.NoteNode); ok && noteNode.Pointer() != "" {
				nodes = append(nodes, noteNode)
			}
		}
	case EntityMediaObject:
		for _, rootNode := range doc.Nodes() {
			if rootNode.Tag().Is(gedcom.TagObject) && rootNode.Pointer() != "" {
				nodes = append(nodes, rootNode)
			}
		}
	}

	return nodes
}

// extractPointerFromValue extracts a pointer from a value like "@N1@"
func extractPointerFromValue(val string) string {
	valLen := len(val)
	if valLen < 3 {
		return ""
	}
	if val[0] == '@' && val[valLen-1] == '@' {
		return val[1 : valLen-1]
	}
	return ""
}


