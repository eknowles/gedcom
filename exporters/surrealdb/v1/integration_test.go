package v1

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
	namespace    = "gedcom_test"
	database     = "test_db"
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
		"REMOVE TABLE child_of;",
		"REMOVE TABLE parent_of;",
		"REMOVE TABLE spouse_of;",
	}

	for _, query := range queries {
		if _, err := c.Query(query); err != nil {
			// Ignore errors if tables don't exist
			continue
		}
	}

	return nil
}

// getLastResult extracts the last result from a SurrealDB query response
// This handles the case where USE statements create additional result objects
func getLastResult(results []map[string]interface{}) map[string]interface{} {
	if len(results) == 0 {
		return nil
	}
	return results[len(results)-1]
}

// TestSurrealDBExport_BasicStructure tests that the export creates correct structure
func TestSurrealDBExport_BasicStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create test document
	doc := gedcom.NewDocument()

	husband := doc.AddIndividual("I1")
	husband.AddName("John /Smith/")
	husband.SetSex(gedcom.SexMale)
	husband.AddBirthDate("1 JAN 1950")

	wife := doc.AddIndividual("I2")
	wife.AddName("Jane /Doe/")
	wife.SetSex(gedcom.SexFemale)
	wife.AddBirthDate("15 MAR 1952")

	child := doc.AddIndividual("I3")
	child.AddName("Bob /Smith/")
	child.SetSex(gedcom.SexMale)
	child.AddBirthDate("10 JUL 1975")

	family := doc.AddFamilyWithHusbandAndWife("F1", husband, wife)
	family.AddChild(child)

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

	// Test 1: Verify people were created
	results, err := client.Query("SELECT * FROM person;")
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Get the last result (actual query result, after any USE statements)
	resultData := getLastResult(results)["result"].([]interface{})
	assert.Len(t, resultData, 3, "Should have 3 people")

	// Test 2: Verify family was created
	results, err = client.Query("SELECT * FROM family;")
	require.NoError(t, err)
	resultData = getLastResult(results)["result"].([]interface{})
	assert.Len(t, resultData, 1, "Should have 1 family")
}

// TestSurrealDBExport_SpouseRelationships tests spouse relationships
func TestSurrealDBExport_SpouseRelationships(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	doc := gedcom.NewDocument()
	husband := doc.AddIndividual("I1")
	husband.AddName("John /Smith/")

	wife := doc.AddIndividual("I2")
	wife.AddName("Jane /Doe/")

	doc.AddFamilyWithHusbandAndWife("F1", husband, wife)

	var buf bytes.Buffer
	exporter := NewExporter(&buf, namespace, database)
	err := exporter.Export(doc)
	require.NoError(t, err)

	client := NewSurrealDBClient()
	defer client.CleanupDatabase()

	err = client.ImportData(buf.String())
	require.NoError(t, err)

	// Test: John's spouse should be Jane
	results, err := client.Query("SELECT ->spouse_of->person.name as spouse FROM person:i1;")
	require.NoError(t, err)

	resultData := getLastResult(results)["result"].([]interface{})
	require.Len(t, resultData, 1)

	spouses := resultData[0].(map[string]interface{})["spouse"].([]interface{})
	require.Len(t, spouses, 1)
	assert.Equal(t, "Jane Doe", spouses[0], "John's spouse should be Jane")

	// Test: Jane's spouse should be John (bidirectional)
	results, err = client.Query("SELECT ->spouse_of->person.name as spouse FROM person:i2;")
	require.NoError(t, err)

	resultData = getLastResult(results)["result"].([]interface{})
	require.Len(t, resultData, 1)

	spouses = resultData[0].(map[string]interface{})["spouse"].([]interface{})
	require.Len(t, spouses, 1)
	assert.Equal(t, "John Smith", spouses[0], "Jane's spouse should be John")
}

