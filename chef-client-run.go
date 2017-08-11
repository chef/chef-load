package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-chef/chef"
	uuid "github.com/satori/go.uuid"
)

func chefClientRun(nodeClient chef.Client, nodeName string, firstRun bool, ohaiJSON map[string]interface{}, convergeJSON map[string]interface{}, complianceJSON map[string]interface{}) {
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
	reportingAvailable := true
	dataCollectorAvailable := true
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

	if config.RunChefClient {
		if firstRun {
			clientBody := map[string]interface{}{
				"name":  nodeName,
				"admin": false,
			}
			res, err := apiRequest(nodeClient, "POST", "clients", clientBody, nil, nil)
			if err != nil {
				if res != nil && res.StatusCode != 409 {
					printError(nodeName, err)
					printRunFailed(nodeName)
					return
				}
			}
		}

		res, err := apiRequest(nodeClient, "GET", "nodes/"+nodeName, nil, &node, nil)
		if err != nil {
			if res != nil && res.StatusCode != 404 {
				printError(nodeName, err)
				printRunFailed(nodeName)
				return
			}
		}
		if res != nil && res.StatusCode == 404 {
			node = chef.Node{Name: nodeName, Environment: chefEnvironment}
			_, err = apiRequest(nodeClient, "POST", "nodes", node, nil, nil)
			if err != nil {
				printError(nodeName, err)
				printRunFailed(nodeName)
				return
			}
			// mimic the real chef-client by setting the automatic attributes after creating the node object
			node.AutomaticAttributes = ohaiJSON
		}
		node.Environment = chefEnvironment
	} else {
		node = chef.Node{Name: nodeName, Environment: chefEnvironment, AutomaticAttributes: ohaiJSON}
	}

	if config.RunChefClient {
		// Expand run_list
		expandedRunList, err = runList.expand(&nodeClient, chefEnvironment)
		if err != nil {
			printError(nodeName, err)
			printRunFailed(nodeName)
			return
		}

		_, err := apiRequest(nodeClient, "GET", "environments/"+chefEnvironment, nil, nil, nil)
		if err != nil {
			printError(nodeName, err)
			printRunFailed(nodeName)
			return
		}

		// Notify Reporting of run start
		if config.EnableReporting {
			res, err := reportingRunStart(nodeClient, nodeName, runUUID, startTime)
			if err != nil {
				if res != nil && res.StatusCode != 404 {
					printError(nodeName, err)
					printRunFailed(nodeName)
					return
				}
			}
			if res != nil && res.StatusCode == 404 {
				reportingAvailable = false
			}
		}
	}

	// Notify Data Collector of run start
	runStartBody := dataCollectorRunStart(nodeName, orgName, runUUID, nodeUUID, startTime)
	if config.DataCollectorURL != "" {
		err := chefAutomateSendMessage(config.DataCollectorToken, config.DataCollectorURL, runStartBody)
		if err != nil {
			dataCollectorAvailable = false
			printError(nodeName, err)
		}
	} else {
		res, err := apiRequest(nodeClient, "POST", "data-collector", runStartBody, nil, nil)
		if err != nil {
			dataCollectorAvailable = false
			if res != nil && res.StatusCode != 404 {
				printError(nodeName, err)
			}
		}
	}

	if config.RunChefClient {
		// Calculate cookbook dependencies
		ckbks, err := solveRunListDependencies(&nodeClient, expandedRunList, chefEnvironment)
		if err != nil {
			printError(nodeName, err)
			printRunFailed(nodeName)
			return
		}

		// Download cookbooks
		if config.DownloadCookbooks == "always" || (config.DownloadCookbooks == "first" && firstRun) {
			err := ckbks.download(&nodeClient)
			if err != nil {
				printError(nodeName, err)
				printRunFailed(nodeName)
				return
			}
		}

		for _, apiGetRequest := range apiGetRequests {
			_, err := apiRequest(nodeClient, "GET", apiGetRequest, nil, nil, nil)
			if err != nil {
				printError(nodeName, err)
				printRunFailed(nodeName)
				return
			}
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
		_, err := apiRequest(nodeClient, "PUT", "nodes/"+nodeName, node, nil, nil)
		if err != nil {
			printError(nodeName, err)
			printRunFailed(nodeName)
			return
		}

		// Notify Reporting of run end
		if config.EnableReporting && reportingAvailable {
			_, err := reportingRunStop(nodeClient, nodeName, runUUID, startTime, endTime, runList)
			if err != nil {
				printError(nodeName, err)
				printRunFailed(nodeName)
				return
			}
		}
	}

	// Notify Data Collector of run end
	runStopBody := dataCollectorRunStop(node, nodeName, orgName, runList, parseRunList(expandedRunList), runUUID, nodeUUID, startTime, endTime, convergeJSON)
	if config.DataCollectorURL != "" {
		if dataCollectorAvailable || !config.RunChefClient {
			err := chefAutomateSendMessage(config.DataCollectorToken, config.DataCollectorURL, runStopBody)
			if err != nil {
				printError(nodeName, err)
				printRunFailed(nodeName)
				return
			}
		}
	} else if dataCollectorAvailable {
		_, err := apiRequest(nodeClient, "POST", "data-collector", runStopBody, nil, nil)
		if err != nil {
			printError(nodeName, err)
			printRunFailed(nodeName)
			return
		}
	}

	// Notify Data Collector of compliance report
	if len(complianceJSON) != 0 {
		complianceReportBody := dataCollectorComplianceReport(nodeName, chefEnvironment, reportUUID, nodeUUID, endTime, complianceJSON)
		if config.DataCollectorURL != "" {
			err := chefAutomateSendMessage(config.DataCollectorToken, config.DataCollectorURL, complianceReportBody)
			if err != nil {
				printError(nodeName, err)
				printRunFailed(nodeName)
				return
			}
		} else {
			_, err := apiRequest(nodeClient, "POST", "data-collector", complianceReportBody, nil, nil)
			if err != nil {
				printError(nodeName, err)
				printRunFailed(nodeName)
				return
			}
		}
	}

	fmt.Println(time.Now().UTC().Format(iso8601DateTime), nodeName, "run_finished")
}
