package httpclient

import (
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/monorepo/go/kit/telemetry"
	"io"
	"net/http"
	"time"
)

type telemetryCli interface {
	Timing(name string, value time.Duration, tags map[string]interface{})
}

var (
	DefaultTimeOut = 3 * time.Second
	DefaultRetries = 3
)

type Config struct {
	TimeOut    time.Duration
	MaxRetries int
}

type Client struct {
	httpClient *http.Client
	telemetry  telemetryCli
}

func NewHttpClient(config Config) *Client {
	timeOut := DefaultTimeOut
	if config.TimeOut != 0 {
		timeOut = config.TimeOut
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = DefaultRetries
	if config.MaxRetries != 0 {
		retryClient.RetryMax = config.MaxRetries
	}

	httpClient := retryClient.StandardClient()

	httpClient.Timeout = timeOut

	return &Client{
		httpClient,
		telemetry.NewTelemetry(),
	}
}

func (c *Client) Post(url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := c.request(http.MethodPost, url, body, headers)
	if err != nil {
		return nil, err
	}

	return c.Do(req)

}

func (c *Client) Get(url string, headers map[string]string) (*http.Response, error) {
	req, err := c.request(http.MethodGet, url, nil, headers)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

func (c *Client) request(method string, url string, body io.Reader, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("crating request %w", err)
	}

	if headers != nil {
		for name, value := range headers {
			req.Header.Set(name, value)
		}
	}

	return req, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	start := time.Now()

	res, err := c.httpClient.Do(req)

	duration := time.Since(start)

	var status_code = 0
	if req.Response != nil {
		status_code = req.Response.StatusCode
	}

	c.telemetry.Timing("http", duration, map[string]interface{}{
		"url":         req.URL.String(),
		"method":      req.Method,
		"status_code": status_code,
	})

	return res, err
}
