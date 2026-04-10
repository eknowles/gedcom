package v2

// EntityType represents the type of entity in the graph database
type EntityType string

const (
	EntityPerson      EntityType = "person"
	EntityFamily      EntityType = "family"
	EntityEvent       EntityType = "event"
	EntityPlace       EntityType = "place"
	EntityNote        EntityType = "note"
	EntityMediaObject EntityType = "media_object"
	EntitySource      EntityType = "source"
)

// InlineStrategy determines how entities are represented
type InlineStrategy int

const (
	// InlineAlways always inline as properties (e.g., simple birth date/place)
	InlineAlways InlineStrategy = iota

	// CreateNodeAndLink create separate node and link (e.g., event with participants)
	CreateNodeAndLink

	// CreateNodeIfPointer create node only if it has pointer (e.g., shared notes)
	CreateNodeIfPointer
)

// RelationshipDirection specifies the direction of a relationship
type RelationshipDirection int

const (
	DirectionUnidirectional RelationshipDirection = iota
	DirectionBidirectional
)

// GraphNode represents a node entity in the graph
type GraphNode struct {
	ID         string                 // Sanitized ID
	Type       EntityType             // Entity type
	Properties map[string]interface{} // Node properties
}

// GraphEdge represents a relationship edge in the graph
type GraphEdge struct {
	FromID     string                 // Source node ID
	ToID       string                 // Target node ID
	FromType   EntityType             // Source entity type
	ToType     EntityType             // Target entity type
	Type       string                 // Relationship type
	Properties map[string]interface{} // Edge properties
}

