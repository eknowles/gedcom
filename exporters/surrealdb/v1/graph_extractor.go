package v1

import (
	"strings"

	"github.com/eknowles/gedcom/v39"
)

// GraphEntity represents a node in the graph database
type GraphEntity struct {
	Type       string                 // "person", "family", "event", "note", etc.
	ID         string                 // Sanitized pointer
	Properties map[string]interface{} // Node properties
	SourceNode gedcom.Node            // Original GEDCOM node
}

// GraphRelationship represents an edge in the graph database
type GraphRelationship struct {
	Type       string                 // "parent_of", "spouse_of", "has_note", etc.
	FromID     string                 // Source node ID
	ToID       string                 // Target node ID
	Properties map[string]interface{} // Edge properties (optional)
}

// GraphExtractor extracts entities and relationships from GEDCOM for graph databases
type GraphExtractor struct {
	document *gedcom.Document

	// Options
	InlineSimpleEvents bool // If true, inline events without notes/sources
	CreatePlaceNodes   bool // If true, create separate place nodes
	CreateEventNodes   bool // If true, create separate event nodes
}

// NewGraphExtractor creates a new graph extractor with default options
func NewGraphExtractor(doc *gedcom.Document) *GraphExtractor {
	return &GraphExtractor{
		document:           doc,
		InlineSimpleEvents: true,  // Default: inline simple events
		CreatePlaceNodes:   false, // Default: inline places
		CreateEventNodes:   false, // Default: inline events
	}
}

// ExtractAll extracts all entities and relationships
func (ge *GraphExtractor) ExtractAll() ([]GraphEntity, []GraphRelationship, error) {
	entities := []GraphEntity{}
	relationships := []GraphRelationship{}

	// Extract person nodes
	personEntities := ge.extractPersonEntities()
	entities = append(entities, personEntities...)

	// Extract family nodes
	familyEntities := ge.extractFamilyEntities()
	entities = append(entities, familyEntities...)

	// Extract note nodes (only those with pointers)
	noteEntities := ge.extractNoteEntities()
	entities = append(entities, noteEntities...)

	// Extract media object nodes (only those with pointers)
	mediaEntities := ge.extractMediaObjectEntities()
	entities = append(entities, mediaEntities...)

	// Extract relationships
	relationships = append(relationships, ge.extractFamilyRelationships()...)
	relationships = append(relationships, ge.extractNoteRelationships()...)
	relationships = append(relationships, ge.extractMediaRelationships()...)

	// Optional: Extract event nodes and relationships
	if ge.CreateEventNodes {
		eventEntities, eventRels := ge.extractEventEntities()
		entities = append(entities, eventEntities...)
		relationships = append(relationships, eventRels...)
	}

	return entities, relationships, nil
}

// extractPersonEntities extracts person nodes with their properties
func (ge *GraphExtractor) extractPersonEntities() []GraphEntity {
	var entities []GraphEntity

	for _, individual := range ge.document.Individuals() {
		properties := make(map[string]interface{})

		// Basic properties
		properties["pointer"] = individual.Pointer()
		properties["is_living"] = individual.IsLiving()

		// Name properties
		if name := individual.Name(); name != nil {
			properties["name"] = name.String()
			if givenName := name.GivenName(); givenName != "" {
				properties["given_name"] = givenName
			}
			if surname := name.Surname(); surname != "" {
				properties["surname"] = surname
			}
		}

		// Sex
		if sex := individual.Sex(); sex != nil {
			properties["sex"] = sex.String()
		}

		// Events - inline or create nodes based on strategy
		if ge.InlineSimpleEvents {
			// Inline simple events
			if birth, _ := individual.Birth(); birth != nil && !ge.isComplexEvent(birth) {
				properties["birth"] = ge.extractEventProperties(birth)
			}
			if death, _ := individual.Death(); death != nil && !ge.isComplexEvent(death) {
				properties["death"] = ge.extractEventProperties(death)
			}
			if baptism, _ := individual.Baptism(); baptism != nil && !ge.isComplexEvent(baptism) {
				properties["baptism"] = ge.extractEventProperties(baptism)
			}
			if burial, _ := individual.Burial(); burial != nil && !ge.isComplexEvent(burial) {
				properties["burial"] = ge.extractEventProperties(burial)
			}
		}

		entity := GraphEntity{
			Type:       "person",
			ID:         sanitizeID(individual.Pointer()),
			Properties: properties,
			SourceNode: individual,
		}

		entities = append(entities, entity)
	}

	return entities
}

