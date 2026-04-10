package v2

import (
	"github.com/eknowles/gedcom/v39"
)

// ParentOfRelationship creates a relationship extractor for parent -> child
func ParentOfRelationship() RelationshipExtractor {
	return RelationshipExtractor{
		RelationType: "parent_of",
		Direction:    DirectionUnidirectional,
		FromEntity:   EntityPerson,
		ToEntity:     EntityPerson,

		ResolveTarget: func(node gedcom.Node, doc *gedcom.Document) []string {
			individual, ok := node.(*gedcom.IndividualNode)
			if !ok {
				return nil
			}

			// Use a map to deduplicate child IDs
			// A parent should only have one relationship to each unique child
			childIDMap := make(map[string]bool)
			families := individual.Families()

			for _, family := range families {
				// Check if this individual is a parent in this family
				isParent := false
				if husband := family.Husband(); husband != nil {
					if husband.Individual() != nil && husband.Individual().Pointer() == individual.Pointer() {
						isParent = true
					}
				}
				if wife := family.Wife(); wife != nil {
					if wife.Individual() != nil && wife.Individual().Pointer() == individual.Pointer() {
						isParent = true
					}
				}

				if isParent {
					children := family.Children()
					for _, child := range children {
						if childInd := child.Individual(); childInd != nil {
							childIDMap[sanitizeID(childInd.Pointer())] = true
						}
					}
				}
			}

			// Convert map keys to slice
			childIDs := make([]string, 0, len(childIDMap))
			for id := range childIDMap {
				childIDs = append(childIDs, id)
			}

			return childIDs
		},

		Properties: []PropertyExtractor{
			{
				Name: "family_id",
				Transform: func(node gedcom.Node) (interface{}, error) {
					// We could extract family context here if needed
					// For now, keeping it simple
					return nil, nil
				},
			},
		},
	}
}

// ChildOfRelationship creates a relationship extractor for child -> parent
func ChildOfRelationship() RelationshipExtractor {
	return RelationshipExtractor{
		RelationType: "child_of",
		Direction:    DirectionUnidirectional,
		FromEntity:   EntityPerson,
		ToEntity:     EntityPerson,

		ResolveTarget: func(node gedcom.Node, doc *gedcom.Document) []string {
			individual, ok := node.(*gedcom.IndividualNode)
			if !ok {
				return nil
			}

			// Use a map to deduplicate parent IDs
			// A person should only have one relationship to each unique parent
			parentIDMap := make(map[string]bool)
			parents := individual.Parents()

			for _, family := range parents {
				if husband := family.Husband(); husband != nil {
					if husbandInd := husband.Individual(); husbandInd != nil {
						parentIDMap[sanitizeID(husbandInd.Pointer())] = true
					}
				}
				if wife := family.Wife(); wife != nil {
					if wifeInd := wife.Individual(); wifeInd != nil {
						parentIDMap[sanitizeID(wifeInd.Pointer())] = true
					}
				}
			}

			// Convert map keys to slice
			parentIDs := make([]string, 0, len(parentIDMap))
			for id := range parentIDMap {
				parentIDs = append(parentIDs, id)
			}

			return parentIDs
		},

		Properties: []PropertyExtractor{
			{
				Name: "family_id",
				Transform: func(node gedcom.Node) (interface{}, error) {
					return nil, nil
				},
			},
		},
	}
}

// SpouseOfRelationship creates a bidirectional relationship extractor for spouses
func SpouseOfRelationship() RelationshipExtractor {
	return RelationshipExtractor{
		RelationType: "spouse_of",
		Direction:    DirectionBidirectional,
		FromEntity:   EntityPerson,
		ToEntity:     EntityPerson,

		ResolveTarget: func(node gedcom.Node, doc *gedcom.Document) []string {
			individual, ok := node.(*gedcom.IndividualNode)
			if !ok {
				return nil
			}

			var spouseIDs []string
			spouses := individual.Spouses()

			for _, spouse := range spouses {
				if spouse != nil {
					spouseIDs = append(spouseIDs, sanitizeID(spouse.Pointer()))
				}
			}

			return spouseIDs
		},

		Properties: []PropertyExtractor{
			{
				Name: "family_id",
				Transform: func(node gedcom.Node) (interface{}, error) {
					return nil, nil
				},
			},
		},
	}
}

