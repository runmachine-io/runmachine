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

// UuidEqual is a helper function that returns a UuidCondition filtering on an
// exact UUID match
func UuidEqual(search string) *UuidCondition {
	return &UuidCondition{
		Op:   OP_EQUAL,
		Uuid: search,
	}
}

// UuidsCondition describes an IN(<uuid>, <uuid>, ...) or NOT IN (<uuid>,
// <uuid>, ...) selection
type UuidsCondition struct {
	Op    Op
	Uuids []string
}

func (c *UuidsCondition) Matches(obj HasUuid) bool {
	if c == nil || c.Uuids == nil || len(c.Uuids) == 0 {
		return true
	}
	cmp := obj.GetUuid()
	switch c.Op {
	case OP_IN:
		for _, uuid := range c.Uuids {
			if uuid == cmp {
				return true
			}
		}
		return false
	case OP_NOT_IN:
		for _, uuid := range c.Uuids {
			if uuid == cmp {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// UuidIn is a helper function that returns a UuidsCondition matching objects
// with an IN (<uuid>, <uuid>, ...) expression
func UuidIn(any []string) *UuidsCondition {
	return &UuidsCondition{
		Op:    OP_IN,
		Uuids: any,
	}
}

// UuidNotIn is a helper function that returns a UuidsCondition matching objects
// with a NOT IN (<uuid>, <uuid>, ...) expression
func UuidNotIn(any []string) *UuidsCondition {
	return &UuidsCondition{
		Op:    OP_NOT_IN,
		Uuids: any,
	}
}
