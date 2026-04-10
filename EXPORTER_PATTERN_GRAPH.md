# Graph Exporter Pattern for GEDCOM Data

## Overview

This extends the property extractor pattern for **graph databases** like SurrealDB, where tags can become:
1. **Separate nodes** (entities) - e.g., Event, Note, Media
2. **Relationships** (edges) - e.g., ATTENDED_EVENT, HAS_NOTE
3. **Inline properties** - e.g., name, sex embedded in person node

## Core Concepts for Graph Databases

### 1. Entity Types

Define what becomes a separate node in the graph:

```go
type EntityType string

const (
    EntityPerson       EntityType = "person"
    EntityFamily       EntityType = "family"
    EntityEvent        EntityType = "event"
    EntityPlace        EntityType = "place"
    EntityNote         EntityType = "note"
    EntityMediaObject  EntityType = "media_object"
    EntitySource       EntityType = "source"
)
```

### 2. Entity Extractor

Extracts entities that become their own nodes:

```go
type EntityExtractor struct {
    // Type of entity to create (person, event, note, etc.)
    EntityType EntityType
    
    // Tags that indicate this entity should be extracted
    Tags []gedcom.Tag
    
    // Whether to create nodes for each occurrence
    CreateNode bool
    
    // Whether to inline as properties or create separate nodes
    InlineStrategy InlineStrategy
    
    // Properties for the entity node
    Properties []PropertyExtractor
    
    // Relationships this entity creates
    Relationships []RelationshipExtractor
}

type InlineStrategy int

const (
    // Always inline as properties (e.g., simple birth date/place)
    InlineAlways InlineStrategy = iota
    
    // Create separate node and link (e.g., event with participants)
    CreateNodeAndLink
    
    // Create node only if it has pointer (e.g., shared notes)
    CreateNodeIfPointer
)
```

### 3. Relationship Extractor

Defines edges between nodes:

```go
type RelationshipExtractor struct {
    // Relationship type (spouse_of, parent_of, attended_event, etc.)
    RelationType string
    
    // Direction of relationship
    Direction RelationshipDirection
    
    // Source and target entity types
    FromEntity EntityType
    ToEntity   EntityType
    
    // Tags that trigger this relationship
    Tags []gedcom.Tag
    
    // Function to resolve the target entity ID from node
    ResolveTarget func(node gedcom.Node, doc *gedcom.Document) []string
    
    // Properties on the relationship edge
    Properties []PropertyExtractor
}

type RelationshipDirection int

const (
    DirectionUnidirectional RelationshipDirection = iota
    DirectionBidirectional
)
```

## SurrealDB-Specific Pattern

### Strategy 1: Inline Events (Current Approach)

Events like birth/death stored as nested objects in person:

```go
var PersonWithInlineEvents = EntityExtractor{
    EntityType: EntityPerson,
    Tags:       []gedcom.Tag{gedcom.TagIndividual},
    CreateNode: true,
    
    Properties: []PropertyExtractor{
        // Basic properties
        {Name: "pointer", /* ... */},
        {Name: "name", Tags: []gedcom.Tag{gedcom.TagName}, /* ... */},
        {Name: "sex", Tags: []gedcom.Tag{gedcom.TagSex}, /* ... */},
        
        // Inline events as objects
        {
            Name: "birth",
            Tags: []gedcom.Tag{gedcom.TagBirth},
            Transform: func(node gedcom.Node) (interface{}, error) {
                birthNode := node.(*gedcom.BirthNode)
                return map[string]interface{}{
                    "date":  ExtractDate(birthNode),
                    "place": ExtractPlace(birthNode),
                }, nil
            },
        },
        {
            Name: "death",
            Tags: []gedcom.Tag{gedcom.TagDeath},
            Transform: func(node gedcom.Node) (interface{}, error) {
                deathNode := node.(*gedcom.DeathNode)
                return map[string]interface{}{
                    "date":  ExtractDate(deathNode),
                    "place": ExtractPlace(deathNode),
                }, nil
            },
        },
    },
}
```

