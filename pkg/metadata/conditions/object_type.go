package conditions

type HasObjectTypeCode interface {
	GetObjectTypeCode() string
}

type ObjectTypeCondition struct {
	Op      Op
	Operand string
}

func (c *ObjectTypeCondition) Matches(obj HasObjectTypeCode) bool {
	if c == nil || c.Operand == "" {
		return true
	}
	cmp := obj.GetObjectTypeCode()
	switch c.Op {
	case OP_EQUAL:
		return c.Operand == cmp
	case OP_NOT_EQUAL:
		return c.Operand != cmp
	default:
		return false
	}
}

// ObjectTypeEqual is a helper function that returns a ObjectTypeCondition
// filtering on an exact ObjectType object match
func ObjectTypeEqual(search string) *ObjectTypeCondition {
	return &ObjectTypeCondition{
		Op:      OP_EQUAL,
		Operand: search,
	}
}
