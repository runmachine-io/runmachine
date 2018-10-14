package cursor

import (
	"errors"
	"fmt"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/golang/protobuf/proto"
)

var errNilPtr = errors.New("destination pointer is nil") // embedded in descriptive error

// implements abstract Cursor interface for an etcd key-value store that is
// storing protobuffer messages as values.
// NOTE(jaypipes): It is NOT safe to share these objects between threads. The
// idxScan member is not protected.
type EtcdPBCursor struct {
	resp    *etcd.GetResponse
	idxScan int // The last record that was Scan()'d
}

func NewEtcdPBCursor(resp *etcd.GetResponse) *EtcdPBCursor {
	return &EtcdPBCursor{
		resp:    resp,
		idxScan: 0,
	}
}

func (c *EtcdPBCursor) Scan(dest ...interface{}) error {
	idx := c.idxScan
	if idx > len(c.resp.Kvs) {
		return fmt.Errorf("attempted to read past end of etcd cursor.")
	}
	if err := copyToDest(dest[0], c.resp.Kvs[idx].Key); err != nil {
		return err
	}
	if err := proto.Unmarshal(c.resp.Kvs[idx].Value, dest[1].(proto.Message)); err != nil {
		return err
	}
	c.idxScan += 1
	return nil
}

func copyToDest(dest interface{}, src []byte) error {
	switch d := dest.(type) {
	case *string:
		if d == nil {
			return errNilPtr
		}
		*d = string(src)
		return nil
	case *[]byte:
		if d == nil {
			return errNilPtr
		}
		*d = cloneBytes(src)
		return nil
	}
	return fmt.Errorf("unsupported Scan from src type %T into dest type %T", src, dest)
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

func (c *EtcdPBCursor) Next() bool {
	cnt := len(c.resp.Kvs)
	return cnt > 0 && c.idxScan < cnt
}

func (c *EtcdPBCursor) Close() error {
	return nil
}
