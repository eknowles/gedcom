package v2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

// TestMediaObjectsIntegration verifies that media objects are correctly
// connected in the SurrealDB output, including those attached to events
func TestMediaObjectsIntegration(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create media objects with pointers
	photoMedia := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "portrait.jpg", ""),
		gedcom.NewNode(gedcom.TagTitle, "Family portrait", ""),
	)
	doc.AddNode(photoMedia)

	birthCertMedia := gedcom.NewNode(gedcom.TagObject, "", "M2",
		gedcom.NewNode(gedcom.TagFile, "birth_cert.pdf", ""),
		gedcom.NewNode(gedcom.TagTitle, "Birth certificate", ""),
	)
	doc.AddNode(birthCertMedia)

	deathCertMedia := gedcom.NewNode(gedcom.TagObject, "", "M3",
		gedcom.NewNode(gedcom.TagFile, "death_cert.pdf", ""),
	)
	doc.AddNode(deathCertMedia)

	// Create person with:
	// - Direct media reference (portrait)
	// - Birth event with media (birth certificate)
	// - Death event with media (death certificate)
	person := doc.AddIndividual("I1",
		gedcom.NewNameNode("John /Doe/"),
	)

	// Add direct media reference
	person.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))

	// Add birth with media
	birth := gedcom.NewBirthNode("",
		gedcom.NewDateNode("1 Jan 1900"),
		gedcom.NewNode(gedcom.TagPlace, "New York", ""),
	)
	birth.AddNode(gedcom.NewNode(gedcom.TagObject, "@M2@", ""))
	person.AddNode(birth)

	// Add death with media
	death := gedcom.NewDeathNode("",
		gedcom.NewDateNode("1 Jan 1980"),
	)
	death.AddNode(gedcom.NewNode(gedcom.TagObject, "@M3@", ""))
	person.AddNode(death)

	// Export to SurrealDB format
	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "test")
	err := exporter.Export(doc)

	assert.NoError(t, err)
	output := buf.String()

	// Verify media objects are created
	assert.Contains(t, output, "CREATE media_object:m1", "Should create portrait media object")
	assert.Contains(t, output, "CREATE media_object:m2", "Should create birth cert media object")
	assert.Contains(t, output, "CREATE media_object:m3", "Should create death cert media object")

	// Verify media relationships are created
	// Should have 3 has_media relationships: I1->M1, I1->M2, I1->M3
	mediaRelCount := strings.Count(output, "RELATE person:i1->has_media->media_object:")
	assert.Equal(t, 3, mediaRelCount, "Should create 3 has_media relationships for person I1")

	// Verify specific relationships
	assert.Contains(t, output, "RELATE person:i1->has_media->media_object:m1", "Should link person to portrait")
	assert.Contains(t, output, "RELATE person:i1->has_media->media_object:m2", "Should link person to birth cert")
	assert.Contains(t, output, "RELATE person:i1->has_media->media_object:m3", "Should link person to death cert")
}

// TestFamilyMediaIntegration verifies media objects attached to families
func TestFamilyMediaIntegration(t *testing.T) {
	doc := gedcom.NewDocument()

	// Create family media
	weddingPhoto := gedcom.NewNode(gedcom.TagObject, "", "M1",
		gedcom.NewNode(gedcom.TagFile, "wedding.jpg", ""),
	)
	doc.AddNode(weddingPhoto)

	marriageCert := gedcom.NewNode(gedcom.TagObject, "", "M2",
		gedcom.NewNode(gedcom.TagFile, "marriage_cert.pdf", ""),
	)
	doc.AddNode(marriageCert)

	// Create family with media
	husband := doc.AddIndividual("I1", gedcom.NewNameNode("John /Doe/"))
	wife := doc.AddIndividual("I2", gedcom.NewNameNode("Jane /Smith/"))
	family := doc.AddFamilyWithHusbandAndWife("F1", husband, wife)

	// Add direct media to family
	family.AddNode(gedcom.NewNode(gedcom.TagObject, "@M1@", ""))

	// Add marriage event with media
	marriage := gedcom.NewNode(gedcom.TagMarriage, "", "",
		gedcom.NewDateNode("15 Jun 1920"),
	)
	marriage.AddNode(gedcom.NewNode(gedcom.TagObject, "@M2@", ""))
	family.AddNode(marriage)

	// Export
	var buf bytes.Buffer
	exporter := NewExporter(&buf, "test", "test")
	err := exporter.Export(doc)

	assert.NoError(t, err)
	output := buf.String()

	// Verify family media relationships
	assert.Contains(t, output, "RELATE family:f1->has_media->media_object:m1", "Should link family to wedding photo")
	assert.Contains(t, output, "RELATE family:f1->has_media->media_object:m2", "Should link family to marriage cert")
}


