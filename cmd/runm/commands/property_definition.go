package commands

import (
	"fmt"
	"strconv"

	apitypes "github.com/runmachine-io/runmachine/pkg/api/types"
	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var propertyDefinitionCommand = &cobra.Command{
	Use:   "property-definition",
	Short: "Manipulate property definition information",
}

func init() {
	propertyDefinitionCommand.AddCommand(propertyDefinitionListCommand)
	propertyDefinitionCommand.AddCommand(propertyDefinitionGetCommand)
	propertyDefinitionCommand.AddCommand(propertyDefinitionSetCommand)
	propertyDefinitionCommand.AddCommand(propertyDefinitionDeleteCommand)
}

func printPropertyDefinition(obj *pb.PropertyDefinition) {
	fmt.Printf("Partition:    %s\n", obj.Partition)
	fmt.Printf("Type:         %s\n", obj.Type)
	fmt.Printf("Key:          %s\n", obj.Key)
	fmt.Printf("Required:     %s\n", strconv.FormatBool(obj.IsRequired))
	if len(obj.Permissions) > 0 {
		fmt.Printf("Permissions:\n")
		for x, perm := range obj.Permissions {
			permStr := ""
			if perm.Project != nil {
				permStr += "project: " + perm.Project.Value
			}
			if perm.Role != nil {
				if len(permStr) > 0 {
					permStr += " "
				}
				permStr += "role: " + perm.Role.Value
			}
			if len(permStr) > 0 {
				permStr += " "
			}
			permStr += "permission: "
			readBit := perm.Permission & apitypes.PERMISSION_READ
			writeBit := perm.Permission & apitypes.PERMISSION_WRITE
			if readBit != 0 {
				if writeBit != 0 {
					permStr += "READ/WRITE"
				} else {
					permStr += "READ"
				}
			} else if writeBit != 0 {
				permStr += "WRITE"
			} else {
				permStr += "NONE (Deny)"
			}
			fmt.Printf("  %d: %s\n", x+1, permStr)
		}
	}
	fmt.Printf("Schema:\n%s", obj.Schema)
}