// HasNoteRelationship creates a relationship extractor for entity -> note
func HasNoteRelationship() RelationshipExtractor {
	return RelationshipExtractor{
		RelationType: "has_note",
		Direction:    DirectionUnidirectional,
		FromEntity:   EntityPerson, // Can be used for person or family
		ToEntity:     EntityNote,
		Tags:         []gedcom.Tag{gedcom.TagNote},

		ResolveTarget: func(node gedcom.Node, doc *gedcom.Document) []string {
			var noteIDs []string

			// Direct note references
			noteNodes := gedcom.NodesWithTag(node, gedcom.TagNote)
			for _, noteNode := range noteNodes {
				notePointer := extractPointerFromValue(noteNode.Value())
				if notePointer == "" && noteNode.Pointer() != "" {
					notePointer = noteNode.Pointer()
				}

				if notePointer != "" {
					noteIDs = append(noteIDs, sanitizeID(notePointer))
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
							noteIDs = append(noteIDs, sanitizeID(notePointer))
						}
					}
				}
			}

			return noteIDs
		},

		Properties: []PropertyExtractor{
			// Context could be added here if we track event relationships differently
		},
	}
}

// HasMediaRelationship creates a relationship extractor for entity -> media_object
func HasMediaRelationship() RelationshipExtractor {
	return RelationshipExtractor{
		RelationType: "has_media",
		Direction:    DirectionUnidirectional,
		FromEntity:   EntityPerson, // Can be used for person or family
		ToEntity:     EntityMediaObject,
		Tags:         []gedcom.Tag{gedcom.TagObject},

		ResolveTarget: func(node gedcom.Node, doc *gedcom.Document) []string {
			var mediaIDs []string

			// Direct media references
			mediaNodes := gedcom.NodesWithTag(node, gedcom.TagObject)
			for _, mediaNode := range mediaNodes {
				mediaPointer := extractPointerFromValue(mediaNode.Value())
				if mediaPointer == "" && mediaNode.Pointer() != "" {
					mediaPointer = mediaNode.Pointer()
				}

				if mediaPointer != "" {
					mediaIDs = append(mediaIDs, sanitizeID(mediaPointer))
				}
			}

			// Media attached to events
			for _, child := range node.Nodes() {
				if child.Tag().IsEvent() {
					eventMediaNodes := gedcom.NodesWithTag(child, gedcom.TagObject)
					for _, eventMedia := range eventMediaNodes {
						mediaPointer := extractPointerFromValue(eventMedia.Value())
						if mediaPointer == "" && eventMedia.Pointer() != "" {
							mediaPointer = eventMedia.Pointer()
						}

						if mediaPointer != "" {
							mediaIDs = append(mediaIDs, sanitizeID(mediaPointer))
						}
					}
				}
			}

			return mediaIDs
		},

		Properties: []PropertyExtractor{
			// Context could be added here if we track event relationships differently
		},
	}
}

// FamilyHasNoteRelationship creates a relationship extractor for family -> note
func FamilyHasNoteRelationship() RelationshipExtractor {
	rel := HasNoteRelationship()
	rel.FromEntity = EntityFamily
	return rel
}

// FamilyHasMediaRelationship creates a relationship extractor for family -> media_object
func FamilyHasMediaRelationship() RelationshipExtractor {
	rel := HasMediaRelationship()
	rel.FromEntity = EntityFamily
	return rel
}

// AllRelationshipExtractors returns all standard relationship extractors
func AllRelationshipExtractors() []RelationshipExtractor {
	return []RelationshipExtractor{
		ParentOfRelationship(),
		ChildOfRelationship(),
		SpouseOfRelationship(),
		HasNoteRelationship(),
		HasMediaRelationship(),
		FamilyHasNoteRelationship(),
		FamilyHasMediaRelationship(),
	}
}
