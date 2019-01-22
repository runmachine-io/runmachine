package commands

import (
	"fmt"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

const (
	// TODO(jaypipes): Move these to a generic location?
	PERMISSION_NONE  = uint32(0)
	PERMISSION_READ  = uint32(1)
	PERMISSION_WRITE = uint32(1) << 1
)

func printPropertyPermissions(obj *pb.PropertyPermissions) {
	fmt.Printf("  Key: %s\n", obj.Key)
	for x, perm := range obj.Permissions {
		fmt.Printf("    %d: ", x)
		printPropertyPermission(perm)
	}

}

func printPropertyPermission(obj *pb.PropertyPermission) {
	if obj.Project == "" && obj.Role == "" {
		fmt.Printf("GLOBAL ")
	} else {
		if obj.Project != "" {
			fmt.Printf("PROJECT(" + obj.Project + ") ")
		}
		if obj.Role != "" {
			fmt.Printf("ROLE(" + obj.Role + ") ")
		}
	}
	readBit := obj.Permission & PERMISSION_READ
	writeBit := obj.Permission & PERMISSION_WRITE
	if readBit != 0 {
		if writeBit != 0 {
			fmt.Printf("READ/WRITE\n")
		} else {
			fmt.Printf("READ\n")
		}
	} else if writeBit != 0 {
		fmt.Printf("WRITE\n")
	} else {
		fmt.Printf("NONE (Deny)\n")
	}
}

func printObjectDefinition(obj *pb.ObjectDefinition) {
	fmt.Printf("Schema:\n%s", obj.Schema)
	if obj.PropertyPermissions != nil && len(obj.PropertyPermissions) > 0 {
		fmt.Printf("Property permissions:\n")
		for _, propPerms := range obj.PropertyPermissions {
			printPropertyPermissions(propPerms)
		}
	}
}
