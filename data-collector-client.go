package main

// Cheers! https://github.com/go-chef/chef/blob/master/http.go

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// DataCollectorConfig holds our configuration for the Data Collector
type DataCollectorConfig struct {
	Token   string
	URL     string
	SkipSSL bool
	Timeout time.Duration
}

// DataCollectorClient has our configured HTTP client, our Token and the URL
type DataCollectorClient struct {
	Client *http.Client
	Token  string
	URL    *url.URL
}

// NewDataCollectorClient builds our Client struct with our Config
func NewDataCollectorClient(cfg *DataCollectorConfig) (*DataCollectorClient, error) {
	URL, _ := url.Parse(cfg.URL)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.SkipSSL},
	}

	c := &DataCollectorClient{
		Client: &http.Client{
			Transport: tr,
			Timeout:   cfg.Timeout * time.Second,
		},
		URL:   URL,
		Token: cfg.Token,
	}
	return c, nil
}

// Update the data collector endpoint with our map
func (dcc *DataCollectorClient) Update(body map[string]interface{}) error {
	// Convert our body to encoded JSON
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(body)
	encodedBody := bytes.NewReader(buf.Bytes())

	// Create an HTTP Request
	req, err := http.NewRequest("POST", dcc.URL.String(), encodedBody)
	if err != nil {
		return err
	}

	// Set our headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-data-collector-auth", "version=1.0")
	req.Header.Set("x-data-collector-token", dcc.Token)

	// Do request
	res, err := dcc.Client.Do(req)

	// Handle response
	if res != nil {
		defer res.Body.Close()
	}

	return err
}
