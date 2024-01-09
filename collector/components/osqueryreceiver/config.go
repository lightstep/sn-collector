package osqueryreceiver

type Config struct {
	ExtensionsSocket string   `mapstructure:"extensions_socket"`
	Queries          []string `mapstructure:"queries"`
}
