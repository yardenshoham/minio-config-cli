package validation

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schema.json
var schema []byte

// validator a compiled JSON schema validator.
var validator *jsonschema.Schema

func init() {
	compiler := jsonschema.NewCompiler()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewBuffer(schema))
	if err != nil {
		panic(err)
	}
	err = compiler.AddResource("schema.json", doc)
	if err != nil {
		panic(err)
	}
	validator = compiler.MustCompile("schema.json")
}

// ValidateConfig validates the config file against the JSON schema.
func ValidateConfig(config io.Reader) error {
	var dataToValidate any
	err := yaml.NewDecoder(config).Decode(&dataToValidate)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return validator.Validate(dataToValidate)
}
