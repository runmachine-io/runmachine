package types

import (
	"strings"

	pb "github.com/runmachine-io/runmachine/proto"
)

type Op int

const (
	OP_EQUAL              Op = 0
	OP_NOT_EQUAL             = 1
	OP_GREATER_THAN          = 2
	OP_GREATER_THAN_EQUAL    = 3
	OP_LESSER_THAN           = 4
	OP_LESSER_THAN_EQUAL     = 5
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
