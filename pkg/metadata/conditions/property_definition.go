package conditions

import (
	"fmt"
	"strconv"
	"strings"

	pb "github.com/runmachine-io/runmachine/proto"
)

type HasKey interface {
	GetKey() string
}

type PropertyKeyCondition struct {
	Op          Op
	PropertyKey string
}

func (c *PropertyKeyCondition) Matches(obj HasKey) bool {
	if c == nil || c.PropertyKey == "" {
		return true
	}
	cmp := obj.GetKey()
	switch c.Op {
	case OP_EQUAL:
		return c.PropertyKey == cmp
	case OP_NOT_EQUAL:
		return c.PropertyKey != cmp
	case OP_GREATER_THAN_EQUAL:
		return strings.HasPrefix(cmp, c.PropertyKey)
	default:
		return false
	}
}

// PropertyDefinitionCondition is a class used in filtering property definitions.
// Optional partition and object type PKs have already been expanded from
// user-supplied partition and type filter strings
type PropertyDefinitionCondition struct {
	PartitionCondition   *PartitionCondition
	ObjectTypeCondition  *ObjectTypeCondition
	UuidCondition        *UuidCondition
	PropertyKeyCondition *PropertyKeyCondition
}

func (f *PropertyDefinitionCondition) Matches(obj *pb.PropertyDefinition) bool {
	if !f.UuidCondition.Matches(obj) {
		return false
	}
	if !f.PartitionCondition.Matches(obj) {
		return false
	}
	if !f.ObjectTypeCondition.Matches(obj) {
		return false
	}
	if !f.PropertyKeyCondition.Matches(obj) {
		return false
	}
	return true
}

func (f *PropertyDefinitionCondition) IsEmpty() bool {
	return f.PartitionCondition == nil &&
		f.ObjectTypeCondition == nil &&
		f.PropertyKeyCondition == nil &&
		f.UuidCondition == nil
}

func (f *PropertyDefinitionCondition) String() string {
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
	if f.PropertyKeyCondition != nil {
		attrMap["key"] = f.PropertyKeyCondition.PropertyKey
		attrMap["use_prefix"] = strconv.FormatBool(
			f.PropertyKeyCondition.Op == OP_GREATER_THAN_EQUAL,
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
	return fmt.Sprintf("PropertyDefinitionCondition(%s)", attrs)
}
