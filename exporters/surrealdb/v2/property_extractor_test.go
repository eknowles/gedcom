package v2

import (
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

func TestNodesWithTagPath(t *testing.T) {
	tests := []struct {
		name        string
		setupNode   func() gedcom.Node
		tagPath     []gedcom.Tag
		expectValue string
		expectNil   bool
	}{
		{
			name: "Single level path",
			setupNode: func() gedcom.Node {
				doc := gedcom.NewDocument()
				person := doc.AddIndividual("I1")
				person.AddNode(gedcom.NewNameNode("John /Smith/"))
				return person
			},
			tagPath:     []gedcom.Tag{gedcom.TagName},
			expectValue: "John /Smith/",
			expectNil:   false,
		},
		{
			name: "Two level path - Birth Date",
			setupNode: func() gedcom.Node {
				doc := gedcom.NewDocument()
				person := doc.AddIndividual("I1")
				person.AddNode(gedcom.NewBirthNode("",
					gedcom.NewDateNode("1 Jan 1900"),
				))
				return person
			},
			tagPath:     []gedcom.Tag{gedcom.TagBirth, gedcom.TagDate},
			expectValue: "1 Jan 1900",
			expectNil:   false,
		},
		{
			name: "Two level path - Birth Place",
			setupNode: func() gedcom.Node {
				doc := gedcom.NewDocument()
				person := doc.AddIndividual("I1")
				person.AddNode(gedcom.NewBirthNode("",
					gedcom.NewPlaceNode("London, England"),
				))
				return person
			},
			tagPath:     []gedcom.Tag{gedcom.TagBirth, gedcom.TagPlace},
			expectValue: "London, England",
			expectNil:   false,
		},
		{
			name: "Path not found",
			setupNode: func() gedcom.Node {
				doc := gedcom.NewDocument()
				person := doc.AddIndividual("I1")
				return person
			},
			tagPath:   []gedcom.Tag{gedcom.TagBirth, gedcom.TagDate},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := tt.setupNode()
			result := NodesWithTagPath(node, tt.tagPath)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectValue, result.Value())
			}
		})
	}
}

func TestPropertyExtractor_Extract(t *testing.T) {
	tests := []struct {
		name      string
		setupNode func() gedcom.Node
		extractor PropertyExtractor
		expected  interface{}
	}{
		{
			name: "Extract direct value",
			setupNode: func() gedcom.Node {
				return gedcom.NewNameNode("John /Smith/")
			},
			extractor: PropertyExtractor{
				Name: "name",
			},
			expected: "John /Smith/",
		},
		{
			name: "Extract with tag path",
			setupNode: func() gedcom.Node {
				doc := gedcom.NewDocument()
				person := doc.AddIndividual("I1")
				person.AddNode(gedcom.NewBirthNode("",
					gedcom.NewDateNode("1 Jan 1900"),
				))
				return person
			},
			extractor: PropertyExtractor{
				Name:    "birth_date",
				TagPath: []gedcom.Tag{gedcom.TagBirth, gedcom.TagDate},
			},
			expected: "1 Jan 1900",
		},
		{
			name: "Extract with transform",
			setupNode: func() gedcom.Node {
				return gedcom.NewNameNode("John /Smith/")
			},
			extractor: PropertyExtractor{
				Name: "name_upper",
				Transform: func(node gedcom.Node) (interface{}, error) {
					if nameNode, ok := node.(*gedcom.NameNode); ok {
						return nameNode.String(), nil
					}
					return nil, nil
				},
			},
			expected: "John Smith",
		},
		{
			name: "Node not found returns default",
			setupNode: func() gedcom.Node {
				doc := gedcom.NewDocument()
				person := doc.AddIndividual("I1")
				return person
			},
			extractor: PropertyExtractor{
				Name:    "birth_date",
				TagPath: []gedcom.Tag{gedcom.TagBirth, gedcom.TagDate},
				Default: "Unknown",
			},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := tt.setupNode()
			result, err := tt.extractor.Extract(node)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractProperties(t *testing.T) {
	doc := gedcom.NewDocument()
	person := doc.AddIndividual("I1")
	person.AddNode(gedcom.NewNameNode("John /Smith/"))
	person.AddNode(gedcom.NewSexNode("M"))
	person.AddNode(gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1900"),
		gedcom.NewPlaceNode("London"),
	))

	extractors := []PropertyExtractor{
		{
			Name:    "pointer",
			Transform: func(node gedcom.Node) (interface{}, error) {
				return node.Pointer(), nil
			},
		},
		{
			Name: "sex",
			Tags: []gedcom.Tag{gedcom.TagSex},
		},
		{
			Name:    "birth_date",
			TagPath: []gedcom.Tag{gedcom.TagBirth, gedcom.TagDate},
		},
		{
			Name:    "birth_place",
			TagPath: []gedcom.Tag{gedcom.TagBirth, gedcom.TagPlace},
		},
		{
			Name:    "death_date",
			TagPath: []gedcom.Tag{gedcom.TagDeath, gedcom.TagDate},
			Default: nil, // Should not be included
		},
	}

	properties := ExtractProperties(person, extractors)

	assert.Equal(t, "I1", properties["pointer"])
	assert.Equal(t, "M", properties["sex"])
	assert.Equal(t, "1 Jan 1900", properties["birth_date"])
	assert.Equal(t, "London", properties["birth_place"])
	assert.NotContains(t, properties, "death_date") // Should be excluded
}

