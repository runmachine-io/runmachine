package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/runmachine-io/runmachine/pkg/api/types"
)

var (
	_one = 1
)

func TestPropertySchemaDocumentYAML(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		doc    string
		expect types.PropertySchemaDocument
	}{
		// Simple single string type
		{
			doc: `
type: string
`,
			expect: types.PropertySchemaDocument{
				Types: []string{
					"string",
				},
			},
		},
		// Array of multiple type strings
		{
			doc: `
type:
  - string
  - integer
`,
			expect: types.PropertySchemaDocument{
				Types: []string{
					"string", "integer",
				},
			},
		},
		// Simple single string type
		{
			doc: `
maximum: 1
`,
			expect: types.PropertySchemaDocument{
				Maximum: &_one,
			},
		},
	}

	for _, test := range tests {
		got := types.PropertySchemaDocument{}
		if err := yaml.Unmarshal([]byte(test.doc), &got); err != nil {
			t.Fatalf("failed unmarshalling %s: %v", test.doc, err)
		}
		assert.Equal(test.expect, got)
	}
}
