package servicenowexporter

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type midClient struct {
	config     *Config
	httpClient *http.Client
	logger     *zap.Logger
}

func newMidClient(config *Config, l *zap.Logger) *midClient {
	return &midClient{
		config: config,
		logger: l,
		httpClient: &http.Client{
			Timeout: config.TimeoutSettings.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: config.InsecureSkipVerify,
				},
			},
		},
	}
}

func handleNon200Response(res *http.Response) error {
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return fmt.Errorf("ServiceNow API returned non-200 status code: %d (%s)", res.StatusCode, string(bodyBytes))
}

func (c *midClient) Close() {
	c.httpClient.CloseIdleConnections()
}

func (c *midClient) sendEvents(payload []ServiceNowEvent) error {
	url := c.config.PushEventsURL
	request := ServiceNowEventRequestBody{Records: payload}
	c.logger.Info("Sending events to ServiceNow", zap.String("url", url), zap.Any("request", request))
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	r, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	r.SetBasicAuth(c.config.Username, string(c.config.Password))

	res, err := c.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return handleNon200Response(res)
	}

	return nil
}

func (c *midClient) sendLogs(payload []ServiceNowLog) error {
	url := c.config.PushLogsURL
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	r, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")

	if len(c.config.Username) > 0 {
		r.SetBasicAuth(c.config.Username, string(c.config.Password))
	} else if len(c.config.ApiKey) > 0 {
		r.Header.Set("Authorization", "key "+string(c.config.ApiKey))
	}

	if err != nil {
		return err
	}

	res, err := c.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return handleNon200Response(res)
	}

	return nil
}

func (c *midClient) sendMetrics(payload []ServiceNowMetric) error {
	url := c.config.PushMetricsURL
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	r, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	r.SetBasicAuth(c.config.Username, string(c.config.Password))
	if err != nil {
		return err
	}

	res, err := c.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return handleNon200Response(res)
	}

	return nil
}
