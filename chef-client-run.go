package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-chef/chef"
	"github.com/satori/go.uuid"
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

	run_uuid := uuid.NewV4()
	start_time := timestamp()

	reportsStatusCode := func() int {
		startRunBody := map[string]interface{}{
			"action":     "start",
			"run_id":     run_uuid,
			"start_time": start_time,
		}
		data, err := chef.JSONReader(startRunBody)
		if err != nil {
			fmt.Println(err)
		}

		req, err := nodeClient.NewRequest("POST", "reports/nodes/"+nodeName+"/runs", data)
		req.Header.Set("X-Ops-Reporting-Protocol-Version", "0.1.0")
		res, err := nodeClient.Do(req, nil)
		if err != nil && res.StatusCode != 404 {
			// can't print res here if it is nil
			// fmt.Println(res.StatusCode)
			fmt.Println(err)
			return res.StatusCode
		}
		defer res.Body.Close()
		return res.StatusCode
	}()

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
	node.AutomaticAttributes["ohai_time"] = time.Now().Unix()

	_, err = nodeClient.Nodes.Put(node)
	if err != nil {
		fmt.Println("Couldn't update node: ", err)
	}

	if reportsStatusCode == 201 {
		err = func() error {
			end_time := timestamp()
			endRunBody := map[string]interface{}{
				"action":          "end",
				"data":            map[string]interface{}{},
				"end_time":        end_time,
				"resources":       []interface{}{},
				"run_list":        "[]",
				"start_time":      start_time,
				"status":          "success",
				"total_res_count": "0",
			}
			data, err := chef.JSONReader(endRunBody)
			if err != nil {
				fmt.Println(err)
			}

			req, err := nodeClient.NewRequest("POST", "reports/nodes/"+nodeName+"/runs/"+run_uuid.String(), data)
			req.Header.Set("X-Ops-Reporting-Protocol-Version", "0.1.0")
			res, err := nodeClient.Do(req, nil)
			if err != nil {
				// can't print res here if it is nil
				// fmt.Println(res.StatusCode)
				fmt.Println(err)
				return err
			}
			defer res.Body.Close()
			return err
		}()
	}
}
