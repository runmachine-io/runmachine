package metadata

import (
	pb "github.com/runmachine-io/runmachine/proto"
)

func checkSession(sess *pb.Session) error {
	if sess.User == "" {
		return ErrSessionUserRequired
	}
	if sess.Partition == "" {
		return ErrSessionPartitionRequired
	}
	if sess.Project == "" {
		return ErrSessionProjectRequired
	}
	return nil
}
