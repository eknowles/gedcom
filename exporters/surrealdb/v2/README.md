# SurrealDB Exporter v2

A systematic graph-based exporter for GEDCOM data following the **Graph Exporter Pattern**.

## Overview

This v2 exporter follows a pattern-based approach for exporting GEDCOM genealogical data to SurrealDB, implementing the architecture described in `/EXPORTER_PATTERN_GRAPH.md`. It provides a flexible, extensible system for mapping GEDCOM entities to graph database nodes and relationships.

## Key Features

- **Pattern-based architecture**: Uses EntityExtractor and RelationshipExtractor abstractions
- **Hybrid event strategy**: Inline simple events, create nodes for complex ones
- **Property extraction**: Systematic TagPath navigation for nested properties
- **Bidirectional relationships**: Automatic handling of spouse relationships
- **Full test coverage**: Comprehensive TDD approach with 100% passing tests

## Architecture

### Core Components

1. **EntityExtractor**: Defines how GEDCOM entities become graph nodes
2. **RelationshipExtractor**: Defines how entities are connected
3. **PropertyExtractor**: Extracts properties using TagPath navigation
4. **GraphMapper**: Orchestrates entity and relationship extraction
5. **Exporter**: Outputs SurrealQL or JSON format

### Entity Types

- `person` - Individuals with inline birth/death/baptism/burial events
- `family` - Family units linking spouses and children
- `note` - Notes with pointers (shared references)
- `media_object` - Media objects with pointers (photos, documents)

### Relationship Types

- `spouse_of` - Bidirectional spouse relationships
- `parent_of` - Parent to child relationships
- `child_of` - Child to parent relationships
- `has_note` - Links to note entities
- `has_media` - Links to media objects

## Usage

### Basic Export

```go
package main

import (
    "os"
    "github.com/eknowles/gedcom/v39"
    "github.com/eknowles/gedcom/v39/exporters/surrealdb/v2"
)

func main() {
    // Load GEDCOM file
    f, _ := os.Open("family.ged")
    defer f.Close()
    
    decoder := gedcom.NewDecoder(f)
    doc, _ := decoder.Decode()
    
    // Create exporter
    out, _ := os.Create("output.surql")
    defer out.Close()
    
    exporter := v2.NewExporter(out, "genealogy", "family_tree")
    
    // Export
    err := exporter.Export(doc)
    if err != nil {
        panic(err)
    }
}
```

### Custom Mapper

You can customize which entities and relationships are exported:

```go
// Create custom mapper with only specific extractors
customMapper := &v2.GraphMapper{
    Entities: []v2.EntityExtractor{
        v2.PersonEntityExtractor(),
        v2.FamilyEntityExtractor(),
        // Exclude notes and media
    },
    Relationships: []v2.RelationshipExtractor{
        v2.SpouseOfRelationship(),
        v2.ParentOfRelationship(),
        v2.ChildOfRelationship(),
        // Exclude note/media relationships
    },
    EventStrategy: v2.DefaultEventStrategy(),
}

exporter := v2.NewExporter(out, "test", "db")
exporter.SetMapper(customMapper)
exporter.Export(doc)
```

### JSON Output

For bulk import or debugging:

```go
exporter := v2.NewExporter(out, "test", "db")
exporter.SetUseJSON(true)
err := exporter.Export(doc)
```

## Output Format

### SurrealQL (Default)

```sql
-- SurrealDB Export for GEDCOM (v2)
-- Generated: 2026-04-10T10:03:36+01:00

OPTION IMPORT;
USE NS genealogy DB family_tree;

-- Schema definitions...
DEFINE TABLE person SCHEMAFULL;
DEFINE FIELD pointer ON person TYPE string;
DEFINE FIELD name ON person TYPE option<string>;
-- ...

-- Entity creation
CREATE person:i1 CONTENT {
    "pointer": "I1",
    "name": "John Smith",
    "given_name": "John",
    "surname": "Smith",
    "sex": "M",
    "is_living": false,
    "birth": {
        "date": "1 Jan 1900",
        "date_range_start": "1900-01-01T00:00:00Z",
        "date_range_end": "1900-01-01T23:59:59Z",
        "place": "London"
    }
};

-- Relationships
RELATE person:i1->spouse_of->person:i2 CONTENT {"family_id": family:f1};
RELATE person:i1->parent_of->person:i3 CONTENT {"family_id": family:f1};
```

