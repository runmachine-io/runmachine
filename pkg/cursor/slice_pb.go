package cursor

import (
	"fmt"

	"github.com/golang/protobuf/proto"
)

// implements abstract Cursor interface for a slice of protobuffer message
// struct pointers
// NOTE(jaypipes): It is NOT safe to share these objects between threads. The
// idxScan member is not protected.
type SlicePBMessageCursor struct {
	vals    []proto.Message
	idxScan int // The last record that was Scan()'d
}

func NewFromSlicePBMessages(vals []proto.Message) *SlicePBMessageCursor {
	return &SlicePBMessageCursor{
		vals:    vals,
		idxScan: 0,
	}
}

func (c *SlicePBMessageCursor) Scan(msg proto.Message) error {
	idx := c.idxScan
	if idx > len(c.vals) {
		return fmt.Errorf("attempted to read past end of slice of protobuffer messages cursor.")
	}

	proto.Merge(msg, c.vals[c.idxScan])
	c.idxScan += 1
	return nil
}

func (c *SlicePBMessageCursor) Next() bool {
	cnt := len(c.vals)
	return cnt > 0 && c.idxScan < cnt
}

func (c *SlicePBMessageCursor) Close() error {
	return nil
}
