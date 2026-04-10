# Exporter Pattern for GEDCOM Data

## Overview

This document describes an efficient, testable pattern for creating GEDCOM exporters using property extractors. This approach follows TDD principles and leverages the tag system defined in `tag.go`.

## Core Concepts

### 1. Property Extractor Interface

Instead of manually checking each property (as the current SurrealDB exporter does), we define a systematic way to extract properties:

```go
// PropertyExtractor defines how to extract a specific property from a node
type PropertyExtractor struct {
    // Name of the property in the export format (e.g., "given_name", "birthDate")
    Name string
    
    // Tags to search for (e.g., TagBirth, TagDeath)
    Tags []Tag
    
    // Optional: Path through nested tags (e.g., [TagBirth, TagDate])
    TagPath []Tag
    
    // Transform function to convert node value to export format
    Transform func(node Node) (interface{}, error)
}
```

### 2. Entity Mapper

Maps GEDCOM entities (Individual, Family, etc.) to export format:

```go
// EntityMapper defines how to map a GEDCOM entity to an exportable format
type EntityMapper struct {
    // Entity type name (e.g., "person", "family", "event")
    EntityType string
    
    // Properties to extract
    Properties []PropertyExtractor
    
    // Optional: Nested entity mappers
    NestedMappers []EntityMapper
}
```

### 3. Tag-Based Discovery

Use the tag system to automatically discover all possible properties:

```go
// PropertyDiscoverer helps identify all extractable properties
type PropertyDiscoverer struct {
    // Map of Tag to PropertyExtractor
    tagRegistry map[Tag]PropertyExtractor
}

func NewPropertyDiscoverer() *PropertyDiscoverer {
    return &PropertyDiscoverer{
        tagRegistry: make(map[Tag]PropertyExtractor),
    }
}

func (pd *PropertyDiscoverer) RegisterTag(tag Tag, name string, transform func(Node) (interface{}, error)) {
    pd.tagRegistry[tag] = PropertyExtractor{
        Name:      name,
        Tags:      []Tag{tag},
        Transform: transform,
    }
}
```

## TDD Approach

### Step 1: Write Test Data Builders

Create helper functions to build test documents with known data:

```go
// testdata/builders.go
package testdata

func NewMinimalIndividual(pointer string) *Document {
    doc := gedcom.NewDocument()
    individual := doc.AddIndividual(pointer)
    return doc
}

func NewIndividualWithName(pointer, givenName, surname string) *Document {
    doc := NewMinimalIndividual(pointer)
    individual := doc.Individuals()[0]
    individual.AddNode(gedcom.NewNameNode(fmt.Sprintf("%s /%s/", givenName, surname)))
    return doc
}

func NewIndividualWithBirth(pointer string, date string, place string) *Document {
    doc := NewMinimalIndividual(pointer)
    individual := doc.Individuals()[0]
    birth := gedcom.NewBirthNode("",
        gedcom.NewDateNode(date),
        gedcom.NewPlaceNode(place),
    )
    individual.AddNode(birth)
    return doc
}
```

### Step 2: Write Property Extraction Tests

Test individual property extractors before building the full exporter:

```go
func TestExtractGivenName(t *testing.T) {
    tests := []struct {
        name     string
        doc      *gedcom.Document
        expected string
    }{
        {
            name:     "Extract given name from NAME tag",
            doc:      testdata.NewIndividualWithName("I1", "John", "Smith"),
            expected: "John",
        },
        {
            name:     "No name returns empty string",
            doc:      testdata.NewMinimalIndividual("I1"),
            expected: "",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            extractor := PropertyExtractor{
                Name: "given_name",
                Tags: []Tag{gedcom.TagName},
                Transform: func(node gedcom.Node) (interface{}, error) {
                    if nameNode, ok := node.(*gedcom.NameNode); ok {
                        return nameNode.GivenName(), nil
                    }
                    return "", nil
                },
            }
            
            individual := tt.doc.Individuals()[0]
            nameNodes := gedcom.NodesWithTag(individual, gedcom.TagName)
            
            var result string
            if len(nameNodes) > 0 {
                val, err := extractor.Transform(nameNodes[0])
                assert.NoError(t, err)
                result = val.(string)
            }
            
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Step 3: Write Mapper Tests

Test the entity mapper with various property combinations:

```go
func TestIndividualMapper(t *testing.T) {
    tests := []struct {
        name     string
        doc      *gedcom.Document
        expected map[string]interface{}
    }{
        {
            name: "Map individual with name and birth",
            doc: func() *gedcom.Document {
                doc := gedcom.NewDocument()
                ind := doc.AddIndividual("I1")
                ind.AddNode(gedcom.NewNameNode("John /Smith/"))
                ind.AddNode(gedcom.NewBirthNode("", gedcom.NewDateNode("1 Jan 1900")))
                return doc
            }(),
            expected: map[string]interface{}{
                "pointer":    "I1",
                "name":       "John Smith",
                "given_name": "John",
                "surname":    "Smith",
                "birth_date": "1 Jan 1900",
            },
        },
    }
    
    mapper := NewIndividualMapper()
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            individual := tt.doc.Individuals()[0]
            result := mapper.Map(individual)
            
            for key, expectedValue := range tt.expected {
                assert.Equal(t, expectedValue, result[key], "Mismatch for key: %s", key)
            }
        })
    }
}
```

### Step 4: Write Exporter Tests

Finally, test the complete exporter output:

```go
func TestExporter_Export(t *testing.T) {
    tests := []struct {
        name           string
        doc            *gedcom.Document
        expectedOutput []string  // Strings that should appear in output
        notExpected    []string  // Strings that should NOT appear
    }{
        {
            name: "Export single individual",
            doc:  testdata.NewIndividualWithName("I1", "Jane", "Doe"),
            expectedOutput: []string{
                "Jane",
                "Doe",
                "I1",
            },
            notExpected: []string{
                "John",
                "Smith",
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            exporter := NewMyExporter(&buf)
            
            err := exporter.Export(tt.doc)
            assert.NoError(t, err)
            
            output := buf.String()
            
            for _, expected := range tt.expectedOutput {
                assert.Contains(t, output, expected)
            }
            
            for _, notExpected := range tt.notExpected {
                assert.NotContains(t, output, notExpected)
            }
        })
    }
}
```

## Implementation Pattern

### Define Property Extractors Using Tags

```go
// extractors.go
package myexporter