// extractEventProperties extracts date and place from an event node
func (ge *GraphExtractor) extractEventProperties(eventNode gedcom.Node) map[string]interface{} {
	props := make(map[string]interface{})

	// Extract date
	dateNodes := gedcom.NodesWithTag(eventNode, gedcom.TagDate)
	if len(dateNodes) > 0 {
		if dateNode, ok := dateNodes[0].(*gedcom.DateNode); ok && dateNode != nil {
			props["date"] = dateNode.String()
			if dateNode.IsValid() {
				props["date_range_start"] = dateNode.DateRange().StartDate().Time()
				props["date_range_end"] = dateNode.DateRange().EndDate().Time()
			}
		}
	}

	// Extract place
	placeNodes := gedcom.NodesWithTag(eventNode, gedcom.TagPlace)
	if len(placeNodes) > 0 {
		if placeNode, ok := placeNodes[0].(*gedcom.PlaceNode); ok && placeNode != nil {
			props["place"] = placeNode.String()
		}
	}

	return props
}

// isComplexEvent checks if an event has notes, sources, or media
func (ge *GraphExtractor) isComplexEvent(eventNode gedcom.Node) bool {
	// Check for notes
	if len(gedcom.NodesWithTag(eventNode, gedcom.TagNote)) > 0 {
		return true
	}

	// Check for sources
	if len(gedcom.NodesWithTag(eventNode, gedcom.TagSource)) > 0 {
		return true
	}

	// Check for media
	if len(gedcom.NodesWithTag(eventNode, gedcom.TagObject)) > 0 {
		return true
	}

	return false
}

// extractFamilyEntities extracts family nodes
func (ge *GraphExtractor) extractFamilyEntities() []GraphEntity {
	var entities []GraphEntity

	for _, family := range ge.document.Families() {
		properties := make(map[string]interface{})

		properties["pointer"] = family.Pointer()

		// Add husband reference
		if husband := family.Husband(); husband != nil {
			if husbandInd := husband.Individual(); husbandInd != nil {
				properties["husband_id"] = "person:" + sanitizeID(husbandInd.Pointer())
			}
		}

		// Add wife reference
		if wife := family.Wife(); wife != nil {
			if wifeInd := wife.Individual(); wifeInd != nil {
				properties["wife_id"] = "person:" + sanitizeID(wifeInd.Pointer())
			}
		}

		// Optional: Add marriage event inline
		marriageNodes := gedcom.NodesWithTag(family, gedcom.TagMarriage)
		if len(marriageNodes) > 0 {
			properties["marriage"] = ge.extractEventProperties(marriageNodes[0])
		}

		entity := GraphEntity{
			Type:       "family",
			ID:         sanitizeID(family.Pointer()),
			Properties: properties,
			SourceNode: family,
		}

		entities = append(entities, entity)
	}

	return entities
}

// extractNoteEntities extracts note nodes (only with pointers)
func (ge *GraphExtractor) extractNoteEntities() []GraphEntity {
	var entities []GraphEntity

	for _, rootNode := range ge.document.Nodes() {
		if noteNode, ok := rootNode.(*gedcom.NoteNode); ok && noteNode.Pointer() != "" {
			properties := make(map[string]interface{})
			properties["pointer"] = noteNode.Pointer()

			// Extract full text with CONT/CONC
			fullText := noteNode.Value()
			for _, child := range noteNode.Nodes() {
				if child.Tag().Is(gedcom.TagContinued) {
					fullText += "\n" + child.Value()
				} else if child.Tag().Is(gedcom.TagConcatenation) {
					fullText += child.Value()
				}
			}
			properties["text"] = fullText

			entity := GraphEntity{
				Type:       "note",
				ID:         sanitizeID(noteNode.Pointer()),
				Properties: properties,
				SourceNode: noteNode,
			}

			entities = append(entities, entity)
		}
	}

	return entities
}

