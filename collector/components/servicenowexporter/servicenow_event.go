package servicenowexporter

type ServiceNowEventRequestBody struct {
	Records []ServiceNowEvent `json:"records"`
}

// https://docs.servicenow.com/bundle/vancouver-it-operations-management/page/product/event-management/task/send-events-via-web-service.html
type ServiceNowEvent struct {
	Resource       string `json:"resource"`
	Node           string `json:"node"`
	Severity       string `json:"severity"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	Timestamp      string `json:"time_of_event"`             // yyyy-MM-dd HH:mm:ss
	AdditionalInfo string `json:"additional_info,omitempty"` // actually a json string
	Source         string `json:"source"`
}
