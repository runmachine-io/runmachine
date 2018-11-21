package cursor

import (
	"fmt"

	"github.com/golang/protobuf/proto"
)

// implements abstract Cursor interface for an empty list of things
type EmptyCursor struct{}

func Empty() *EmptyCursor {
	return &EmptyCursor{}
}

func (c *EmptyCursor) Scan(msg proto.Message) error {
	return fmt.Errorf("attempted to Scan an empty cursor.")
}

func (c *EmptyCursor) Next() bool {
	return false
}

func (c *EmptyCursor) Close() error {
	return nil
}
