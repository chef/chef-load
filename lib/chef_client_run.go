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

import (
	"net/url"
	"strings"
	"time"

	"github.com/go-chef/chef"
	uuid "github.com/satori/go.uuid"
)

func ChefClientRun(config *Config, nodeClient chef.Client, nodeName string, firstRun bool, ohaiJSON map[string]interface{}, convergeJSON map[string]interface{}, complianceJSON map[string]interface{}) {
	var (
		chefEnvironment        = config.ChefEnvironment
		runList                = parseRunList(config.RunList)
		apiGetRequests         = config.APIGetRequests
		sleepDuration          = config.SleepDuration
		runUUID, _             = uuid.NewV4()
		reportUUID, _          = uuid.NewV4()
		nodeUUID               = uuid.NewV3(uuid.NamespaceDNS, nodeName)
		startTime              = time.Now().UTC()
		url, _                 = url.ParseRequestURI(config.ChefServerURL)
		chefServerURL, _       = url.Parse(config.ChefServerURL)
		chefServerFQDN         = chefServerURL.Host
		status                 = "success"
		orgName                = strings.Split(url.Path, "/")[2]
		reportingAvailable     = true
		dataCollectorAvailable = true
		expandedRunList        []string
		node                   chef.Node
	)

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
				"admin":     false,
				"name":      nodeName,
				"validator": false,
			}
			if config.ChefServerCreatesClientKey {
				clientBody["create_key"] = config.ChefServerCreatesClientKey
			}
			apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "clients", clientBody, nil, nil)
		}

		res, err := apiRequest(nodeClient, nodeName, config.ChefVersion, "GET", "nodes/"+nodeName, nil, &node, nil)
		if err != nil {
			if res != nil && res.StatusCode != 404 {
				node = chef.Node{Name: nodeName}
			}
		}
		if res != nil && res.StatusCode == 404 {
			node = chef.Node{Name: nodeName, Environment: chefEnvironment}
			_, err = apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "nodes", node, nil, nil)
			if err != nil {
				node = chef.Node{Name: nodeName}
			}
		}
	} else {
		node = chef.Node{Name: nodeName}
	}
	node.Environment = chefEnvironment
	node.AutomaticAttributes = ohaiJSON

	if config.RunChefClient {
		// Expand run_list
		expandedRunList = runList.expand(&nodeClient, nodeName, config.ChefVersion, chefEnvironment)

		apiRequest(nodeClient, nodeName, config.ChefVersion, "GET", "environments/"+chefEnvironment, nil, nil, nil)

		// Notify Reporting of run start
		if config.EnableReporting {
			res, _ := reportingRunStart(nodeClient, nodeName, config.ChefVersion, runUUID, startTime)
			if res != nil && res.StatusCode == 404 {
				reportingAvailable = false
			}
		}
	}

	// Notify Data Collector of run start
	runStartBody := dataCollectorRunStart(config, nodeName, "", orgName, runUUID, nodeUUID, startTime)
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(nodeName, config.DataCollectorToken, config.DataCollectorURL, runStartBody)
	} else {
		res, err := apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "data-collector", runStartBody, nil, nil)
		if err != nil {
			if res != nil {
				if res.StatusCode == 404 {
					dataCollectorAvailable = false
				}
			}
		}
	}

	if config.RunChefClient {
		// Calculate cookbook dependencies
		ckbks := solveRunListDependencies(&nodeClient, nodeName, config.ChefVersion, chefEnvironment, expandedRunList)

		// Download cookbooks
		if config.DownloadCookbooks == "always" || (config.DownloadCookbooks == "first" && firstRun) {
			ckbks.download(&nodeClient, nodeName, config.ChefVersion)
		}

		for _, apiGetRequest := range apiGetRequests {
			apiRequest(nodeClient, nodeName, config.ChefVersion, "GET", apiGetRequest, nil, nil, nil)
		}
	} else {
		expandedRunList = runList.toStringSlice()
	}

	time.Sleep(time.Duration(sleepDuration) * time.Second)

	node.RunList = runList.toStringSlice()

	// Ensure that at least an empty set of tags is set for the node's normal attributes
	if node.NormalAttributes == nil {
		node.NormalAttributes = map[string]interface{}{"tags": []interface{}{}}
	} else {
		if node.NormalAttributes["tags"] == nil {
			node.NormalAttributes["tags"] = []interface{}{}
		}
	}
	// Ensure that what we post at the end of the run is different from previous runs
	endTime := time.Now().UTC()
	node.AutomaticAttributes["ohai_time"] = endTime.Unix()

	if config.RunChefClient {
		apiRequest(nodeClient, nodeName, config.ChefVersion, "PUT", "nodes/"+nodeName, node, nil, nil)

		// Notify Reporting of run end
		if config.EnableReporting && reportingAvailable {
			reportingRunStop(nodeClient, nodeName, config.ChefVersion, runUUID, startTime, endTime, runList)
		}
	}

	// Notify Data Collector of run end
	runStopBody := dataCollectorRunStop(config, node, nodeName, chefServerFQDN, orgName, status, runList,
		parseRunList(expandedRunList), runUUID, nodeUUID, startTime, endTime, convergeJSON)
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(nodeName, config.DataCollectorToken, config.DataCollectorURL, runStopBody)
	} else if dataCollectorAvailable {
		apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "data-collector", runStopBody, nil, nil)
	}

	// Notify Data Collector of compliance report
	if len(complianceJSON) != 0 {
		complianceReportBody := dataCollectorComplianceReport(nodeName, chefEnvironment, reportUUID, nodeUUID, endTime, complianceJSON)
		if config.DataCollectorURL != "" {
			chefAutomateSendMessage(nodeName, config.DataCollectorToken, config.DataCollectorURL, complianceReportBody)
		} else {
			apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "data-collector", complianceReportBody, nil, nil)
		}
	}
}
