package types

type Provider struct {
	// Identifier of the partition the object belongs to
	Partition string `yaml:"partition"`
	// Code for the type of provider this is
	ProviderType string `yaml:"provider_type"`
	// Optional identifier of the provider the provider is a child of. Leave
	// empty if the provider has no parents (it's a "root provider")
	Parent string `yaml:"parent"`
	// The UUID of the provider. Expected to be blank when a user is creating a
	// new provider.
	Uuid string `yaml:"uuid"`
	// Human-readable name for the provider. Uniqueness is guaranteed in the
	// scope of the partition the provider belongs to.
	Name string `yaml:"name"`
	// Map of key/value properties associated with this provider. Properties can
	// have a structure and be validated against a schema.
	Properties map[string]string `yaml:"properties"`
	// Array of string tags. Tags are unstructured and unvalidated and any user
	// with write access to the provider can add or remove any tag.
	Tags []string `yaml:"tags"`
}
