package server

import (
	"github.com/runmachine-io/runmachine/pkg/logging"
	"github.com/runmachine-io/runmachine/pkg/metadata/server/storage"
	pb "github.com/runmachine-io/runmachine/proto"
)

// ObjectTypeCache is a simple cache of object type messages that are very
// frequently accessed to answer things like whether a particular object is
// partition or project-scoped
//
// NOTE(jaypipes): This struct is deliberately not protected by a mutex since
// it is meant to be attached to the Server struct which itself is protected by
// the gRPC server-side mutex mechanisms already.
type ObjectTypeCache struct {
	log       *logging.Logs
	store     *storage.Store
	cache     map[string]*pb.ObjectType
	CountHit  uint64
	CountMiss uint64
}

// NewObjectTypeCache returns an initialized object type cache
func NewObjectTypeCache(
	log *logging.Logs,
	store *storage.Store,
) *ObjectTypeCache {
	return &ObjectTypeCache{
		log:       log,
		store:     store,
		cache:     make(map[string]*pb.ObjectType, 0),
		CountHit:  0,
		CountMiss: 0,
	}
}

// Get returns the ObjectType protobuffer message for the supplied code,
// retrieving the message from backend storage if not in the cache. Returns nil
// if no such object type could be found.
func (c *ObjectTypeCache) Get(code string) *pb.ObjectType {
	obj, ok := c.cache[code]
	if ok {
		c.CountHit += 1
		return obj
	}
	c.CountMiss += 1
	// Try looking up in backend storage and setting our cache entry if found
	obj, err := c.store.ObjectTypeGetByCode(code)
	if err != nil {
		c.log.ERR(
			"failed to retrieve object type of %s: %s",
			code, err,
		)
		return nil
	}
	c.cache[code] = obj
	return obj
}

// ScopeOf returns the object type's scope given an object type code. This
// method has no mechanism for returning any error state and is intended to be
// used when the caller absolutely knows that the object type with the supplied
// code actually exists. If the object type with that code does not exist or
// there was some failure in retrieving the object type from backend storage,
// the function returns ObjectTypeScope_PARTITION.
func (c *ObjectTypeCache) ScopeOf(code string) pb.ObjectTypeScope {
	ot := c.Get(code)
	if ot == nil {
		return pb.ObjectTypeScope_PARTITION
	}
	return ot.Scope
}
