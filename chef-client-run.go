package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-chef/chef"
	"github.com/satori/go.uuid"
)

const iso8601DateTime = "2006-01-02T15:04:05Z"
const rubyDateTime = "2006-01-02 15:04:05 -0700"

func newChefNode(nodeName, ohaiJSONFile string) (node chef.Node) {
	node = chef.Node{
		Name:                nodeName,
		Environment:         "_default",
		ChefType:            "node",
		JsonClass:           "Chef::Node",
		RunList:             []string{},
		AutomaticAttributes: map[string]interface{}{},
		NormalAttributes:    map[string]interface{}{},
		DefaultAttributes:   map[string]interface{}{},
		OverrideAttributes:  map[string]interface{}{},
	}

	if ohaiJSONFile != "" {
		file, err := os.Open(ohaiJSONFile)
		if err != nil {
			fmt.Println("Couldn't open ohai JSON file ", ohaiJSONFile, ": ", err)
			return
		}
		defer file.Close()

		ohaiJSON := map[string]interface{}{}

		err = json.NewDecoder(file).Decode(&ohaiJSON)
		if err != nil {
			fmt.Println("Couldn't decode ohai JSON file ", ohaiJSONFile, ": ", err)
			return
		}
		node.AutomaticAttributes = ohaiJSON
	}

	return
}

func reportingRunStart(nodeClient chef.Client, nodeName string, runUUID uuid.UUID, startTime time.Time) int {
	startRunBody := map[string]string{
		"action":     "start",
		"run_id":     runUUID.String(),
		"start_time": startTime.Format(rubyDateTime),
	}
	data, err := chef.JSONReader(startRunBody)
	if err != nil {
		fmt.Println(err)
	}

	req, err := nodeClient.NewRequest("POST", "reports/nodes/"+nodeName+"/runs", data)
	req.Header.Set("X-Ops-Reporting-Protocol-Version", "0.1.0")
	res, err := nodeClient.Do(req, nil)
	if err != nil && res.StatusCode != 404 {
		// can't print res here if it is nil
		// fmt.Println(res.StatusCode)
		fmt.Println(err)
		return res.StatusCode
	}
	defer res.Body.Close()
	return res.StatusCode
}

func reportingRunStop(nodeClient chef.Client, nodeName string, runUUID uuid.UUID, startTime time.Time, endTime time.Time) int {
	endRunBody := map[string]interface{}{
		"action":          "end",
		"data":            map[string]interface{}{},
		"end_time":        endTime.Format(rubyDateTime),
		"resources":       []interface{}{},
		"run_list":        "[]",
		"start_time":      startTime.Format(rubyDateTime),
		"status":          "success",
		"total_res_count": "0",
	}
	data, err := chef.JSONReader(endRunBody)
	if err != nil {
		fmt.Println(err)
	}

	req, err := nodeClient.NewRequest("POST", "reports/nodes/"+nodeName+"/runs/"+runUUID.String(), data)
	req.Header.Set("X-Ops-Reporting-Protocol-Version", "0.1.0")
	res, err := nodeClient.Do(req, nil)
	if err != nil && res.StatusCode != 404 {
		// can't print res here if it is nil
		// fmt.Println(res.StatusCode)
		fmt.Println(err)
		return res.StatusCode
	}
	defer res.Body.Close()
	return res.StatusCode
}

func dataCollectorRunStart(nodeName string, orgName string, runUUID uuid.UUID, nodeUUID uuid.UUID, startTime time.Time, config chefLoadConfig) error {
	msgBody := map[string]interface{}{
		"chef_server_fqdn": config.ChefServerUrl,
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
		URL:     config.DataCollectorUrl,
		SkipSSL: true,
	})

	if err != nil {
		fmt.Printf("Error creating DataCollectorClient: %+v \n", err)
	}

	res := client.Update(msgBody)

	return res
}

