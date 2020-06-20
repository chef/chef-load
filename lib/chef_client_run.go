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
	"github.com/google/uuid"
)

func ChefClientRun(config *Config, nodeName string, firstRun bool, requests chan *request, nodeNumber uint32) {
	var (
		nodeClient             chef.Client
		ohaiJSON               = map[string]interface{}{}
		convergeJSON           = map[string]interface{}{}
		complianceJSON         = map[string]interface{}{}
		chefEnvironment        = config.ChefEnvironment
		runList                = parseRunList(config.RunList)
		apiGetRequests         = config.APIGetRequests
		sleepDuration          = config.SleepDuration
		runUUID, _             = uuid.NewRandom()
		reportUUID, _          = uuid.NewRandom()
		roles                  = getRandomStringArray(compRoles)
		recipes                = getRandomStringArray(compRecipes)
		nodeUUID               = uuid.NewMD5(uuid.NameSpaceDNS, []byte(nodeName))
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
		nodeDetails            = NodeDetails{
			name:        nodeName,
			ipAddr:      int2ip(nodeNumber).String(),
			environment: chefEnvironment,
			roles:       roles,
			recipes:     recipes,
			nodeUUID:    nodeUUID,
			sourceFqdn:  chefServerFQDN,
			fqdn:        node.Name,
			orgName:     orgName,
			policyGroup: "hello_policy_group",
			policyName:  "hello_policy_name",
			chefTags:    []string{"tag1", "tag2", "tag3"},
		}
	)

	if config.RunChefClient {
		nodeClient = getAPIClient(config.ClientName, config.ClientKey, config.ChefServerURL)
	}

	if config.OhaiJSONFile != "" {
		ohaiJSON = parseJSONFile(config.OhaiJSONFile)
	}
	if config.ConvergeStatusJSONFile != "" {
		convergeJSON = parseJSONFile(config.ConvergeStatusJSONFile)
	}

	if config.ComplianceStatusJSONFile != "" {
		complianceJSON = parseJSONFile(config.ComplianceStatusJSONFile)
	}

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
			apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "clients", clientBody, nil, nil, requests)
		}

		res, err := apiRequest(nodeClient, nodeName, config.ChefVersion, "GET", "nodes/"+nodeName, nil, &node, nil, requests)
		if err != nil {
			if res != nil && res.StatusCode != 404 {
				node = chef.Node{Name: nodeName}
			}
		}
		if res != nil && res.StatusCode == 404 {
			node = chef.Node{Name: nodeName, Environment: chefEnvironment}
			_, err = apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "nodes", node, nil, nil, requests)
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
		expandedRunList = runList.expand(&nodeClient, nodeName, config.ChefVersion, chefEnvironment, requests)

		apiRequest(nodeClient, nodeName, config.ChefVersion, "GET", "environments/"+chefEnvironment, nil, nil, nil, requests)

		// Notify Reporting of run start
		if config.EnableReporting {
			res, _ := reportingRunStart(nodeClient, nodeName, config.ChefVersion, runUUID, startTime, requests)
			if res != nil && res.StatusCode == 404 {
				reportingAvailable = false
			}
		}
	}

	// TODO: Check all the errors!
	dataCollectorClient, _ := NewDataCollectorClient(&DataCollectorConfig{
		Token:   config.DataCollectorToken,
		URL:     config.DataCollectorURL,
		SkipSSL: true,
	}, requests)
	//if err != nil {
	//return errors.New(fmt.Sprintf("Error creating DataCollectorClient: %+v \n", err))
	//}

	// Notify Data Collector of run start
	runStartBody := dataCollectorRunStart(config, nodeName, "", orgName, runUUID, nodeUUID, startTime)
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(dataCollectorClient, nodeName, runStartBody)
	} else {
		res, err := apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "data-collector", runStartBody, nil, nil, requests)
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
		ckbks := solveRunListDependencies(&nodeClient, nodeName, config.ChefVersion, chefEnvironment, expandedRunList, requests)

		// Download cookbooks
		if config.DownloadCookbooks == "always" || (config.DownloadCookbooks == "first" && firstRun) {
			ckbks.download(&nodeClient, nodeName, config.ChefVersion, requests)
		}

		for _, apiGetRequest := range apiGetRequests {
			apiRequest(nodeClient, nodeName, config.ChefVersion, "GET", apiGetRequest, nil, nil, nil, requests)
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
		apiRequest(nodeClient, nodeName, config.ChefVersion, "PUT", "nodes/"+nodeName, node, nil, nil, requests)

		// Notify Reporting of run end
		if config.EnableReporting && reportingAvailable {
			reportingRunStop(nodeClient, nodeName, config.ChefVersion, runUUID, startTime, endTime, runList, requests)
		}
	}

	// Notify Data Collector of run end
	runStopBody := dataCollectorRunStop(config, node, nodeName, chefServerFQDN, orgName, status, runList,
		parseRunList(expandedRunList), runUUID, nodeUUID, startTime, endTime, convergeJSON)
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(dataCollectorClient, nodeName, runStopBody)
	} else if dataCollectorAvailable {
		apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "data-collector", runStopBody, nil, nil, requests)
	}

	// Send an Update Action that we just ran a CCR and the node updated itself
	ccrAction := newActionRequest(nodeAction)
	ccrAction.SetTask(updateTask)
	ccrAction.EntityName = nodeName
	ccrAction.RequestorName = nodeName
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(dataCollectorClient, ccrAction.String(), ccrAction)
	} else if dataCollectorAvailable {
		apiRequest(nodeClient, ccrAction.String(), config.ChefVersion, "POST", "data-collector", ccrAction, nil, nil, requests)
	}

	// Notify Data Collector of compliance report
	if len(complianceJSON) != 0 {
		complianceReportBody := dataCollectorComplianceReport(nodeDetails, reportUUID, endTime, complianceJSON)
		if config.DataCollectorURL != "" {
			chefAutomateSendMessage(dataCollectorClient, nodeName, complianceReportBody)
		} else {
			apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "data-collector", complianceReportBody, nil, nil, requests)
		}
	}
}