### Strategy 2: Event Nodes (Graph-Native Approach)

Events as separate nodes with relationships:

```go
var EventEntityExtractor = EntityExtractor{
    EntityType:     EntityEvent,
    Tags:           AllEventTags(), // Birth, Death, Marriage, etc.
    CreateNode:     true,
    InlineStrategy: CreateNodeAndLink,
    
    Properties: []PropertyExtractor{
        {
            Name: "event_type",
            Transform: func(node gedcom.Node) (interface{}, error) {
                return node.Tag().String(), nil // "Birth", "Death", etc.
            },
        },
        {
            Name: "date",
            TagPath: []gedcom.Tag{gedcom.TagDate},
            Transform: CommonTransforms.DateString,
        },
        {
            Name: "place",
            TagPath: []gedcom.Tag{gedcom.TagPlace},
            Transform: CommonTransforms.PlaceString,
        },
    },
    
    Relationships: []RelationshipExtractor{
        {
            RelationType: "attended_event",
            Direction:    DirectionUnidirectional,
            FromEntity:   EntityPerson,
            ToEntity:     EntityEvent,
        },
    },
}

// Helper to get all event tags
func AllEventTags() []gedcom.Tag {
    var eventTags []gedcom.Tag
    for _, tag := range gedcom.Tags() {
        if tag.IsEvent() {
            eventTags = append(eventTags, tag)
        }
    }
    return eventTags
}
```

### Strategy 3: Hybrid (Recommended)

Inline simple events, create nodes for complex ones:

```go
type EventStrategy struct {
    // Inline if event only has date and place
    InlineIfSimple bool
    
    // Create node if event has these additional tags
    ComplexityTags []gedcom.Tag // e.g., TagNote, TagSource, TagMedia
}

func (es *EventStrategy) ShouldCreateNode(eventNode gedcom.Node) bool {
    if !es.InlineIfSimple {
        return true
    }
    
    // Check if event has complex children
    for _, child := range eventNode.Nodes() {
        for _, complexTag := range es.ComplexityTags {
            if child.Tag().Is(complexTag) {
                return true // Has notes/sources, create separate node
            }
        }
    }
    
    return false // Simple event, inline it
}
```

## Complete SurrealDB Example

```go
type SurrealDBGraphMapper struct {
    // Entity extractors define what becomes nodes
    Entities []EntityExtractor
    
    // Relationship extractors define edges
    Relationships []RelationshipExtractor
    
    // Strategy for event handling
    EventStrategy EventStrategy
}

func NewSurrealDBGraphMapper() *SurrealDBGraphMapper {
    return &SurrealDBGraphMapper{
        Entities: []EntityExtractor{
            PersonEntityExtractor(),
            FamilyEntityExtractor(),
            NoteEntityExtractor(),
            MediaEntityExtractor(),
            // Optionally: EventEntityExtractor(),
            // Optionally: PlaceEntityExtractor(),
        },
        Relationships: []RelationshipExtractor{
            SpouseOfRelationship(),
            ParentOfRelationship(),
            ChildOfRelationship(),
            HasNoteRelationship(),
            HasMediaRelationship(),
            // If using event nodes:
            // AttendedEventRelationship(),
        },
        EventStrategy: EventStrategy{
            InlineIfSimple: true,
            ComplexityTags: []gedcom.Tag{
                gedcom.TagNote,
                gedcom.TagSource,
                gedcom.TagMedia,
            },
        },
    }
}
```

### Entity Extractors