// TestSurrealDBExport_ParentChildRelationships tests parent-child links
func TestSurrealDBExport_ParentChildRelationships(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	doc := gedcom.NewDocument()

	father := doc.AddIndividual("I1")
	father.AddName("John /Smith/")

	mother := doc.AddIndividual("I2")
	mother.AddName("Jane /Smith/")

	child := doc.AddIndividual("I3")
	child.AddName("Bob /Smith/")

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

	// Test 1: Bob's parents should include John and Jane
	results, err := client.Query("SELECT ->child_of->person.name as parents FROM person:i3;")
	require.NoError(t, err)

	resultData := getLastResult(results)["result"].([]interface{})
	require.Len(t, resultData, 1)

	parents := resultData[0].(map[string]interface{})["parents"].([]interface{})
	assert.Len(t, parents, 2, "Bob should have 2 parents")

	parentNames := []string{}
	for _, p := range parents {
		parentNames = append(parentNames, p.(string))
	}
	assert.Contains(t, parentNames, "John Smith")
	assert.Contains(t, parentNames, "Jane Smith")

	// Test 2: John's children should include Bob
	results, err = client.Query("SELECT ->parent_of->person.name as children FROM person:i1;")
	require.NoError(t, err)

	resultData = getLastResult(results)["result"].([]interface{})
	require.Len(t, resultData, 1)

	children := resultData[0].(map[string]interface{})["children"].([]interface{})
	require.Len(t, children, 1)
	assert.Equal(t, "Bob Smith", children[0])
}

// TestSurrealDBExport_MultiGenerational tests multi-generational queries
func TestSurrealDBExport_MultiGenerational(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	doc := gedcom.NewDocument()

	// Grandparents
	grandfather := doc.AddIndividual("I1")
	grandfather.AddName("Grandfather /Smith/")

	grandmother := doc.AddIndividual("I2")
	grandmother.AddName("Grandmother /Smith/")

	// Parents
	father := doc.AddIndividual("I3")
	father.AddName("Father /Smith/")

	mother := doc.AddIndividual("I4")
	mother.AddName("Mother /Jones/")

	// Grandchild
	grandchild := doc.AddIndividual("I5")
	grandchild.AddName("Grandchild /Smith/")

	// Create families
	family1 := doc.AddFamilyWithHusbandAndWife("F1", grandfather, grandmother)
	family1.AddChild(father)

	family2 := doc.AddFamilyWithHusbandAndWife("F2", father, mother)
	family2.AddChild(grandchild)

	var buf bytes.Buffer
	exporter := NewExporter(&buf, namespace, database)
	err := exporter.Export(doc)
	require.NoError(t, err)

	client := NewSurrealDBClient()
	defer client.CleanupDatabase()

	err = client.ImportData(buf.String())
	require.NoError(t, err)

	// Test: Find grandparents of grandchild
	results, err := client.Query(
		"SELECT ->child_of->person->child_of->person.name as grandparents FROM person:i5;")
	require.NoError(t, err)

	resultData := getLastResult(results)["result"].([]interface{})
	require.Len(t, resultData, 1)

	grandparents := resultData[0].(map[string]interface{})["grandparents"].([]interface{})
	assert.Len(t, grandparents, 2, "Should find 2 grandparents")

	grandparentNames := []string{}
	for _, gp := range grandparents {
		grandparentNames = append(grandparentNames, gp.(string))
	}
	assert.Contains(t, grandparentNames, "Grandfather Smith")
	assert.Contains(t, grandparentNames, "Grandmother Smith")
}

// TestSurrealDBExport_PersonData tests that person data is exported correctly
func TestSurrealDBExport_PersonData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	doc := gedcom.NewDocument()

	person := doc.AddIndividual("I1")
	person.AddName("John /Smith/")
	person.SetSex(gedcom.SexMale)
	person.AddBirthDate("15 MAR 1950")

	var buf bytes.Buffer
	exporter := NewExporter(&buf, namespace, database)
	err := exporter.Export(doc)
	require.NoError(t, err)

	client := NewSurrealDBClient()
	defer client.CleanupDatabase()

	err = client.ImportData(buf.String())
	require.NoError(t, err)

	// Query person data
	results, err := client.Query("SELECT * FROM person:i1;")
	require.NoError(t, err)

	resultData := getLastResult(results)["result"].([]interface{})
	require.Len(t, resultData, 1)

	personData := resultData[0].(map[string]interface{})

	// Verify fields
	assert.Equal(t, "John Smith", personData["name"])
	assert.Equal(t, "John", personData["given_name"])
	assert.Equal(t, "Smith", personData["surname"])
	assert.Equal(t, "Male", personData["sex"])
	assert.Equal(t, true, personData["is_living"])
	assert.Equal(t, "I1", personData["pointer"])

	// Verify birth data
	birth := personData["birth"].(map[string]interface{})
	assert.Equal(t, "15 Mar 1950", birth["date"])
}

