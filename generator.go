//
// Copyright:: Copyright 2017 Chef Software, Inc.
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

package main

// This file will have the functions that will generate random data,
// it involve creating fake Chef Nodes and Chef Runs that can be sent
// to the data-collector endpoint

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/go-chef/chef"
	uuid "github.com/satori/go.uuid"
)

func generateRandomData(nodeClient chef.Client, ohaiJSON, convergeJSON, complianceJSON map[string]interface{}, requests amountOfRequests) (err error) {
	channels := make([]<-chan error, config.NumNodes)

	for i := 0; i < config.NumNodes; i++ {
		nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i+1)
		fmt.Printf(".")
		channels[i] = ccr(nodeClient, nodeName, ohaiJSON, convergeJSON, complianceJSON)
	}

	fmt.Println("\n")

	for n := range merge(channels...) {
		if n != nil {
			fmt.Println("Error: ", n)
			err = n
		}
	}

	printAPIRequestProfile(requests)

	return err
}

func ccr(nodeClient chef.Client, nodeName string,
	ohaiJSON, convergeJSON, complianceJSON map[string]interface{}) <-chan error {
	out := make(chan error)
	go func() {
		randomChefClientRun(nodeClient, nodeName, ohaiJSON, convergeJSON, complianceJSON)
		close(out)
	}()
	return out
}

func merge(cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan error) {
		for err := range c {
			out <- err
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func randomChefClientRun(
	nodeClient chef.Client,
	nodeName string,
	ohaiJSON map[string]interface{},
	convergeJSON map[string]interface{},
	complianceJSON map[string]interface{}) {
	// ALL_ROLES = ['admin', 'windows_builder', 'stash']
	// ALL_EVENT_ACTIONS = ['created', 'updated', 'unknown']
	// ALL_POLICY_NAMES = ['policy1', 'policy2', 'policy3']
	// ALL_POLICY_GROUPS = ['dev', 'prod', 'audit']

	environments := make([]string, 0)
	environments = append(environments,
		"arctic",
		"coast",
		"desert",
		"forest",
		"grassland",
		"mountain",
		"swamp",
		"underdark",
		"astral plane",
		"ethereal plane",
		"plane of shadow",
		"feywild",
		"shadowfell",
		"mirror plane",
		"outer space",
		"acceptance-org-proj-master")

	chefEnvironment := environments[rand.Intn(len(environments))]

	runList := parseRunList(config.RunList)

	apiGetRequests := config.APIGetRequests
	sleepDuration := config.SleepDuration
	runUUID := uuid.NewV4()
	reportUUID := uuid.NewV4()
	nodeUUID := uuid.NewV3(uuid.NamespaceDNS, nodeName)
	startTime := time.Now().UTC()

	organizations := make([]string, 0)
	organizations = append(organizations,
		"The Avengers",
		"The Defenders",
		"Justice League of America",
		"The Great Lakes Avengers",
		"The Fantastic Four",
		"Astonishing X-Men",
		"Justice League of Antarctica",
		"The Misfits",
		"The Secret Six",
		"Teen Titans",
		"Watchmen",
		"Guardians of the Galaxy",
		"S.H.I.E.L.D.",
		"Howling Commandos",
		"Ultimates",
		"X-Factor",
		"Uncanny X-Men",
		"Next Wave")

	orgName := organizations[rand.Intn(len(organizations))]
	reportingAvailable := true
	dataCollectorAvailable := true
	var expandedRunList []string
	var node chef.Node

	// not normally in sample data
	ohaiJSON["fqdn"] = nodeName

	platforms := make([]string, 0)
	platforms = append(platforms,
		"centos",
		"ubuntu",
		"oracle",
		"solaris",
		"windows",
		"mac_os_x",
		"salim",
		"kyleen",
		"lance",
		"rachel",
		"shadae",
		"maggie",
		"elizabeth",
		"platform 14")

	ohaiJSON["platform"] = platforms[rand.Intn(len(platforms))]

	if config.RunChefClient {
		clientBody := map[string]interface{}{
			"admin":     false,
			"name":      nodeName,
			"validator": false,
		}
		if config.ChefServerCreatesClientKey {
			clientBody["create_key"] = config.ChefServerCreatesClientKey
		}
		apiRequest(nodeClient, nodeName, "POST", "clients", clientBody, nil, nil)

		res, err := apiRequest(nodeClient, nodeName, "GET", "nodes/"+nodeName, nil, &node, nil)
		if err != nil {
			if res != nil && res.StatusCode != 404 {
				node = chef.Node{Name: nodeName}
			}
		}
		if res != nil && res.StatusCode == 404 {
			node = chef.Node{Name: nodeName, Environment: chefEnvironment}
			_, err = apiRequest(nodeClient, nodeName, "POST", "nodes", node, nil, nil)
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
		expandedRunList = runList.expand(&nodeClient, nodeName, chefEnvironment)

		apiRequest(nodeClient, nodeName, "GET", "environments/"+chefEnvironment, nil, nil, nil)

		// Notify Reporting of run start
		if config.EnableReporting {
			res, _ := reportingRunStart(nodeClient, nodeName, runUUID, startTime)
			if res != nil && res.StatusCode == 404 {
				reportingAvailable = false
			}
		}
	}

	// Notify Data Collector of run start
	runStartBody := dataCollectorRunStart(nodeName, orgName, runUUID, nodeUUID, startTime)
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(nodeName, config.DataCollectorToken, config.DataCollectorURL, runStartBody)
	} else {
		res, err := apiRequest(nodeClient, nodeName, "POST", "data-collector", runStartBody, nil, nil)
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
		ckbks := solveRunListDependencies(&nodeClient, nodeName, expandedRunList, chefEnvironment)

		// Download cookbooks
		if config.DownloadCookbooks == "always" || (config.DownloadCookbooks == "first") {
			ckbks.download(&nodeClient, nodeName)
		}

		for _, apiGetRequest := range apiGetRequests {
			apiRequest(nodeClient, nodeName, "GET", apiGetRequest, nil, nil, nil)
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
		apiRequest(nodeClient, nodeName, "PUT", "nodes/"+nodeName, node, nil, nil)

		// Notify Reporting of run end
		if config.EnableReporting && reportingAvailable {
			reportingRunStop(nodeClient, nodeName, runUUID, startTime, endTime, runList)
		}
	}

	// Notify Data Collector of run end
	runStopBody := dataCollectorRunStop(node, nodeName, orgName, runList, parseRunList(expandedRunList), runUUID, nodeUUID, startTime, endTime, convergeJSON)
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(nodeName, config.DataCollectorToken, config.DataCollectorURL, runStopBody)
	} else if dataCollectorAvailable {
		apiRequest(nodeClient, nodeName, "POST", "data-collector", runStopBody, nil, nil)
	}

	// Notify Data Collector of compliance report
	if len(complianceJSON) != 0 {
		complianceReportBody := dataCollectorComplianceReport(nodeName, chefEnvironment, reportUUID, nodeUUID, endTime, complianceJSON)
		if config.DataCollectorURL != "" {
			chefAutomateSendMessage(nodeName, config.DataCollectorToken, config.DataCollectorURL, complianceReportBody)
		} else {
			apiRequest(nodeClient, nodeName, "POST", "data-collector", complianceReportBody, nil, nil)
		}
	}
}
