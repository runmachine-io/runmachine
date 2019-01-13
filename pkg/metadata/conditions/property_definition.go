package conditions

import (
	"strings"
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

// PropertyKeyEqual is a helper function that returns a PropertyKeyCondition
// filtering on an exact property key match
func PropertyKeyEqual(search string) *PropertyKeyCondition {
	return &PropertyKeyCondition{
		Op:          OP_EQUAL,
		PropertyKey: search,
	}
}

// PropertyKeyLike is a helper function that returns a PropertyKeyCondition
// filtering on a prefixed property key match
func PropertyKeyLike(search string) *PropertyKeyCondition {
	return &PropertyKeyCondition{
		Op:          OP_GREATER_THAN_EQUAL,
		PropertyKey: search,
	}
}
