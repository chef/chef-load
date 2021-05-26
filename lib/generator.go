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
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/go-chef/chef"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func GenerateData(config *Config) error {
	var (
		numRequests = make(amountOfRequests)
		requests    = make(chan *request)
	)

	go func() {
		for {
			select {
			case req := <-requests:
				numRequests.addRequest(request{Method: req.Method, Url: req.Url, StatusCode: req.StatusCode})
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		GenerateComplianceData(config, requests)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		GenerateChefActions(config, requests)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		GenerateCCRs(config, requests)
	}()
	if config.LivenessAgent {
		wg.Add(1)
		go func() {
			defer wg.Done()
			GenerateLivenessData(config, requests)
		}()
	}

	wg.Wait()

	printAPIRequestProfile(numRequests)

	return nil
}

func GenerateCCRs(config *Config, requests chan *request) (err error) {
	var (
		chefClient   chef.Client
		channels     []<-chan int
		ccrsPerDay   int64 = 1
		ccrsTotal    int64 = 1
		c            int64 = 0
		ccrsIngested int64 = 0
		ccrsRejected int64 = 0
		batches      int   = 1
		code         int   = 999
		rejects      bool  = false
		timeMarker         = time.Now()
	)

	// Calculate how many chef-client runs we need to trigger
	//
	// Example: If we have 10 nodes with 30 days of data converging every 30 minutes
	//
	// ccrsPerDay = 1440m / 30m = 40 CCR a day
	// ccrsTotal = ccrsPerDay * 30d = 1200 per Node
	if config.DaysBack > 0 {
		ccrsPerDay = 1440 / int64(config.Interval)
		ccrsTotal = ccrsPerDay * int64(config.DaysBack)
	}

	if config.RunChefClient {
		chefClient = getAPIClient(config.ClientName, config.ClientKey, config.ChefServerURL)
	}

	log.WithFields(log.Fields{
		"nodes":        config.NumNodes,
		"days_back":    config.DaysBack,
		"ccr_per_node": ccrsTotal,
		"total_ccrs":   ccrsTotal * int64(config.NumNodes),
		"random_data":  config.RandomData,
		"goroutines":   config.Threads,
	}).Info("Generating chef-client runs")

	rand.Seed(time.Now().UTC().UnixNano())

	// Lets try to use a smaller number of goroutines
	if config.NumNodes > config.Threads {
		// If the number of nodes is bigger than the channel
		// size, lets calculate how many batches we need to run
		batches = int(config.NumNodes / config.Threads)
		if config.NumNodes%config.Threads != 0 {
			batches++
		}
		channels = make([]<-chan int, config.Threads)
	} else {
		channels = make([]<-chan int, config.NumNodes)
	}

	// For the total of CCRs per node, run a converge
	for c = 0; c < ccrsTotal; c++ {

		// batches * config.Threads = NumNodes (ish)
		for j := 0; j < batches; j++ {

			if ((j + 1) * config.Threads) > config.NumNodes {
				size := config.NumNodes % config.Threads
				channels = make([]<-chan int, size)
			}

			for i := 0; i < config.Threads; i++ {
				nodeNum := i + (j * config.Threads)
				// The trick here is to stop the last loop when we reach
				// the total number of nodes that we want to load
				if nodeNum >= config.NumNodes {
					break
				}
				nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(nodeNum+1)
				channels[i] = ccr(config, chefClient, nodeName, requests)
			}

			for code = range merge(channels...) {
				if code != 200 {
					rejects = true
					ccrsRejected++
					// err = n
				} else {
					ccrsIngested++
				}
			}

			// When we start rejecting/dropping messages we will wait
			// an interval of time to let the system digest
			if config.DaysBack > 0 {
				if rejects {
					log.WithFields(log.Fields{
						"ccrs_per_node":                   ccrsTotal,
						"total_ccrs":                      ccrsTotal * int64(config.NumNodes),
						"total_ccrs_ingested":             (c * int64(config.NumNodes)) + int64(j*config.Threads) + ccrsIngested,
						"sleep":                           fmt.Sprintf("%ds", config.SleepTimeOnFailure),
						"time_elapsed_since_last_failure": time.Now().Sub(timeMarker),
						"ccr_ingested_since_last_failure": ccrsIngested,
						"ccr_rejected_since_last_failure": ccrsRejected,
						"goroutines":                      config.Threads,
						"nodes":                           config.NumNodes,
						"days_back":                       config.DaysBack,
					}).Info("Sleeping")
					time.Sleep(time.Second * time.Duration(config.SleepTimeOnFailure))

					rejects = false
					ccrsIngested = 0
					ccrsRejected = 0
					timeMarker = time.Now()
					if config.NumNodes > config.Threads {
						channels = make([]<-chan int, config.Threads)
					} else {
						channels = make([]<-chan int, config.NumNodes)
					}
				}
			}
		}
	}

	return
}

func ccr(config *Config, chefClient chef.Client, nodeName string, requests chan *request) <-chan int {
	out := make(chan int)
	go func() {
		code, _ := randomChefClientRun(config, chefClient, nodeName, requests)
		out <- code
		close(out)
	}()
	return out
}

func merge(cs ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	out := make(chan int)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan int) {
		for code := range c {
			out <- code
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
	case "source_fqdn":
		return sourceFqdns[rand.Intn(len(sourceFqdns))]
	case "status":
		return ccrStatus[rand.Intn(len(ccrStatus))]
	case "cookbook":
		return randCookbooks[rand.Intn(len(randCookbooks))]
	case compEnvironments:
		return complianceEnv[rand.Intn(len(complianceEnv))]
	default:
		return ""
	}
}

func getRandomStringArray(kind string) []string {
	switch kind {
	case compRecipes:
		return complianceRecipes[rand.Intn(len(complianceRecipes))]
	case compRoles:
		return complianceRoles[rand.Intn(len(complianceRoles))]
	default:
		return []string{}
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
		cb := getRandom("cookbook")
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

func genRandomTags() []string {
	const instances = 3
	tagsSize := rand.Intn(10) + 1
	ts := make([]string, tagsSize)
	perm := rand.Perm(len(tags))
	for i := range ts {
		tag := tags[perm[i]]
		instance := rand.Intn(instances)
		ts[i] = fmt.Sprintf("%s%d", tag, instance)
	}

	return ts
}

func genStartEndTime(config *Config) (time.Time, time.Time) {
	var (
		sTime time.Time
		eTime time.Time
	)
	if config.DaysBack > 0 {
		hours := rand.Intn(config.DaysBack) * 24
		historyDuration, _ := time.ParseDuration(fmt.Sprintf("%dh", hours))
		sTime = time.Now().UTC().Add(-historyDuration).UTC()
	} else {
		sTime = time.Now().UTC()
	}
	minutes := rand.Intn(60)
	randDuration, _ := time.ParseDuration(fmt.Sprintf("%dm", minutes))
	eTime = sTime.Add(randDuration).UTC()

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

func randomChefClientRun(config *Config, chefClient chef.Client, nodeName string, requests chan *request) (int, error) {
	var (
		startTime, endTime     = genStartEndTime(config)
		runUUID                = uuid.New()
		nodeUUID               = uuid.NewMD5(uuid.NameSpaceDNS, []byte(nodeName))
		orgName                = getRandom("organization")
		chefServerFQDN         = getRandom("source_fqdn")
		status                 = getRandom("status")
		node                   = chef.NewNode(nodeName) // Our Random Chef Node
		reportingAvailable     = true
		dataCollectorAvailable = true
		code                   int
		expandedRunList        []string
		convergeJSON           = map[string]interface{}{ // This is used just for the list of resources
			"resources": genRandomResourcesTree(),
		}
		randRunList, randRecipes = genRandomRunList()
	)

	node.Environment = getRandom("environment")
	node.RunList = randRunList
	if config.OhaiJSONFile != "" {
		node.AutomaticAttributes = parseJSONFile(config.OhaiJSONFile)
	} else {
		node.AutomaticAttributes = map[string]interface{}{}
	}
	node.AutomaticAttributes["fqdn"] = nodeName
	node.AutomaticAttributes["roles"] = []string{getRandom("role")}
	node.AutomaticAttributes["platform"] = getRandom("platform")
	// TODO: (@afiune) Do we need platform version and family?
	//"platform_version": "7.1",
	//"platform_family": "rhel",

	node.AutomaticAttributes["recipes"] = randRecipes
	node.AutomaticAttributes["cookbooks"] = cookbooksData
	node.AutomaticAttributes["uptime_seconds"] = 0
	node.NormalAttributes = genRandomAttributes()
	node.NormalAttributes["tags"] = genRandomTags()
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
		apiRequest(chefClient, nodeName, config.ChefVersion, "POST", "clients", clientBody, nil, nil, requests)

		res, _ := apiRequest(chefClient, nodeName, config.ChefVersion, "GET", "nodes/"+nodeName, nil, &node, nil, requests)
		if res != nil && res.StatusCode == 404 {
			apiRequest(chefClient, nodeName, config.ChefVersion, "POST", "nodes", node, nil, nil, requests)
		}
	}

	if config.RunChefClient {
		// Expand run_list
		expandedRunList = runList.expand(&chefClient, nodeName, config.ChefVersion, node.Environment, requests)

		// TODO Check error?
		apiRequest(chefClient, nodeName, config.ChefVersion, "GET", "environments/"+node.Environment, nil, nil, nil, requests)

		// Notify Reporting of run start
		if config.EnableReporting {
			res, _ := reportingRunStart(chefClient, nodeName, config.ChefVersion, runUUID, startTime, requests)
			if res != nil && res.StatusCode == 404 {
				reportingAvailable = false
			}
		}
	}

	dataCollectorClient, err := NewDataCollectorClient(&DataCollectorConfig{
		Token:   config.DataCollectorToken,
		URL:     config.DataCollectorURL,
		SkipSSL: true,
	}, requests)
	if err != nil {
		return 999, errors.New(fmt.Sprintf("Error creating DataCollectorClient: %+v \n", err))
	}

	// Notify Data Collector of run start
	runStartBody := dataCollectorRunStart(config, nodeName, chefServerFQDN, orgName, runUUID, nodeUUID, startTime)
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(dataCollectorClient, nodeName, runStartBody)
	} else {
		// TODO Check error?
		apiRequest(chefClient, nodeName, config.ChefVersion, "POST", "data-collector", runStartBody, nil, nil, requests)
	}

	if config.RunChefClient {
		// Calculate cookbook dependencies
		ckbks := solveRunListDependencies(&chefClient, nodeName, config.ChefVersion, node.Environment, expandedRunList, requests)

		// Download cookbooks
		if config.DownloadCookbooks == "always" || (config.DownloadCookbooks == "first") {
			ckbks.download(&chefClient, nodeName, config.ChefVersion, requests)
		}
	} else {
		expandedRunList = runList.toStringSlice()
	}

	if config.RunChefClient {
		apiRequest(chefClient, nodeName, config.ChefVersion, "PUT", "nodes/"+nodeName, node, nil, nil, requests)

		// Notify Reporting of run end
		if config.EnableReporting && reportingAvailable {
			reportingRunStop(chefClient, nodeName, config.ChefVersion, runUUID, startTime, endTime, runList, requests)
		}
	}

	// Notify Data Collector of run end
	runStopBody := dataCollectorRunStop(config, node, nodeName, chefServerFQDN, orgName, status, runList,
		parseRunList(expandedRunList), runUUID, nodeUUID, startTime, endTime, convergeJSON)
	if config.DataCollectorURL != "" {
		chefAutomateSendMessage(dataCollectorClient, nodeName, runStopBody)
	} else if dataCollectorAvailable {
		apiRequest(chefClient, nodeName, config.ChefVersion, "POST", "data-collector", runStopBody, nil, nil, requests)
	}

	// Send an Update Action that we just ran a CCR and the node updated itself
	ccrAction := newActionRequest(nodeAction)
	ccrAction.SetTask(updateTask)
	ccrAction.EntityName = nodeName
	ccrAction.RequestorName = nodeName
	if config.DataCollectorURL != "" {
		code, err = chefAutomateSendMessage(dataCollectorClient, ccrAction.String(), ccrAction)
	} else if dataCollectorAvailable {
		apiRequest(chefClient, ccrAction.String(), config.ChefVersion, "POST", "data-collector", ccrAction, nil, nil, requests)
	}

	// TODO: (@afiune) Notify Data Collector of compliance report
	//reportUUID, _ := uuid.NewV4()
	//
	//if len(config.ComplianceStatusJSONFile) != 0 {
	//	complianceJSON := parseJSONFile(config.ComplianceStatusJSONFile)
	//	complianceReportBody := dataCollectorComplianceReport(nodeName, "chefEnvironment", reportUUID, nodeUUID, endTime, complianceJSON)
	//	if config.DataCollectorURL != "" {
	//		chefAutomateSendMessage(dataCollectorClient, nodeName, complianceReportBody)
	//	} else {
	//		apiRequest(chefClient, nodeName, config.ChefVersion, "POST", "data-collector", complianceReportBody, nil, nil, requests)
	//	}
	//}
	return code, err
}
