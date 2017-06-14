package main

// Cheers! https://github.com/go-chef/chef/blob/master/http.go

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chef/chef"
	uuid "github.com/satori/go.uuid"
)

const iso8601DateTime = "2006-01-02T15:04:05Z"

const updatedResourcesJSON = `
  [
		{
			"type": "file",
			"name": "/tmp/test.txt",
			"id": "/tmp/test.txt",
			"after": {
				"owner": null,
				"group": null,
				"mode": null,
				"path": "/tmp/test.txt"
			},
			"before": {},
			"duration": "0",
			"delta": "",
			"result": "nothing",
			"status": "skipped",
			"cookbook_name": "insights-test",
			"cookbook_version": "0.1.1",
			"conditional": "not_if { action == :nothing }"
		},
		{
			"type": "execute",
			"name": "ls",
			"id": "ls",
			"after": {
				"command": "ls"
			},
			"before": {},
			"duration": "16",
			"delta": "",
			"result": "run",
			"status": "updated",
			"cookbook_name": "insights-test",
			"cookbook_version": "0.1.1"
		},
		{
			"type": "file",
			"name": "/tmp/test.txt",
			"id": "/tmp/test.txt",
			"after": {
				"owner": "aleff",
				"group": "wheel",
				"mode": "0644",
				"path": "/tmp/test.txt",
				"verifications": [],
				"checksum": "fa6a85e9eaf51901151bcc24f85e3b38b9693790877d47bbee3ebf82ee2d2336"
			},
			"before": {
				"checksum": "d152e4660ca0d02cd5145792c89eff662be03aad3e43b94a7fa4be75dbe80896",
				"owner": "aleff",
				"group": "wheel",
				"mode": "0644",
				"path": "/tmp/test.txt"
			},
			"duration": "9",
			"delta": "--- /tmp/test.txt\t2016-06-28 09:05:45.000000000 -0400\\n+++ /tmp/.chef-test.txt20160628-74223-lx2wnj\t2016-06-28 11:13:22.000000000 -0400\\n@@ -1,2 +1,2 @@\\n-5214b6f7-ee24-4202-9ff0-73fd4caca9dd\\n+5f8e07e8-b90f-4123-828a-5e9b2afbccfb",
			"result": "create",
			"status": "updated",
			"cookbook_name": "insights-test",
			"cookbook_version": "0.1.1"
		},
		{
			"type": "execute",
			"name": "ls -l",
			"id": "ls -l",
			"after": {
				"command": "ls -l"
			},
			"before": {},
			"duration": "11",
			"delta": "",
			"result": "run",
			"status": "updated",
			"cookbook_name": "insights-test",
			"cookbook_version": "0.1.1"
		},
		{
			"type": "file",
			"name": "/tmp/test.txt",
			"id": "/tmp/test.txt",
			"after": {
				"owner": "aleff",
				"group": "wheel",
				"mode": "0644",
				"path": "/tmp/test.txt",
				"verifications": [],
				"checksum": "fa6a85e9eaf51901151bcc24f85e3b38b9693790877d47bbee3ebf82ee2d2336"
			},
			"before": {
				"checksum": "fa6a85e9eaf51901151bcc24f85e3b38b9693790877d47bbee3ebf82ee2d2336",
				"owner": "aleff",
				"group": "wheel",
				"mode": "0644",
				"path": "/tmp/test.txt"
			},
			"duration": "10",
			"delta": "",
			"result": "create",
			"status": "up-to-date",
			"cookbook_name": "insights-test",
			"cookbook_version": "0.1.1"
		},
		{
			"type": "file",
			"name": "/tmp/always-updated.txt",
			"id": "/tmp/always-updated.txt",
			"after": {
				"owner": "aleff",
				"group": "wheel",
				"mode": "0644",
				"path": "/tmp/always-updated.txt",
				"verifications": [],
				"checksum": "f27d6dba15d380b69870a8dc7704f383bc37dd43728d2556b572bbc9e8137b97"
			},
			"before": {
				"checksum": "0f92fd4284f031a9b163bf5207cde0470e522b3d3b6292ced654905a3bdc8f57",
				"owner": "aleff",
				"group": "wheel",
				"mode": "0644",
				"path": "/tmp/always-updated.txt"
			},
			"duration": "8",
			"delta": "--- /tmp/always-updated.txt\t2016-06-28 09:05:45.000000000 -0400\\n+++ /tmp/.chef-always-updated.txt20160628-74223-6pcmxs\t2016-06-28 11:13:22.000000000 -0400\\n@@ -1,2 +1,2 @@\\n-369adb69-87f6-44a1-b4d5-7aa4fafe45e2\\n+aaa6b8a6-a300-4a7a-a285-f4746ac716fe",
			"result": "create",
			"status": "updated",
			"cookbook_name": "insights-test",
			"cookbook_version": "0.1.1"
		},
		{
			"type": "file",
			"name": "/failed/file/resource",
			"id": "/failed/file/resource",
			"after": {
				"owner": null,
				"group": null,
				"mode": null,
				"path": "/failed/file/resource"
			},
			"before": {},
			"duration": "0",
			"delta": "",
			"result": "create",
			"status": "skipped",
			"cookbook_name": "insights-test",
			"cookbook_version": "0.1.1",
			"conditional": "not_if { #code block }"
		},
		{
			"type": "file",
			"name": "/tmp/do-not-write.txt",
			"id": "/tmp/do-not-write.txt",
			"after": {
				"owner": null,
				"group": null,
				"mode": null,
				"path": "/tmp/do-not-write.txt"
			},
			"before": {},
			"duration": "0",
			"delta": "",
			"result": "create",
			"status": "skipped",
			"cookbook_name": "insights-test",
			"cookbook_version": "0.1.1",
			"conditional": "not_if { #code block }"
		},
		{
			"type": "file",
			"name": "/path/does/not/exist/but/we/will/ignore/this/failure.txt",
			"id": "/path/does/not/exist/but/we/will/ignore/this/failure.txt",
			"after": {
				"owner": null,
				"group": null,
				"mode": null,
				"path": "/path/does/not/exist/but/we/will/ignore/this/failure.txt"
			},
			"before": {},
			"duration": "0",
			"delta": "",
			"result": "create",
			"status": "skipped",
			"cookbook_name": "insights-test",
			"cookbook_version": "0.1.1",
			"conditional": "not_if { #code block }"
		},
		{
			"type": "file",
			"name": "/path/does/not/exist/so/this/should/fail.txt",
			"id": "/path/does/not/exist/so/this/should/fail.txt",
			"after": {
				"owner": null,
				"group": null,
				"mode": null,
				"path": "/path/does/not/exist/so/this/should/fail.txt"
			},
			"before": {},
			"duration": "0",
			"delta": "",
			"result": "create",
			"status": "skipped",
			"cookbook_name": "insights-test",
			"cookbook_version": "0.1.1",
			"conditional": "not_if { #code block }"
		}
	]
`

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

