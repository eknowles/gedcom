package v2

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/eknowles/gedcom/v39"
)

// Exporter exports GEDCOM data to SurrealDB format using the graph pattern
type Exporter struct {
	writer    io.Writer
	mapper    *GraphMapper
	namespace string
	database  string
	useJSON   bool
	err       error // Sticky error
}

// NewExporter creates a new v2 SurrealDB exporter
func NewExporter(w io.Writer, ns, db string) *Exporter {
	if ns == "" {
		ns = "main"
	}
	if db == "" {
		db = "main"
	}

	return &Exporter{
		writer:    w,
		mapper:    NewGraphMapper(),
		namespace: ns,
		database:  db,
		useJSON:   false,
	}
}

// SetMapper sets a custom graph mapper
func (e *Exporter) SetMapper(mapper *GraphMapper) {
	e.mapper = mapper
}

// SetUseJSON sets whether to use JSON format
func (e *Exporter) SetUseJSON(useJSON bool) {
	e.useJSON = useJSON
}

// w writes formatted output, storing errors
func (e *Exporter) w(format string, a ...interface{}) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintf(e.writer, format, a...)
}

// schemaDefinition contains the SurrealDB schema
const schemaDefinition = `-- Define tables with explicit schemas
DEFINE TABLE person SCHEMAFULL;
DEFINE FIELD pointer ON person TYPE string;
DEFINE FIELD name ON person TYPE option<object>;
DEFINE FIELD name.prefix ON person TYPE option<string>;
DEFINE FIELD name.given ON person TYPE option<string>;
DEFINE FIELD name.middle ON person TYPE option<string>;
DEFINE FIELD name.surname ON person TYPE option<string>;
DEFINE FIELD name.suffix ON person TYPE option<string>;
DEFINE FIELD name.nickname ON person TYPE option<string>;
DEFINE FIELD name.full ON person TYPE option<string>;
DEFINE FIELD sex ON person TYPE option<string>;
DEFINE FIELD is_living ON person TYPE bool;
DEFINE FIELD birth ON person TYPE option<object>;
DEFINE FIELD birth.date ON person TYPE option<string>;
DEFINE FIELD birth.date_range_start ON person TYPE option<datetime>;
DEFINE FIELD birth.date_range_end ON person TYPE option<datetime>;
DEFINE FIELD birth.place ON person TYPE option<string>;
DEFINE FIELD death ON person TYPE option<object>;
DEFINE FIELD death.date ON person TYPE option<string>;
DEFINE FIELD death.date_range_start ON person TYPE option<datetime>;
DEFINE FIELD death.date_range_end ON person TYPE option<datetime>;
DEFINE FIELD death.place ON person TYPE option<string>;
DEFINE FIELD baptism ON person TYPE option<object>;
DEFINE FIELD baptism.date ON person TYPE option<string>;
DEFINE FIELD baptism.date_range_start ON person TYPE option<datetime>;
DEFINE FIELD baptism.date_range_end ON person TYPE option<datetime>;
DEFINE FIELD baptism.place ON person TYPE option<string>;
DEFINE FIELD burial ON person TYPE option<object>;
DEFINE FIELD burial.date ON person TYPE option<string>;
DEFINE FIELD burial.date_range_start ON person TYPE option<datetime>;
DEFINE FIELD burial.date_range_end ON person TYPE option<datetime>;
DEFINE FIELD burial.place ON person TYPE option<string>;

DEFINE TABLE family SCHEMAFULL;
DEFINE FIELD pointer ON family TYPE string;
DEFINE FIELD husband_id ON family TYPE option<record<person>>;
DEFINE FIELD wife_id ON family TYPE option<record<person>>;

DEFINE TABLE note SCHEMAFULL;
DEFINE FIELD pointer ON note TYPE string;
DEFINE FIELD text ON note TYPE option<string>;

DEFINE TABLE media_object SCHEMAFULL;
DEFINE FIELD pointer ON media_object TYPE string;
DEFINE FIELD files ON media_object TYPE option<array<string>>;
DEFINE FIELD format ON media_object TYPE option<string>;
DEFINE FIELD title ON media_object TYPE option<string>;

-- Define relationship tables
DEFINE TABLE parent_of TYPE RELATION IN person OUT person;
DEFINE FIELD family_id ON parent_of TYPE option<record<family>>;

DEFINE TABLE spouse_of TYPE RELATION IN person OUT person;
DEFINE FIELD family_id ON spouse_of TYPE option<record<family>>;

DEFINE TABLE child_of TYPE RELATION IN person OUT person;
DEFINE FIELD family_id ON child_of TYPE option<record<family>>;

DEFINE TABLE has_note TYPE RELATION IN person | family OUT note;
DEFINE FIELD context ON has_note TYPE option<string>;

DEFINE TABLE has_media TYPE RELATION IN person | family OUT media_object;
DEFINE FIELD context ON has_media TYPE option<string>;

`

