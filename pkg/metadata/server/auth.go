package server

import (
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
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
