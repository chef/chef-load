//
// Copyright:: Copyright 2017-2018 Chef Software, Inc.
// License:: Apache License, Version 2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package chef_load

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
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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
	Client   *http.Client
	Token    string
	URL      *url.URL
	Requests chan *request
}

type expandedRunListItem struct {
	ItemType string  `json:"type"`
	Name     string  `json:"name"`
	Version  *string `json:"version"`
	Skipped  bool    `json:"skipped"`
}

// NewDataCollectorClient builds our Client struct with our Config
func NewDataCollectorClient(cfg *DataCollectorConfig, reqChan chan *request) (*DataCollectorClient, error) {
	URL, _ := url.Parse(cfg.URL)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.SkipSSL},
	}

	c := &DataCollectorClient{
		Client: &http.Client{
			Transport: tr,
			Timeout:   cfg.Timeout * time.Second,
		},
		URL:      URL,
		Token:    cfg.Token,
		Requests: reqChan,
	}
	return c, nil
}

// Update the data collector endpoint with our map
func (dcc *DataCollectorClient) Update(nodeName string, body interface{}) (*http.Response, error) {
	var bodyJSON io.Reader = nil
	if body != nil {
		var err error
		// TODO: @afiune check panic!?
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
	if dcc.Token != "dev" {
		req.Header.Set("x-data-collector-auth", "version=1.0")
		req.Header.Set("x-data-collector-token", dcc.Token)
	} else {
		req.Header.Set("Authorization", "Bearer dev")
	}

	// Do request
	t0 := time.Now()
	res, err := dcc.Client.Do(req)
	request_time := time.Now().Sub(t0)
	statusCode := 999
	if res != nil {
		defer res.Body.Close()
		statusCode = res.StatusCode
	}
	dcc.Requests <- &request{Method: req.Method, Url: req.URL.String(), StatusCode: statusCode}
	logger.WithFields(log.Fields{
		"name":                 nodeName,
		"method":               req.Method,
		"url":                  req.URL.String(),
		"status_code":          statusCode,
		"headers":              req.Header,
		"request_time_seconds": float64(request_time.Nanoseconds()/1e6) / 1000,
	}).Info("API Request")

	//logger.Infof(req.)

	if res != nil {
		if !(res.StatusCode >= 200 && res.StatusCode <= 299) {
			return res, errors.New(fmt.Sprintf("POST %s: %s", dcc.URL.String(), res.Status))
		}
	}

	if err != nil {
		return res, err
	}

	ioutil.ReadAll(res.Body)
	return res, err
}

func chefAutomateSendMessage(client *DataCollectorClient, nodeName string, body interface{}) (int, error) {
	code := 999
	res, err := client.Update(nodeName, body)
	if res != nil {
		code = res.StatusCode
	}
	return code, err
}

func dataCollectorRunStart(config *Config, nodeName, chefServerFQDN, orgName string,
	runUUID, nodeUUID uuid.UUID, startTime time.Time) interface{} {

	if chefServerFQDN == "" {
		chefServerURL, _ := url.Parse(config.ChefServerURL)
		chefServerFQDN = chefServerURL.Host
	}

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
		"start_time":        startTime.Format(DateTimeFormat),
	}
	return body
}

// TODO: (@afiune) Refactor this so we dont pass so many arguments
func dataCollectorRunStop(config *Config, node chef.Node, nodeName, chefServerFQDN, orgName, status string,
	runList, expandedRunList runList, runUUID, nodeUUID uuid.UUID,
	startTime, endTime time.Time, convergeJSON map[string]interface{}) interface{} {

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
		"start_time":             startTime.Format(DateTimeFormat),
		"end_time":               endTime.Format(DateTimeFormat),
		"status":                 status,
		"run_list":               convergedRunList,
		"expanded_run_list":      convergedExpandedRunListMap,
		"node":                   node,
		"resources":              resourcesJSON,
		"total_resource_count":   0,
		"updated_resource_count": 0,
	}
	return body
}

func dataCollectorComplianceReport(node NodeDetails, reportUUID uuid.UUID, endTime time.Time, complianceJSON map[string]interface{}) interface{} {
	body := complianceJSON
	body["type"] = "inspec_report"
	body["node_name"] = node.name
	body["environment"] = node.environment
	body["report_uuid"] = reportUUID
	body["node_uuid"] = node.nodeUUID
	body["roles"] = node.roles
	body["recipes"] = node.recipes
	body["end_time"] = endTime.Format(DateTimeFormat)
	body["source_fqdn"] = node.sourceFqdn
	body["fqdn"] = node.fqdn
	body["organization_name"] = node.orgName
	body["policy_group"] = node.policyGroup
	body["policy_name"] = node.policyName
	body["chef_tags"] = node.chefTags
	body["ipaddress"] = node.ipAddr

	if body["controls"] != nil {
		delete(body, "controls")
	}
	return body
}
