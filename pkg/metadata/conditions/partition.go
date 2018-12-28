package conditions

import pb "github.com/runmachine-io/runmachine/proto"

type HasPartition interface {
	GetPartition() string
}

type PartitionCondition struct {
	Op        Op
	Partition *pb.Partition
}

func (c *PartitionCondition) Matches(obj HasPartition) bool {
	if c == nil || c.Partition == nil {
		return true
	}
	cmp := obj.GetPartition()
	switch c.Op {
	case OP_EQUAL:
		return c.Partition.Uuid == cmp
	case OP_NOT_EQUAL:
		return c.Partition.Uuid != cmp
	default:
		return false
	}
}

// PartitionEqual is a helper function that returns a PartitionCondition
// filtering on an exact Partition object match
func PartitionEqual(search *pb.Partition) *PartitionCondition {
	return &PartitionCondition{
		Op:        OP_EQUAL,
		Partition: search,
	}
}
