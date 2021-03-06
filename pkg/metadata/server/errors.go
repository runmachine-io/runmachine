package server

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrUnknown = status.Errorf(
		codes.Unknown,
		"an unknown error occurred.",
	)
	ErrDuplicate = status.Errorf(
		codes.AlreadyExists,
		"duplicate record.",
	)
	ErrNotFound = status.Errorf(
		codes.NotFound,
		"object could not be found.",
	)
	ErrSessionUserRequired = status.Errorf(
		codes.FailedPrecondition,
		"user is required in session.",
	)
	ErrSessionPartitionRequired = status.Errorf(
		codes.FailedPrecondition,
		"partition is required in session.",
	)
	ErrSessionProjectRequired = status.Errorf(
		codes.FailedPrecondition,
		"project is required in session.",
	)
	ErrFailedExpandObjectFilters = status.Errorf(
		codes.FailedPrecondition,
		"failed to expand object filters.",
	)
	ErrCodeFilterRequired = status.Errorf(
		codes.FailedPrecondition,
		"code filter is required",
	)
	ErrObjectFilterRequired = status.Errorf(
		codes.FailedPrecondition,
		"object filter is required when fetching object.",
	)
	ErrFailedExpandPropertyDefinitionFilters = status.Errorf(
		codes.FailedPrecondition,
		"failed to expand property definition filters.",
	)
	ErrMultipleRecordsFound = status.Errorf(
		codes.FailedPrecondition,
		"multiple records found (expected single record match).",
	)
	ErrPartitionUnknown = status.Errorf(
		codes.FailedPrecondition,
		"unknown partition.",
	)
	ErrNameRequired = status.Errorf(
		codes.FailedPrecondition,
		"name is required.",
	)
	ErrUuidRequired = status.Errorf(
		codes.FailedPrecondition,
		"UUID is required.",
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
	ErrPropertyDefinitionFilterRequired = status.Errorf(
		codes.FailedPrecondition,
		"property definition filter is required.",
	)
	ErrPropertyDefinitionObjectRequired = status.Errorf(
		codes.FailedPrecondition,
		"property definition object is required.",
	)
	ErrSearchRequired = status.Errorf(
		codes.FailedPrecondition,
		"Either UUID or name to search for is required.",
	)
	ErrCodeRequired = status.Errorf(
		codes.FailedPrecondition,
		"A code to search for is required.",
	)
	ErrObjectTypeCodeRequired = status.Errorf(
		codes.FailedPrecondition,
		"object type code is required.",
	)
	ErrBootstrapTokenRequired = status.Errorf(
		codes.FailedPrecondition,
		"bootstrap token is required.",
	)
	ErrPartitionUuidRequired = status.Errorf(
		codes.FailedPrecondition,
		"partition UUID is required.",
	)
	ErrPartitionNameRequired = status.Errorf(
		codes.FailedPrecondition,
		"partition name is required.",
	)
	ErrProviderTypeCodeRequired = status.Errorf(
		codes.FailedPrecondition,
		"provider type code is required.",
	)
	ErrAtLeastOneUuidRequired = status.Errorf(
		codes.FailedPrecondition,
		"at least one UUID is required.",
	)
	ErrObjectDeleteFailed = status.Errorf(
		codes.FailedPrecondition,
		"failed to delete object (check response errors collection).",
	)
	ErrAtLeastOneObjectDefinitionFilterRequired = status.Errorf(
		codes.FailedPrecondition,
		"at least one object definition filter is required.",
	)
	ErrObjectDefinitionFilterInvalid = status.Errorf(
		codes.FailedPrecondition,
		"invalid object definition filter.",
	)
	ErrObjectDefinitionDeleteFailed = status.Errorf(
		codes.FailedPrecondition,
		"failed to delete object definition (check response errors collection).",
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

func errProviderTypeNotFound(providerType string) error {
	return status.Errorf(
		codes.FailedPrecondition,
		"Provider type %s not found", providerType,
	)
}

func errSessionUnknownPartition(partition string) error {
	return status.Errorf(
		codes.FailedPrecondition,
		"Unknown partition '%s' specified in session", partition,
	)
}
