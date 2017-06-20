package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-chef/chef"
)

func startNode(nodeName string, config chefLoadConfig) {
	if config.Splay > 0 {
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		splay := r1.Intn(config.Splay)
		fmt.Printf("%v Sleeping %v seconds\n", nodeName, splay)
		time.Sleep(time.Duration(splay) * time.Second)
	}

	var nodeClient chef.Client
	if config.Mode == "chef-client" {
		nodeClient = getAPIClient(config.ClientName, config.ClientKey, config.ChefServerURL)
	}

	ohaiJSON := map[string]interface{}{}
	if config.OhaiJSONFile != "" {
		ohaiJSON = parseJSONFile(config.OhaiJSONFile)
	}

	resourcesJSON := []interface{}{}
	if config.ConvergeStatusJSONFile != "" {
		convergeStatusJSON := map[string]interface{}{}
		convergeStatusJSON = parseJSONFile(config.ConvergeStatusJSONFile)
		resourcesJSON = convergeStatusJSON["resources"].([]interface{})
	}

	complianceJSON := map[string]interface{}{}
	if config.ComplianceStatusJSONFile != "" {
		complianceJSON = parseJSONFile(config.ComplianceStatusJSONFile)
	}

	switch config.Runs {
	case 0:
		for run := 1; true; run++ {
			manageChefClientRun(nodeName, config, nodeClient, ohaiJSON, resourcesJSON, complianceJSON, run)
		}
	default:
		for run := 1; run <= config.Runs; run++ {
			manageChefClientRun(nodeName, config, nodeClient, ohaiJSON, resourcesJSON, complianceJSON, run)
		}
	}
	quit <- 1
}

func manageChefClientRun(nodeName string, config chefLoadConfig, nodeClient chef.Client, ohaiJSON map[string]interface{}, resourcesJSON []interface{}, complianceJSON map[string]interface{}, run int) {
	fmt.Println(nodeName, "Started")
	var getCookbooks bool
	switch config.DownloadCookbooks {
	case "always":
		getCookbooks = true
	case "first":
		if run == 1 {
			getCookbooks = true
		} else {
			getCookbooks = false
		}
	case "never":
		getCookbooks = false
	default:
		getCookbooks = true
	}
	chefClientRun(nodeClient, nodeName, getCookbooks, ohaiJSON, resourcesJSON, complianceJSON, config)
	fmt.Println(nodeName, "Finished")
	if config.Runs == 0 || (config.Runs > 1 && run < config.Runs) {
		splay := 0
		if config.Splay > 0 {
			s1 := rand.NewSource(time.Now().UnixNano())
			r1 := rand.New(s1)
			splay = r1.Intn(config.Splay)
		}
		delay := config.Interval + splay
		fmt.Printf("%v Sleeping %v seconds\n", nodeName, delay)
		time.Sleep(time.Duration(delay) * time.Second)
	}
}
