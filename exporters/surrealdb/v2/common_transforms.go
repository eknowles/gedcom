package v2

import (
	"time"

	"github.com/eknowles/gedcom/v39"
)

// CommonTransforms contains reusable transform functions
var CommonTransforms = struct {
	DateString      func(gedcom.Node) (interface{}, error)
	DateRangeStart  func(gedcom.Node) (interface{}, error)
	DateRangeEnd    func(gedcom.Node) (interface{}, error)
	PlaceString     func(gedcom.Node) (interface{}, error)
	NameString      func(gedcom.Node) (interface{}, error)
	GivenName       func(gedcom.Node) (interface{}, error)
	Surname         func(gedcom.Node) (interface{}, error)
	FullText        func(gedcom.Node) (interface{}, error)
	PointerValue    func(gedcom.Node) (interface{}, error)
	SexString       func(gedcom.Node) (interface{}, error)
	IsLiving        func(gedcom.Node) (interface{}, error)
}{
	DateString: func(node gedcom.Node) (interface{}, error) {
		if dateNode, ok := node.(*gedcom.DateNode); ok && dateNode != nil {
			return dateNode.String(), nil
		}
		return nil, nil
	},

	DateRangeStart: func(node gedcom.Node) (interface{}, error) {
		if dateNode, ok := node.(*gedcom.DateNode); ok && dateNode != nil && dateNode.IsValid() {
			return dateNode.DateRange().StartDate().Time().Format(time.RFC3339), nil
		}
		return nil, nil
	},

	DateRangeEnd: func(node gedcom.Node) (interface{}, error) {
		if dateNode, ok := node.(*gedcom.DateNode); ok && dateNode != nil && dateNode.IsValid() {
			return dateNode.DateRange().EndDate().Time().Format(time.RFC3339), nil
		}
		return nil, nil
	},

	PlaceString: func(node gedcom.Node) (interface{}, error) {
		if placeNode, ok := node.(*gedcom.PlaceNode); ok && placeNode != nil {
			return placeNode.String(), nil
		}
		return nil, nil
	},

	NameString: func(node gedcom.Node) (interface{}, error) {
		if nameNode, ok := node.(*gedcom.NameNode); ok && nameNode != nil {
			return nameNode.String(), nil
		}
		return nil, nil
	},

	GivenName: func(node gedcom.Node) (interface{}, error) {
		if nameNode, ok := node.(*gedcom.NameNode); ok && nameNode != nil {
			givenName := nameNode.GivenName()
			if givenName != "" {
				return givenName, nil
			}
		}
		return nil, nil
	},

	Surname: func(node gedcom.Node) (interface{}, error) {
		if nameNode, ok := node.(*gedcom.NameNode); ok && nameNode != nil {
			surname := nameNode.Surname()
			if surname != "" {
				return surname, nil
			}
		}
		return nil, nil
	},

	FullText: func(node gedcom.Node) (interface{}, error) {
		if node == nil {
			return nil, nil
		}

		// Get base text
		fullText := node.Value()

		// Add continuation lines (CONT/CONC)
		for _, child := range node.Nodes() {
			if child.Tag().Is(gedcom.TagContinued) {
				fullText += "\n" + child.Value()
			} else if child.Tag().Is(gedcom.TagConcatenation) {
				fullText += child.Value()
			}
		}

		if fullText == "" {
			return nil, nil
		}

		return fullText, nil
	},

	PointerValue: func(node gedcom.Node) (interface{}, error) {
		if node == nil {
			return nil, nil
		}
		return node.Pointer(), nil
	},

	SexString: func(node gedcom.Node) (interface{}, error) {
		if sexNode, ok := node.(*gedcom.SexNode); ok && sexNode != nil {
			return sexNode.Value(), nil
		}
		return nil, nil
	},

	IsLiving: func(node gedcom.Node) (interface{}, error) {
		if individual, ok := node.(*gedcom.IndividualNode); ok && individual != nil {
			return individual.IsLiving(), nil
		}
		return false, nil
	},
}

// ExtractEventObject extracts an event as a nested object with date and place
func ExtractEventObject(eventNode gedcom.Node) map[string]interface{} {
	if eventNode == nil {
		return nil
	}

	props := make(map[string]interface{})

	// Extract date
	dateNodes := gedcom.NodesWithTag(eventNode, gedcom.TagDate)
	if len(dateNodes) > 0 {
		if dateNode, ok := dateNodes[0].(*gedcom.DateNode); ok && dateNode != nil {
			if dateStr, _ := CommonTransforms.DateString(dateNode); dateStr != nil {
				props["date"] = dateStr
			}
			if dateNode.IsValid() {
				if start, _ := CommonTransforms.DateRangeStart(dateNode); start != nil {
					props["date_range_start"] = start
				}
				if end, _ := CommonTransforms.DateRangeEnd(dateNode); end != nil {
					props["date_range_end"] = end
				}
			}
		}
	}

	// Extract place
	placeNodes := gedcom.NodesWithTag(eventNode, gedcom.TagPlace)
	if len(placeNodes) > 0 {
		if placeNode, ok := placeNodes[0].(*gedcom.PlaceNode); ok && placeNode != nil {
			if placeStr, _ := CommonTransforms.PlaceString(placeNode); placeStr != nil {
				props["place"] = placeStr
			}
		}
	}

	if len(props) == 0 {
		return nil
	}

	return props
}

// ExtractNameObject extracts a name as a nested object with all name parts
func ExtractNameObject(nameNode gedcom.Node) map[string]interface{} {
	if nameNode == nil {
		return nil
	}

	name, ok := nameNode.(*gedcom.NameNode)
	if !ok || name == nil {
		return nil
	}

	obj := make(map[string]interface{})

	if prefix := name.Prefix(); prefix != "" {
		obj["prefix"] = prefix
	}
	if given := name.GivenName(); given != "" {
		obj["given"] = given
	}
	if secondary := name.SecondaryGivenName(); secondary != "" {
		obj["middle"] = secondary
	}
	if surname := name.Surname(); surname != "" {
		obj["surname"] = surname
	}
	if suffix := name.Suffix(); suffix != "" {
		obj["suffix"] = suffix
	}

	// Always include full name
	obj["full"] = name.FullName()

	return obj
}

// ExtractNameObjectWithNickname extracts the primary name object with nickname support.
// It extracts the primary name (first NAME entry) and includes nickname from:
// 1. Any NAME entry with TYPE "nick"
// 2. Direct NICK tag under the individual
func ExtractNameObjectWithNickname(individual *gedcom.IndividualNode) map[string]interface{} {
	if individual == nil {
		return nil
	}

	names := individual.Names()
	if len(names) == 0 {
		return nil
	}

	// Extract primary name (first NAME entry)
	obj := ExtractNameObject(names[0])
	if obj == nil {
		obj = make(map[string]interface{})
	}

	// Look for nickname in additional NAME entries with TYPE "nick"
	for _, name := range names {
		if name.Type() == gedcom.NameTypeNickname {
			if nickname := name.GivenName(); nickname != "" {
				obj["nickname"] = nickname
				break
			}
		}
	}

	// If no nickname found in NAME entries, check for direct NICK tag
	if _, hasNickname := obj["nickname"]; !hasNickname {
		nickNodes := gedcom.NodesWithTag(individual, gedcom.TagNickname)
		if len(nickNodes) > 0 && nickNodes[0].Value() != "" {
			obj["nickname"] = nickNodes[0].Value()
		}
	}

	return obj
}


