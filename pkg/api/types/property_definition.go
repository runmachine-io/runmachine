package types

import (
	"fmt"
)

var (
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
	// Identifier of the partition the object belongs to
	Partition string `yaml:"partition"`
	// Code for the type of object this is
	Type string `yaml:"type"`
	// The key of the property this schema will apply to
	Key string `yaml:"key"`
	// JSONSchema property type document represented in YAML, dictating the
	// constraints applied by this schema to the property's value
	Schema *PropertySchema `yaml:"schema"`
	// TODO(jaypipes): Add access permissions
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
	// Indicates the property is required for all objects of this property
	// schema's object type
	Required bool `yaml:"required"`
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
func (doc *PropertySchema) Validate() error {
	if len(doc.Types) > 0 {
		typeFound := make(map[string]bool, len(doc.Types))
		for _, docType := range doc.Types {
			typeFound[docType] = false
			for _, t := range validTypes {
				if t == docType {
					typeFound[docType] = true
				}
			}
		}
		for docType, found := range typeFound {
			if !found {
				return fmt.Errorf(
					"invalid type %s. valid types are %v",
					docType,
					validTypes,
				)
			}
		}
	}
	if doc.Format != "" {
		found := false
		for _, f := range validFormats {
			if f == doc.Format {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(
				"invalid format %s. valid formats are %v",
				doc.Format,
				validFormats,
			)
		}
	}
	return nil
}

// Returns a JSONSchema (DRAFT-7) document representing the schema for the
// object type and key pair.
func (doc *PropertySchema) JSONSchemaString() string {
	if doc == nil {
		return ""
	}
	res := "required: "
	if doc.Required {
		res += "true\n"
	} else {
		res += "false\n"
	}
	switch len(doc.Types) {
	case 0:
		break
	case 1:
		res += "type: " + doc.Types[0] + "\n"
	default:
		res += "type:\n"
		for _, t := range doc.Types {
			res += "  - " + t + "\n"
		}
	}
	if len(doc.Enum) > 0 {
		res += "enum:\n"
		for _, val := range doc.Enum {
			res += "  - " + val + "\n"
		}
	}
	if doc.MultipleOf != nil {
		res += fmt.Sprintf("multipleOf: %d\n", *doc.MultipleOf)
	}
	if doc.Minimum != nil {
		res += fmt.Sprintf("minimum: %d\n", *doc.Minimum)
	}
	if doc.Maximum != nil {
		res += fmt.Sprintf("maximum: %d\n", *doc.Maximum)
	}
	if doc.MinLength != nil {
		res += fmt.Sprintf("minLength: %d\n", *doc.MinLength)
	}
	if doc.MaxLength != nil {
		res += fmt.Sprintf("maxLength: %d\n", *doc.MaxLength)
	}
	if doc.Format != "" {
		res += "format: " + doc.Format + "\n"
	}
	if doc.Pattern != "" {
		res += "pattern: " + doc.Pattern + "\n"
	}
	return res
}
