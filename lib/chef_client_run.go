//
// Copyright:: Copyright 2026 Progress Software Corporation, Inc.
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
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/go-chef/chef"
	"github.com/google/uuid"
)

func ChefClientRun(config *Config, nodeName string, firstRun bool, requests chan *request, done chan int, nodeNumber uint32) {
	var (
		nodeClient             chef.Client
		ohaiJSON               = map[string]interface{}{}
		convergeJSON           = map[string]interface{}{}
		complianceJSON         = map[string]interface{}{}
		chefEnvironment        = config.ChefEnvironment
		runList                = parseRunList(config.RunList)
		runLists               = parseRunLists(config.RunLists)
		apiGetRequests         = config.APIGetRequests
		sleepDuration          = config.SleepDuration
		runUUID, _             = uuid.NewRandom()
		reportUUID, _          = uuid.NewRandom()
		skipClientCreation     = config.SkipClientCreation
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
		expandedRunListID      string
		policy                 PolicyDocument
		node                   chef.Node
		nodeDetails            = NodeDetails{
			name:        nodeName,
			ipAddr:      int2ip(nodeNumber).String(),
			environment: chefEnvironment,
			roles:       roles,
			recipes:     recipes,
			nodeUUID:    nodeUUID,
			sourceFqdn:  chefServerFQDN,
			// fqdn is the node's own hostname; use nodeName as the best
			// available value at construction time (node hasn't been fetched yet).
			fqdn:        nodeName,
			orgName:     orgName,
			policyGroup: "hello_policy_group",
			policyName:  "hello_policy_name",
			chefTags:    []string{"tag1", "tag2", "tag3"},
		}
	)

	// Determine whether this node slot should simulate a policyfile-based
	// chef-client run.  Three load modes are supported:
	//   "legacy"     – traditional run-list/roles/environments (original behavior)
	//   "policyfile" – every node uses the policyfile API
	//   "mixed"      – a deterministic fraction of node slots use policyfile;
	//                  the rest follow the traditional path.
	usePolicyfile := false
	switch config.LoadMode {
	case "policyfile":
		usePolicyfile = true
	case "mixed":
		usePolicyfile = float64(nodeNumber)/float64(config.NumNodes) < config.PolicyfilePercentage
	}

	// When multiple policyfiles are configured, pick one at random for this run.
	// This mirrors how run_lists works for legacy nodes — each simulated run
	// gets a randomly selected policy name from the pool. All names are fetched
	// from the same policy_group; policy_name is ignored when the list is set.
	activePolicyName := config.PolicyName
	activePolicyGroup := config.PolicyGroup
	if len(config.Policyfiles) > 0 {
		activePolicyName = config.Policyfiles[rand.Intn(len(config.Policyfiles))] //nolint:gosec
	}

	// For policyfile nodes update nodeDetails to carry the real policy identity
	// (used in compliance reports sent to Automate) and set the correct
	// expanded_run_list id for the data-collector run_converge message.
	if usePolicyfile {
		nodeDetails.policyName = activePolicyName
		nodeDetails.policyGroup = activePolicyGroup
		// For policyfile nodes chef-client sets chef_environment to the policy group.
		nodeDetails.environment = activePolicyGroup
		expandedRunListID = "_policy_node"
	} else {
		expandedRunListID = chefEnvironment
	}
	// Notify orchestrator when done.  THere's probably a cleaner way to
	// do this.
	closer := func() {
		// log.info(fmt.Printf("[node: %s] simulated converge time: %s", nodeName, time.Since(startTime)))
		done <- int(nodeNumber)
	}
	defer closer()

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

		if firstRun && !skipClientCreation {
			clientBody := map[string]interface{}{
				"admin":     false,
				"name":      nodeName,
				"validator": false,
			}
			if config.ChefServerCreatesClientKey {
				clientBody["create_key"] = config.ChefServerCreatesClientKey
			}
			_, _ = apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "clients", clientBody, nil, nil, requests) //nolint:bodyclose
		}

		res, err := apiRequest(nodeClient, nodeName, config.ChefVersion, "GET", "nodes/"+nodeName, nil, &node, nil, requests) //nolint:bodyclose
		if err != nil {
			if res != nil && res.StatusCode != 404 {
				node = chef.Node{Name: nodeName}
			}
		}
		if res != nil && res.StatusCode == 404 {
			// Create the node with the correct identity for the run mode.
			if usePolicyfile {
				node = chef.Node{Name: nodeName, PolicyName: activePolicyName, PolicyGroup: activePolicyGroup}
			} else {
				node = chef.Node{Name: nodeName, Environment: chefEnvironment}
			}
			_, err = apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "nodes", node, nil, nil, requests) //nolint:bodyclose
			if err != nil {
				node = chef.Node{Name: nodeName}
			}
		}
	} else {
		node = chef.Node{Name: nodeName}
	}
	node.AutomaticAttributes = ohaiJSON

	// Apply node identity fields based on the run mode.
	if usePolicyfile {
		node.PolicyName = activePolicyName
		node.PolicyGroup = activePolicyGroup
		// chef-client sets chef_environment to the policy group for policyfile nodes.
		node.Environment = activePolicyGroup
	} else {
		node.Environment = chefEnvironment
	}

	if config.RunChefClient {
		if usePolicyfile {
			// --- Policyfile API flow ---
			// Fetch the policyfile document from the Chef Server.
			// This replaces run-list expansion + environment fetch.
			policy, _ = fetchPolicy(nodeClient, nodeName, config.ChefVersion, activePolicyGroup, activePolicyName, requests)

			// Set policyfile-specific automatic attributes on the node so that the
			// node object saved to the server (PUT nodes/<name>) and the data
			// collector run_converge message carry the correct metadata.
			node.AutomaticAttributes["policy_name"] = activePolicyName
			node.AutomaticAttributes["policy_group"] = activePolicyGroup
			node.AutomaticAttributes["chef_environment"] = activePolicyGroup
			node.AutomaticAttributes["policy_revision"] = policy.RevisionID
			node.AutomaticAttributes["roles"] = []string{}
			node.AutomaticAttributes["recipes"] = policyRunListRecipes(policy.RunList)

			// The node's run_list is derived directly from the policy's run_list.
			expandedRunList = policy.RunList

			// Reporting is not supported for policyfile-based chef-client runs.
		} else {
			// --- Traditional roles/environments flow ---
			// Expand run_list (resolving roles through the Chef Server).
			numLists := len(runLists)
			rl := runList
			if numLists > 0 {
				rl = runLists[rand.Intn(numLists)] //nolint:gosec
			}
			expandedRunList = rl.expand(&nodeClient, nodeName, config.ChefVersion, chefEnvironment, requests)
			_, _ = apiRequest(nodeClient, nodeName, config.ChefVersion, "GET", "environments/"+chefEnvironment, nil, nil, nil, requests) //nolint:bodyclose

			// Notify Reporting of run start.
			if config.EnableReporting {
				res, _ := reportingRunStart(nodeClient, nodeName, config.ChefVersion, runUUID, startTime, requests) //nolint:bodyclose
				if res != nil && res.StatusCode == 404 {
					reportingAvailable = false
				}
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
		_, _ = chefAutomateSendMessage(dataCollectorClient, nodeName, runStartBody)
	} else {
		res, err := apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "data-collector", runStartBody, nil, nil, requests) //nolint:bodyclose
		if err != nil {
			if res != nil {
				if res.StatusCode == 404 {
					dataCollectorAvailable = false
				}
			}
		}
	}

	var dlCookbookFileChance = config.DownloadCookbooksScaleFactor
	var doDownload = false
	if config.DownloadCookbooks == "first" && firstRun {
		doDownload = true
		dlCookbookFileChance = 1.0
	}

	if config.RunChefClient {
		if usePolicyfile {
			// --- Policyfile cookbook resolution ---
			// Fetch each cookbook artifact manifest from cookbook_artifacts/<name>/<identifier>.
			// This replaces the POST environments/<env>/cookbook_versions call used by the
			// traditional flow.  File downloads are gated by the same download_cookbooks
			// and download_cookbooks_scale_factor settings as the traditional path.
			resolveAndDownloadPolicyfileCookbooks(
				&nodeClient, nodeName, config.ChefVersion,
				policy, doDownload, dlCookbookFileChance, requests)
		} else {
			// --- Traditional cookbook resolution ---
			// Request resolved cookbook versions from environments/<env>/cookbook_versions.
			ckbks := solveRunListDependencies(&nodeClient, nodeName, config.ChefVersion, chefEnvironment, expandedRunList, requests)
			if doDownload || config.DownloadCookbooks == "always" {
				ckbks.download(&nodeClient, nodeName, config.ChefVersion, dlCookbookFileChance, requests)
			}

			for _, apiGetRequest := range apiGetRequests {
				apiRequest(nodeClient, nodeName, config.ChefVersion, "GET", apiGetRequest, nil, nil, nil, requests) //nolint:bodyclose,errcheck,gosec
			}
		}
	} else {
		expandedRunList = runList.toStringSlice()
	}

	time.Sleep(time.Duration(sleepDuration) * time.Second)

	// For policyfile nodes the node's run_list comes from the policy document.
	// For traditional nodes it comes from the configured run_list.
	if usePolicyfile {
		node.RunList = expandedRunList
	} else {
		node.RunList = runList.toStringSlice()
	}

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
		if rand.Float64() <= config.NodeSaveFrequency { //nolint:gosec
			_, _ = apiRequest(nodeClient, nodeName, config.ChefVersion, "PUT", "nodes/"+nodeName, node, nil, nil, requests) //nolint:bodyclose
		}

		// Reporting is only supported for traditional (non-policyfile) runs.
		if !usePolicyfile && config.EnableReporting && reportingAvailable {
			_, _ = reportingRunStop(nodeClient, nodeName, config.ChefVersion, runUUID, startTime, endTime, runList, requests) //nolint:bodyclose
		}
	}

	// For the data collector run_converge message, policyfile nodes report their
	// run_list as the policy's recipe list (already fully expanded).  Traditional
	// nodes use the config run_list.
	convergeRunList := runList
	if usePolicyfile {
		convergeRunList = parseRunList(expandedRunList)
	}

	// Notify Data Collector of run end
	runStopBody := dataCollectorRunStop(config, node, nodeName, chefServerFQDN, orgName, status, convergeRunList,
		parseRunList(expandedRunList), runUUID, nodeUUID, startTime, endTime, convergeJSON, expandedRunListID)
	if config.DataCollectorURL != "" {
		_, _ = chefAutomateSendMessage(dataCollectorClient, nodeName, runStopBody)
	} else if dataCollectorAvailable {
		_, _ = apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "data-collector", runStopBody, nil, nil, requests) //nolint:bodyclose
	}

	// Send an Update Action that we just ran a CCR and the node updated itself
	ccrAction := newActionRequest(nodeAction)
	ccrAction.SetTask(updateTask)
	ccrAction.EntityName = nodeName
	ccrAction.RequestorName = nodeName
	if config.DataCollectorURL != "" {
		_, _ = chefAutomateSendMessage(dataCollectorClient, ccrAction.String(), ccrAction)
	} else if dataCollectorAvailable {
		_, _ = apiRequest(nodeClient, ccrAction.String(), config.ChefVersion, "POST", "data-collector", ccrAction, nil, nil, requests) //nolint:bodyclose
	}

	// Notify Data Collector of compliance report.
	//
	// The compliance phase fires when:
	//   1. A compliance_status_json_file was loaded (legacy file-gated behavior), OR
	//   2. enable_compliance_phase is true AND this node slot falls within the
	//      compliance_phase_percentage fraction of the pool.
	//
	// This mirrors chef-client's built-in compliance phase (Chef Infra Client 17+)
	// which runs InSpec at the end of every converge and sends an inspec_report
	// message linked to the CCR via run_uuid.
	runCompliancePhase := len(complianceJSON) != 0 ||
		(config.EnableCompliancePhase && config.NumNodes > 0 &&
			float64(nodeNumber) < float64(config.NumNodes)*config.CompliancePhasePercentage)

	if runCompliancePhase {
		compBody := complianceJSON
		if len(compBody) == 0 {
			// No file provided — use a minimal synthetic InSpec report that
			// represents a compliance phase run with no profiles configured.
			compBody = syntheticComplianceReport()
		}
		complianceReportBody := dataCollectorComplianceReport(nodeDetails, reportUUID, runUUID, endTime, compBody)
		if config.DataCollectorURL != "" {
			_, _ = chefAutomateSendMessage(dataCollectorClient, nodeName, complianceReportBody)
		} else {
			_, _ = apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "data-collector", complianceReportBody, nil, nil, requests) //nolint:bodyclose
		}
	}
}