func dataCollectorRunStart(nodeName string, orgName string, runUUID uuid.UUID, nodeUUID uuid.UUID, startTime time.Time, config chefLoadConfig) error {
	msgBody := map[string]interface{}{
		"chef_server_fqdn": config.ChefServerURL,
		"entity_uuid":      nodeUUID.String(),
		"id":               runUUID.String(),
		"message_version":  "1.0.0",
		"message_type":     "run_start",
		"node_name":        nodeName,
		"organization":     orgName,
		"run_id":           runUUID.String(),
		"source":           "chef_client",
		"start_time":       startTime.Format(iso8601DateTime),
	}

	client, err := NewDataCollectorClient(&DataCollectorConfig{
		Token:   config.DataCollectorToken,
		URL:     config.DataCollectorURL,
		SkipSSL: true,
	})

	if err != nil {
		fmt.Printf("Error creating DataCollectorClient: %+v \n", err)
	}

	res := client.Update(msgBody)

	return res
}

func dataCollectorRunStop(node chef.Node, nodeName string, orgName string, runList runList, expandedRunList runList, runUUID uuid.UUID, nodeUUID uuid.UUID, startTime time.Time, endTime time.Time, config chefLoadConfig) error {
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

	expandedRunListMap := map[string]interface{}{
		"id":       config.ChefEnvironment,
		"run_list": expandedRunListItems,
	}

	updatedResources := []interface{}{}
	dec := json.NewDecoder(strings.NewReader(updatedResourcesJSON))
	err := dec.Decode(&updatedResources)
	if err != nil {
		fmt.Println("Couldn't decode updated resources JSON: ", err)
	}

	msgBody := map[string]interface{}{
		"chef_server_fqdn":       config.ChefServerURL,
		"entity_uuid":            nodeUUID.String(),
		"id":                     runUUID.String(),
		"message_version":        "1.0.0",
		"message_type":           "run_converge",
		"node_name":              nodeName,
		"organization":           orgName,
		"run_id":                 runUUID.String(),
		"source":                 "chef_client",
		"start_time":             startTime.Format(iso8601DateTime),
		"end_time":               endTime.Format(iso8601DateTime),
		"status":                 "success",
		"run_list":               runList.toStringSlice(),
		"expanded_run_list":      expandedRunListMap,
		"node":                   node,
		"resources":              updatedResources,
		"total_resource_count":   10,
		"updated_resource_count": 4,
	}

	client, err := NewDataCollectorClient(&DataCollectorConfig{
		Token:   config.DataCollectorToken,
		URL:     config.DataCollectorURL,
		SkipSSL: true,
	})

	if err != nil {
		fmt.Printf("Error creating DataCollectorClient: %+v \n", err)
	}

	res := client.Update(msgBody)

	return res
}
