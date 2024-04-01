package servicenowexporter

type ServiceNowEventRequestBody struct {
	Records []ServiceNowEvent `json:"records"`
}

// https://docs.servicenow.com/bundle/vancouver-it-operations-management/page/product/event-management/task/send-events-via-web-service.html
type ServiceNowEvent struct {
	// The resource on the node impacted
	Resource string `json:"resource"`
	// CI associated w/ event
	Node        string `json:"node"`
	Severity    string `json:"severity"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Timestamp   string `json:"time_of_event"` // yyyy-MM-dd HH:mm:ss
	// k8s.cluster.name:test-cluster,k8s.cluster.uid=12345
	AdditionalInfo string `json:"additional_info,omitempty"` // actually a json string
	Source         string `json:"source"`
}