// extractMediaObjectEntities extracts media object nodes
func (ge *GraphExtractor) extractMediaObjectEntities() []GraphEntity {
	var entities []GraphEntity

	for _, rootNode := range ge.document.Nodes() {
		if rootNode.Tag().Is(gedcom.TagObject) && rootNode.Pointer() != "" {
			properties := make(map[string]interface{})
			properties["pointer"] = rootNode.Pointer()

			// Extract files
			fileNodes := gedcom.NodesWithTag(rootNode, gedcom.TagFile)
			if len(fileNodes) > 0 {
				files := []string{}
				for _, fileNode := range fileNodes {
					if fileNode.Value() != "" {
						files = append(files, fileNode.Value())
					}
				}
				properties["files"] = files
			}

			// Extract format
			formatNodes := gedcom.NodesWithTag(rootNode, gedcom.TagFormat)
			if len(formatNodes) > 0 && formatNodes[0].Value() != "" {
				properties["format"] = formatNodes[0].Value()
			}

			// Extract title
			titleNodes := gedcom.NodesWithTag(rootNode, gedcom.TagTitle)
			if len(titleNodes) > 0 && titleNodes[0].Value() != "" {
				properties["title"] = titleNodes[0].Value()
			}

			entity := GraphEntity{
				Type:       "media_object",
				ID:         sanitizeID(rootNode.Pointer()),
				Properties: properties,
				SourceNode: rootNode,
			}

			entities = append(entities, entity)
		}
	}

	return entities
}

// extractFamilyRelationships extracts spouse and parent-child relationships
func (ge *GraphExtractor) extractFamilyRelationships() []GraphRelationship {
	var relationships []GraphRelationship

	for _, family := range ge.document.Families() {
		familyID := sanitizeID(family.Pointer())

		husband := family.Husband()
		wife := family.Wife()

		// Create spouse relationships
		if husband != nil && wife != nil {
			husbandInd := husband.Individual()
			wifeInd := wife.Individual()

			if husbandInd != nil && wifeInd != nil {
				husbandID := sanitizeID(husbandInd.Pointer())
				wifeID := sanitizeID(wifeInd.Pointer())

				// Bidirectional spouse relationship
				relationships = append(relationships,
					GraphRelationship{
						Type:   "spouse_of",
						FromID: husbandID,
						ToID:   wifeID,
						Properties: map[string]interface{}{
							"family_id": "family:" + familyID,
						},
					},
					GraphRelationship{
						Type:   "spouse_of",
						FromID: wifeID,
						ToID:   husbandID,
						Properties: map[string]interface{}{
							"family_id": "family:" + familyID,
						},
					},
				)
			}
		}

		// Create parent-child relationships
		children := family.Children()
		for _, child := range children {
			childInd := child.Individual()
			if childInd == nil {
				continue
			}
			childID := sanitizeID(childInd.Pointer())

			// Parent -> Child relationships
			if husband != nil && husband.Individual() != nil {
				husbandID := sanitizeID(husband.Individual().Pointer())
				relationships = append(relationships,
					GraphRelationship{
						Type:   "parent_of",
						FromID: husbandID,
						ToID:   childID,
						Properties: map[string]interface{}{
							"family_id": "family:" + familyID,
						},
					},
					GraphRelationship{
						Type:   "child_of",
						FromID: childID,
						ToID:   husbandID,
						Properties: map[string]interface{}{
							"family_id": "family:" + familyID,
						},
					},
				)
			}

			if wife != nil && wife.Individual() != nil {
				wifeID := sanitizeID(wife.Individual().Pointer())
				relationships = append(relationships,
					GraphRelationship{
						Type:   "parent_of",
						FromID: wifeID,
						ToID:   childID,
						Properties: map[string]interface{}{
							"family_id": "family:" + familyID,
						},
					},
					GraphRelationship{
						Type:   "child_of",
						FromID: childID,
						ToID:   wifeID,
						Properties: map[string]interface{}{
							"family_id": "family:" + familyID,
						},
					},
				)
			}
		}
	}

	return relationships
}

// extractNoteRelationships extracts has_note relationships
func (ge *GraphExtractor) extractNoteRelationships() []GraphRelationship {
	var relationships []GraphRelationship

	// From individuals
	for _, individual := range ge.document.Individuals() {
		relationships = append(relationships,
			ge.extractNoteRelationshipsFromNode("person", individual)...)
	}

	// From families
	for _, family := range ge.document.Families() {
		relationships = append(relationships,
			ge.extractNoteRelationshipsFromNode("family", family)...)
	}

	return relationships
}

