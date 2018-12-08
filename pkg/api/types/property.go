package types

type PropertySchema struct {
	// Identifier of the partition the object belongs to
	Partition string `yaml:"partition"`
	// Code for the type of object this is
	Type string `yaml:"type"`
	// The key of the property this schema will apply to
	Key string `yaml:"key"`
	// JSONSchema document represented in YAML, dictating the constraints
	// applied by this schema to the property's value
	Schema string `yaml:"schema"`
}