```go
func PersonEntityExtractor() EntityExtractor {
    return EntityExtractor{
        EntityType: EntityPerson,
        Tags:       []gedcom.Tag{gedcom.TagIndividual},
        CreateNode: true,
        
        Properties: []PropertyExtractor{
            {Name: "pointer", /* ... */},
            {Name: "name", Tags: []gedcom.Tag{gedcom.TagName}, /* ... */},
            {Name: "given_name", Tags: []gedcom.Tag{gedcom.TagName}, /* ... */},
            {Name: "surname", Tags: []gedcom.Tag{gedcom.TagName}, /* ... */},
            {Name: "sex", Tags: []gedcom.Tag{gedcom.TagSex}, /* ... */},
            {Name: "is_living", /* computed */ },
            // Inline simple events
            {Name: "birth", Tags: []gedcom.Tag{gedcom.TagBirth}, /* ... */},
            {Name: "death", Tags: []gedcom.Tag{gedcom.TagDeath}, /* ... */},
        },
    }
}

func NoteEntityExtractor() EntityExtractor {
    return EntityExtractor{
        EntityType:     EntityNote,
        Tags:           []gedcom.Tag{gedcom.TagNote},
        CreateNode:     true,
        InlineStrategy: CreateNodeIfPointer, // Only if has pointer
        
        Properties: []PropertyExtractor{
            {Name: "pointer", /* ... */},
            {
                Name: "text",
                Transform: func(node gedcom.Node) (interface{}, error) {
                    // Concatenate CONT and CONC tags
                    return ExtractFullText(node), nil
                },
            },
        },
    }
}

func MediaEntityExtractor() EntityExtractor {
    return EntityExtractor{
        EntityType:     EntityMediaObject,
        Tags:           []gedcom.Tag{gedcom.TagObject},
        CreateNode:     true,
        InlineStrategy: CreateNodeIfPointer,
        
        Properties: []PropertyExtractor{
            {Name: "pointer", /* ... */},
            {
                Name: "files",
                Tags: []gedcom.Tag{gedcom.TagFile},
                Transform: func(node gedcom.Node) (interface{}, error) {
                    // Collect all FILE tags
                    files := gedcom.NodesWithTag(node.Parent(), gedcom.TagFile)
                    var fileList []string
                    for _, f := range files {
                        fileList = append(fileList, f.Value())
                    }
                    return fileList, nil
                },
            },
            {Name: "format", Tags: []gedcom.Tag{gedcom.TagFormat}, /* ... */},
            {Name: "title", Tags: []gedcom.Tag{gedcom.TagTitle}, /* ... */},
        },
    }
}
```

### Relationship Extractors

```go
func HasNoteRelationship() RelationshipExtractor {
    return RelationshipExtractor{
        RelationType: "has_note",
        Direction:    DirectionUnidirectional,
        FromEntity:   EntityPerson, // or EntityFamily
        ToEntity:     EntityNote,
        Tags:         []gedcom.Tag{gedcom.TagNote},
        
        ResolveTarget: func(node gedcom.Node, doc *gedcom.Document) []string {
            noteNodes := gedcom.NodesWithTag(node, gedcom.TagNote)
            var noteIDs []string
            
            for _, noteNode := range noteNodes {
                pointer := ExtractPointerFromValue(noteNode.Value())
                if pointer != "" {
                    noteIDs = append(noteIDs, pointer)
                }
            }
            
            return noteIDs
        },
        
        Properties: []PropertyExtractor{
            {
                Name: "context",
                Transform: func(node gedcom.Node) (interface{}, error) {
                    // If note is under an event, include event type
                    parent := node.Parent()
                    if parent != nil && parent.Tag().IsEvent() {
                        return parent.Tag().String(), nil
                    }
                    return nil, nil
                },
            },
        },
    }
}

func ParentOfRelationship() RelationshipExtractor {
    return RelationshipExtractor{
        RelationType: "parent_of",
        Direction:    DirectionUnidirectional,
        FromEntity:   EntityPerson,
        ToEntity:     EntityPerson,
        
        ResolveTarget: func(node gedcom.Node, doc *gedcom.Document) []string {
            individual := node.(*gedcom.IndividualNode)
            families := individual.Families()
            
            var childIDs []string
            for _, family := range families {
                // Only if this individual is a parent (husband/wife)
                if family.Husband().IsIndividual(individual) || 
                   family.Wife().IsIndividual(individual) {
                    children := family.Children()
                    for _, child := range children {
                        childIDs = append(childIDs, child.Individual().Pointer())
                    }
                }
            }
            
            return childIDs
        },
        
        Properties: []PropertyExtractor{
            {
                Name: "family_id",
                Transform: func(node gedcom.Node) (interface{}, error) {
                    // Could extract family context if needed
                    return nil, nil
                },
            },
        },
    }
}
```

