package conditions

import pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"

type HasObjectType interface {
	GetObjectType() string
}

type ObjectTypeCondition struct {
	Op         Op
	ObjectType *pb.ObjectType
}

func (c *ObjectTypeCondition) Matches(obj HasObjectType) bool {
	if c == nil || c.ObjectType == nil {
		return true
	}
	cmp := obj.GetObjectType()
	switch c.Op {
	case OP_EQUAL:
		return c.ObjectType.Code == cmp
	case OP_NOT_EQUAL:
		return c.ObjectType.Code != cmp
	default:
		return false
	}
}

// ObjectTypeEqual is a helper function that returns a ObjectTypeCondition
// filtering on an exact ObjectType object match
func ObjectTypeEqual(search *pb.ObjectType) *ObjectTypeCondition {
	return &ObjectTypeCondition{
		Op:         OP_EQUAL,
		ObjectType: search,
	}
}
