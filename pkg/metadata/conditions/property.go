package conditions

import (
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

type HasProperties interface {
	GetProperties() []*pb.Property
}

type PropertyCondition struct {
	RequireKeys  []string
	RequireItems []*pb.Property
	AnyKeys      []string
	AnyItems     []*pb.Property
	ForbidKeys   []string
	ForbidItems  []*pb.Property
}

func (c *PropertyCondition) Matches(obj HasProperties) bool {
	if c == nil {
		return true
	}
	if (c.RequireKeys == nil || len(c.RequireKeys) == 0) &&
		(c.RequireItems == nil || len(c.RequireItems) == 0) &&
		(c.AnyKeys == nil || len(c.AnyKeys) == 0) &&
		(c.AnyItems == nil || len(c.AnyItems) == 0) &&
		(c.ForbidKeys == nil || len(c.ForbidKeys) == 0) &&
		(c.ForbidItems == nil || len(c.ForbidItems) == 0) {
		return true
	}
	props := obj.GetProperties()
	if props == nil {
		return false
	}
	if len(c.RequireKeys) > 0 {
		foundKeys := 0
		for _, reqKey := range c.RequireKeys {
			for _, prop := range props {
				if reqKey == prop.Key {
					foundKeys += 1
				}
			}
		}
		if foundKeys != len(c.RequireKeys) {
			return false
		}
	}
	if len(c.RequireItems) > 0 {
		foundItems := 0
		for _, reqItem := range c.RequireItems {
			for _, prop := range props {
				if reqItem.Key == prop.Key && reqItem.Value == prop.Value {
					foundItems += 1
				}
			}
		}
		if foundItems != len(c.RequireItems) {
			return false
		}
	}
	if len(c.AnyKeys) > 0 {
		found := false
		for _, anyKey := range c.ForbidKeys {
			for _, prop := range props {
				if anyKey == prop.Key {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(c.AnyItems) > 0 {
		found := false
		for _, anyItem := range c.ForbidItems {
			for _, prop := range props {
				if anyItem.Key == prop.Key && anyItem.Value == prop.Value {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(c.ForbidKeys) > 0 {
		for _, forbidKey := range c.ForbidKeys {
			for _, prop := range props {
				if forbidKey == prop.Key {
					return false
				}
			}
		}
	}
	if len(c.ForbidItems) > 0 {
		for _, forbidItem := range c.ForbidItems {
			for _, prop := range props {
				if forbidItem.Key == prop.Key && forbidItem.Value == prop.Value {
					return false
				}
			}
		}
	}
	return true
}

// PropertyEqual is a helper function that returns a PropertyCondition
// filtering on an exact property item match
func PropertyConditionFromFilter(filter *pb.PropertyFilter) *PropertyCondition {
	return &PropertyCondition{
		RequireKeys:  filter.RequireKeys,
		RequireItems: filter.RequireItems,
		AnyKeys:      filter.AnyKeys,
		AnyItems:     filter.AnyItems,
		ForbidKeys:   filter.ForbidKeys,
		ForbidItems:  filter.ForbidItems,
	}
}
