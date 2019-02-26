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
	UuidsCondition      *UuidsCondition
	NameCondition       *NameCondition
	ProjectCondition    string
	PropertyCondition   *PropertyCondition
	// TODO(jaypipes): Add support for tag filters
}

func (f *ObjectCondition) Matches(obj *pb.Object) bool {
	if !f.UuidCondition.Matches(obj) {
		return false
	}
	if !f.UuidsCondition.Matches(obj) {
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
	if !f.PropertyCondition.Matches(obj) {
		return false
	}
	return true
}

func (f *ObjectCondition) IsEmpty() bool {
	return f.PartitionCondition == nil &&
		f.ObjectTypeCondition == nil &&
		f.UuidCondition == nil &&
		f.UuidsCondition == nil &&
		f.ProjectCondition == "" &&
		f.PropertyCondition == nil &&
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
	if f.UuidsCondition != nil {
		attrMap["uuids"] = fmt.Sprintf("%s", f.UuidsCondition.Uuids)
	}
	if f.ProjectCondition != "" {
		attrMap["project"] = f.ProjectCondition
	}
	if f.NameCondition != nil {
		attrMap["name"] = f.NameCondition.Name
		attrMap["use_prefix"] = strconv.FormatBool(
			f.NameCondition.Op == OP_GREATER_THAN_EQUAL,
		)
	}
	if f.PropertyCondition != nil {
		attrMap["properties"] = fmt.Sprintf(
			"reqkeys=%s,reqitems=%s,anykeys=%s,anyitems=%s,forbidkeys=%s,forbiditems=%s",
			f.PropertyCondition.RequireKeys,
			f.PropertyCondition.RequireItems,
			f.PropertyCondition.AnyKeys,
			f.PropertyCondition.AnyItems,
			f.PropertyCondition.ForbidKeys,
			f.PropertyCondition.ForbidItems,
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
