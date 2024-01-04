package resourcegraphconnector

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SchemaPath string `mapstructure:"schema_path"`
}

func (c Config) loadResourceSchema() (*ResourceSchema, error) {
	schemaFile, err := os.ReadFile(c.SchemaPath)
	if err != nil {
		return nil, err
	}

	var resourceSchema ResourceSchema
	err = yaml.Unmarshal(schemaFile, &resourceSchema)
	if err != nil {
		return nil, err
	}

	return &resourceSchema, nil
}

func (c Config) Validate() error {
	if c.SchemaPath == "" {
		return errors.New("schema_path cannot be empty")
	}

	return nil
}
