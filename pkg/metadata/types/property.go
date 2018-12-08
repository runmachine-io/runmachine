package types

import (
	"fmt"
	"strconv"

	pb "github.com/runmachine-io/runmachine/proto"
)

// A specialized filter class that has already looked up specific partition and
// object types (expanded from user-supplied partition and type filter
// strings). Users pass pb.PropertySchemaFilter messages which contain optional
// pb.PartitionFilter and pb.ObjectTypeFilter messages. Those may be expanded
// (due to UsePrefix = true) to a set of partition UUIDs and/or object type
// codes. We then create zero or more of these ObjectListFilter structs
// that represent a specific filter on partition UUID and object type, along
// with the the property schema's key
type PropertySchemaFilter struct {
	Partition *pb.Partition
	Type      *pb.ObjectType
	Search    string
	UsePrefix bool
}

func (f *PropertySchemaFilter) IsEmpty() bool {
	return f.Partition == nil && f.Type == nil && f.Search == ""
}

func (f *PropertySchemaFilter) String() string {
	attrMap := make(map[string]string, 0)
	if f.Partition != nil {
		attrMap["partition"] = f.Partition.Uuid
	}
	if f.Type != nil {
		attrMap["object_type"] = f.Type.Code
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
	return fmt.Sprintf("PropertySchemaFilter(%s)", attrs)
}

// PropertySchemaWithReferences is a concrete struct containing pointers to
// already-constructed and validated Partition and ObjectType messages. This is
// the struct that is passed to backend storage when creating new property
// schemas, not the protobuffer PropertySchema message or the
// api/types/PropertySchema struct, neither of which are guaranteed to be
// pre-validated and their relations already expanded.
type PropertySchemaWithReferences struct {
	PropertySchema *pb.PropertySchema
	Partition      *pb.Partition
	Type           *pb.ObjectType
}