// TestSurrealDBExport_SiblingQuery tests finding siblings
func TestSurrealDBExport_SiblingQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	doc := gedcom.NewDocument()

	father := doc.AddIndividual("I1")
	father.AddName("Father /Smith/")

	mother := doc.AddIndividual("I2")
	mother.AddName("Mother /Smith/")

	child1 := doc.AddIndividual("I3")
	child1.AddName("Alice /Smith/")

	child2 := doc.AddIndividual("I4")
	child2.AddName("Bob /Smith/")

	child3 := doc.AddIndividual("I5")
	child3.AddName("Charlie /Smith/")

	family := doc.AddFamilyWithHusbandAndWife("F1", father, mother)
	family.AddChild(child1)
	family.AddChild(child2)
	family.AddChild(child3)

	var buf bytes.Buffer
	exporter := NewExporter(&buf, namespace, database)
	err := exporter.Export(doc)
	require.NoError(t, err)

	client := NewSurrealDBClient()
	defer client.CleanupDatabase()

	err = client.ImportData(buf.String())
	require.NoError(t, err)

	// Find Alice's siblings
	results, err := client.Query(`
		SELECT ->child_of->person->parent_of->person.name as siblings 
		FROM person:i3
	`)
	require.NoError(t, err)

	resultData := getLastResult(results)["result"].([]interface{})
	require.Len(t, resultData, 1)

	siblings := resultData[0].(map[string]interface{})["siblings"].([]interface{})

	// Should include Alice herself plus her siblings (3 total)
	// We can filter out self in the query if needed
	assert.GreaterOrEqual(t, len(siblings), 2, "Should find at least 2 siblings")
}

// TestSurrealDBExport_ComplexFamily tests a complex family structure
func TestSurrealDBExport_ComplexFamily(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	doc := gedcom.NewDocument()

	// Create 10 individuals and 3 families
	individuals := make([]*gedcom.IndividualNode, 10)
	for i := 0; i < 10; i++ {
		individuals[i] = doc.AddIndividual(fmt.Sprintf("I%d", i+1))
		individuals[i].AddName(fmt.Sprintf("Person%d /Test/", i+1))
	}

	// Family 1: individuals[0] + individuals[1] -> individuals[2], individuals[3]
	family1 := doc.AddFamilyWithHusbandAndWife("F1", individuals[0], individuals[1])
	family1.AddChild(individuals[2])
	family1.AddChild(individuals[3])

	// Family 2: individuals[2] + individuals[4] -> individuals[5], individuals[6]
	family2 := doc.AddFamilyWithHusbandAndWife("F2", individuals[2], individuals[4])
	family2.AddChild(individuals[5])
	family2.AddChild(individuals[6])

	// Family 3: individuals[7] + individuals[8] -> individuals[9]
	family3 := doc.AddFamilyWithHusbandAndWife("F3", individuals[7], individuals[8])
	family3.AddChild(individuals[9])

	var buf bytes.Buffer
	exporter := NewExporter(&buf, namespace, database)
	err := exporter.Export(doc)
	require.NoError(t, err)

	client := NewSurrealDBClient()
	defer client.CleanupDatabase()

	err = client.ImportData(buf.String())
	require.NoError(t, err)

	// Test: Count total relationships
	results, err := client.Query("SELECT * FROM parent_of;")
	require.NoError(t, err)

	resultData := getLastResult(results)["result"].([]interface{})

	// Each child has 2 parent_of links (from each parent)
	// Family 1: 2 children * 2 parents = 4 links
	// Family 2: 2 children * 2 parents = 4 links
	// Family 3: 1 child * 2 parents = 2 links
	// Total: 10 links
	assert.Len(t, resultData, 10, "Should have 10 parent_of relationships")
}
