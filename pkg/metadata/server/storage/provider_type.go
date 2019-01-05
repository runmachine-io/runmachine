package storage

import (
	"strings"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/golang/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

// TODO(jaypipes): This file has lots of copied code from
// storage/object_type.go. DRY that up.

const (
	// $ROOT/types/runm.provider/ is a key namespace containing valued keys
	// where the key is the provider type's code and the value is the
	// serialized ProviderType protobuffer message
	_PROVIDER_TYPES_KEY = "types/runm.provider/"
)

var (
	// The collection of well-known runm object types
	runmProviderTypes = []*pb.ProviderType{
		&pb.ProviderType{
			Code:        "runm.compute",
			Description: "A provider of CPU, memory, local block storage, etc",
		},
		&pb.ProviderType{
			Code:        "runm.storage.block",
			Description: "A provider of block storage",
		},
	}
)

// ensureProviderTypes is responsible for making sure etcd has the well-known
// runm provider types in storage.
func (s *Store) ensureProviderTypes() error {
	ctx, cancel := s.requestCtx()
	defer cancel()

	s.log.L3("ensuring provider types...")

	resp, err := s.kv.Get(
		ctx,
		_PROVIDER_TYPES_KEY,
		etcd.WithPrefix(),
		etcd.WithKeysOnly(),
	)
	if err != nil {
		s.log.ERR("error listing provider types: %v", err)
		return err
	}
	all := make(map[string]bool, 0)
	for _, k := range resp.Kvs {
		ptCode := strings.TrimPrefix(string(k.Key), _PROVIDER_TYPES_KEY)
		all[ptCode] = true
	}

	for _, ot := range runmProviderTypes {
		if _, ok := all[ot.Code]; !ok {
			s.log.L3("provider type %s not in storage. adding...", ot.Code)
			if err = s.providerTypeCreate(ot); err != nil {
				if err == errors.ErrDuplicate {
					// some other thread created the type... just ignore
					continue
				}
				return err
			}
			s.log.L2("created provider type %s", ot.Code)
		}
	}
	return nil
}

// ProviderTypeGet returns an ProviderType protobuffer message having the
// supplied code
func (s *Store) ProviderTypeGet(
	code string,
) (*pb.ProviderType, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	key := _PROVIDER_TYPES_KEY + code
	resp, err := s.kv.Get(ctx, key)
	if err != nil {
		s.log.ERR("error getting key %s: %v", key, err)
		return nil, err
	}

	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}

	obj := &pb.ProviderType{}
	if err = proto.Unmarshal(resp.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// ProviderTypeList returns a slice of pointers to ProviderType protobuffer
// messages matching a set of supplied filters.
func (s *Store) ProviderTypeList(
	any []*pb.ProviderTypeFilter,
) ([]*pb.ProviderType, error) {
	if len(any) == 0 {
		// Just return all object types
		return s.providerTypesGetByCode("", true)
	}

	// Each filter is evaluated in an OR fashion, so we keep a hashmap of
	// provider type codes in order to return unique results
	objs := make(map[string]*pb.ProviderType, 0)
	for _, filter := range any {
		if filter.CodeFilter != nil {
			filterObjs, err := s.providerTypesGetByCode(
				filter.CodeFilter.Code,
				filter.CodeFilter.UsePrefix,
			)
			if err != nil {
				return nil, err
			}
			for _, obj := range filterObjs {
				objs[obj.Code] = obj
			}
		}
	}
	res := make([]*pb.ProviderType, len(objs))
	x := 0
	for _, obj := range objs {
		res[x] = obj
		x += 1
	}
	return res, nil
}

func (s *Store) providerTypesGetByCode(
	code string,
	usePrefix bool,
) ([]*pb.ProviderType, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	key := _PROVIDER_TYPES_KEY + code

	opts := []etcd.OpOption{
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
	}

	if usePrefix {
		opts = append(opts, etcd.WithPrefix())
	}

	resp, err := s.kv.Get(ctx, key, opts...)
	if err != nil {
		s.log.ERR("error listing provider types: %v", err)
		return nil, err
	}

	if resp.Count == 0 {
		return []*pb.ProviderType{}, nil
	}

	res := make([]*pb.ProviderType, resp.Count)
	for x, kv := range resp.Kvs {
		msg := &pb.ProviderType{}
		if err := proto.Unmarshal(kv.Value, msg); err != nil {
			return nil, err
		}
		res[x] = msg
	}

	return res, nil
}

// providerTypeCreate writes the supplied ObjectType provider to the key at
// $ROOT/provider-types/{provider_type_code}
func (s *Store) providerTypeCreate(
	obj *pb.ProviderType,
) error {
	ctx, cancel := s.requestCtx()
	defer cancel()

	key := _PROVIDER_TYPES_KEY + obj.Code
	value, err := proto.Marshal(obj)
	if err != nil {
		return err
	}
	// create the provider type using a transaction that ensures another thread
	// hasn't created an provider type with the same key underneath us
	onSuccess := etcd.OpPut(key, string(value))
	// Ensure the key doesn't yet exist
	compare := etcd.Compare(etcd.Version(key), "=", 0)
	resp, err := s.kv.Txn(ctx).If(compare).Then(onSuccess).Commit()

	if err != nil {
		s.log.ERR("failed to create txn in etcd: %v", err)
		return err
	} else if resp.Succeeded == false {
		return errors.ErrDuplicate
	}
	return nil
}
