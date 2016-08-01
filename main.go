package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
)

const AppVersion = "0.3.0" // Application Version

var quit = make(chan int)

func main() {
	fConfig := flag.String("config", "", "Configuration file to load")
	fDataCollector := flag.Bool("data-collector", false, "Load test the Chef Automate Data Collector")
	fHelp := flag.Bool("help", false, "Print this help")
	fNodes := flag.String("nodes", "", "Number of nodes making chef-client runs")
	fRuns := flag.String("runs", "", "Number of chef-client runs each node should make, 0 value will make infinite runs")
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

	if *fNodes != "" {
		config.Nodes, _ = strconv.Atoi(*fNodes)
	}

	if *fRuns != "" {
		config.Runs, _ = strconv.Atoi(*fRuns)
	}

	// Early exit if we can't read the client_key, avoiding a messy stacktrace
	f, err := os.Open(config.ClientKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read client key %v:\n\t%v\n", config.ClientKey, err)
		os.Exit(1)
	}
	f.Close()

	// If the data collector is enabled, validate the we've configured the URL and Token
	if *fDataCollector {
		_, err := url.ParseRequestURI(config.DataCollectorUrl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "data_collector_url is not a valid URL %v:\n\t%v\n", config.DataCollectorUrl, err)
			os.Exit(1)
		}

		if len(config.DataCollectorToken) == 0 {
			fmt.Fprintf(os.Stderr, "data_collector_token is not a valid token: %v\n", config.DataCollectorToken)
			os.Exit(1)
		}
	}

	sem := make(chan int, config.BootstrapNodesConcurrency)

	numNodes := config.Nodes
	for i := 0; i < numNodes; i++ {
		nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
		go setupChefLoad(nodeName, *config, sem)
	}
	for i := 0; i < numNodes; i++ {
		<-quit // Wait to be told to exit.
	}
	for i := 0; i < numNodes; i++ {
		nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
		go startNode(nodeName, *config)
	}
	for i := 0; i < numNodes; i++ {
		<-quit // Wait to be told to exit.
	}
}
