package validation

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
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

func ValidateConfig(config io.Reader) error {
	var dataToValidate any
	err := yaml.NewDecoder(config).Decode(&dataToValidate)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return validator.Validate(dataToValidate)
}
