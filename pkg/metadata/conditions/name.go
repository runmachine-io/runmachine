package conditions

import "strings"

type HasName interface {
	GetName() string
}

type NameCondition struct {
	Op   Op
	Name string
}

func (c *NameCondition) Matches(obj HasName) bool {
	if c == nil || c.Name == "" {
		return true
	}
	cmp := obj.GetName()
	switch c.Op {
	case OP_EQUAL:
		return c.Name == cmp
	case OP_NOT_EQUAL:
		return c.Name != cmp
	case OP_GREATER_THAN_EQUAL:
		return strings.HasPrefix(cmp, c.Name)
	default:
		return false
	}
}
