package types

import "fmt"

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
