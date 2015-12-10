package main

import (
	"fmt"
	"time"

	"github.com/go-chef/chef"
)

func startNode(nodeName string, config chefLoadConfig) {
	adminClient := getApiClient(config.ClientName, config.ClientKey, config.ChefServerUrl)

	adminClient.Clients.Delete(nodeName)
	adminClient.Nodes.Delete(nodeName)
	createClient(adminClient, nodeName, getPublicKey(config.ClientKey))

	nodeClient := getApiClient(nodeName, config.ClientKey, config.ChefServerUrl)

	switch config.Runs {
	case 0:
		for run := 1; true; run++ {
			manageChefClientRun(nodeName, config, nodeClient, run)
		}
	default:
		for run := 1; run <= config.Runs; run++ {
			manageChefClientRun(nodeName, config, nodeClient, run)
		}
	}
	quit <- 1
}

func manageChefClientRun(nodeName string, config chefLoadConfig, nodeClient chef.Client, run int) {
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
	chefClientRun(nodeClient, nodeName, config.RunList, getCookbooks, config.ApiGetRequests, config.SleepDuration)
	fmt.Println(nodeName, "Finished")
	if config.Runs == 0 || (config.Runs > 1 && run < config.Runs) {
		fmt.Printf("%v Sleeping %v seconds\n", nodeName, config.Interval)
		time.Sleep(time.Duration(config.Interval) * time.Second)
	}
}

func createClient(adminClient chef.Client, clientName, publicKey string) {
	apiClient := chef.ApiClient{
		Name:       clientName,
		ClientName: clientName,
		PublicKey:  publicKey,
		Admin:      false,
		Validator:  false,
	}
	data, err := chef.JSONReader(apiClient)
	if err != nil {
		return
	}
	req, err := adminClient.NewRequest("POST", "clients", data)
	res, err := adminClient.Do(req, nil)
	if err != nil {
		// can't print res here if it is nil
		// fmt.Println(res.StatusCode)
		// TODO: need to handle errors better
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
}
