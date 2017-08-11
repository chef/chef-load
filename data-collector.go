package main

// Cheers! https://github.com/go-chef/chef/blob/master/http.go

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chef/chef"
	uuid "github.com/satori/go.uuid"
)

const iso8601DateTime = "2006-01-02T15:04:05Z"

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

type expandedRunListItem struct {
	ItemType string  `json:"type"`
	Name     string  `json:"name"`
	Version  *string `json:"version"`
	Skipped  bool    `json:"skipped"`
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
func (dcc *DataCollectorClient) Update(body interface{}) (*http.Response, error) {
	var bodyJSON io.Reader = nil
	if body != nil {
		var err error
		bodyJSON, err = chef.JSONReader(body)
		if err != nil {
			return nil, err
		}
	}

	// Create an HTTP Request
	req, err := http.NewRequest("POST", dcc.URL.String(), bodyJSON)
	if err != nil {
		return nil, err
	}

	// Set our headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-data-collector-auth", "version=1.0")
	req.Header.Set("x-data-collector-token", dcc.Token)

	// Do request
	res, err := dcc.Client.Do(req)
	if err != nil {
		return res, err
	}
	if res != nil {
		defer res.Body.Close()
		if !(res.StatusCode >= 200 && res.StatusCode <= 299) {
			return res, errors.New(fmt.Sprintf("POST %s: %s", dcc.URL.String(), res.Status))
		}
	}
	ioutil.ReadAll(res.Body)
	return res, err
}

func chefAutomateSendMessage(dataCollectorToken string, dataCollectorURL string, body interface{}) error {
	client, err := NewDataCollectorClient(&DataCollectorConfig{
		Token:   dataCollectorToken,
		URL:     dataCollectorURL,
		SkipSSL: true,
	})

	if err != nil {
		return errors.New(fmt.Sprintf("Error creating DataCollectorClient: %+v \n", err))
	}

	_, err = client.Update(body)
	return err
}

func dataCollectorRunStart(nodeName string, orgName string, runUUID uuid.UUID, nodeUUID uuid.UUID, startTime time.Time) interface{} {
	chefServerURL, _ := url.Parse(config.ChefServerURL)
	chefServerFQDN := chefServerURL.Host

	body := map[string]interface{}{
		"chef_server_fqdn":  chefServerFQDN,
		"entity_uuid":       nodeUUID.String(),
		"id":                runUUID.String(),
		"message_version":   "1.1.0",
		"message_type":      "run_start",
		"node_name":         nodeName,
		"organization_name": orgName,
		"run_id":            runUUID.String(),
		"source":            "chef_client",
		"start_time":        startTime.Format(iso8601DateTime),
	}
	return body
}

func dataCollectorRunStop(node chef.Node, nodeName string, orgName string, runList runList, expandedRunList runList, runUUID uuid.UUID, nodeUUID uuid.UUID, startTime time.Time, endTime time.Time, convergeJSON map[string]interface{}) interface{} {
	chefServerURL, _ := url.Parse(config.ChefServerURL)
	chefServerFQDN := chefServerURL.Host

	convergedRunList := []interface{}{}
	convergedExpandedRunListMap := map[string]interface{}{}
	if convergeJSON["run_list"] != nil && convergeJSON["expanded_run_list"] != nil {
		convergedRunList = convergeJSON["run_list"].([]interface{})
		convergedExpandedRunListMap = convergeJSON["expanded_run_list"].(map[string]interface{})
	} else {
		for _, v := range runList.toStringSlice() {
			convergedRunList = append(convergedRunList, v)
		}

		var expandedRunListItems []expandedRunListItem
		for _, runListItem := range expandedRunList {
			erli := expandedRunListItem{
				Name:     runListItem.name,
				ItemType: runListItem.itemType,
				Skipped:  false,
			}
			if runListItem.version != "" {
				version := runListItem.version
				erli.Version = &version
			}
			expandedRunListItems = append(expandedRunListItems, erli)
		}

		expandedRunListItemsInterface := []interface{}{}
		for _, v := range expandedRunListItems {
			expandedRunListItemsInterface = append(expandedRunListItemsInterface, v)
		}

		convergedExpandedRunListMap = map[string]interface{}{
			"id":       config.ChefEnvironment,
			"run_list": expandedRunListItemsInterface,
		}
	}

	resourcesJSON := []interface{}{}
	if convergeJSON["resources"] != nil {
		resourcesJSON = convergeJSON["resources"].([]interface{})
	}

	body := map[string]interface{}{
		"chef_server_fqdn":       chefServerFQDN,
		"entity_uuid":            nodeUUID.String(),
		"id":                     runUUID.String(),
		"message_version":        "1.1.0",
		"message_type":           "run_converge",
		"node_name":              nodeName,
		"organization_name":      orgName,
		"run_id":                 runUUID.String(),
		"source":                 "chef_client",
		"start_time":             startTime.Format(iso8601DateTime),
		"end_time":               endTime.Format(iso8601DateTime),
		"status":                 "success",
		"run_list":               convergedRunList,
		"expanded_run_list":      convergedExpandedRunListMap,
		"node":                   node,
		"resources":              resourcesJSON,
		"total_resource_count":   0,
		"updated_resource_count": 0,
	}
	return body
}

func dataCollectorComplianceReport(nodeName string, chefEnvironment string, reportUUID uuid.UUID, nodeUUID uuid.UUID, endTime time.Time, complianceJSON map[string]interface{}) interface{} {
	body := complianceJSON
	body["type"] = "inspec_report"
	body["node_name"] = nodeName
	body["environment"] = chefEnvironment
	body["report_uuid"] = reportUUID
	body["node_uuid"] = nodeUUID
	body["end_time"] = endTime.Format(iso8601DateTime)

	if body["controls"] != nil {
		delete(body, "controls")
	}
	return body
}
