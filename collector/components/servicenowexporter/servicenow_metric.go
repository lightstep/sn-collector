package servicenowexporter

// https://docs.servicenow.com/bundle/vancouver-api-reference/page/integrate/inbound-rest/concept/push-metrics-MID-server.html
// https://support.servicenow.com/kb?id=kb_article_view&sysparm_article=KB0853084
type ServiceNowMetric struct {
	MetricType   string            `json:"metric_type"`
	ResourcePath string            `json:"resource_path"`
	Node         string            `json:"node"`
	CiSysId      string            `json:"ci,omitempty"`
	Value        float64           `json:"value"`
	Timestamp    uint64            `json:"timestamp"`
	Ci2MetricID  map[string]string `json:"ci2metric_id,omitempty"`
	Source       string            `json:"source"`
}
