package v2

import (
	"strings"

	"github.com/eknowles/gedcom/v39"
)

// EntityExtractor extracts entities (nodes) from GEDCOM data
type EntityExtractor struct {
	// Type of entity to create (person, event, note, etc.)
	EntityType EntityType

	// Tags that indicate this entity should be extracted
	Tags []gedcom.Tag

	// Whether to create nodes for each occurrence
	CreateNode bool

	// Strategy for inline vs separate nodes
	InlineStrategy InlineStrategy

	// Properties for the entity node
	Properties []PropertyExtractor

	// Filter function to determine if entity should be created
	// Returns true if the entity should be created
	Filter func(gedcom.Node) bool
}

// EventStrategy determines when to create separate event nodes
type EventStrategy struct {
	// InlineIfSimple inlines events that only have date and place
	InlineIfSimple bool

	// ComplexityTags are tags that make an event "complex"
	ComplexityTags []gedcom.Tag
}

// DefaultEventStrategy returns the recommended hybrid strategy
func DefaultEventStrategy() EventStrategy {
	return EventStrategy{
		InlineIfSimple: true,
		ComplexityTags: []gedcom.Tag{
			gedcom.TagNote,
			gedcom.TagSource,
			gedcom.TagObject,
		},
	}
}

// ShouldCreateNode determines if an event should be a separate node
func (es *EventStrategy) ShouldCreateNode(eventNode gedcom.Node) bool {
	if !es.InlineIfSimple {
		return true // Always create nodes
	}

	// Check if event has complex children
	for _, child := range eventNode.Nodes() {
		for _, complexTag := range es.ComplexityTags {
			if child.Tag().Is(complexTag) {
				return true // Has notes/sources/media, create separate node
			}
		}
	}

	return false // Simple event, inline it
}

// Extract extracts entities from a document
func (ee *EntityExtractor) Extract(doc *gedcom.Document) []GraphNode {
	var nodes []GraphNode

	// Find all matching root nodes
	for _, rootNode := range doc.Nodes() {
		nodes = append(nodes, ee.extractFromNode(rootNode)...)
	}

	return nodes
}

// extractFromNode extracts entities from a single node
func (ee *EntityExtractor) extractFromNode(node gedcom.Node) []GraphNode {
	var nodes []GraphNode

	// Check if this node matches our tags
	matches := false
	for _, tag := range ee.Tags {
		if node.Tag().Is(tag) {
			matches = true
			break
		}
	}

	if !matches {
		return nodes
	}

	// Apply filter if provided
	if ee.Filter != nil && !ee.Filter(node) {
		return nodes
	}

	// Apply inline strategy
	if ee.InlineStrategy == CreateNodeIfPointer && node.Pointer() == "" {
		return nodes // Skip nodes without pointers
	}

	// Extract properties
	properties := ExtractProperties(node, ee.Properties)

	// Create graph node
	graphNode := GraphNode{
		ID:         sanitizeID(node.Pointer()),
		Type:       ee.EntityType,
		Properties: properties,
	}

	nodes = append(nodes, graphNode)

	return nodes
}

// FindNodesWithTags finds all nodes in the tree matching any of the given tags
func FindNodesWithTags(root gedcom.Node, tags []gedcom.Tag) []gedcom.Node {
	var matches []gedcom.Node

	for _, tag := range tags {
		matches = append(matches, gedcom.NodesWithTag(root, tag)...)
	}

	return matches
}

// sanitizeID sanitizes a GEDCOM pointer for use as a database ID
func sanitizeID(pointer string) string {
	if pointer == "" {
		return ""
	}

	// Remove @ symbols
	id := strings.Trim(pointer, "@")

	// Replace spaces and dashes
	id = strings.ReplaceAll(id, " ", "_")
	id = strings.ReplaceAll(id, "-", "_")

	// Convert to lowercase
	id = strings.ToLower(id)

	// Ensure ID starts with a letter (not digit)
	if len(id) > 0 && id[0] >= '0' && id[0] <= '9' {
		id = "i" + id
	}

	return id
}
