package osqueryreceiver

import (
	"errors"

	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	ExtensionsSocket                        string   `mapstructure:"extensions_socket"`
	Queries                                 []string `mapstructure:"queries"`
}

func (c Config) Validate() error {
	if len(c.Queries) == 0 {
		return errors.New("queries cannot be empty")
	}
	return nil
}