## Export Process Flow

```go
type GraphExporter struct {
    mapper     *SurrealDBGraphMapper
    writer     io.Writer
    nodeCache  map[string]bool // Track created nodes
    edgeCache  map[string]bool // Track created edges
}

func (ge *GraphExporter) Export(doc *gedcom.Document) error {
    // Phase 1: Extract and create all entity nodes
    for _, entityExtractor := range ge.mapper.Entities {
        entities := ge.extractEntities(doc, entityExtractor)
        for _, entity := range entities {
            if err := ge.createNode(entity); err != nil {
                return err
            }
        }
    }
    
    // Phase 2: Create all relationships
    for _, relExtractor := range ge.mapper.Relationships {
        relationships := ge.extractRelationships(doc, relExtractor)
        for _, rel := range relationships {
            if err := ge.createEdge(rel); err != nil {
                return err
            }
        }
    }
    
    return nil
}

type GraphNode struct {
    ID         string
    Type       EntityType
    Properties map[string]interface{}
}

type GraphEdge struct {
    FromID     string
    ToID       string
    Type       string
    Properties map[string]interface{}
}

func (ge *GraphExporter) extractEntities(doc *gedcom.Document, extractor EntityExtractor) []GraphNode {
    var nodes []GraphNode
    
    // Find all nodes matching the entity tags
    for _, rootNode := range doc.Nodes() {
        matches := FindNodesWithTags(rootNode, extractor.Tags)
        
        for _, match := range matches {
            // Check inline strategy
            if extractor.InlineStrategy == CreateNodeIfPointer {
                if match.Pointer() == "" {
                    continue // Skip inline nodes
                }
            }
            
            // Extract properties
            properties := ExtractProperties(match, extractor.Properties)
            
            node := GraphNode{
                ID:         SanitizeID(match.Pointer()),
                Type:       extractor.EntityType,
                Properties: properties,
            }
            
            nodes = append(nodes, node)
        }
    }
    
    return nodes
}

func (ge *GraphExporter) extractRelationships(doc *gedcom.Document, extractor RelationshipExtractor) []GraphEdge {
    var edges []GraphEdge
    
    // Find source entities
    sourceNodes := ge.findEntitiesByType(doc, extractor.FromEntity)
    
    for _, sourceNode := range sourceNodes {
        // Resolve target IDs using the extractor's logic
        targetIDs := extractor.ResolveTarget(sourceNode, doc)
        
        for _, targetID := range targetIDs {
            // Extract edge properties
            properties := ExtractProperties(sourceNode, extractor.Properties)
            
            edge := GraphEdge{
                FromID:     SanitizeID(sourceNode.Pointer()),
                ToID:       SanitizeID(targetID),
                Type:       extractor.RelationType,
                Properties: properties,
            }
            
            edges = append(edges, edge)
            
            // Create reverse edge if bidirectional
            if extractor.Direction == DirectionBidirectional {
                reverseEdge := GraphEdge{
                    FromID:     edge.ToID,
                    ToID:       edge.FromID,
                    Type:       edge.Type,
                    Properties: edge.Properties,
                }
                edges = append(edges, reverseEdge)
            }
        }
    }
    
    return edges
}
```

## TDD Approach for Graph Exporters

