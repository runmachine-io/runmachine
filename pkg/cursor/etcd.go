package cursor

import (
	"fmt"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/golang/protobuf/proto"
)

// implements abstract Cursor interface for an etcd key-value store that is
// storing protobuffer messages as values.
// NOTE(jaypipes): It is NOT safe to share these objects between threads. The
// idxScan member is not protected.
type EtcdGetResponseCursor struct {
	resp    *etcd.GetResponse
	idxScan int64 // The last record that was Scan()'d
}

func NewFromEtcdGetResponse(resp *etcd.GetResponse) *EtcdGetResponseCursor {
	return &EtcdGetResponseCursor{
		resp:    resp,
		idxScan: 0,
	}
}

func (c *EtcdGetResponseCursor) Scan(msg proto.Message) error {
	idx := c.idxScan
	if idx > c.resp.Count {
		return fmt.Errorf("attempted to read past end of etcd get response cursor.")
	}
	if err := proto.Unmarshal(c.resp.Kvs[idx].Value, msg); err != nil {
		return err
	}
	c.idxScan += 1
	return nil
}

func (c *EtcdGetResponseCursor) Next() bool {
	cnt := c.resp.Count
	return cnt > 0 && c.idxScan < cnt
}

func (c *EtcdGetResponseCursor) Close() error {
	return nil
}
