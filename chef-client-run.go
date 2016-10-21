package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-chef/chef"
	uuid "github.com/satori/go.uuid"
)

func newChefNode(nodeName, chefEnvironment, ohaiJsonFile string) (node chef.Node) {
	node = chef.Node{
		Name:                nodeName,
		Environment:         chefEnvironment,
		ChefType:            "node",
		JsonClass:           "Chef::Node",
		RunList:             []string{},
		AutomaticAttributes: map[string]interface{}{},
		NormalAttributes:    map[string]interface{}{},
		DefaultAttributes:   map[string]interface{}{},
		OverrideAttributes:  map[string]interface{}{},
	}
	if ohaiJsonFile != "" {
		file, err := os.Open(ohaiJsonFile)
		if err != nil {
			fmt.Println("Couldn't open ohai JSON file ", ohaiJsonFile, ": ", err)
			return
		}
		defer file.Close()

		ohai_json := map[string]interface{}{}

		err = json.NewDecoder(file).Decode(&ohai_json)
		if err != nil {
			fmt.Println("Couldn't decode ohai JSON file ", ohaiJsonFile, ": ", err)
			return
		}
		node.AutomaticAttributes = ohai_json
	}

	return
}

func chefClientRun(nodeClient chef.Client, nodeName string, getCookbooks bool, config chefLoadConfig) {
	ohaiJsonFile := config.OhaiJsonFile
	chefEnvironment := config.ChefEnvironment
	runList := parseRunList(config.RunList)
	apiGetRequests := config.ApiGetRequests
	sleepDuration := config.SleepDuration
	runUUID := uuid.NewV4()
	startTime := time.Now().UTC()

	node, err := nodeClient.Nodes.Get(nodeName)
	if err != nil {
		statusCode := getStatusCode(err)
		if statusCode == 404 {
			// Create a Node object
			// TODO: should have a constructor for this
			node = newChefNode(nodeName, chefEnvironment, ohaiJsonFile)

			_, err = nodeClient.Nodes.Post(node)
			if err != nil {
				fmt.Println("Couldn't create node. ", err)
			}
		} else {
			fmt.Println("Couldn't get node: ", err)
		}
	}

	nodeClient.Environments.Get(chefEnvironment)

	// Notify Reporting of run start
	reportsStatusCode := reportingRunStart(nodeClient, nodeName, runUUID, startTime)

	// Expand run_list
	expandedRunList := runList.expand(&nodeClient, chefEnvironment)

	// Calculate cookbook dependencies
	ckbks := solveRunListDependencies(&nodeClient, expandedRunList, chefEnvironment)

	// Download cookbooks
	if getCookbooks {
		ckbks.download(&nodeClient)
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

	node.RunList = runList.toStringSlice()

	// Ensure that what we post at the end of the run is different from previous runs
	endTime := time.Now().UTC()
	node.AutomaticAttributes["ohai_time"] = endTime.Unix()

	_, err = nodeClient.Nodes.Put(node)
	if err != nil {
		fmt.Println("Couldn't update node: ", err)
	}

	// Notify Reporting of run end
	if reportsStatusCode == 201 {
		reportingRunStop(nodeClient, nodeName, runUUID, startTime, endTime, runList)
	}
}
