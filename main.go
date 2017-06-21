package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/go-chef/chef"
)

// AppVersion - Application Version
const AppVersion = "0.5.0"

func main() {
	fConfig := flag.String("config", "", "Configuration file to load")
	fHelp := flag.Bool("help", false, "Print this help")
	fNodeNamePrefix := flag.String("prefix", "", "This prefix will go at the beginning of each node name")
	fRunsPerMinute := flag.String("rpm", "", "The number of Chef Client runs to make per minute")
	fInterval := flag.String("interval", "", "Interval between a node's chef-client runs, in minutes")
	fSampleConfig := flag.Bool("sample-config", false, "Print out full sample configuration")
	fVersion := flag.Bool("version", false, "Print chef-load version")
	flag.Parse()

	if *fHelp {
		fmt.Println("Usage of chef-load:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *fVersion {
		fmt.Println("chef-load", AppVersion)
		os.Exit(0)
	}

	if *fSampleConfig {
		printSampleConfig()
		os.Exit(0)
	}

	var (
		config *chefLoadConfig
		err    error
	)

	if *fConfig != "" {
		config, err = loadConfig(*fConfig)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("Usage of chef-load:")
		flag.PrintDefaults()
		return
	}

	if *fRunsPerMinute != "" {
		config.RunsPerMinute, _ = strconv.Atoi(*fRunsPerMinute)
	}

	if *fInterval != "" {
		config.Interval, _ = strconv.Atoi(*fInterval)
	}

	if *fNodeNamePrefix != "" {
		config.NodeNamePrefix = *fNodeNamePrefix
	}

	if config.ChefServerURL != "" {
		config.RunChefClient = true
		if config.ClientName == "" || config.ClientKey == "" {
			fmt.Println("ERROR: You must set client_name and client_key if chef_server_url is set")
			os.Exit(1)
		}
	}

	if config.DataCollectorURL != "" {
		config.RunDataCollector = true
	}

	if !config.RunChefClient && !config.RunDataCollector {
		fmt.Println("ERROR: You must set chef_server_url or data_collector_url or both")
		os.Exit(1)
	}

	if config.RunDataCollector && config.ChefServerURL == "" {
		// make sure config.ChefServerURL is set to something because it is used
		// even when only in data-collector mode
		config.ChefServerURL = "https://chef.example.com/organizations/demo/"
	}

	if config.RunChefClient {
		// Early exit if we can't read the client_key, avoiding a messy stacktrace
		f, err := os.Open(config.ClientKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not read client key %v:\n\t%v\n", config.ClientKey, err)
			os.Exit(1)
		}
		f.Close()
	}

	var nodeClient chef.Client
	if config.RunChefClient {
		nodeClient = getAPIClient(config.ClientName, config.ClientKey, config.ChefServerURL)
	}

	ohaiJSON := map[string]interface{}{}
	if config.OhaiJSONFile != "" {
		ohaiJSON = parseJSONFile(config.OhaiJSONFile)
	}

	convergeJSON := map[string]interface{}{}
	if config.ConvergeStatusJSONFile != "" {
		convergeJSON = parseJSONFile(config.ConvergeStatusJSONFile)
	}

	complianceJSON := map[string]interface{}{}
	if config.ComplianceStatusJSONFile != "" {
		complianceJSON = parseJSONFile(config.ComplianceStatusJSONFile)
	}

	var getCookbooks bool
	if config.DownloadCookbooks == "never" {
		getCookbooks = false
	} else {
		getCookbooks = true
	}

	numNodes := config.RunsPerMinute * config.Interval
	delayBetweenNodes := time.Duration(math.Ceil(float64(time.Minute/time.Nanosecond)/float64(config.RunsPerMinute))) * time.Nanosecond
	for {
		for i := 1; i <= numNodes; i++ {
			nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
			go chefClientRun(nodeClient, nodeName, getCookbooks, ohaiJSON, convergeJSON, complianceJSON, *config)
			time.Sleep(delayBetweenNodes)
		}
		if config.DownloadCookbooks == "first" {
			getCookbooks = false
		}
	}
}
