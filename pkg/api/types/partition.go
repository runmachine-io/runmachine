package types

import "fmt"

type Partition struct {
	// The UUID of the partition. Expected to be blank when a user is creating a
	// new partition.
	Uuid string `yaml:"uuid"`
	// Human-readable name for the partition. Uniqueness is guaranteed in the
	// scope of the runmachine deployment
	Name string `yaml:"name"`
}

// Validate returns an error if the definition is invalid, nil otherwise
func (p *Partition) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("name required")
	}
	return nil
}
