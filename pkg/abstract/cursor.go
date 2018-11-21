package abstract

import "github.com/golang/protobuf/proto"

// Generic interface for anything that can iterate over some list of things and
// parse a protobuf message. Purposefully made to function like database/sql
// package's Rows interface.  Useful for test mocking and abstracting a storage
// layer.
type Cursor interface {
	Next() bool
	Close() error
	Scan(msg proto.Message) error
}
