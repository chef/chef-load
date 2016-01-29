package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chef/chef"
	"math/rand"
	"time"
)

func chefClientRun(nodeClient chef.Client, nodeName string, runList []string, getCookbooks bool, apiGetRequests []string, sleepDuration int) {
	node, err := nodeClient.Nodes.Get(nodeName)
	if err != nil {
		statusCode := getStatusCode(err)
		if statusCode == 404 {
			// Create a Node object
			// TODO: should have a constructor for this
			node = chef.Node{
				Name:                nodeName,
				Environment:         "_default",
				ChefType:            "node",
				JsonClass:           "Chef::Node",
				RunList:             []string{},
				AutomaticAttributes: map[string]interface{}{},
				NormalAttributes:    map[string]interface{}{},
				DefaultAttributes:   map[string]interface{}{},
				OverrideAttributes:  map[string]interface{}{},
			}

			_, err = nodeClient.Nodes.Post(node)
			if err != nil {
				fmt.Println("Couldn't create node. ", err)
			}
		} else {
			fmt.Println("Couldn't get node: ", err)
		}
	}

	rl := map[string][]string{"run_list": runList}
	data, err := chef.JSONReader(rl)
	if err != nil {
		fmt.Println(err)
	}

	var cookbooks map[string]json.RawMessage

	err = func(cookbooks *map[string]json.RawMessage) error {
		req, err := nodeClient.NewRequest("POST", "environments/_default/cookbook_versions", data)
		res, err := nodeClient.Do(req, nil)
		if err != nil {
			// can't print res here if it is nil
			// fmt.Println(res.StatusCode)
			fmt.Println(err)
			return err
		}
		defer res.Body.Close()

		return json.NewDecoder(res.Body).Decode(cookbooks)
	}(&cookbooks)

	if getCookbooks {
		downloadCookbooks(&nodeClient, cookbooks)
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

	// Ensure that what we post at the end of the run is different from previous runs
	node.AutomaticAttributes["cache-buster"] = fmt.Sprintf("%d-%d-%d-%d",
		rand.Intn(1000),
		rand.Intn(1000),
		rand.Intn(1000),
		rand.Intn(1000))

	_, err = nodeClient.Nodes.Put(node)
	if err != nil {
		fmt.Println("Couldn't update node: ", err)
	}
}
