package types

import (
	pb "github.com/runmachine-io/runmachine/proto"
)

type ObjectDefinitionMatcher interface {
	Matches(obj *pb.ObjectDefinition) bool
}

// ObjectDefinitionWithReferences is a concrete struct containing pointers to
// already-constructed and validated Partition and ObjectType messages. This is
// the struct that is passed to backend storage when creating new object
// schemas, not the protobuffer ObjectDefinition message or the
// api/types/ObjectDefinition struct, neither of which are guaranteed to be
// pre-validated and their relations already expanded.
type ObjectDefinitionWithReferences struct {
	Partition  *pb.Partition
	ObjectType *pb.ObjectType
	Definition *pb.ObjectDefinition
}
