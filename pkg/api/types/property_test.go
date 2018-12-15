package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/runmachine-io/runmachine/pkg/api/types"
)

var (
	_true    = true
	_zero    = 0
	_one     = 1
	_two     = 2
	_zero_us = uint(0)
	_one_us  = uint(1)
	_two_us  = uint(2)
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
		// Check maximum (number-based)
		{
			doc: `
maximum: 1
`,
			expect: types.PropertySchemaDocument{
				Maximum: &_one,
			},
		},
		// Check minimum (number-based)
		{
			doc: `
minimum: 0
`,
			expect: types.PropertySchemaDocument{
				Minimum: &_zero,
			},
		},
		// Check nax length (string)
		{
			doc: `
max_length: 2
`,
			expect: types.PropertySchemaDocument{
				MaxLength: &_two_us,
			},
		},
		// Check min length (string)
		{
			doc: `
min_length: 0
`,
			expect: types.PropertySchemaDocument{
				MinLength: &_zero_us,
			},
		},
		// Check required
		{
			doc: `
required: true
`,
			expect: types.PropertySchemaDocument{
				Required: &_true,
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
