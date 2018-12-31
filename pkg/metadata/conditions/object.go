package conditions

import (
	"fmt"
	"strconv"

	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

// ObjectCondition is a class used in filtering objects.  Optional partition
// and object type PKs have already been expanded from user-supplied partition
// and type filter strings
type ObjectCondition struct {
	PartitionCondition  *PartitionCondition
	ObjectTypeCondition *ObjectTypeCondition
	UuidCondition       *UuidCondition
	NameCondition       *NameCondition
	ProjectCondition    string
	// TODO(jaypipes): Add support for property and tag filters
}

func (f *ObjectCondition) Matches(obj *pb.Object) bool {
	if !f.UuidCondition.Matches(obj) {
		return false
	}
	if !f.PartitionCondition.Matches(obj) {
		return false
	}
	if !f.ObjectTypeCondition.Matches(obj) {
		return false
	}
	if !f.NameCondition.Matches(obj) {
		return false
	}
	if f.ProjectCondition != "" && obj.Project != "" {
		if obj.Project != f.ProjectCondition {
			return false
		}
	}
	return true
}

func (f *ObjectCondition) IsEmpty() bool {
	return f.PartitionCondition == nil &&
		f.ObjectTypeCondition == nil &&
		f.UuidCondition == nil &&
		f.ProjectCondition == "" &&
		f.NameCondition == nil
}

func (f *ObjectCondition) String() string {
	attrMap := make(map[string]string, 0)
	if f.PartitionCondition != nil {
		attrMap["partition"] = f.PartitionCondition.Partition.Uuid
	}
	if f.ObjectTypeCondition != nil {
		attrMap["object_type"] = f.ObjectTypeCondition.ObjectType.Code
	}
	if f.UuidCondition != nil {
		attrMap["uuid"] = f.UuidCondition.Uuid
	}
	if f.ProjectCondition != "" {
		attrMap["project"] = f.ProjectCondition
	}
	if f.NameCondition != nil {
		attrMap["key"] = f.NameCondition.Name
		attrMap["use_prefix"] = strconv.FormatBool(
			f.NameCondition.Op == OP_GREATER_THAN_EQUAL,
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
