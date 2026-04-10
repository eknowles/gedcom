package v2

import (
	"strings"
	"testing"

	"github.com/eknowles/gedcom/v39"
	"github.com/stretchr/testify/assert"
)

func TestExtractNameObjectWithNickname(t *testing.T) {
	tests := []struct {
		name     string
		gedcom   string
		expected map[string]interface{}
	}{
		{
			name: "Full name with SECG and nickname from TYPE nick",
			gedcom: `0 @I1@ INDI
1 NAME Charles William Frederick Lance /Knowles/
2 GIVN Charles William
2 SURN Knowles
2 SECG Frederick Lance
1 NAME William //
2 TYPE nick
2 GIVN William`,
			expected: map[string]interface{}{
				"given":    "Charles William",
				"middle":   "Frederick Lance",
				"surname":     "Knowles",
				"full":     "Charles William Frederick Lance Knowles",
				"nickname": "William",
			},
		},
		{
			name: "Name with nickname from NICK tag",
			gedcom: `0 @I1@ INDI
1 NAME John /Smith/
2 GIVN John
2 SURN Smith
1 NICK Johnny`,
			expected: map[string]interface{}{
				"given":    "John",
				"surname":     "Smith",
				"full":     "John Smith",
				"nickname": "Johnny",
			},
		},
		{
			name: "Name with prefix and suffix",
			gedcom: `0 @I1@ INDI
1 NAME Dr. Robert /Johnson/ Jr.
2 NPFX Dr.
2 GIVN Robert
2 SURN Johnson
2 NSFX Jr.`,
			expected: map[string]interface{}{
				"prefix": "Dr.",
				"given":  "Robert",
				"surname":   "Johnson",
				"suffix": "Jr.",
				"full":   "Dr. Robert Johnson Jr.",
			},
		},
		{
			name: "Simple name without extras",
			gedcom: `0 @I1@ INDI
1 NAME Jane /Doe/
2 GIVN Jane
2 SURN Doe`,
			expected: map[string]interface{}{
				"given": "Jane",
				"surname":  "Doe",
				"full":  "Jane Doe",
			},
		},
		{
			name: "Name with middle name and nickname",
			gedcom: `0 @I1@ INDI
1 NAME Robert James /Wilson/
2 GIVN Robert
2 SURN Wilson
2 SECG James
1 NAME Bob //
2 TYPE nick
2 GIVN Bob`,
			expected: map[string]interface{}{
				"given":    "Robert",
				"middle":   "James",
				"surname":  "Wilson",
				"full":     "Robert James Wilson",
				"nickname": "Bob",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := gedcom.NewDecoder(strings.NewReader(tt.gedcom)).Decode()
			assert.NoError(t, err)
			assert.NotNil(t, doc)

			individuals := doc.Individuals()
			assert.Len(t, individuals, 1)

			result := ExtractNameObjectWithNickname(individuals[0])
			assert.NotNil(t, result)

			// Check each expected field
			for key, expectedValue := range tt.expected {
				assert.Equal(t, expectedValue, result[key], "Field %s should match", key)
			}

			// Ensure no extra fields (except those expected to be missing)
			for key := range result {
				if key == "full" || key == "given" || key == "middle" || key == "surname" ||
				   key == "prefix" || key == "suffix" || key == "nickname" {
					// These are valid fields
					continue
				}
				t.Errorf("Unexpected field in result: %s", key)
			}
		})
	}
}

func TestExtractNameObjectWithNickname_NilOrEmpty(t *testing.T) {
	// Test with nil individual
	result := ExtractNameObjectWithNickname(nil)
	assert.Nil(t, result)

	// Test with individual without names
	doc := gedcom.NewDocument()
	individual := doc.AddIndividual("I1")
	result = ExtractNameObjectWithNickname(individual)
	assert.Nil(t, result)
}





