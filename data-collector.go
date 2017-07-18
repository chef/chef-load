package main

// Cheers! https://github.com/go-chef/chef/blob/master/http.go

import (
	"crypto/tls"
	"fmt"
	"io"
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
func (dcc *DataCollectorClient) Update(msgJSON io.Reader) error {
	// Create an HTTP Request
	req, err := http.NewRequest("POST", dcc.URL.String(), msgJSON)
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

func chefAutomateSendMessage(dataCollectorToken string, dataCollectorURL string, msgJSON io.Reader) error {
	client, err := NewDataCollectorClient(&DataCollectorConfig{
		Token:   dataCollectorToken,
		URL:     dataCollectorURL,
		SkipSSL: true,
	})

	if err != nil {
		fmt.Printf("Error creating DataCollectorClient: %+v \n", err)
	}

	res := client.Update(msgJSON)

	return res
}

func dataCollectorRunStart(nodeName string, orgName string, runUUID uuid.UUID, nodeUUID uuid.UUID, startTime time.Time, config chefLoadConfig) error {
	chefServerURL, _ := url.Parse(config.ChefServerURL)
	chefServerFQDN := chefServerURL.Host

	msgBody := map[string]interface{}{
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

	msgJSON, err := chef.JSONReader(msgBody)
	if err != nil {
		fmt.Println(err)
	}

	res := chefAutomateSendMessage(config.DataCollectorToken, config.DataCollectorURL, msgJSON)
	return res
}

func dataCollectorRunStop(node chef.Node, nodeName string, orgName string, runList runList, expandedRunList runList, runUUID uuid.UUID, nodeUUID uuid.UUID, startTime time.Time, endTime time.Time, convergeJSON map[string]interface{}, config chefLoadConfig) error {
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

	msgBody := map[string]interface{}{
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

	msgJSON, err := chef.JSONReader(msgBody)
	if err != nil {
		fmt.Println(err)
	}

	res := chefAutomateSendMessage(config.DataCollectorToken, config.DataCollectorURL, msgJSON)
	return res
}

func dataCollectorComplianceReport(nodeName string, chefEnvironment string, reportUUID uuid.UUID, nodeUUID uuid.UUID, endTime time.Time, complianceJSON map[string]interface{}, config chefLoadConfig) error {
	msgBody := complianceJSON
	msgBody["type"] = "inspec_report"
	msgBody["node_name"] = nodeName
	msgBody["environment"] = chefEnvironment
	msgBody["report_uuid"] = reportUUID
	msgBody["node_uuid"] = nodeUUID
	msgBody["end_time"] = endTime.Format(iso8601DateTime)

	if msgBody["controls"] != nil {
		delete(msgBody, "controls")
	}

	msgJSON, err := chef.JSONReader(msgBody)
	if err != nil {
		fmt.Println(err)
	}

	res := chefAutomateSendMessage(config.DataCollectorToken, config.DataCollectorURL, msgJSON)
	return res
}
