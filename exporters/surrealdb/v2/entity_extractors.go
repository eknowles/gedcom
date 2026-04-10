package v2

import (
	"fmt"

	"github.com/eknowles/gedcom/v39"
)

// PersonEntityExtractor creates an extractor for person nodes
func PersonEntityExtractor() EntityExtractor {
	return EntityExtractor{
		EntityType: EntityPerson,
		Tags:       []gedcom.Tag{gedcom.TagIndividual},
		CreateNode: true,

		Properties: []PropertyExtractor{
			{
				Name:      "pointer",
				Transform: CommonTransforms.PointerValue,
			},
			{
				Name:      "is_living",
				Transform: CommonTransforms.IsLiving,
			},
			{
				Name: "name",
				Transform: func(node gedcom.Node) (interface{}, error) {
					if individual, ok := node.(*gedcom.IndividualNode); ok {
						return ExtractNameObjectWithNickname(individual), nil
					}
					return nil, nil
				},
			},
			{
				Name:      "sex",
				Tags:      []gedcom.Tag{gedcom.TagSex},
				Transform: CommonTransforms.SexString,
			},
			// Inline simple events
			{
				Name: "birth",
				Transform: func(node gedcom.Node) (interface{}, error) {
					if individual, ok := node.(*gedcom.IndividualNode); ok {
						births := gedcom.NodesWithTag(individual, gedcom.TagBirth)
						if len(births) > 0 {
							return ExtractEventObject(births[0]), nil
						}
					}
					return nil, nil
				},
			},
			{
				Name: "death",
				Transform: func(node gedcom.Node) (interface{}, error) {
					if individual, ok := node.(*gedcom.IndividualNode); ok {
						deaths := gedcom.NodesWithTag(individual, gedcom.TagDeath)
						if len(deaths) > 0 {
							return ExtractEventObject(deaths[0]), nil
						}
					}
					return nil, nil
				},
			},
			{
				Name: "baptism",
				Transform: func(node gedcom.Node) (interface{}, error) {
					if individual, ok := node.(*gedcom.IndividualNode); ok {
						baptisms := gedcom.NodesWithTag(individual, gedcom.TagBaptism)
						if len(baptisms) > 0 {
							return ExtractEventObject(baptisms[0]), nil
						}
					}
					return nil, nil
				},
			},
			{
				Name: "burial",
				Transform: func(node gedcom.Node) (interface{}, error) {
					if individual, ok := node.(*gedcom.IndividualNode); ok {
						burials := gedcom.NodesWithTag(individual, gedcom.TagBurial)
						if len(burials) > 0 {
							return ExtractEventObject(burials[0]), nil
						}
					}
					return nil, nil
				},
			},
		},
	}
}

// FamilyEntityExtractor creates an extractor for family nodes
func FamilyEntityExtractor() EntityExtractor {
	return EntityExtractor{
		EntityType: EntityFamily,
		Tags:       []gedcom.Tag{gedcom.TagFamily},
		CreateNode: true,

		Properties: []PropertyExtractor{
			{
				Name:      "pointer",
				Transform: CommonTransforms.PointerValue,
			},
			{
				Name: "husband_id",
				Transform: func(node gedcom.Node) (interface{}, error) {
					if family, ok := node.(*gedcom.FamilyNode); ok {
						if husband := family.Husband(); husband != nil {
							if husbandInd := husband.Individual(); husbandInd != nil {
								return fmt.Sprintf("person:%s", sanitizeID(husbandInd.Pointer())), nil
							}
						}
					}
					return nil, nil
				},
			},
			{
				Name: "wife_id",
				Transform: func(node gedcom.Node) (interface{}, error) {
					if family, ok := node.(*gedcom.FamilyNode); ok {
						if wife := family.Wife(); wife != nil {
							if wifeInd := wife.Individual(); wifeInd != nil {
								return fmt.Sprintf("person:%s", sanitizeID(wifeInd.Pointer())), nil
							}
						}
					}
					return nil, nil
				},
			},
			{
				Name: "marriage",
				Tags: []gedcom.Tag{gedcom.TagMarriage},
				Transform: func(node gedcom.Node) (interface{}, error) {
					marriages := gedcom.NodesWithTag(node, gedcom.TagMarriage)
					if len(marriages) > 0 {
						return ExtractEventObject(marriages[0]), nil
					}
					return nil, nil
				},
			},
		},
	}
}

// NoteEntityExtractor creates an extractor for note nodes
func NoteEntityExtractor() EntityExtractor {
	return EntityExtractor{
		EntityType:     EntityNote,
		Tags:           []gedcom.Tag{gedcom.TagNote},
		CreateNode:     true,
		InlineStrategy: CreateNodeIfPointer, // Only create nodes for notes with pointers

		Properties: []PropertyExtractor{
			{
				Name:      "pointer",
				Transform: CommonTransforms.PointerValue,
			},
			{
				Name:      "text",
				Transform: CommonTransforms.FullText,
			},
		},
	}
}

// MediaObjectEntityExtractor creates an extractor for media object nodes
func MediaObjectEntityExtractor() EntityExtractor {
	return EntityExtractor{
		EntityType:     EntityMediaObject,
		Tags:           []gedcom.Tag{gedcom.TagObject},
		CreateNode:     true,
		InlineStrategy: CreateNodeIfPointer, // Only create nodes for media with pointers

		Properties: []PropertyExtractor{
			{
				Name:      "pointer",
				Transform: CommonTransforms.PointerValue,
			},
			{
				Name: "files",
				Transform: func(node gedcom.Node) (interface{}, error) {
					fileNodes := gedcom.NodesWithTag(node, gedcom.TagFile)
					if len(fileNodes) == 0 {
						return nil, nil
					}

					files := make([]string, 0, len(fileNodes))
					for _, fileNode := range fileNodes {
						if fileNode.Value() != "" {
							files = append(files, fileNode.Value())
						}
					}

					if len(files) == 0 {
						return nil, nil
					}

					return files, nil
				},
			},
			{
				Name:    "format",
				TagPath: []gedcom.Tag{gedcom.TagFormat},
			},
			{
				Name:    "title",
				TagPath: []gedcom.Tag{gedcom.TagTitle},
			},
		},
	}
}