## Event Strategy

The hybrid event strategy balances simplicity and richness:

- **Simple events** (only date/place): Inlined as nested objects
- **Complex events** (with notes/sources/media): Could be separate nodes (future enhancement)

### Event Detection

```go
strategy := v2.DefaultEventStrategy()
// Returns EventStrategy{
//     InlineIfSimple: true,
//     ComplexityTags: []gedcom.Tag{
//         gedcom.TagNote,
//         gedcom.TagSource,
//         gedcom.TagObject,
//     },
// }

isComplex := strategy.ShouldCreateNode(birthNode)
```

## Property Extraction

Properties are extracted using a systematic approach:

```go
// Direct property
{
    Name: "pointer",
    Transform: CommonTransforms.PointerValue,
}

// Tag path navigation
{
    Name: "birth_date",
    TagPath: []gedcom.Tag{gedcom.TagBirth, gedcom.TagDate},
}

// Custom transform
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
}
```

## Common Transforms

Pre-built transform functions for common patterns:

- `CommonTransforms.DateString` - Extract date as string
- `CommonTransforms.DateRangeStart` - Extract date range start
- `CommonTransforms.DateRangeEnd` - Extract date range end
- `CommonTransforms.PlaceString` - Extract place name
- `CommonTransforms.NameString` - Extract formatted name
- `CommonTransforms.GivenName` - Extract given name
- `CommonTransforms.Surname` - Extract surname
- `CommonTransforms.FullText` - Extract text with CONT/CONC
- `CommonTransforms.PointerValue` - Extract pointer
- `CommonTransforms.SexString` - Extract sex value
- `CommonTransforms.IsLiving` - Compute living status

## Testing

Run the test suite:

```bash
cd exporters/surrealdb/v2
go test -v
```

All tests follow TDD principles:
- Property extraction tests
- Entity extraction tests
- Relationship extraction tests
- Full integration tests

## Schema

The exporter creates a comprehensive SurrealDB schema with:

### Tables
- `person` - Individual people
- `family` - Family units
- `note` - Shared notes
- `media_object` - Media files

### Relationships
- `spouse_of` - Bidirectional
- `parent_of` - Parent → Child
- `child_of` - Child → Parent
- `has_note` - Entity → Note
- `has_media` - Entity → Media

## Comparison with v1

| Feature | v1 | v2 |
|---------|----|----|
| Architecture | Monolithic | Pattern-based |
| Extensibility | Limited | Highly extensible |
| Event handling | Fixed inline | Configurable strategy |
| Property extraction | Manual | Systematic |
| Relationship deduplication | Manual | Automatic |
| Test coverage | Good | Comprehensive |

## Future Enhancements

Potential additions (not yet implemented):

1. **Event nodes**: Create separate nodes for complex events
2. **Place entities**: Deduplicated place nodes
3. **Source entities**: Bibliography and citations
4. **Custom extractors**: Runtime registration of extractors
5. **Context tracking**: Event context in relationships

## Performance

- **Deduplication**: Bidirectional relationships automatically deduplicated
- **Pointer cache**: Document-level caching from base library
- **Two-phase export**: Entities first, then relationships
- **Memory efficient**: Streaming output

## References

- Pattern documentation: `/EXPORTER_PATTERN_GRAPH.md`
- Base GEDCOM library: `github.com/eknowles/gedcom/v39`
- SurrealDB: https://surrealdb.com/

## License

Same as parent project.