// Export exports the document to SurrealDB format
func (e *Exporter) Export(doc *gedcom.Document) error {
	// Write header
	if !e.useJSON {
		e.w("-- SurrealDB Export for GEDCOM (v2)\n")
		e.w("-- Generated: %s\n\n", time.Now().Format(time.RFC3339))
		e.w("OPTION IMPORT;\n\n")
		e.w("USE NS %s DB %s;\n\n", e.namespace, e.database)
		e.w(schemaDefinition)
	}

	// Extract entities and relationships using graph mapper
	entities, relationships, err := e.mapper.ExtractAll(doc)
	if err != nil {
		return err
	}

	// Phase 1: Write all entity nodes
	for _, entity := range entities {
		if err := e.writeNode(entity); err != nil {
			return err
		}
	}

	// Phase 2: Write all relationship edges
	for _, relationship := range relationships {
		if err := e.writeEdge(relationship); err != nil {
			return err
		}
	}

	return e.err
}

// writeNode writes a graph node to the output
func (e *Exporter) writeNode(node GraphNode) error {
	if node.ID == "" {
		return nil
	}

	// Clean properties
	cleanProps := cleanProperties(node.Properties)

	if e.useJSON {
		jsonData, err := json.Marshal(cleanProps)
		if err != nil {
			return err
		}
		e.w("%s\n", jsonData)
		return e.err
	}

	// SurrealQL format
	contentStr := formatContent(cleanProps)
	e.w("CREATE %s:%s CONTENT %s;\n", node.Type, node.ID, contentStr)

	return e.err
}

// writeEdge writes a graph edge to the output
func (e *Exporter) writeEdge(edge GraphEdge) error {
	if edge.FromID == "" || edge.ToID == "" {
		return nil
	}

	// Clean properties
	cleanProps := cleanProperties(edge.Properties)

	if e.useJSON {
		allData := map[string]interface{}{
			"in":   edge.FromID,
			"out":  edge.ToID,
			"data": cleanProps,
		}
		jsonData, err := json.Marshal(allData)
		if err != nil {
			return err
		}
		e.w("%s\n", jsonData)
		return e.err
	}

	// SurrealQL format
	var contentStr string
	if len(cleanProps) > 0 {
		contentStr = fmt.Sprintf(" CONTENT %s", formatContent(cleanProps))
	}

	// Use entity types from edge (don't guess from ID)
	fromTable := string(edge.FromType)
	toTable := string(edge.ToType)

	e.w("RELATE %s:%s->%s->%s:%s%s;\n",
		fromTable, edge.FromID,
		edge.Type,
		toTable, edge.ToID,
		contentStr)

	return e.err
}

// cleanProperties removes nil and empty values
func cleanProperties(props map[string]interface{}) map[string]interface{} {
	cleaned := make(map[string]interface{})

	for k, v := range props {
		if v == nil || v == "" {
			continue
		}

		// Remove empty maps
		if m, ok := v.(map[string]interface{}); ok && len(m) == 0 {
			continue
		}

		// Remove empty slices
		if s, ok := v.([]interface{}); ok && len(s) == 0 {
			continue
		}
		if s, ok := v.([]string); ok && len(s) == 0 {
			continue
		}

		cleaned[k] = v
	}

	return cleaned
}

// formatContent formats properties as SurrealQL content
func formatContent(props map[string]interface{}) string {
	var parts []string

	for key, value := range props {
		var valueStr string

		// Handle record links (ends with _id)
		if strVal, ok := value.(string); ok && strings.HasSuffix(key, "_id") {
			if strings.Contains(strVal, ":") {
				// Already formatted as record link
				valueStr = strVal
			} else {
				jsonBytes, _ := json.Marshal(value)
				valueStr = string(jsonBytes)
			}
		} else {
			// Marshal normally
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				continue
			}
			valueStr = string(jsonBytes)
		}

		parts = append(parts, fmt.Sprintf("%q: %s", key, valueStr))
	}

	// Sort for consistent output
	sort.Strings(parts)

	return "{" + strings.Join(parts, ", ") + "}"
}

// extractTableFromID extracts the table name from an ID like "person:i1" or just "i1"
func extractTableFromID(id string) string {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) == 2 {
		return parts[0]
	}

	// Guess from ID prefix
	if strings.HasPrefix(id, "i") {
		return "person"
	} else if strings.HasPrefix(id, "f") {
		return "family"
	} else if strings.HasPrefix(id, "n") {
		return "note"
	} else if strings.HasPrefix(id, "m") {
		return "media_object"
	}

	return "unknown"
}

// extractIDFromTableID extracts the ID part from "table:id" or returns the whole string
func extractIDFromTableID(id string) string {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return id
}

