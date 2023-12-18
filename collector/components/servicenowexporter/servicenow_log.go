package servicenowexporter

// https://docs.servicenow.com/bundle/vancouver-it-operations-management/page/product/health-log-analytics-admin/task/hla-data-input-rest-api.html
type ServiceNowLog struct {
	ResourcePath string            `json:"resource_path"`
	Node         string            `json:"node"`
	Body         string            `json:"body"`
	CiSysId      string            `json:"ci,omitempty"`
	Timestamp    uint64            `json:"timestamp"`
	Severity     string            `json:"severity"`
	Ci2LogID     map[string]string `json:"ci2log_id,omitempty"`
	Source       string            `json:"source"`
}
