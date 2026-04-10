# AGENTS.md

AI coding agent instructions for the `gedcom` project - a Go library and CLI toolkit for parsing, manipulating, comparing, and publishing GEDCOM genealogy files.

Do not create markdown files unless the user explicitly states to do so, you may offer to create markdown files, but only do so with explicit user approval.

## Architecture Overview

This is a **tree-based parser/processor** for GEDCOM (genealogical data) files with three distinct layers:

1. **Core Package (`gedcom/`)**: Node-based tree structure with typed nodes for GEDCOM entities
2. **Query Language (`q/`)**: Expression parser/engine called "gedcomq" - a jq-inspired query DSL
3. **HTML Publisher (`html/`)**: Static HTML site generator from GEDCOM data
4. **CLI Commands (`cmd/gedcom/`)**: Command-line tools (diff, publish, query, warnings, surrealdb export)

### Node System

The fundamental abstraction is the `Node` interface (see `node.go`). All GEDCOM entities are nodes:

- **SimpleNode**: Base implementation with tag, value, pointer, and children
- **Typed Nodes**: Specialized types like `IndividualNode`, `FamilyNode`, `DateNode`, `BirthNode`, etc.
- **Document**: Root container with pointer cache (`sync.Map`) for O(1) node lookup

Key pattern: Nodes are created via `NewNode()` factory which dispatches to specific constructors based on `Tag` constants (see `decoder.go:271-375`). Tags like `TagBirth`, `TagIndividual`, `TagFamily` determine node type.

### Critical Relationships

**Individual ↔ Family linkage** is central but NOT stored in nodes directly:
- Families reference individuals via pointer values (e.g., `@P1@` in HUSB/WIFE/CHIL tags)
- `Document.NodeByPointer()` resolves these references using the pointer cache
- Methods like `IndividualNode.Families()` traverse `Document.Families()` to find relationships
- Cache invalidation: Adding individuals/families calls `resetCache()` on all affected nodes

**Parent-child structure**:
- Every node can have children via `Nodes()` and `AddNode()`
- Indentation in GEDCOM reflects depth (0 = root, 1 = child of level 0, etc.)

## Testing & Quality

### Test Organization

- Test files are `*_test.go` alongside implementation (standard Go convention)
- Use `github.com/stretchr/testify/assert` for assertions
- Use `github.com/elliotchance/tf` for table-driven function tests (see `age_test.go`, `gedcomer_test.go`)
- Pattern: `var decoderTests = map[string]struct{...}` for test suites (see `decoder_test.go`)

### Running Tests

```bash
# All tests with race detector
make test

# Coverage report
make test-coverage
```

### Warnings System

Built-in validation warnings (NOT errors) detect genealogical issues:
- `IncorrectEventOrderWarning`: Birth after death, burial before death, etc.
- `IndividualTooOldWarning`: Age exceeds `DefaultMaxLivingAge` (100 years)
- `ChildBornBeforeParentWarning`: Child born before parent
- `MarriedOutOfRangeWarning`: Marriage age < 16 or > 100

Warning pattern: Implement `Warner` interface, embed `SimpleWarning`, attach `WarningContext` linking to Individual/Family. See `individual_node.go:1033-1043` for aggregation.

## Build & Development

### Module Structure

- Module path: `github.com/eknowles/gedcom/v39` (Go 1.26)
- Forked from `github.com/elliotchance/gedcom` (note mixed references in code)
- CLI binary: `cmd/gedcom/main.go` with cobra subcommands

### Building

```bash
# Build CLI
make zip   # Creates bin/gedcom and gedcom-*.zip

# Build for SurrealDB workflow
make sql   # Builds, exports GEDCOM to .surql, imports to SurrealDB
```

### CLI Commands

```bash
gedcom diff -left-gedcom a.ged -right-gedcom b.ged -output diff.html
gedcom publish -gedcom tree.ged -output-dir ./site
gedcom query -gedcom tree.ged '.Individuals | Length'
gedcom warnings tree.ged
gedcom surrealdb -gedcom tree.ged -output export.surql
```

## Key Conventions

### Decoder Flags

Two critical decoder options handle malformed GEDCOM:
- `AllowMultiLine`: Permits values spanning multiple lines (invalid per spec but common)
- `AllowInvalidIndents`: Tolerates child indents > parent+1 (see `decoder.go:50-62`)

Both default to false. Enable when dealing with broken files from genealogy software exports.

### Nil Handling

Every accessor method checks for nil and returns safe defaults:
```go
// If the node is nil the result will also be nil.
func (node *IndividualNode) Name() *NameNode
```
Comment pattern "If the node is nil..." appears throughout. Use `IsNil()` helper (handles interface nil).

### GEDCOM String Representation

Two methods for rendering:
- `GEDCOMLine(indent int)`: Single line with tag/value/pointer
- `GEDCOMString(indent int)`: Recursive including children
- `NoIndent` constant (-1): Excludes indent prefix for comparisons

