package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-chef/chef"
	uuid "github.com/satori/go.uuid"
)

func chefClientRun(nodeClient chef.Client, nodeName string, firstRun bool, ohaiJSON map[string]interface{}, convergeJSON map[string]interface{}, complianceJSON map[string]interface{}, config chefLoadConfig) {
	fmt.Println(time.Now().UTC().Format(iso8601DateTime), nodeName, "run_started")

	chefEnvironment := config.ChefEnvironment
	runList := parseRunList(config.RunList)
	apiGetRequests := config.APIGetRequests
	sleepDuration := config.SleepDuration
	runUUID := uuid.NewV4()
	reportUUID := uuid.NewV4()
	nodeUUID := uuid.NewV3(uuid.NamespaceDNS, nodeName)
	startTime := time.Now().UTC()
	chefServerURL := config.ChefServerURL
	url, _ := url.ParseRequestURI(chefServerURL)
	orgName := strings.Split(url.Path, "/")[2]
	var reportsStatusCode int
	var expandedRunList []string
	var node chef.Node
	var err error

	ohaiJSON["fqdn"] = nodeName

	if ohaiJSON["platform"] == nil {
		ohaiJSON["platform"] = "rhel"
	}

	if ohaiJSON["ipaddress"] == nil {
		ohaiJSON["ipaddress"] = "169.254.169.254"
	}

	if config.RunChefClient && config.CreateClients {
		_, err = nodeClient.Clients.Get(nodeName)
		if err != nil {
			statusCode := getStatusCode(err)
			if statusCode == 404 {
				_, err = nodeClient.Clients.Create(nodeName, false)
				if err != nil {
					fmt.Println("Couldn't create client", err)
				}
			}
		}
	}

	if config.RunChefClient {
		if firstRun {
			_, err = nodeClient.Clients.Create(nodeName, false)
			if err != nil && getStatusCode(err) != 409 {
				fmt.Println("Couldn't create client", err)
			}
		}

		node, err = nodeClient.Nodes.Get(nodeName)
		if err != nil {
			statusCode := getStatusCode(err)
			if statusCode == 404 {
				node = chef.Node{Name: nodeName, Environment: chefEnvironment}
				_, err = nodeClient.Nodes.Post(node)
				if err != nil {
					fmt.Println("Couldn't create node. ", err)
				}
				// mimic the real chef-client by setting the automatic attributes after creating the node object
				node.AutomaticAttributes = ohaiJSON
			} else {
				fmt.Println("Couldn't get node: ", err)
			}
		}
	} else {
		node = chef.Node{Name: nodeName, Environment: chefEnvironment, AutomaticAttributes: ohaiJSON}
	}

	if config.RunChefClient {
		nodeClient.Environments.Get(chefEnvironment)

		// Notify Reporting of run start
		if config.EnableReporting {
			reportsStatusCode = reportingRunStart(nodeClient, nodeName, runUUID, startTime)
		}
	}

	// Notify Data Collector of run start
	if config.DataCollectorURL != "" {
		dataCollectorRunStart(nodeName, orgName, runUUID, nodeUUID, startTime, config)
	}

	if config.RunChefClient {
		// Expand run_list
		expandedRunList = runList.expand(&nodeClient, chefEnvironment)

		// Calculate cookbook dependencies
		ckbks := solveRunListDependencies(&nodeClient, expandedRunList, chefEnvironment)

		// Download cookbooks
		if config.DownloadCookbooks == "always" || (config.DownloadCookbooks == "first" && firstRun) {
			ckbks.download(&nodeClient)
		}

		for _, apiGetRequest := range apiGetRequests {
			apiRequest(nodeClient, "GET", apiGetRequest, nil)
		}
	} else {
		expandedRunList = runList.toStringSlice()
	}

	time.Sleep(time.Duration(sleepDuration) * time.Second)

	node.RunList = runList.toStringSlice()

	// Ensure that at least an empty set of tags is set for the node's normal attributes
	node.NormalAttributes = map[string]interface{}{"tags": []interface{}{}}

	// Ensure that what we post at the end of the run is different from previous runs
	endTime := time.Now().UTC()
	node.AutomaticAttributes["ohai_time"] = endTime.Unix()

	if config.RunChefClient {
		_, err = nodeClient.Nodes.Put(node)
		if err != nil {
			fmt.Println("Couldn't update node: ", err)
		}

		// Notify Reporting of run end
		if config.EnableReporting && reportsStatusCode == 201 {
			reportingRunStop(nodeClient, nodeName, runUUID, startTime, endTime, runList)
		}
	}

	// Notify Data Collector of run end
	if config.DataCollectorURL != "" {
		dataCollectorRunStop(node, nodeName, orgName, runList, parseRunList(expandedRunList), runUUID, nodeUUID, startTime, endTime, convergeJSON, config)
		if len(complianceJSON) != 0 {
			dataCollectorComplianceReport(nodeName, chefEnvironment, reportUUID, nodeUUID, endTime, complianceJSON, config)
		}
	}

	fmt.Println(time.Now().UTC().Format(iso8601DateTime), nodeName, "run_finished")
}
