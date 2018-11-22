package metadata

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrUnknown = status.Errorf(
		codes.Unknown,
		"an unknown error occurred.",
	)
	ErrNotFound = status.Errorf(
		codes.NotFound,
		"object could not be found.",
	)
	ErrPartitionUnknown = status.Errorf(
		codes.FailedPrecondition,
		"unknown partition.",
	)
	ErrPartitionRequired = status.Errorf(
		codes.FailedPrecondition,
		"partition is required.",
	)
	ErrObjectTypeRequired = status.Errorf(
		codes.FailedPrecondition,
		"object type is required.",
	)
	ErrPropertyKeyRequired = status.Errorf(
		codes.FailedPrecondition,
		"property key is required.",
	)
	ErrSchemaRequired = status.Errorf(
		codes.FailedPrecondition,
		"schema is required.",
	)
	ErrPropertySchemaObjectRequired = status.Errorf(
		codes.FailedPrecondition,
		"property schema object is required.",
	)
	ErrSearchRequired = status.Errorf(
		codes.FailedPrecondition,
		"Either UUID or name to search for is required.",
	)
	ErrCodeRequired = status.Errorf(
		codes.FailedPrecondition,
		"A code to search for is required.",
	)
	ErrBootstrapTokenRequired = status.Errorf(
		codes.FailedPrecondition,
		"bootstrap token is required.",
	)
	ErrPartitionNameRequired = status.Errorf(
		codes.FailedPrecondition,
		"partition name is required.",
	)
)

func errPartitionNotFound(partition string) error {
	return status.Errorf(
		codes.FailedPrecondition,
		"Partition %s not found", partition,
	)
}

func errObjectTypeNotFound(objectType string) error {
	return status.Errorf(
		codes.FailedPrecondition,
		"Object type %s not found", objectType,
	)
}
