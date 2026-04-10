package v2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	surrealDBURL = "http://localhost:8000"
	namespace    = "gedcom_test_v2"
	database     = "test_db_v2"
	username     = "root"
	password     = "root"
)

// SurrealDBClient is a simple client for testing
type SurrealDBClient struct {
	url       string
	namespace string
	database  string
	username  string
	password  string
}

// NewSurrealDBClient creates a new test client
func NewSurrealDBClient() *SurrealDBClient {
	return &SurrealDBClient{
		url:       surrealDBURL,
		namespace: namespace,
		database:  database,
		username:  username,
		password:  password,
	}
}

// Query executes a SurrealQL query
func (c *SurrealDBClient) Query(query string) ([]map[string]interface{}, error) {
	req, err := http.NewRequest("POST", c.url+"/sql", strings.NewReader(query))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/surql")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("surreal-ns", c.namespace)
	req.Header.Set("surreal-db", c.database)
	req.SetBasicAuth(c.username, c.password)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// ImportData imports SurrealQL data
func (c *SurrealDBClient) ImportData(data string) error {
	// The exporter already includes USE NS ... DB ... in the output,
	// so we don't need to add it again
	_, err := c.Query(data)
	return err
}

// CleanupDatabase removes all data from test database
func (c *SurrealDBClient) CleanupDatabase() error {
	queries := []string{
		"REMOVE TABLE person;",
		"REMOVE TABLE family;",
		"REMOVE TABLE note;",
		"REMOVE TABLE media_object;",
		"REMOVE TABLE child_of;",
		"REMOVE TABLE parent_of;",
		"REMOVE TABLE spouse_of;",
		"REMOVE TABLE has_note;",
		"REMOVE TABLE has_media;",
	}

	for _, query := range queries {
		if _, err := c.Query(query); err != nil {
			// Ignore errors if tables don't exist
			continue
		}
	}

	return nil
}

// TestSurrealDBIntegration_V2_BasicStructure tests the v2 exporter with a real database
func TestSurrealDBIntegration_V2_BasicStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create test document
	doc := gedcom.NewDocument()

	person := doc.AddIndividual("I1",
		gedcom.NewNameNode("John /Smith/"),
	)
	person.AddNode(gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1950"),
		gedcom.NewNode(gedcom.TagPlace, "New York", ""),
	))

	// Export to SurrealDB format
	var buf bytes.Buffer
	exporter := NewExporter(&buf, namespace, database)
	err := exporter.Export(doc)
	require.NoError(t, err)

	// Setup client and cleanup
	client := NewSurrealDBClient()
	defer client.CleanupDatabase()

	// Import data
	err = client.ImportData(buf.String())
	require.NoError(t, err)

	// Test: Verify person was created
	results, err := client.Query("SELECT * FROM person:i1;")
	require.NoError(t, err)
	require.Len(t, results, 1)

	resultData := results[0]["result"].([]interface{})
	require.Len(t, resultData, 1)

	personData := resultData[0].(map[string]interface{})

	// Verify name structure
	nameObj := personData["name"].(map[string]interface{})
	assert.Equal(t, "John Smith", nameObj["full"])
	assert.Equal(t, "John", nameObj["given"])
	assert.Equal(t, "Smith", nameObj["surname"])

	// Verify birth structure
	birthObj := personData["birth"].(map[string]interface{})
	assert.Equal(t, "1 Jan 1950", birthObj["date"])
	assert.Equal(t, "New York", birthObj["place"])
}

