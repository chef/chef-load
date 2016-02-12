package main

import (
	"encoding/json"
	"fmt"
	"time"
	"os"

	"github.com/go-chef/chef"
)


func newChefNode(nodeName, nodeJsonFile string) (node chef.Node) {
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
	if nodeJsonFile != "" {
		file, err := os.Open(nodeJsonFile)
		if err != nil {
			fmt.Println("Couldn't open node JSON file ", nodeJsonFile, ": ", err)
			return
		}
		defer file.Close()

		var json_node chef.Node

		err = json.NewDecoder(file).Decode(&json_node)
		if err != nil {
			fmt.Println("Couldn't decode node JSON file ", nodeJsonFile, ": ", err)
			return
		}

		node.AutomaticAttributes 	= json_node.AutomaticAttributes
		node.NormalAttributes 		= json_node.NormalAttributes
		node.DefaultAttributes 		= json_node.DefaultAttributes
		node.OverrideAttributes 	= json_node.OverrideAttributes
	}

	return
}

func chefClientRun(nodeClient chef.Client, nodeName string, nodeJsonFile string, runList []string, getCookbooks bool, apiGetRequests []string, sleepDuration int) {
	node, err := nodeClient.Nodes.Get(nodeName)
	fmt.Println(node)
	if err != nil {
		statusCode := getStatusCode(err)
		if statusCode == 404 {
			// Create a Node object
			// TODO: should have a constructor for this
			node = newChefNode(nodeName, nodeJsonFile)

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

	_, err = nodeClient.Nodes.Put(node)
	if err != nil {
		fmt.Println("Couldn't update node: ", err)
	}
}
