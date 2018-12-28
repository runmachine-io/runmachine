package conditions

type HasUuid interface {
	GetUuid() string
}

type UuidCondition struct {
	Op   Op
	Uuid string
}

func (c *UuidCondition) Matches(obj HasUuid) bool {
	if c == nil || c.Uuid == "" {
		return true
	}
	cmp := obj.GetUuid()
	switch c.Op {
	case OP_EQUAL:
		return c.Uuid == cmp
	case OP_NOT_EQUAL:
		return c.Uuid != cmp
	default:
		return false
	}
}
