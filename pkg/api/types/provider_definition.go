package types

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

var (
	baseProviderRequiredFields = []string{
		"partition",
		"provider_type",
	}
	providerSchemaTemplateContents = `
{
  "$id": "https://runmachine.io/runm.provider.schema.json",
  "$schema": "http://json-schema.org/draft-07/schema#",
  "description": "A provider of resources",
  "type": "object",
  "properties": {
    "partition": {
      "type": "string"
    },
    "provider_type": {
      "type": "string"
    },
    "properties": {
      "type": "object",
      {{ if len .PropertySchemas -}}
      "properties": {
{{- range .PropertySchemas -}}
{{ template "property-schema" . }}
{{- end }}
      },{{ end }}
      "patternProperties": {
        "^[a-zA-Z0-9]*$": {
          "type": "string"
        }
      }
    }
  },
  "required": [{{ quote_join .RequiredFields ", " }}],
  "additionalProperties": false
}
`
	providerSchemaTemplate *template.Template
	templateFuncMap        = template.FuncMap{
		"join":  strings.Join,
		"quote": strconv.Quote,
		"quote_join": func(elems []string, delim string) string {
			quoted := make([]string, len(elems))
			for x, elem := range elems {
				quoted[x] = strconv.Quote(elem)
			}
			return strings.Join(quoted, delim)
		},
		"deref_int": func(x *int) int {
			return *x
		},
		"deref_uint": func(x *uint) uint {
			return *x
		},
	}
)

func init() {
	providerSchemaTemplate = template.Must(
		template.New(
			"provider-schema",
		).Funcs(
			templateFuncMap,
		).Parse(
			providerSchemaTemplateContents,
		),
	)
	// include the property schema template. I wish golang's template
	// construction wasn't so bonkers...
	_, err := providerSchemaTemplate.Parse(
		propertySchemaTemplateContents,
	)
	if err != nil {
		panic(err)
	}
}

// ProviderDefinition is used by runmachine system administrators to constrain
// the properties that may be set on provider objects
type ProviderDefinition struct {
	// Identifier of the partition the object belongs to
	Partition string `yaml:"partition"`
	// Named properties may have their values constrained by a property
	// definition. The map key is the key of the property to apply the
	// property definition to
	PropertyDefinitions map[string]*PropertyDefinition `yaml:"property_definitions"`
}

// Validate returns an error if the definition is invalid, nil otherwise
func (def *ProviderDefinition) Validate() error {
	if def.Partition == "" {
		return fmt.Errorf("partition required")
	}
	for _, pd := range def.PropertyDefinitions {
		if err := pd.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type templateVars struct {
	RequiredFields  []string
	PropertySchemas []*propertySchemaWithKey
}

// JSONSchemaString returns a valid JSONSchema DRAFT-07 document describing the
// fields and properties that may be set for the providers described by the
// provider definition
func (def *ProviderDefinition) JSONSchemaString() string {
	vars := &templateVars{
		RequiredFields:  make([]string, 0),
		PropertySchemas: make([]*propertySchemaWithKey, 0),
	}
	for _, field := range baseProviderRequiredFields {
		vars.RequiredFields = append(vars.RequiredFields, field)
	}
	for key, prop := range def.PropertyDefinitions {
		if prop.Required {
			vars.RequiredFields = append(vars.RequiredFields, key)
		}
		ps := &propertySchemaWithKey{
			PropertySchema: prop.Schema,
			Key:            key,
		}
		vars.PropertySchemas = append(vars.PropertySchemas, ps)
	}
	var b bytes.Buffer
	if err := providerSchemaTemplate.Execute(&b, vars); err != nil {
		return fmt.Sprintf("TEMPLATE ERROR: %s", err)
	}
	return b.String()
}