// extractNoteRelationshipsFromNode extracts note relationships from any node
func (ge *GraphExtractor) extractNoteRelationshipsFromNode(sourceType string, node gedcom.Node) []GraphRelationship {
	var relationships []GraphRelationship

	sourceID := sanitizeID(node.Pointer())

	// Direct note references
	noteNodes := gedcom.NodesWithTag(node, gedcom.TagNote)
	for _, noteNode := range noteNodes {
		notePointer := extractPointerFromValue(noteNode.Value())
		if notePointer == "" && noteNode.Pointer() != "" {
			notePointer = noteNode.Pointer()
		}

		if notePointer != "" {
			relationships = append(relationships, GraphRelationship{
				Type:       "has_note",
				FromID:     sourceID,
				ToID:       sanitizeID(notePointer),
				Properties: map[string]interface{}{},
			})
		}
	}

	// Notes attached to events
	for _, child := range node.Nodes() {
		if child.Tag().IsEvent() {
			eventNoteNodes := gedcom.NodesWithTag(child, gedcom.TagNote)
			for _, eventNote := range eventNoteNodes {
				notePointer := extractPointerFromValue(eventNote.Value())
				if notePointer == "" && eventNote.Pointer() != "" {
					notePointer = eventNote.Pointer()
				}

				if notePointer != "" {
					relationships = append(relationships, GraphRelationship{
						Type:   "has_note",
						FromID: sourceID,
						ToID:   sanitizeID(notePointer),
						Properties: map[string]interface{}{
							"context": child.Tag().String(), // e.g., "Birth", "Death"
						},
					})
				}
			}
		}
	}

	return relationships
}

// extractMediaRelationships extracts has_media relationships
func (ge *GraphExtractor) extractMediaRelationships() []GraphRelationship {
	var relationships []GraphRelationship

	// From individuals
	for _, individual := range ge.document.Individuals() {
		relationships = append(relationships,
			ge.extractMediaRelationshipsFromNode("person", individual)...)
	}

	// From families
	for _, family := range ge.document.Families() {
		relationships = append(relationships,
			ge.extractMediaRelationshipsFromNode("family", family)...)
	}

	return relationships
}

// extractMediaRelationshipsFromNode extracts media relationships from any node
func (ge *GraphExtractor) extractMediaRelationshipsFromNode(sourceType string, node gedcom.Node) []GraphRelationship {
	var relationships []GraphRelationship

	sourceID := sanitizeID(node.Pointer())

	// Direct media references
	mediaNodes := gedcom.NodesWithTag(node, gedcom.TagObject)
	for _, mediaNode := range mediaNodes {
		mediaPointer := extractPointerFromValue(mediaNode.Value())
		if mediaPointer == "" && mediaNode.Pointer() != "" {
			mediaPointer = mediaNode.Pointer()
		}

		if mediaPointer != "" {
			relationships = append(relationships, GraphRelationship{
				Type:       "has_media",
				FromID:     sourceID,
				ToID:       sanitizeID(mediaPointer),
				Properties: map[string]interface{}{},
			})
		}
	}

	return relationships
}

// extractEventEntities creates separate event nodes (optional strategy)
func (ge *GraphExtractor) extractEventEntities() ([]GraphEntity, []GraphRelationship) {
	var entities []GraphEntity
	var relationships []GraphRelationship

	// This is a more advanced pattern - create event nodes for all events
	// and link them to individuals

	eventCounter := 1

	for _, individual := range ge.document.Individuals() {
		individualID := sanitizeID(individual.Pointer())

		// Process all event nodes
		for _, child := range individual.Nodes() {
			if child.Tag().IsEvent() {
				// Create event entity
				eventID := sanitizeID(individual.Pointer()) + "_event_" + string(rune(eventCounter))
				eventCounter++

				properties := ge.extractEventProperties(child)
				properties["event_type"] = child.Tag().String()

				entities = append(entities, GraphEntity{
					Type:       "event",
					ID:         eventID,
					Properties: properties,
					SourceNode: child,
				})

				// Create relationship
				relationships = append(relationships, GraphRelationship{
					Type:   "attended_event",
					FromID: individualID,
					ToID:   eventID,
					Properties: map[string]interface{}{
						"role": "primary", // Could be "primary", "witness", etc.
					},
				})
			}
		}
	}

	return entities, relationships
}

// Helper functions

func sanitizeID(pointer string) string {
	// Implementation from existing exporter
	id := strings.Trim(pointer, "@")
	id = strings.ReplaceAll(id, " ", "_")
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ToLower(id)

	if len(id) > 0 && id[0] >= '0' && id[0] <= '9' {
		id = "i" + id
	}

	return id
}

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
