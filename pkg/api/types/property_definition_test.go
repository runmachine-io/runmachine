package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/runmachine-io/runmachine/pkg/api/types"
)

var (
	_zero    = 0
	_one     = 1
	_two     = 2
	_zero_us = uint(0)
	_one_us  = uint(1)
	_two_us  = uint(2)
)

func TestPropertyDefinitionYAML(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		doc    string
		expect types.PropertyDefinition
	}{
		// Required missing
		{
			doc: `
type: runm.image
key: architecture
`,
			expect: types.PropertyDefinition{
				Type:     "runm.image",
				Key:      "architecture",
				Required: false,
			},
		},
		// Required false
		{
			doc: `
type: runm.image
key: architecture
required: false
`,
			expect: types.PropertyDefinition{
				Type:     "runm.image",
				Key:      "architecture",
				Required: false,
			},
		},
		// Required true
		{
			doc: `
type: runm.image
key: architecture
required: true
`,
			expect: types.PropertyDefinition{
				Type:     "runm.image",
				Key:      "architecture",
				Required: true,
			},
		},
		// Project access permission specified
		{
			doc: `
type: runm.image
key: architecture
permissions:
  - project: proj1
    permission: rw
`,
			expect: types.PropertyDefinition{
				Type: "runm.image",
				Key:  "architecture",
				Permissions: []*types.PropertyPermission{
					&types.PropertyPermission{
						Project:    "proj1",
						Permission: "rw",
					},
				},
			},
		},
		// Role access permission specified
		{
			doc: `
type: runm.image
key: architecture
permissions:
  - role: admin
    permission: rw
`,
			expect: types.PropertyDefinition{
				Type: "runm.image",
				Key:  "architecture",
				Permissions: []*types.PropertyPermission{
					&types.PropertyPermission{
						Role:       "admin",
						Permission: "rw",
					},
				},
			},
		},
		// Project and role access permission specified
		{
			doc: `
type: runm.image
key: architecture
permissions:
  - role: member
    project: proj2
    permission: r
`,
			expect: types.PropertyDefinition{
				Type: "runm.image",
				Key:  "architecture",
				Permissions: []*types.PropertyPermission{
					&types.PropertyPermission{
						Role:       "member",
						Project:    "proj2",
						Permission: "r",
					},
				},
			},
		},
		// Permission with blank permission string (used for revoking all
		// permissions on a property)
		{
			doc: `
type: runm.image
key: architecture
permissions:
  - role: member
    project: blacklisted
    permission:
`,
			expect: types.PropertyDefinition{
				Type: "runm.image",
				Key:  "architecture",
				Permissions: []*types.PropertyPermission{
					&types.PropertyPermission{
						Role:       "member",
						Project:    "blacklisted",
						Permission: "",
					},
				},
			},
		},
	}

	for _, test := range tests {
		got := types.PropertyDefinition{}
		if err := yaml.Unmarshal([]byte(test.doc), &got); err != nil {
			t.Fatalf("failed unmarshalling %s: %v", test.doc, err)
		}
		assert.Equal(test.expect, got)
	}
}

func TestPropertySchemaYAML(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		doc    string
		expect types.PropertySchema
	}{
		// Simple single string type
		{
			doc: `
type: string
`,
			expect: types.PropertySchema{
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
			expect: types.PropertySchema{
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
			expect: types.PropertySchema{
				Maximum: &_one,
			},
		},
		// Check minimum (number-based)
		{
			doc: `
minimum: 0
`,
			expect: types.PropertySchema{
				Minimum: &_zero,
			},
		},
		// Check nax length (string)
		{
			doc: `
max_length: 2
`,
			expect: types.PropertySchema{
				MaxLength: &_two_us,
			},
		},
		// Check min length (string)
		{
			doc: `
min_length: 0
`,
			expect: types.PropertySchema{
				MinLength: &_zero_us,
			},
		},
	}

	for _, test := range tests {
		got := types.PropertySchema{}
		if err := yaml.Unmarshal([]byte(test.doc), &got); err != nil {
			t.Fatalf("failed unmarshalling %s: %v", test.doc, err)
		}
		assert.Equal(test.expect, got)
	}
}

func TestPropertySchemaValidate(t *testing.T) {
	tests := []struct {
		doc       *types.PropertySchema
		expectErr bool
	}{
		// Good type
		{
			doc: &types.PropertySchema{
				Types: []string{"string"},
			},
			expectErr: false,
		},
		// Bad type
		{
			doc: &types.PropertySchema{
				Types: []string{"array", "string"},
			},
			expectErr: true,
		},
		// Good format
		{
			doc: &types.PropertySchema{
				Format: "date-time",
			},
			expectErr: false,
		},
		// Bad format
		{
			doc: &types.PropertySchema{
				Format: "datetime",
			},
			expectErr: true,
		},
	}

	for x, test := range tests {
		if err := test.doc.Validate(); err != nil {
			if !test.expectErr {
				t.Fatalf("in test %d expected no error but got: %v", x, err)
			}
		} else {
			if test.expectErr {
				t.Fatalf("in test %d expected error but got none", x)
			}
		}
	}
}
