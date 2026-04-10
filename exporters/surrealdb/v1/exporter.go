package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/eknowles/gedcom/v39"
)

// Exporter exports a GEDCOM document to SurrealDB format.
type Exporter struct {
	writer    io.Writer
	namespace string
	database  string
	useJSON   bool
	err       error // Sticky error to reduce boilerplate
}

// NewExporter creates a new SurrealDB exporter.
// If namespace and database are empty, they default to "main" and "main" respectively.
func NewExporter(w io.Writer, ns, db string) *Exporter {
	if ns == "" {
		ns = "main"
	}
	if db == "" {
		db = "main"
	}
	return &Exporter{
		writer:    w,
		namespace: ns,
		database:  db,
		useJSON:   false,
	}
}

// SetUseJSON sets whether to use JSON format for SurrealDB import.
// Default is false (uses SurrealQL statements).
func (e *Exporter) SetUseJSON(useJSON bool) {
	e.useJSON = useJSON
}

// w is a helper to reduce 'if err != nil' repetition.
// It writes formatted output to the writer, storing any error in e.err.
// If e.err is already set, the write is skipped.
func (e *Exporter) w(format string, a ...interface{}) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintf(e.writer, format, a...)
}

// schemaDefinition contains the SurrealDB schema for GEDCOM data.
const schemaDefinition = `-- Define tables with explicit schemas
DEFINE TABLE person SCHEMAFULL;
DEFINE FIELD pointer ON person TYPE string;
DEFINE FIELD name ON person TYPE option<string>;
DEFINE FIELD given_name ON person TYPE option<string>;
DEFINE FIELD surname ON person TYPE option<string>;
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
DEFINE FIELD security_group ON person TYPE option<string>;
DEFINE FIELD physical_description ON person TYPE option<object>;
DEFINE FIELD physical_description.height ON person TYPE option<string>;
DEFINE FIELD physical_description.eyes ON person TYPE option<string>;
DEFINE FIELD physical_description.hair ON person TYPE option<string>;
DEFINE FIELD residency ON person TYPE option<array<object>>;
DEFINE FIELD residency.*.date ON person TYPE option<string>;
DEFINE FIELD residency.*.place ON person TYPE option<record<place>>;
DEFINE FIELD education ON person TYPE option<array<object>>;
DEFINE FIELD education.*.date ON person TYPE option<string>;
DEFINE FIELD education.*.place ON person TYPE option<record<place>>;

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
DEFINE FIELD notes ON media_object TYPE option<array<string>>;

DEFINE TABLE place SCHEMAFULL;
DEFINE FIELD name ON place TYPE string;
DEFINE FIELD address ON place TYPE option<string>;
DEFINE FIELD coordinates ON place TYPE option<geometry<point>>;

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

// Export exports the entire document to SurrealDB format.
// Export exports the entire document to SurrealDB format.
func (e *Exporter) Export(doc *gedcom.Document) error {
	if !e.useJSON {
		e.w("-- SurrealDB Export for GEDCOM\n")
		e.w("-- Generated: %s\n\n", time.Now().Format(time.RFC3339))
		e.w("OPTION IMPORT;\n\n")
		e.w("USE NS %s DB %s;\n\n", e.namespace, e.database)
		e.w(schemaDefinition)
	}

	// Export individuals as person records
	for _, individual := range doc.Individuals() {
		if err := e.exportIndividual(individual); err != nil {
			return err
		}
	}

	// Export families and relationships
	for _, family := range doc.Families() {
		if err := e.exportFamily(family); err != nil {
			return err
		}
	}

	// Export notes
	for _, node := range doc.Nodes() {
		if noteNode, ok := node.(*gedcom.NoteNode); ok {
			if err := e.exportNote(noteNode); err != nil {
				return err
			}
		}
	}

	// Export media objects (OBJE)
	for _, node := range doc.Nodes() {
		if node.Tag().Is(gedcom.TagObject) && node.Pointer() != "" {
			if err := e.exportMediaObject(node); err != nil {
				return err
			}
		}
	}

	return e.err
}

// mapEvent is a generic helper to map GEDCOM events (Birth, Death, Baptism, Burial)
// to a common structure with date and place information.
func (e *Exporter) mapEvent(date *gedcom.DateNode, place *gedcom.PlaceNode) map[string]interface{} {
	if date == nil && place == nil {
		return nil
	}
	m := make(map[string]interface{})
	if date != nil && date.IsValid() {
		m["date"] = date.String()
		m["date_range_start"] = date.DateRange().StartDate().Time().Format(time.RFC3339)
		m["date_range_end"] = date.DateRange().EndDate().Time().Format(time.RFC3339)
	}
	if place != nil {
		m["place"] = place.String()
	}
	return m
}

// exportIndividual exports a single individual to SurrealDB.
func (e *Exporter) exportIndividual(individual *gedcom.IndividualNode) error {
	if individual == nil {
		return nil
	}

	id := e.sanitizeID(individual.Pointer())
	if id == "" {
		return nil
	}

	data := map[string]interface{}{
		"pointer":   individual.Pointer(),
		"is_living": individual.IsLiving(),
	}

	// Name
	if name := individual.Name(); name != nil {
		data["name"] = name.String()
		if givenName := name.GivenName(); givenName != "" {
			data["given_name"] = givenName
		}
		if surname := name.Surname(); surname != "" {
			data["surname"] = surname
		}
	}

	// Sex
	if sex := individual.Sex(); sex != nil {
		data["sex"] = sex.String()
	}

	// Map events using the generic helper
	data["birth"] = e.mapEvent(individual.Birth())
	data["death"] = e.mapEvent(individual.Death())
	data["baptism"] = e.mapEvent(individual.Baptism())
	data["burial"] = e.mapEvent(individual.Burial())

	// Write person record first
	if err := e.writeRecord("person", id, data); err != nil {
		return err
	}

	// Export note relationships for the individual
	if err := e.exportNodeNoteRelationships("person", id, individual); err != nil {
		return err
	}

	// Export media object relationships for the individual
	if err := e.exportNodeMediaRelationships("person", id, individual); err != nil {
		return err
	}

	return nil
}

// exportFamily exports family relationships to SurrealDB.
func (e *Exporter) exportFamily(family *gedcom.FamilyNode) error {
	if family == nil {
		return nil
	}

	familyID := e.sanitizeID(family.Pointer())
	if familyID == "" {
		return nil
	}

	// Create family record
	familyData := map[string]interface{}{
		"pointer": family.Pointer(),
	}

	husband := family.Husband()
	wife := family.Wife()

	if husband != nil {
		husbandIndividual := husband.Individual()
		if husbandIndividual != nil {
			familyData["husband_id"] = fmt.Sprintf("person:%s", e.sanitizeID(husbandIndividual.Pointer()))
		}
	}

	if wife != nil {
		wifeIndividual := wife.Individual()
		if wifeIndividual != nil {
			familyData["wife_id"] = fmt.Sprintf("person:%s", e.sanitizeID(wifeIndividual.Pointer()))
		}
	}

	// Write family record
	if err := e.writeRecord("family", familyID, familyData); err != nil {
		return err
	}

	// Create spouse relationships
	if husband != nil && wife != nil {
		husbandIndividual := husband.Individual()
		wifeIndividual := wife.Individual()

		if husbandIndividual != nil && wifeIndividual != nil {
			husbandID := e.sanitizeID(husbandIndividual.Pointer())
			wifeID := e.sanitizeID(wifeIndividual.Pointer())

			// Create SPOUSE_OF relationship
			relData := map[string]interface{}{
				"family_id": fmt.Sprintf("family:%s", familyID),
			}

			if err := e.writeRelation("spouse_of", fmt.Sprintf("person:%s", husbandID), fmt.Sprintf("person:%s", wifeID), relData); err != nil {
				return err
			}
			if err := e.writeRelation("spouse_of", fmt.Sprintf("person:%s", wifeID), fmt.Sprintf("person:%s", husbandID), relData); err != nil {
				return err
			}
		}
	}

	// Create parent-child relationships
	children := family.Children()
	for _, child := range children {
		childIndividual := child.Individual()
		if childIndividual == nil {
			continue
		}

		childID := e.sanitizeID(childIndividual.Pointer())

		// CHILD_OF relationships
		if husband != nil {
			husbandIndividual := husband.Individual()
			if husbandIndividual != nil {
				husbandID := e.sanitizeID(husbandIndividual.Pointer())
				relData := map[string]interface{}{
					"family_id": fmt.Sprintf("family:%s", familyID),
				}
				if err := e.writeRelation("child_of", fmt.Sprintf("person:%s", childID), fmt.Sprintf("person:%s", husbandID), relData); err != nil {
					return err
				}
				// PARENT_OF relationship (reverse)
				if err := e.writeRelation("parent_of", fmt.Sprintf("person:%s", husbandID), fmt.Sprintf("person:%s", childID), relData); err != nil {
					return err
				}
			}
		}

		if wife != nil {
			wifeIndividual := wife.Individual()
			if wifeIndividual != nil {
				wifeID := e.sanitizeID(wifeIndividual.Pointer())
				relData := map[string]interface{}{
					"family_id": fmt.Sprintf("family:%s", familyID),
				}
				if err := e.writeRelation("child_of", fmt.Sprintf("person:%s", childID), fmt.Sprintf("person:%s", wifeID), relData); err != nil {
					return err
				}
				// PARENT_OF relationship (reverse)
				if err := e.writeRelation("parent_of", fmt.Sprintf("person:%s", wifeID), fmt.Sprintf("person:%s", childID), relData); err != nil {
					return err
				}
			}
		}
	}

	// Export note relationships for the family
	if err := e.exportNodeNoteRelationships("family", familyID, family); err != nil {
		return err
	}

	// Export media object relationships for the family
	if err := e.exportNodeMediaRelationships("family", familyID, family); err != nil {
		return err
	}

	return nil
}

// exportNote exports a single note to SurrealDB.
func (e *Exporter) exportNote(note *gedcom.NoteNode) error {
	if note == nil {
		return nil
	}

	pointer := note.Pointer()
	if pointer == "" {
		return nil // Skip inline notes without pointers
	}

	id := e.sanitizeID(pointer)
	if id == "" {
		return nil
	}

	data := map[string]interface{}{
		"pointer": pointer,
	}

	// Add the note text content
	if note.Value() != "" {
		data["text"] = note.Value()
	}

	// Extract any continuation lines (CONT/CONC tags)
	var fullText string
	fullText = note.Value()
	for _, child := range note.Nodes() {
		if child.Tag().Is(gedcom.TagContinued) {
			fullText += "\n" + child.Value()
		} else if child.Tag().Is(gedcom.TagConcatenation) {
			fullText += child.Value()
		}
	}
	if fullText != "" {
		data["text"] = fullText
	}

	return e.writeRecord("note", id, data)
}

// exportMediaObject exports a single media object (OBJE) to SurrealDB.
func (e *Exporter) exportMediaObject(node gedcom.Node) error {
	if node == nil {
		return nil
	}

	pointer := node.Pointer()
	if pointer == "" {
		return nil
	}

	id := e.sanitizeID(pointer)
	if id == "" {
		return nil
	}

	data := map[string]interface{}{
		"pointer": pointer,
	}

	// Extract FILE tags
	files := gedcom.NodesWithTag(node, gedcom.TagFile)
	if len(files) > 0 {
		fileList := make([]string, 0, len(files))
		for _, fileNode := range files {
			if fileNode.Value() != "" {
				fileList = append(fileList, fileNode.Value())
			}
		}
		if len(fileList) > 0 {
			data["files"] = fileList
		}
	}

	// Extract FORM (format) tag
	formatNodes := gedcom.NodesWithTag(node, gedcom.TagFormat)
	if len(formatNodes) > 0 && formatNodes[0].Value() != "" {
		data["format"] = formatNodes[0].Value()
	}

	// Extract TITL (title) tag
	titleNodes := gedcom.NodesWithTag(node, gedcom.TagTitle)
	if len(titleNodes) > 0 && titleNodes[0].Value() != "" {
		data["title"] = titleNodes[0].Value()
	}

	// Extract any notes associated with the media object
	noteNodes := gedcom.NodesWithTag(node, gedcom.TagNote)
	if len(noteNodes) > 0 {
		noteList := make([]string, 0, len(noteNodes))
		for _, noteNode := range noteNodes {
			if noteNode.Value() != "" {
				noteList = append(noteList, noteNode.Value())
			} else if noteNode.Pointer() != "" {
				// Reference to another note record
				noteList = append(noteList, fmt.Sprintf("note:%s", e.sanitizeID(noteNode.Pointer())))
			}
		}
		if len(noteList) > 0 {
			data["notes"] = noteList
		}
	}

	return e.writeRecord("media_object", id, data)
}

// writeRecord writes a SurrealDB record.
func (e *Exporter) writeRecord(table, id string, data map[string]interface{}) error {
	// Cleanup nil entries before writing
	for k, v := range data {
		if v == nil || v == "" {
			delete(data, k)
		}
		// Also remove empty maps
		if m, ok := v.(map[string]interface{}); ok && len(m) == 0 {
			delete(data, k)
		}
	}

	if e.useJSON {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(e.writer, "%s\n", jsonData)
		return err
	}

	// SurrealQL format with proper record link handling
	var parts []string
	for key, value := range data {
		var valueStr string

		// Check if this is a record link field (ends with _id and value is a string starting with "person:" or "family:")
		if strVal, ok := value.(string); ok && strings.HasSuffix(key, "_id") {
			if strings.HasPrefix(strVal, "person:") || strings.HasPrefix(strVal, "family:") {
				// This is a record link - output without quotes
				valueStr = strVal
			} else {
				// Regular string
				jsonBytes, _ := json.Marshal(value)
				valueStr = string(jsonBytes)
			}
		} else {
			// Marshal any other type normally
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				return err
			}
			valueStr = string(jsonBytes)
		}

		parts = append(parts, fmt.Sprintf("%q: %s", key, valueStr))
	}

	// Sort for consistent output
	sort.Strings(parts)

	contentStr := "{" + strings.Join(parts, ", ") + "}"
	_, err := fmt.Fprintf(e.writer, "CREATE %s:%s CONTENT %s;\n", table, id, contentStr)
	return err
}

// writeRelation writes a SurrealDB relationship edge.
func (e *Exporter) writeRelation(relationType, fromID, toID string, data map[string]interface{}) error {
	if e.useJSON {
		allData := map[string]interface{}{
			"in":   fromID,
			"out":  toID,
			"data": data,
		}
		jsonData, err := json.Marshal(allData)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(e.writer, "%s\n", jsonData)
		return err
	}

	// SurrealQL format with proper record link handling
	var contentStr string
	if len(data) > 0 {
		var parts []string
		for key, value := range data {
			var valueStr string

			// Check if this is a record link field (ends with _id and value is a string starting with "person:" or "family:")
			if strVal, ok := value.(string); ok && strings.HasSuffix(key, "_id") {
				if strings.HasPrefix(strVal, "person:") || strings.HasPrefix(strVal, "family:") {
					// This is a record link - output without quotes
					valueStr = strVal
				} else {
					// Regular string
					jsonBytes, _ := json.Marshal(value)
					valueStr = string(jsonBytes)
				}
			} else {
				// Marshal any other type normally
				jsonBytes, err := json.Marshal(value)
				if err != nil {
					return err
				}
				valueStr = string(jsonBytes)
			}

			parts = append(parts, fmt.Sprintf("%q: %s", key, valueStr))
		}

		// Sort for consistent output
		sort.Strings(parts)

		contentStr = fmt.Sprintf(" CONTENT {%s}", strings.Join(parts, ", "))
	}

	_, err := fmt.Fprintf(e.writer, "RELATE %s->%s->%s%s;\n", fromID, relationType, toID, contentStr)
	return err
}

// exportNodeNoteRelationships exports relationships between a node (person/family) and its notes.
func (e *Exporter) exportNodeNoteRelationships(sourceTable, sourceID string, node gedcom.Node) error {
	if node == nil {
		return nil
	}

	// Get all NOTE tags from the node
	noteNodes := gedcom.NodesWithTag(node, gedcom.TagNote)

	for _, noteNode := range noteNodes {
		var notePointer string

		// Check if this is a pointer-based reference (either in Pointer() or Value())
		if noteNode.Pointer() != "" {
			// Direct pointer reference (less common for NOTE tags)
			notePointer = noteNode.Pointer()
		} else if noteNode.Value() != "" {
			// Value-based reference (e.g., "@N1@")
			notePointer = valueToPointer(noteNode.Value())
		}

		// Skip inline notes that don't have pointers
		if notePointer == "" {
			continue
		}

		noteID := e.sanitizeID(notePointer)
		if noteID == "" {
			continue
		}

		// Create has_note relationship
		relData := map[string]interface{}{}

		if err := e.writeRelation("has_note",
			fmt.Sprintf("%s:%s", sourceTable, sourceID),
			fmt.Sprintf("note:%s", noteID),
			relData); err != nil {
			return err
		}
	}

	// Also check for notes attached to events (BIRT, DEAT, MARR, etc.)
	for _, childNode := range node.Nodes() {
		if childNode.Tag().IsEvent() {
			eventNotes := gedcom.NodesWithTag(childNode, gedcom.TagNote)
			for _, eventNote := range eventNotes {
				if eventNote.Pointer() != "" {
					noteID := e.sanitizeID(eventNote.Pointer())
					if noteID == "" {
						continue
					}

					// Create has_note relationship with event context
					relData := map[string]interface{}{
						"context": childNode.Tag().String(), // e.g., "Birth", "Death", "Marriage"
					}

					if err := e.writeRelation("has_note",
						fmt.Sprintf("%s:%s", sourceTable, sourceID),
						fmt.Sprintf("note:%s", noteID),
						relData); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// exportNodeMediaRelationships exports relationships between a node (person/family) and its media objects.
func (e *Exporter) exportNodeMediaRelationships(sourceTable, sourceID string, node gedcom.Node) error {
	if node == nil {
		return nil
	}

	// Get all OBJE tags from the node
	objeNodes := gedcom.NodesWithTag(node, gedcom.TagObject)

	for _, objeNode := range objeNodes {
		var mediaPointer string

		// Check if this is a pointer-based reference (either in Pointer() or Value())
		if objeNode.Pointer() != "" {
			// Direct pointer reference (common for top-level OBJE records)
			mediaPointer = objeNode.Pointer()
		} else if objeNode.Value() != "" {
			// Value-based reference (e.g., "@O1@")
			mediaPointer = valueToPointer(objeNode.Value())
		}

		// Skip inline objects without pointers
		if mediaPointer == "" {
			continue
		}

		mediaID := e.sanitizeID(mediaPointer)
		if mediaID == "" {
			continue
		}

		// Create has_media relationship
		relData := map[string]interface{}{}

		if err := e.writeRelation("has_media",
			fmt.Sprintf("%s:%s", sourceTable, sourceID),
			fmt.Sprintf("media_object:%s", mediaID),
			relData); err != nil {
			return err
		}
	}

	// Also check for media objects attached to events (BIRT, DEAT, MARR, etc.)
	for _, childNode := range node.Nodes() {
		if childNode.Tag().IsEvent() {
			eventObjes := gedcom.NodesWithTag(childNode, gedcom.TagObject)
			for _, eventObje := range eventObjes {
				var mediaPointer string

				// Check if this is a pointer-based reference (either in Pointer() or Value())
				if eventObje.Pointer() != "" {
					mediaPointer = eventObje.Pointer()
				} else if eventObje.Value() != "" {
					// Value-based reference (e.g., "@O1@")
					mediaPointer = valueToPointer(eventObje.Value())
				}

				// Skip inline objects
				if mediaPointer == "" {
					continue
				}

				mediaID := e.sanitizeID(mediaPointer)
				if mediaID == "" {
					continue
				}

				// Create has_media relationship with event context
				relData := map[string]interface{}{
					"context": childNode.Tag().String(), // e.g., "Birth", "Death", "Marriage"
				}

				if err := e.writeRelation("has_media",
					fmt.Sprintf("%s:%s", sourceTable, sourceID),
					fmt.Sprintf("media_object:%s", mediaID),
					relData); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// sanitizeID sanitizes a GEDCOM pointer to be used as a SurrealDB ID.
func (e *Exporter) sanitizeID(pointer string) string {
	// Remove @ symbols that wrap GEDCOM pointers
	id := strings.Trim(pointer, "@")
	// Replace any remaining invalid characters
	id = strings.NewReplacer(" ", "_", "-", "_").Replace(id)
	id = strings.ToLower(id)

	// Ensure ID starts with a letter (SurrealDB best practice)
	// If it starts with a digit, prefix with 'i'
	if len(id) > 0 && id[0] >= '0' && id[0] <= '9' {
		id = "i" + id
	}

	return id
}

// valueToPointer extracts the pointer from a GEDCOM value if it's in @XXX@ format.
func valueToPointer(val string) string {
	valLen := len(val)
	if valLen < 3 {
		return ""
	}
	firstCharIsAt := val[0] == '@'
	lastCharIsAt := val[valLen-1] == '@'
	if firstCharIsAt && lastCharIsAt {
		return val[1 : valLen-1]
	}
	return ""
}
