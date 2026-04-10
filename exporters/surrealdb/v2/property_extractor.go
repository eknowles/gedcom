package v2

import (
	"github.com/eknowles/gedcom/v39"
)

// PropertyExtractor extracts a single property from a GEDCOM node
type PropertyExtractor struct {
	// Name of the property in the output
	Name string

	// Tags to search for at the current level
	Tags []gedcom.Tag

	// TagPath for nested tag navigation (e.g., [TagBirth, TagDate])
	TagPath []gedcom.Tag

	// Transform function to convert node to output value
	Transform func(gedcom.Node) (interface{}, error)

	// Default value if node not found
	Default interface{}
}

// Extract extracts the property value from a node
func (pe *PropertyExtractor) Extract(node gedcom.Node) (interface{}, error) {
	if node == nil {
		return pe.Default, nil
	}

	var targetNode gedcom.Node

	// Use TagPath if provided
	if len(pe.TagPath) > 0 {
		targetNode = NodesWithTagPath(node, pe.TagPath)
	} else if len(pe.Tags) > 0 {
		// Use Tags at current level
		nodes := gedcom.NodesWithTag(node, pe.Tags[0])
		if len(nodes) > 0 {
			targetNode = nodes[0]
		}
	} else {
		targetNode = node
	}

	if targetNode == nil {
		return pe.Default, nil
	}

	// Apply transform if provided
	if pe.Transform != nil {
		return pe.Transform(targetNode)
	}

	// Default: return node value
	return targetNode.Value(), nil
}

// NodesWithTagPath navigates through a tag path and returns the first matching node
// Example: NodesWithTagPath(individual, [TagBirth, TagDate]) finds DATE under BIRT
func NodesWithTagPath(node gedcom.Node, tagPath []gedcom.Tag) gedcom.Node {
	if node == nil || len(tagPath) == 0 {
		return nil
	}

	current := node
	for _, tag := range tagPath {
		nodes := gedcom.NodesWithTag(current, tag)
		if len(nodes) == 0 {
			return nil
		}
		current = nodes[0]
	}

	return current
}

// ExtractProperties applies multiple property extractors to a node
func ExtractProperties(node gedcom.Node, extractors []PropertyExtractor) map[string]interface{} {
	properties := make(map[string]interface{})

	for _, extractor := range extractors {
		value, err := extractor.Extract(node)
		if err != nil || value == nil {
			continue
		}

		properties[extractor.Name] = value
	}

	return properties
}
