package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

const (
	// TODO(jaypipes): Move these to a generic location?
	PERMISSION_NONE  = uint32(0)
	PERMISSION_READ  = uint32(1)
	PERMISSION_WRITE = uint32(1) << 1
)

var providerDefinitionCommand = &cobra.Command{
	Use:   "definition",
	Short: "Manipulate provider definitions",
}

func init() {
	providerDefinitionCommand.AddCommand(providerDefinitionGetCommand)
	providerDefinitionCommand.AddCommand(providerDefinitionSetCommand)
}

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

func printProviderDefinition(obj *pb.ProviderDefinition) {
	fmt.Printf("Partition: %s\n", obj.Partition)
	fmt.Printf("Schema:\n%s", obj.Schema)
	if len(obj.PropertyPermissions) > 0 {
		fmt.Printf("Property permissions:\n")
		for _, propPerms := range obj.PropertyPermissions {
			printPropertyPermissions(propPerms)
		}
	}
}
