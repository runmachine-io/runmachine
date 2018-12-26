package types

// I would much prefer to be able to decorate our primary Protobuffer models
// (in proto/defs) with tags that indicate YAML input validation, however the
// developers of Google Protobuffers are not interested in supporting this
// feature.
//
// Therefore, we need to have separately-defined structs that can be used for
// YAML input validation and marshaling.
//
// :see: https://github.com/golang/protobuf/issues/52

type Object struct {
	// Identifier of the partition the object belongs to
	Partition string `yaml:"partition"`
	// Code for the type of object this is
	ObjectType string `yaml:"object_type"`
	// Optional identifier of the project the object belongs to. Only required
	// if the object's type is project-scoped
	Project string `yaml:"project"`
	// The UUID of the object. Expected to be blank when a user is creating a
	// new object.
	Uuid string `yaml:"uuid"`
	// Human-readable name for the object. Uniqueness is guaranteed in the
	// scope of the type of object. If the type is project-scoped, then
	// uniqueness is guaranteed within the scope of the partition, object type
	// and project. If partition-scoped, uniqueness is guaranteed within the
	// scope of the partition and object type.
	Name string `yaml:"name"`
	// Map of key/value properties associated with this object. Properties can
	// have a structure and be validated against a schema.
	Properties map[string]string `yaml:"properties"`
	// Array of string tags. Tags are unstructured and unvalidated and any user
	// belonging to the owning project can add or remove any tag.
	Tags []string `yaml:"tags"`
}
