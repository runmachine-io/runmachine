package types

import (
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

type ObjectMatcher interface {
	Matches(obj *pb.Object) bool
}

// ObjectWithReferences is a concrete struct containing pointers to
// already-constructed and validated Partition and ObjectType messages. This is
// the struct that is passed to backend storage when creating new objects, not
// the protobuffer Object message or the api/types/Object struct, neither of
// which are guaranteed to be pre-validated and their relations already
// expanded.
type ObjectWithReferences struct {
	Object     *pb.Object
	Partition  *pb.Partition
	ObjectType *pb.ObjectType
}
