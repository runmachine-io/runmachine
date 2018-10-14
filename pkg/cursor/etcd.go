package cursor

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	etcd "go.etcd.io/etcd/clientv3"
)

// implements abstract Cursor interface for an etcd key-value store that is
// storing protobuffer messages as values.
// NOTE(jaypipes): It is NOT safe to share these objects between threads. The
// idxScan member is not protected.
type EtcdPBCursor struct {
	resp    *etcd.GetResponse
	idxScan int // The last record that was Scan()'d
}

func (c *EtcdPBCursor) Scan(key *[]byte, value proto.Message) error {
	idx := c.idxScan
	if idx > len(c.resp.Kvs) {
		return fmt.Errorf("attempted to read past end of etcd cursor.")
	}
	*key = c.resp.Kvs[idx].Key
	if err := proto.Unmarshal(c.resp.Kvs[idx].Value, value); err != nil {
		return err
	}
	c.idxScan += 1
	return nil
}

func (c *EtcdPBCursor) Next() bool {
	cnt := len(c.resp.Kvs)
	return cnt > 0 && c.idxScan < cnt
}

func (c *EtcdPBCursor) Close() error {
	return nil
}