var IndividualPropertyExtractors = []PropertyExtractor{
    {
        Name: "name",
        Tags: []Tag{gedcom.TagName},
        Transform: func(node gedcom.Node) (interface{}, error) {
            if n, ok := node.(*gedcom.NameNode); ok {
                return n.String(), nil
            }
            return nil, nil
        },
    },
    {
        Name: "given_name",
        Tags: []Tag{gedcom.TagName},
        Transform: func(node gedcom.Node) (interface{}, error) {
            if n, ok := node.(*gedcom.NameNode); ok {
                return n.GivenName(), nil
            }
            return nil, nil
        },
    },
    {
        Name: "surname",
        Tags: []Tag{gedcom.TagName},
        Transform: func(node gedcom.Node) (interface{}, error) {
            if n, ok := node.(*gedcom.NameNode); ok {
                return n.Surname(), nil
            }
            return nil, nil
        },
    },
    {
        Name: "sex",
        Tags: []Tag{gedcom.TagSex},
        Transform: func(node gedcom.Node) (interface{}, error) {
            if n, ok := node.(*gedcom.SexNode); ok {
                return n.String(), nil
            }
            return nil, nil
        },
    },
    {
        Name: "birth_date",
        TagPath: []Tag{gedcom.TagBirth, gedcom.TagDate},
        Transform: func(node gedcom.Node) (interface{}, error) {
            if n, ok := node.(*gedcom.DateNode); ok {
                return n.String(), nil
            }
            return nil, nil
        },
    },
    {
        Name: "birth_place",
        TagPath: []Tag{gedcom.TagBirth, gedcom.TagPlace},
        Transform: func(node gedcom.Node) (interface{}, error) {
            if n, ok := node.(*gedcom.PlaceNode); ok {
                return n.String(), nil
            }
            return nil, nil
        },
    },
}
```

### Generic Property Extraction

```go
// mapper.go
package myexporter

func ExtractProperties(node gedcom.Node, extractors []PropertyExtractor) map[string]interface{} {
    result := make(map[string]interface{})
    
    for _, extractor := range extractors {
        var nodes gedcom.Nodes
        
        if len(extractor.TagPath) > 0 {
            // Navigate through tag path
            nodes = NodesWithTagPath(node, extractor.TagPath...)
        } else if len(extractor.Tags) > 0 {
            // Direct child tags
            nodes = gedcom.NodesWithTag(node, extractor.Tags[0])
        }
        
        if len(nodes) > 0 {
            value, err := extractor.Transform(nodes[0])
            if err == nil && value != nil {
                // Only include non-nil values
                result[extractor.Name] = value
            }
        }
    }
    
    return result
}

// Helper to navigate nested tag paths
func NodesWithTagPath(node gedcom.Node, tagPath ...gedcom.Tag) gedcom.Nodes {
    if len(tagPath) == 0 {
        return nil
    }
    
    current := gedcom.NodesWithTag(node, tagPath[0])
    if len(tagPath) == 1 {
        return current
    }
    
    // Recursively search nested tags
    var results gedcom.Nodes
    for _, n := range current {
        nested := NodesWithTagPath(n, tagPath[1:]...)
        results = append(results, nested...)
    }
    
    return results
}
```

## Benefits of This Approach

1. **Testability**: Each component can be tested independently
2. **Discoverability**: Use `Tags()` function to iterate all possible tags
3. **Reusability**: Property extractors can be shared across exporters
4. **Maintainability**: Adding new properties is declarative
5. **Type Safety**: Transform functions provide clear type conversions
6. **TDD Friendly**: Write tests for each extractor before implementation

## Example: Adding a New Property

To add support for occupation:

1. **Write the test first**:
```go
func TestExtractOccupation(t *testing.T) {
    doc := gedcom.NewDocument()
    ind := doc.AddIndividual("I1")
    ind.AddNode(gedcom.NewNode(gedcom.TagOccupation, "Software Engineer", ""))
    
    extractor := PropertyExtractor{
        Name: "occupation",
        Tags: []Tag{gedcom.TagOccupation},
        Transform: func(node gedcom.Node) (interface{}, error) {
            return node.Value(), nil
        },
    }
    
    // Test extraction logic
    // ...
}
```

2. **Implement the extractor**:
```go
{
    Name: "occupation",
    Tags: []Tag{gedcom.TagOccupation},
    Transform: func(node gedcom.Node) (interface{}, error) {
        return node.Value(), nil
    },
}
```

3. **Add to the mapper**:
```go
IndividualPropertyExtractors = append(IndividualPropertyExtractors, occupationExtractor)
```

4. **Run tests to verify**

## Discovering All Available Properties

Use the tag system to generate a complete list of extractable properties:

```go
func ListAllPossibleProperties() {
    for _, tag := range gedcom.Tags() {
        if tag.IsOfficial() {
            fmt.Printf("Tag: %s (%s)\n", tag.Tag(), tag.String())
            fmt.Printf("  - Is Event: %v\n", tag.IsEvent())
        }
    }
}
```

This allows you to systematically build extractors for all properties you want to support.

