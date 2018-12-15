package types

type PropertySchema struct {
	// Identifier of the partition the object belongs to
	Partition string `yaml:"partition"`
	// Code for the type of object this is
	Type string `yaml:"type"`
	// The key of the property this schema will apply to
	Key string `yaml:"key"`
	// JSONSchema property type document represented in YAML, dictating the
	// constraints applied by this schema to the property's value
	Schema string `yaml:"schema"`
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

type PropertySchemaDocument struct {
	// May only be a JSON scalar type (string, integer, number, etc)
	Types StringArray `yaml:"type"`
	// Property value must be one of these enumerated list of values. If this
	// exists in the schema document and no types are specified, type is
	// assumed to be string
	Enum []string `yaml:"enum"`
	// Indicates the property is required for all objects of this property
	// schema's object type
	Required *bool `yaml:"required"`
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
