package types

import (
	"fmt"
	"strconv"

	pb "github.com/runmachine-io/runmachine/proto"
)

// A specialized filter class that has already looked up specific partition and
// object types (expanded from user-supplied partition and type filter
// strings). Users pass pb.ObjectFilter messages which contain optional
// pb.PartitionFilter and pb.ObjectTypeFilter messages. Those may be expanded
// (due to UsePrefix = true) to a set of partition UUIDs and/or object type
// codes. We then create zero or more of these ObjectListFilter structs
// that represent a specific filter on partition UUID and object type, along
// with the the object's name/UUID and UsePrefix flag.
type ObjectListFilter struct {
	Partition *pb.Partition
	Type      *pb.ObjectType
	Project   string
	Search    string
	UsePrefix bool
	// TODO(jaypipes): Add support for property and tag filters
}

func (f *ObjectListFilter) IsEmpty() bool {
	return f.Partition == nil && f.Type == nil && f.Project == "" && f.Search == ""
}

func (f *ObjectListFilter) String() string {
	attrMap := make(map[string]string, 0)
	if f.Partition != nil {
		attrMap["partition"] = f.Partition.Uuid
	}
	if f.Type != nil {
		attrMap["object_type"] = f.Type.Code
	}
	if f.Project != "" {
		attrMap["project"] = f.Project
	}
	if f.Search != "" {
		attrMap["search"] = f.Search
		attrMap["use_prefix"] = strconv.FormatBool(f.UsePrefix)
	}
	attrs := ""
	x := 0
	for k, v := range attrMap {
		if x > 0 {
			attrs += ","
		}
		attrs += k + "=" + v
	}
	return fmt.Sprintf("ObjectListFilter(%s)", attrs)
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
