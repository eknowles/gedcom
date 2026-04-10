package main

import (
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

func TestPublishWarningsForMissingNames_None(t *testing.T) {
	doc := gedcom.NewDocument()
	doc.AddIndividual("P1").AddName("Ada /Lovelace/")

	warnings := publishWarningsForMissingNames(doc)
	assert.Empty(t, warnings)
}

func TestPublishWarningsForMissingNames_NoNameTag(t *testing.T) {
	doc := gedcom.NewDocument()
	doc.AddIndividual("P1")

	warnings := publishWarningsForMissingNames(doc)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "P1")
}

func TestPublishWarningsForMissingNames_BlankNameValue(t *testing.T) {
	doc := gedcom.NewDocument()
	doc.AddIndividual("P1").AddName("")

	warnings := publishWarningsForMissingNames(doc)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "no usable NAME")
}

func TestPublishWarningsForMissingNames_WhitespaceOnlyName(t *testing.T) {
	doc := gedcom.NewDocument()
	doc.AddIndividual("P1").AddName("   ")

	warnings := publishWarningsForMissingNames(doc)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "Unknown")
}
