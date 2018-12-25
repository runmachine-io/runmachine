package types

import pb "github.com/runmachine-io/runmachine/proto"

type Op int

const (
	OP_EQUAL            Op = 0
	OP_NOT_EQUAL           = 1
	OP_GREAT_THAN          = 2
	OP_GREAT_THAN_EQUAL    = 3
	OP_LESS_THAN           = 4
	OP_LESS_THAN_EQUAL     = 5
)

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

// TODO(jaypipes): Change this back to ObjectType
type HasType interface {
	GetType() string
}

type ObjectTypeCondition struct {
	Op         Op
	ObjectType *pb.ObjectType
}

func (c *ObjectTypeCondition) Matches(obj HasType) bool {
	if c == nil || c.ObjectType == nil {
		return true
	}
	cmp := obj.GetType()
	switch c.Op {
	case OP_EQUAL:
		return c.ObjectType.Code == cmp
	case OP_NOT_EQUAL:
		return c.ObjectType.Code != cmp
	default:
		return false
	}
}