func dataCollectorRunStop(node chef.Node, nodeName string, orgName string, runUUID uuid.UUID, nodeUUID uuid.UUID, startTime time.Time, endTime time.Time, config chefLoadConfig) error {

	// Expand our run_list
	expandedRunList := make([]map[string]interface{}, 0, len(config.RunList))
	runListRegExp, _ := regexp.Compile("(\\w+)\\[(.+)\\]")
	var nullString *string

	for idx, runListItem := range config.RunList {
		matches := runListRegExp.FindStringSubmatch(runListItem)
		expandedRunListItem := map[string]interface{}{
			"type":    matches[1],
			"name":    matches[2],
			"version": nullString,
			"skipped": false,
		}
		expandedRunList[idx] = expandedRunListItem
	}

	expandedRunListMap := map[string]interface{}{
		"id":       "_default",
		"run_list": expandedRunList,
	}

	msgBody := map[string]interface{}{
		"chef_server_fqdn":       config.ChefServerUrl,
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
		"run_list":               []interface{}{strings.Join(config.RunList, "")},
		"expanded_run_list":      expandedRunListMap,
		"node":                   node,
		"resources":              []interface{}{},
		"total_resource_count":   0,
		"updated_resource_count": 0,
	}

	client, err := NewDataCollectorClient(&DataCollectorConfig{
		Token:   config.DataCollectorToken,
		URL:     config.DataCollectorUrl,
		SkipSSL: true,
	})

	if err != nil {
		fmt.Printf("Error creating DataCollectorClient: %+v \n", err)
	}

	res := client.Update(msgBody)

	return res
}

func chefClientRun(nodeClient chef.Client, nodeName string, getCookbooks bool, config chefLoadConfig) {
	ohaiJSONFile := config.OhaiJsonFile
	runList := config.RunList
	apiGetRequests := config.ApiGetRequests
	sleepDuration := config.SleepDuration
	runUUID := uuid.NewV4()
	nodeUUID := uuid.NewV3(uuid.NamespaceDNS, nodeName)
	startTime := time.Now().UTC()
	chefServerURL := config.ChefServerUrl
	url, _ := url.ParseRequestURI(chefServerURL)
	orgName := strings.Split(url.Path, "/")[2]

	// Create the Node
	node, err := nodeClient.Nodes.Get(nodeName)
	if err != nil {
		statusCode := getStatusCode(err)
		if statusCode == 404 {
			// Create a Node object
			// TODO: should have a constructor for this
			node = newChefNode(nodeName, ohaiJSONFile)

			_, err = nodeClient.Nodes.Post(node)
			if err != nil {
				fmt.Println("Couldn't create node. ", err)
			}
		} else {
			fmt.Println("Couldn't get node: ", err)
		}
	}

	// Notify Reporting of run start
	reportsStatusCode := reportingRunStart(nodeClient, nodeName, runUUID, startTime)

	// Notify Data Collector of run start
	if config.DataCollectorUrl != "" {
		dataCollectorRunStart(nodeName, orgName, runUUID, nodeUUID, startTime, config)
	}

	// Download Cookbooks
	rl := map[string][]string{"run_list": runList}
	data, err := chef.JSONReader(rl)
	if err != nil {
		fmt.Println(err)
	}

	var cookbooks map[string]json.RawMessage

	err = func(cookbooks *map[string]json.RawMessage) error {
		req, err := nodeClient.NewRequest("POST", "environments/_default/cookbook_versions", data)
		res, err := nodeClient.Do(req, nil)
		if err != nil {
			// can't print res here if it is nil
			// fmt.Println(res.StatusCode)
			fmt.Println(err)
			return err
		}
		defer res.Body.Close()

		return json.NewDecoder(res.Body).Decode(cookbooks)
	}(&cookbooks)

	if getCookbooks {
		downloadCookbooks(&nodeClient, cookbooks)
	}

	for _, apiGetRequest := range apiGetRequests {
		req, _ := nodeClient.NewRequest("GET", apiGetRequest, nil) //, data)
		res, err := nodeClient.Do(req, nil)
		if err != nil {
			// can't print res here if it is nil
			// fmt.Println(res.StatusCode)
			// TODO: should this be handled better than just skipping over it?
			fmt.Println(err)
			continue
		}
		defer res.Body.Close()
		// res.Body.Close()
	}

	time.Sleep(time.Duration(sleepDuration) * time.Second)

	// Ensure that what we post at the end of the run is different from previous runs
	endTime := time.Now().UTC()
	node.AutomaticAttributes["ohai_time"] = endTime.Unix()

	// Update the node
	_, err = nodeClient.Nodes.Put(node)
	if err != nil {
		fmt.Println("Couldn't update node: ", err)
	}

	// Notify Reporting of run end
	if reportsStatusCode == 201 {
		reportingRunStop(nodeClient, nodeName, runUUID, startTime, endTime)
	}

	// Notify Data Collector of run end
	if config.DataCollectorUrl != "" {
		dataCollectorRunStop(node, nodeName, orgName, runUUID, nodeUUID, startTime, endTime, config)
	}
}
