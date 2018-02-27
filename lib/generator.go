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

func generateRandomData(chefClient chef.Client, requests amountOfRequests) (err error) {
	channels := make([]<-chan error, config.NumNodes)

	for i := 0; i < config.NumNodes; i++ {
		nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i+1)
		fmt.Printf(".")
		channels[i] = ccr(chefClient, nodeName)
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

func ccr(chefClient chef.Client, nodeName string) <-chan error {
	out := make(chan error)
	go func() {
		randomChefClientRun(chefClient, nodeName)
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

func getRandom(kind string) string {
	switch kind {
	case "environment":
		return environments[rand.Intn(len(environments))]
	case "organization":
		return organizations[rand.Intn(len(organizations))]
	case "role":
		return roles[rand.Intn(len(roles))]
	case "platform":
		return platforms[rand.Intn(len(platforms))]
	case "tag":
		return tags[rand.Intn(len(tags))]
	case "source_fqdn":
		return sourceFqdns[rand.Intn(len(sourceFqdns))]
	case "status":
		return ccrStatus[rand.Intn(len(ccrStatus))]
	default:
		return ""
	}
}

func genRandomResourcesTree() []interface{} {
	resourcesSize := rand.Intn(8)
	randResources := make([]interface{}, resourcesSize)
	for i := 0; i < resourcesSize; i++ {
		randResources[i] = resources[rand.Intn(len(resources))]
	}
	return randResources
}

func genRandomRunList() ([]string, []string) {
	runListSize := rand.Intn(3) + 1
	runList := make([]string, runListSize)
	recipeList := make([]string, runListSize)
	for i := 0; i < runListSize; i++ {
		cb := randCookbooks[rand.Intn(len(randCookbooks))]
		runList[i] = fmt.Sprintf("recipe[%s::default]", cb)
		recipeList[i] = fmt.Sprintf("%s::default", cb)
	}
	return runList, recipeList
}

func genRandomAttributes() map[string]interface{} {
	attrSize := rand.Intn(10) + 1
	randAttributes := make(map[string]interface{}, attrSize)
	for i := 0; i < attrSize; i++ {
		k := randAttributeMapKey(attributes)
		randAttributes[k] = attributes[k]
	}
	return randAttributes
}

func genRandomStartEndTime() (time.Time, time.Time) {
	var (
		minutes         = rand.Intn(60)
		randDuration, _ = time.ParseDuration(fmt.Sprintf("%dm", minutes))
		sTime           = time.Now().UTC()
		eTime           = sTime.Add(randDuration).UTC()
	)
	return sTime, eTime
}

func randAttributeMapKey(m map[string]interface{}) string {
	i := rand.Intn(len(m))
	for k, _ := range m {
		if i == 0 {
			return k
		}
		i--
	}
	return ""
}

func randomChefClientRun(chefClient chef.Client, nodeName string) {
	var (
		startTime, endTime     = genRandomStartEndTime()
		runUUID, _             = uuid.NewV4()
		nodeUUID               = uuid.NewV3(uuid.NamespaceDNS, nodeName)
		orgName                = getRandom("organization")
		chefServerFQDN         = getRandom("source_fqdn")
		status                 = getRandom("status")
		node                   = chef.NewNode(nodeName) // Our Random Chef Node
		reportingAvailable     = true
		dataCollectorAvailable = true
		expandedRunList        []string
		convergeJSON           = map[string]interface{}{ // This is used just for the list of resources
			"resources": genRandomResourcesTree(),
		}
		randRunList, randRecipes = genRandomRunList()
	)

	node.Environment = getRandom("environment")
	node.RunList = randRunList
	node.AutomaticAttributes = map[string]interface{}{}
	node.AutomaticAttributes["fqdn"] = nodeName
	node.AutomaticAttributes["roles"] = []string{getRandom("role")}
	node.AutomaticAttributes["platform"] = getRandom("platform")
	// TODO: (@afiune) Do we need platform version and family?
	//"platform_version": "7.1",
	//"platform_family": "rhel",

	node.AutomaticAttributes["recipes"] = randRecipes
	node.AutomaticAttributes["cookbooks"] = map[string]interface{}{}
	node.AutomaticAttributes["uptime_seconds"] = 0
	node.NormalAttributes = genRandomAttributes()
	node.NormalAttributes["tags"] = []string{getRandom("tag")}

	// This run_list is used by the RunChefClient flag, when there is a ChefServerUrl specified
	runList := parseRunList(node.RunList)

	if config.RunChefClient {
		clientBody := map[string]interface{}{
			"admin":     false,
			"name":      nodeName,
			"validator": false,
		}
		if config.ChefServerCreatesClientKey {
			clientBody["create_key"] = config.ChefServerCreatesClientKey
		}
		apiRequest(chefClient, nodeName, "POST", "clients", clientBody, nil, nil)

		res, _ := apiRequest(chefClient, nodeName, "GET", "nodes/"+nodeName, nil, &node, nil)
		if res != nil && res.StatusCode == 404 {
			apiRequest(chefClient, nodeName, "POST", "nodes", node, nil, nil)
		}
	}

	if config.RunChefClient {
		// Expand run_list
		expandedRunList = runList.expand(&chefClient, nodeName, node.Environment)

		// TODO Check error?
		apiRequest(chefClient, nodeName, "GET", "environments/"+node.Environment, nil, nil, nil)

		// Notify Reporting of run start
		if config.EnableReporting {
			res, _ := reportingRunStart(chefClient, nodeName, runUUID, startTime)
			if res != nil && res.StatusCode == 404 {
				reportingAvailable = false
			}
		}
	}

	// Notify Data Collector of run start
	runStartBody := dataCollectorRunStart(nodeName, chefServerFQDN, orgName, runUUID, nodeUUID, startTime)
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(nodeName, config.DataCollectorToken, config.DataCollectorURL, runStartBody)
	} else {
		// TODO Check error?
		apiRequest(chefClient, nodeName, "POST", "data-collector", runStartBody, nil, nil)
	}

	if config.RunChefClient {
		// Calculate cookbook dependencies
		ckbks := solveRunListDependencies(&chefClient, nodeName, expandedRunList, node.Environment)

		// Download cookbooks
		if config.DownloadCookbooks == "always" || (config.DownloadCookbooks == "first") {
			ckbks.download(&chefClient, nodeName)
		}
	} else {
		expandedRunList = runList.toStringSlice()
	}

	if config.RunChefClient {
		apiRequest(chefClient, nodeName, "PUT", "nodes/"+nodeName, node, nil, nil)

		// Notify Reporting of run end
		if config.EnableReporting && reportingAvailable {
			reportingRunStop(chefClient, nodeName, runUUID, startTime, endTime, runList)
		}
	}

	// Notify Data Collector of run end
	runStopBody := dataCollectorRunStop(node, nodeName, chefServerFQDN, orgName, status, runList,
		parseRunList(expandedRunList), runUUID, nodeUUID, startTime, endTime, convergeJSON)
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(nodeName, config.DataCollectorToken, config.DataCollectorURL, runStopBody)
	} else if dataCollectorAvailable {
		apiRequest(chefClient, nodeName, "POST", "data-collector", runStopBody, nil, nil)
	}

	// TODO: (@afiune) Notify Data Collector of compliance report
	//reportUUID := uuid.NewV4()
	//if len(complianceJSON) != 0 {
	//complianceReportBody := dataCollectorComplianceReport(nodeName, chefEnvironment, reportUUID, nodeUUID, endTime, complianceJSON)
	//if config.DataCollectorURL != "" {
	//chefAutomateSendMessage(nodeName, config.DataCollectorToken, config.DataCollectorURL, complianceReportBody)
	//} else {
	//apiRequest(chefClient, nodeName, "POST", "data-collector", complianceReportBody, nil, nil)
	//}
	//}
}
