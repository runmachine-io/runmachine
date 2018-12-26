package types

import (
	"fmt"
	"strconv"

	pb "github.com/runmachine-io/runmachine/proto"
)

type ObjectMatcher interface {
	Matches(obj *pb.Object) bool
}

// ObjectCondition is a class used in filtering objects.  Optional partition
// and object type PKs have already been expanded from user-supplied partition
// and type filter strings
type ObjectCondition struct {
	Partition  *PartitionCondition
	ObjectType *ObjectTypeCondition
	Uuid       *UuidCondition
	Name       *NameCondition
	Project    string
	// TODO(jaypipes): Add support for property and tag filters
}

func (f *ObjectCondition) Matches(obj *pb.Object) bool {
	if !f.Uuid.Matches(obj) {
		return false
	}
	if !f.Partition.Matches(obj) {
		return false
	}
	if !f.ObjectType.Matches(obj) {
		return false
	}
	if !f.Name.Matches(obj) {
		return false
	}
	if f.Project != "" && obj.Project != "" {
		if obj.Project != f.Project {
			return false
		}
	}
	return true
}

func (f *ObjectCondition) IsEmpty() bool {
	return f.Partition == nil && f.ObjectType == nil && f.Uuid == nil && f.Project == "" && f.Name == nil
}

func (f *ObjectCondition) String() string {
	attrMap := make(map[string]string, 0)
	if f.Partition != nil {
		attrMap["partition"] = f.Partition.Partition.Uuid
	}
	if f.ObjectType != nil {
		attrMap["object_type"] = f.ObjectType.ObjectType.Code
	}
	if f.Uuid != nil {
		attrMap["uuid"] = f.Uuid.Uuid
	}
	if f.Project != "" {
		attrMap["project"] = f.Project
	}
	if f.Name != nil {
		attrMap["key"] = f.Name.Name
		attrMap["use_prefix"] = strconv.FormatBool(
			f.Name.Op == OP_GREATER_THAN_EQUAL,
		)
	}
	attrs := ""
	x := 0
	for k, v := range attrMap {
		if x > 0 {
			attrs += ","
		}
		attrs += k + "=" + v
	}
	return fmt.Sprintf("ObjectCondition(%s)", attrs)
}

// ObjectWithReferences is a concrete struct containing pointers to
// already-constructed and validated Partition and ObjectType messages. This is
// the struct that is passed to backend storage when creating new objects, not
// the protobuffer Object message or the api/types/Object struct, neither of
// which are guaranteed to be pre-validated and their relations already
// expanded.
type ObjectWithReferences struct {
	Object    *pb.Object
	Partition *pb.Partition
	Type      *pb.ObjectType
}
