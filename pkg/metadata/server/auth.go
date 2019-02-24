package server

import (
	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
)

func (s *Server) checkSession(sess *pb.Session) error {
	if sess.User == "" {
		return ErrSessionUserRequired
	}
	if sess.Partition == "" {
		return ErrSessionPartitionRequired
	} else {
		// If the Session's partition identifier isn't a UUID, convert it to a
		// partition UUID from a name and return an error if no partition with
		// that name exists
		if !util.IsUuidLike(sess.Partition) {
			part, err := s.store.PartitionGetByName(sess.Partition)
			if err != nil {
				if err == errors.ErrNotFound {
					return errSessionUnknownPartition(sess.Partition)
				}
				return err
			}
			sess.Partition = part.Uuid
		}
	}
	if sess.Project == "" {
		return ErrSessionProjectRequired
	}
	return nil
}