// TestSurrealDBIntegration_V2_MediaAndNotes tests media and note relationships
func TestSurrealDBIntegration_V2_MediaAndNotes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	doc := gedcom.NewDocument()

	// Create note with non-standard ID
	note := gedcom.NewNode(gedcom.TagNote, "Test note", "I999")
	doc.AddNode(note)

	// Create media with non-standard ID
	media := gedcom.NewNode(gedcom.TagObject, "", "I888",
		gedcom.NewNode(gedcom.TagFile, "photo.jpg", ""),
	)
	doc.AddNode(media)

	// Create person referencing both
	person := doc.AddIndividual("I1",
		gedcom.NewNameNode("Jane /Doe/"),
	)
	person.AddNode(gedcom.NewNode(gedcom.TagNote, "@I999@", ""))
	person.AddNode(gedcom.NewNode(gedcom.TagObject, "@I888@", ""))

	var buf bytes.Buffer
	exporter := NewExporter(&buf, namespace, database)
	err := exporter.Export(doc)
	require.NoError(t, err)

	client := NewSurrealDBClient()
	defer client.CleanupDatabase()

	err = client.ImportData(buf.String())
	require.NoError(t, err)

	// Test: Verify note relationship uses correct entity type
	results, err := client.Query("SELECT ->has_note->note.* AS notes FROM person:i1;")
	require.NoError(t, err)

	resultData := results[0]["result"].([]interface{})
	require.Len(t, resultData, 1)

	notes := resultData[0].(map[string]interface{})["notes"].([]interface{})
	require.Len(t, notes, 1, "Should have 1 note relationship")

	noteData := notes[0].(map[string]interface{})
	assert.Equal(t, "Test note", noteData["text"])

	// Test: Verify media relationship uses correct entity type
	results, err = client.Query("SELECT ->has_media->media_object.* AS media FROM person:i1;")
	require.NoError(t, err)

	resultData = results[0]["result"].([]interface{})
	require.Len(t, resultData, 1)

	mediaList := resultData[0].(map[string]interface{})["media"].([]interface{})
	require.Len(t, mediaList, 1, "Should have 1 media relationship")

	mediaData := mediaList[0].(map[string]interface{})
	files := mediaData["files"].([]interface{})
	assert.Contains(t, files, "photo.jpg")
}

// TestSurrealDBIntegration_V2_Relationships tests family relationships
func TestSurrealDBIntegration_V2_Relationships(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	doc := gedcom.NewDocument()

	father := doc.AddIndividual("I1",
		gedcom.NewNameNode("John /Smith/"),
	)
	mother := doc.AddIndividual("I2",
		gedcom.NewNameNode("Jane /Smith/"),
	)
	child := doc.AddIndividual("I3",
		gedcom.NewNameNode("Bob /Smith/"),
	)

	family := doc.AddFamilyWithHusbandAndWife("F1", father, mother)
	family.AddChild(child)

	var buf bytes.Buffer
	exporter := NewExporter(&buf, namespace, database)
	err := exporter.Export(doc)
	require.NoError(t, err)

	client := NewSurrealDBClient()
	defer client.CleanupDatabase()

	err = client.ImportData(buf.String())
	require.NoError(t, err)

	// Test: Child's parents
	results, err := client.Query(`
		SELECT ->child_of->person.name.full AS parents FROM person:i3;
	`)
	require.NoError(t, err)

	resultData := results[0]["result"].([]interface{})
	require.Len(t, resultData, 1)

	parents := resultData[0].(map[string]interface{})["parents"].([]interface{})
	assert.Len(t, parents, 2, "Child should have 2 parents")

	parentNames := []string{}
	for _, p := range parents {
		parentNames = append(parentNames, p.(string))
	}
	assert.Contains(t, parentNames, "John Smith")
	assert.Contains(t, parentNames, "Jane Smith")

	// Test: Spouse relationship
	results, err = client.Query(`
		SELECT ->spouse_of->person.name.full AS spouses FROM person:i1;
	`)
	require.NoError(t, err)

	resultData = results[0]["result"].([]interface{})
	require.Len(t, resultData, 1)

	spouses := resultData[0].(map[string]interface{})["spouses"].([]interface{})
	require.Len(t, spouses, 1)
	assert.Equal(t, "Jane Smith", spouses[0])
}
