package resourcegraphconnector

type ResourceSchema struct {
	APIVersion         string              `yaml:"apiVersion"`
	Name               string              `yaml:"name"`
	Description        string              `yaml:"description"`
	TelemetryResources []TelemetryResource `yaml:"resources"`
}

type TelemetryResource struct {
	Name                  string   `yaml:"name"`
	CI                    string   `yaml:"ci"`
	Sources               []string `yaml:"sources"`
	IDAttributes          []string `yaml:"id_attributes"`
	Attributes            []string `yaml:"attributes"`
	MetricSource          string   `yaml:"metric_source"`
	MetricInstrumentation string   `yaml:"metric_instrumentation"`
}
