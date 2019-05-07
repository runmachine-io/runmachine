package conditions

type HasPartitionUuid interface {
	GetPartitionUuid() string
}

type PartitionCondition struct {
	Op      Op
	Operand string
}

func (c *PartitionCondition) Matches(obj HasPartitionUuid) bool {
	if c == nil || c.Operand == "" {
		return true
	}
	cmp := obj.GetPartitionUuid()
	switch c.Op {
	case OP_EQUAL:
		return c.Operand == cmp
	case OP_NOT_EQUAL:
		return c.Operand != cmp
	default:
		return false
	}
}

// PartitionEqual is a helper function that returns a PartitionCondition
// filtering on an exact Partition object match
func PartitionEqual(search string) *PartitionCondition {
	return &PartitionCondition{
		Op:      OP_EQUAL,
		Operand: search,
	}
}