### Test Node Creation

```go
func TestExtractPersonNode(t *testing.T) {
    doc := gedcom.NewDocument()
    person := doc.AddIndividual("I1")
    person.AddNode(gedcom.NewNameNode("John /Smith/"))
    person.AddNode(gedcom.NewSexNode("M"))
    
    extractor := PersonEntityExtractor()
    nodes := extractEntities(doc, extractor)
    
    assert.Len(t, nodes, 1)
    assert.Equal(t, "i1", nodes[0].ID)
    assert.Equal(t, EntityPerson, nodes[0].Type)
    assert.Equal(t, "John Smith", nodes[0].Properties["name"])
    assert.Equal(t, "M", nodes[0].Properties["sex"])
}
```

### Test Relationship Extraction

```go
func TestExtractParentChildRelationship(t *testing.T) {
    doc := gedcom.NewDocument()
    father := doc.AddIndividual("I1")
    child := doc.AddIndividual("I2")
    
    family := doc.AddFamilyWithHusbandAndWife("F1", father, nil)
    family.AddChild(child)
    
    relExtractor := ParentOfRelationship()
    edges := extractRelationships(doc, relExtractor)
    
    assert.Len(t, edges, 1)
    assert.Equal(t, "i1", edges[0].FromID)
    assert.Equal(t, "i2", edges[0].ToID)
    assert.Equal(t, "parent_of", edges[0].Type)
}
```

### Test Inline vs Separate Nodes

```go
func TestEventInlineStrategy(t *testing.T) {
    tests := []struct {
        name              string
        setupEvent        func(*gedcom.IndividualNode)
        shouldCreateNode  bool
    }{
        {
            name: "Simple birth - inline",
            setupEvent: func(ind *gedcom.IndividualNode) {
                ind.AddNode(gedcom.NewBirthNode("",
                    gedcom.NewDateNode("1 Jan 1900"),
                    gedcom.NewPlaceNode("London"),
                ))
            },
            shouldCreateNode: false,
        },
        {
            name: "Birth with note - create node",
            setupEvent: func(ind *gedcom.IndividualNode) {
                birth := gedcom.NewBirthNode("",
                    gedcom.NewDateNode("1 Jan 1900"),
                    gedcom.NewPlaceNode("London"),
                )
                birth.AddNode(gedcom.NewNode(gedcom.TagNote, "@N1@", ""))
                ind.AddNode(birth)
            },
            shouldCreateNode: true,
        },
    }
    
    strategy := EventStrategy{
        InlineIfSimple: true,
        ComplexityTags: []gedcom.Tag{gedcom.TagNote, gedcom.TagSource},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            doc := gedcom.NewDocument()
            ind := doc.AddIndividual("I1")
            tt.setupEvent(ind)
            
            birthNode := gedcom.NodesWithTag(ind, gedcom.TagBirth)[0]
            shouldCreate := strategy.ShouldCreateNode(birthNode)
            
            assert.Equal(t, tt.shouldCreateNode, shouldCreate)
        })
    }
}
```

## Benefits for Graph Databases

1. **Flexible topology**: Events can be nodes or properties based on complexity
2. **Rich relationships**: Properties on edges (e.g., relationship context)
3. **Traversal queries**: `SELECT * FROM person WHERE ->attended_event->event.type = "Birth"`
4. **Shared entities**: Notes/places referenced by multiple people become central nodes
5. **Evolution**: Start with inline events, migrate to event nodes later without rewriting

## Recommendation for SurrealDB

For the GEDCOM → SurrealDB exporter, I recommend:

1. **Always nodes**: Person, Family, Note (with pointer), Media (with pointer)
2. **Inline by default**: Birth, Death, Baptism, Burial (simple events)
3. **Conditional nodes**: Events with sources/notes/citations
4. **Always relationships**: parent_of, child_of, spouse_of, has_note, has_media

This balances query simplicity with data richness.

