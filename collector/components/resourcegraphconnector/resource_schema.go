package resourcegraphconnector

type ResourceSchema struct {
	APIVersion         string              `yaml:"apiVersion"`
	Name               string              `yaml:"name"`
	Description        string              `yaml:"description"`
	TelemetryResources []TelemetryResource `yaml:"resources"`
}

type TelemetryResource struct {
	Name                string   `yaml:"name"`
	CI                  string   `yaml:"ci"`
	Sources             []string `yaml:"sources"`
	IDAttributes        []string `yaml:"id_attributes"`
	Attributes          []string `yaml:"attributes"`
	Conditions          []string `yaml:"contitions"`
	MetricName          string   `yaml:"metric_name"`
	InstrumentationName string   `yaml:"instrumentation_name"`
}
