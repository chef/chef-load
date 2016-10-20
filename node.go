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
	chefClientRun(nodeClient, nodeName, config.OhaiJsonFile, config.ChefEnvironment, config.RunList, getCookbooks, config.ApiGetRequests, config.SleepDuration)
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
