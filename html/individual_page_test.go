package html_test

import (
	"bytes"
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/eknowles/gedcom/v39/html"
	"github.com/eknowles/gedcom/v39/html/core"
	"github.com/stretchr/testify/assert"
)

func renderHTML(t *testing.T, c core.Component) string {
	t.Helper()

	buf := bytes.NewBuffer(nil)
	_, err := c.WriteHTMLTo(buf)
	assert.NoError(t, err)

	return buf.String()
}

func TestIndividualPage_WriteHTMLTo_NoNames(t *testing.T) {
	doc := gedcom.NewDocument()
	individual := doc.AddIndividual("P1")

	component := html.NewIndividualPage(
		doc,
		individual,
		"",
		&html.PublishShowOptions{LivingVisibility: html.LivingVisibilityShow},
		nil,
		nil,
	)

	out := renderHTML(t, component)
	assert.Contains(t, out, "Unknown")
}

func TestIndividualNameAndSex_WriteHTMLTo_NoNames(t *testing.T) {
	doc := gedcom.NewDocument()
	individual := doc.AddIndividual("P1")

	out := renderHTML(t, html.NewIndividualNameAndSex(individual))
	assert.Contains(t, out, "Name &amp; Sex")
	assert.Contains(t, out, "Sex")
}

func TestIndividualAdditionalNames_WriteHTMLTo_NoNames(t *testing.T) {
	doc := gedcom.NewDocument()
	individual := doc.AddIndividual("P1")

	out := renderHTML(t, html.NewIndividualAdditionalNames(individual))
	assert.Contains(t, out, "Additional Names")
}
