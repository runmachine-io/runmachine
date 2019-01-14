package types

import "fmt"

var (
	baseProviderRequiredFields = []string{
		"partition",
		"provider_type",
	}
	baseProviderSchema = `{
  "$id": "https://runmachine.io/runm.provider.schema.json",
  "$schema": "http://json-schema.org/draft-07/schema#",
  "description": "A provider of resources",
  "type": "object",
  "properties": {
    "partition": {
      "type": "string"
    },
    "provider_type": {
      "type": "string"
    },
`
)

// ProviderDefinition is used by runmachine system administrators to constrain
// the properties that may be set on provider objects
type ProviderDefinition struct {
	// Identifier of the partition the object belongs to
	Partition string `yaml:"partition"`
	// Named properties may have their values constrained by a property
	// definition. The map key is the key of the property to apply the
	// property definition to
	PropertyDefinitions map[string]*PropertyDefinition `yaml:"property_definitions"`
}

// Validate returns an error if the definition is invalid, nil otherwise
func (def *ProviderDefinition) Validate() error {
	if def.Partition == "" {
		return fmt.Errorf("partition required")
	}
	for _, pd := range def.PropertyDefinitions {
		if err := pd.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// JSONSchemaString returns a valid JSONSchema DRAFT-07 document describing the
// fields and properties that may be set for the providers described by the
// provider definition
func (def *ProviderDefinition) JSONSchemaString() string {
	reqFields := make([]string, 0)
	for _, field := range baseProviderRequiredFields {
		reqFields = append(reqFields, field)
	}
	s := baseProviderSchema
	if len(def.PropertyDefinitions) > 0 {
		s += `    "properties": {
      "type": "object",
      "properties": {
`
		numPropDefs := len(def.PropertyDefinitions)
		x := 0
		for key, prop := range def.PropertyDefinitions {
			if prop.Required {
				reqFields = append(reqFields, key)
			}
			s += "        \"" + key + "\": {\n"
			s += prop.Schema.JSONSchemaString()
			s += "        }"
			if (x + 1) < numPropDefs {
				s += ",\n"
			}
		}
		s += `
      }
    }`
	}
	s += `
  },
  "required": [
`
	numReqFields := len(reqFields)
	for x, reqField := range reqFields {
		s += `    "` + reqField + `"`
		if (x + 1) < numReqFields {
			s += ",\n"
		}
	}
	s += `
  ]
}
`
	return s
}