### Tags System

Tags are strongly typed constants (see `tag.go`):
- Official: `TagBirth`, `TagDeath`, `TagMarriage` (GEDCOM 5.5 standard)
- Unofficial: `UnofficialTagFamilySearchID1`, `UnofficialTagUniqueID` (prefix `_`)
- Test with `tag.IsOfficial()`, `tag.IsEvent()`
- Convert: `TagFromString("BIRT")` → `TagBirth`

Event tags (`tagOptionEvent` flag) typically have DATE/PLAC children.

## Comparison & Merging

### Similarity Algorithm

`IndividualNodes.Compare()` uses **surrounding similarity** - not just name/date matching:
1. Individual similarity (name, birth, death)
2. Parents similarity
3. Spouses similarity  
4. Children similarity

Weighted combination in `SurroundingSimilarity` (see `individual_node.go:467-515`). Default minimum: `DefaultMinimumSimilarity = 0.733`.

### Merge Process

```go
merged, err := individuals1.Merge(individuals2, doc, compareOptions)
```

- Pointer preference: `PreferPointerAbove` threshold forces pointer matches
- Unmatched individuals appended to result
- Uses `MergeNodes()` to combine matching individuals' data
- Document required for node attachment

Pattern: Always provide `IndividualNodesCompareOptions` with `SimilarityOptions`.

## Query Language (gedcomq)

Located in `q/` package. See `q/doc.go` for extensive documentation.

Expression types:
- **Accessors**: `.Individuals`, `.Name`, `.Birth`
- **Functions**: `Length`, `First(n)`, `Last(n)`, `Only(condition)`, `Combine(...)`
- **Operators**: `=`, `!=`, `>`, `>=`, `<`, `<=`
- **Variables**: `Names are .Individuals | .Name; Names | .String`
- **Objects**: `{name: .Name | .String, born: .Birth | .String}`

Formatters: JSON, pretty JSON, CSV, GEDCOM (see `q/*_formatter.go`).

## Common Patterns

### Creating Documents

```go
doc := gedcom.NewDocument()
person := doc.AddIndividual("P1",
    gedcom.NewNameNode("John /Doe/"),
    gedcom.NewBirthNode("", gedcom.NewDateNode("1 Jan 1900")),
)
```

### Traversing Relationships

```go
// Get all families individual is part of
families := individual.Families()

// Get spouses
spouses := individual.Spouses()

// Get parents
parents := individual.Parents()  // Returns FamilyNodes

// Get children grouped by spouse
children := individual.ChildrenWithSpouses()  // Returns SpouseChildren map
```

### Date Handling

`DateNode` supports constraints:
- `DateConstraintExact`: Precise date
- `DateConstraintAbout`: Approximate (~)
- `DateConstraintBefore`: Before date (<)
- `DateConstraintAfter`: After date (>)

Parse with `DateConstraintFromString("Abt")`.

### Working with Nodes

```go
// Get children by tag
births := gedcom.NodesWithTag(individual, gedcom.TagBirth)

// Deep copy
copied := gedcom.DeepCopy(node, document)

// Shallow copy
shallow := node.ShallowCopy()

// Equality
if node.Equals(other) { }       // Shallow
if gedcom.DeepEqual(a, b) { }   // Recursive
```

## Performance Considerations

- Pointer cache built once on decode: `document.buildPointerCache()`
- Individual/Family lookups cached on nodes (invalidated on document mutations)
- Comparison is O(n²) - use `IndividualNodesCompareOptions.Jobs` to parallelize
- Large trees: Use `gedcomq` with filters to subset data before processing

## Exporters

See `/EXPORTER_PATTERN.md` and `/EXPORTER_PATTERN_GRAPH.md` for patterns on building exporters:

- **Property Extractor Pattern**: Systematic approach using tag.go for flat exports (CSV, JSON)
- **Graph Exporter Pattern**: Extended pattern for graph databases (SurrealDB) with nodes and relationships
- **TDD Approach**: Write tests before implementation at each layer (extractors → mappers → exporters)

The tag system (`tag.go`) is central to exporters:
- `Tags()` function lists all ~100 available tags
- `tag.IsEvent()` identifies event tags (Birth, Death, Marriage, etc.)
- `tag.IsOfficial()` distinguishes standard vs unofficial tags
- TagPath navigation: `[TagBirth, TagDate]` finds DATE under BIRT

## File Locations

- Core types: `individual_node.go`, `family_node.go`, `document.go`, `node.go`
- Parser: `decoder.go`, `encoder.go`
- Comparison: `individual_nodes.go`, `individual_comparison.go`, `merge.go`
- Warnings: `*_warning.go` files
- CLI: `cmd/gedcom/*.go`
- Exporters: `exporters/surrealdb/` (reference implementation)
- Tests: `*_test.go` everywhere


