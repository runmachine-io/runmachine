package types

import (
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
)

type PropertyDefinition struct {
	// JSONSchema property type document represented in YAML, dictating the
	// constraints applied by this schema to the property's value
	Schema *PropertySchema `yaml:"schema"`
	// Indicates the property is required for all objects of this object type
	Required bool `yaml:"required"`
	// Set of project/role specific permissions for the property
	Permissions []*PropertyPermission `yaml:"permissions"`
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
	Project string `yaml:"project"`
	// Optional role identifier to control access for
	Role string `yaml:"role"`
	// A string containing the permissions:
	//
	// "" indicates the project/role should have no read or write access to the
	// property
	// "r" indicates the project/role should have read access
	// "w" indicates the project/role should have write access
	// "rw" indicates the project/role should have read and write access
	Permission string `yaml:"permission"`
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

// NOTE(jaypipes): A type that can be represented in YAML as *either* a string
// *or* an array of strings, which is what JSONSchema's type field needs.
// see: https://github.com/go-yaml/yaml/issues/100
type StringArray []string

func (a *StringArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var multi []string
	err := unmarshal(&multi)
	if err != nil {
		var single string
		err := unmarshal(&single)
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
	Types StringArray `yaml:"type"`
	// Property value must be one of these enumerated list of values. If this
	// exists in the schema document and no types are specified, type is
	// assumed to be string
	Enum []string `yaml:"enum"`
	// Indicates the property's value must be a multiple of this number. The
	// property's type must be either "number" or "integer"
	MultipleOf *uint `yaml:"multiple_of"`
	// Indicates the property's numeric value must be greater than or equal to
	// this number The property's type must be either "number" or "integer"
	Minimum *int `yaml:"minimum"`
	// Indicates the property's numeric value must be less than or equal to
	// this number The property's type must be either "number" or "integer"
	Maximum *int `yaml:"maximum"`
	// Indicates the property's value must be a string and that string must
	// have a length greater than or equal to this number
	MinLength *uint `yaml:"min_length"`
	// Indicates the property's value must be a string and that string must
	// have a length less than or equal to this number
	MaxLength *uint `yaml:"max_length"`
	// Indicates the property's value must be a string and must match this
	// regex pattern
	Pattern string `yaml:"pattern"`
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
	Format string `yaml:"format"`
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

// Returns a JSONSchema (DRAFT-7) document representing the schema for the
// object type and key pair.
func (schema *PropertySchema) JSONSchemaString() string {
	if schema == nil {
		return ""
	}
	res := ""
	switch len(schema.Types) {
	case 0:
		break
	case 1:
		res += "type: " + schema.Types[0] + "\n"
	default:
		res += "type:\n"
		for _, t := range schema.Types {
			res += "  - " + t + "\n"
		}
	}
	if len(schema.Enum) > 0 {
		res += "enum:\n"
		for _, val := range schema.Enum {
			res += "  - " + val + "\n"
		}
	}
	if schema.MultipleOf != nil {
		res += fmt.Sprintf("multipleOf: %d\n", *schema.MultipleOf)
	}
	if schema.Minimum != nil {
		res += fmt.Sprintf("minimum: %d\n", *schema.Minimum)
	}
	if schema.Maximum != nil {
		res += fmt.Sprintf("maximum: %d\n", *schema.Maximum)
	}
	if schema.MinLength != nil {
		res += fmt.Sprintf("minLength: %d\n", *schema.MinLength)
	}
	if schema.MaxLength != nil {
		res += fmt.Sprintf("maxLength: %d\n", *schema.MaxLength)
	}
	if schema.Format != "" {
		res += "format: " + schema.Format + "\n"
	}
	if schema.Pattern != "" {
		res += "pattern: " + schema.Pattern + "\n"
	}
	return res
}
