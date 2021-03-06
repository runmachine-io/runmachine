package types

import (
	"encoding/json"
	"fmt"
)

const (
	// TODO(jaypipes): Move these to a generic location?
	PERMISSION_NONE  = uint32(0)
	PERMISSION_READ  = uint32(1)
	PERMISSION_WRITE = uint32(1) << 1
)

var (
	validPermissions = []string{"", "r", "rw", "w"}
	// the set of valid type strings that may appear in the property schema
	// document "type" field
	validTypes = []string{
		"string",
		"integer",
		"number",
		"boolean",
	}
	// the set of valid format strings that may be specified in property schema
	// document's "format" field
	validFormats = []string{
		"date-time",
		"date",
		"time",
		"email",
		"idn-email",
		"hostname",
		"idn-hostname",
		"ipv4",
		"ipv6",
		"uri",
		"uri-reference",
		"iri",
		"iri-reference",
		"uri-template",
	}
	propertySchemaTemplateContents = `{{ define "property-schema" }}
        {{ quote .Key}}: {
          "type": [{{ quote_join .Types ", " }}]
{{- if .MultipleOf }}
          , "multipleOf": {{ deref_uint .MultipleOf }}
{{- end }}
{{- if .Minimum }}
          , "minimum": {{ deref_int .Minimum }}
{{- end }}
{{- if .Maximum }}
          , "maximum": {{ deref_int .Maximum }}
{{- end }}
{{- if .MinLength }}
          , "minLength": {{ deref_uint .MinLength }}
{{- end }}
{{- if .MaxLength }}
          , "maxLength": {{ deref_uint .MaxLength }}
{{- end }}
{{- if .Pattern }}
          , "pattern": {{ .Pattern }}
{{- end }}
{{- if .Format }}
          , "format": {{ .Formet }}
{{- end }}
        }
{{- end -}}
`
)

type PropertyDefinition struct {
	// JSONSchema property type document represented in YAML, dictating the
	// constraints applied by this schema to the property's value
	Schema *PropertySchema `json:"schema"`
	// Indicates the property is required for all objects of this object type
	Required bool `json:"required"`
	// Set of project/role specific permissions for the property
	Permissions []*PropertyPermission `json:"permissions"`
}

// Validate returns an error if the definition is invalid, nil otherwise
func (def *PropertyDefinition) Validate() error {
	for _, perm := range def.Permissions {
		if err := perm.Validate(); err != nil {
			return err
		}
	}
	return def.Schema.Validate()
}

// PropertyPermission describes the permission that a project and/or role have
// to read or write a property on an object
type PropertyPermission struct {
	// Optional project identifier to control access for
	Project string `json:"project,omitempty"`
	// Optional role identifier to control access for
	Role string `json:"role,omitempty"`
	// A string containing the permissions:
	//
	// "" indicates the project/role should have no read or write access to the
	// property
	// "r" indicates the project/role should have read access
	// "w" indicates the project/role should have write access
	// "rw" indicates the project/role should have read and write access
	Permission string `json:"permission"`
}

// Validate returns an error if the permission is invalid, nil otherwise
func (perm *PropertyPermission) Validate() error {
	switch perm.Permission {
	case "":
	case "r":
	case "w":
	case "rw":
		return nil
	default:
		return fmt.Errorf(
			"unknown permission string %s. valid choices are %v",
			perm.Permission, validPermissions,
		)
	}
	return nil
}

// ToUint32 converts the string representation of the access permission to the
// unsigned integer flag used for storage and comparisons internally
func (perm *PropertyPermission) PermissionUint32() uint32 {
	switch perm.Permission {
	case "":
		return 0
	case "r":
		return PERMISSION_READ
	case "w":
		return PERMISSION_WRITE
	case "rw":
		return PERMISSION_READ | PERMISSION_WRITE
	default:
		return 0
	}
}

// NOTE(jaypipes): A type that can be represented in JSON as *either* a string
// *or* an array of strings, which is what JSONSchema's type field needs.
// see: https://github.com/go-yaml/yaml/issues/100
type StringArray []string

func (a *StringArray) UnmarshalJSON(b []byte) error {
	var multi []string
	err := json.Unmarshal(b, &multi)
	if err != nil {
		var single string
		err := json.Unmarshal(b, &single)
		if err != nil {
			return err
		}
		*a = []string{single}
	} else {
		*a = multi
	}
	return nil
}

type PropertySchema struct {
	// May only be a JSON scalar type (string, integer, number, etc)
	Types StringArray `json:"type"`
	// Property value must be one of these enumerated list of values. If this
	// exists in the schema document and no types are specified, type is
	// assumed to be string
	Enum []string `json:"enum,omitempty"`
	// Indicates the property's value must be a multiple of this number. The
	// property's type must be either "number" or "integer"
	MultipleOf *uint `json:"multiple_of,omitempty"`
	// Indicates the property's numeric value must be greater than or equal to
	// this number The property's type must be either "number" or "integer"
	Minimum *int `json:"minimum,omitempty"`
	// Indicates the property's numeric value must be less than or equal to
	// this number The property's type must be either "number" or "integer"
	Maximum *int `json:"maximum,omitempty"`
	// Indicates the property's value must be a string and that string must
	// have a length greater than or equal to this number
	MinLength *uint `json:"min_length,omitempty"`
	// Indicates the property's value must be a string and that string must
	// have a length less than or equal to this number
	MaxLength *uint `json:"max_length,omitempty"`
	// Indicates the property's value must be a string and must match this
	// regex pattern
	Pattern string `json:"pattern,omitempty"`
	// A pre-defined regex that will validate the incoming property value.
	// Possible string values for "format" are:
	//
	// * "date-time"
	// * "date"
	// * "time"
	// * "email"
	// * "idn-email"
	// * "hostname"
	// * "idn-hostname"
	// * "ipv4"
	// * "ipv6"
	// * "uri"
	// * "uri-reference"
	// * "iri"
	// * "iri-reference"
	// * "uri-template"
	Format string `json:"format"`
}

type propertySchemaWithKey struct {
	*PropertySchema
	Key string
}

// Validate returns an error if the schema document isn't valid, or nil
// otherwise
func (schema *PropertySchema) Validate() error {
	if schema == nil {
		return nil
	}
	if len(schema.Types) > 0 {
		typeFound := make(map[string]bool, len(schema.Types))
		for _, schemaType := range schema.Types {
			typeFound[schemaType] = false
			for _, t := range validTypes {
				if t == schemaType {
					typeFound[schemaType] = true
				}
			}
		}
		for schemaType, found := range typeFound {
			if !found {
				return fmt.Errorf(
					"invalid type '%s'. valid types are %v",
					schemaType,
					validTypes,
				)
			}
		}
	}
	if schema.Format != "" {
		found := false
		for _, f := range validFormats {
			if f == schema.Format {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(
				"invalid format '%s'. valid formats are %v",
				schema.Format,
				validFormats,
			)
		}
	}
	return nil
}
