package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/runmachine-io/runmachine/pkg/util"
)

var (
	key = "TESTING"
)

func TestNormalizeUuid(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		val    string
		expect string
	}{
		{
			val:    "764f59ca-595d-4bb0-b140-00ae16a6ccb8",
			expect: "764f59ca595d4bb0b14000ae16a6ccb8",
		},
		{
			val:    "764F59CA-595D-4BB0-B140-00AE16A6CCB8",
			expect: "764f59ca595d4bb0b14000ae16a6ccb8",
		},
		{
			val:    "764F59CA595D4BB0B14000AE16A6CCB8",
			expect: "764f59ca595d4bb0b14000ae16a6ccb8",
		},
		{
			val:    "764f59ca595d4bb0b14000ae16a6ccb8",
			expect: "764f59ca595d4bb0b14000ae16a6ccb8",
		},
	}

	for _, t := range tests {
		assert.Equal(t.expect, util.NormalizeUuid(t.val))
	}
}

func TestIsUuidLike(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		val    string
		expect bool
	}{
		{
			val:    "764f59ca-595d-4bb0-b140-00ae16a6ccb8",
			expect: true,
		},
		{
			val:    "764F59CA-595D-4BB0-B140-00AE16A6CCB8",
			expect: true,
		},
		{
			val:    "764F59CA595D4BB0B14000AE16A6CCB8",
			expect: true,
		},
		{
			val:    "764f59ca595d4bb0b14000ae16a6ccb8",
			expect: true,
		},
		{
			// 36 chars but not a UUID...
			val:    "764f59ca595d4bb0b14000ae16a6ccb80000",
			expect: false,
		},
		{
			// 32 chars but not a UUID...
			val:    "xxxx59ca595d4bb0b14000ae16a6ccb8",
			expect: false,
		},
		{
			val:    "not a uuid",
			expect: false,
		},
	}

	for _, t := range tests {
		assert.Equal(t.expect, util.IsUuidLike(t.val))
	}
}
