package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chef/chef"
	log "github.com/sirupsen/logrus"
)

// AppVersion - Application Version
const AppVersion = "3.0.0"

const iso8601DateTime = "2006-01-02T15:04:05Z"

var config *chefLoadConfig

var logFiles []string

type request struct {
	Method     string `json:"method"`
	Url        string `json:"url"`
	StatusCode int    `json:"status_code"`
}

var requests = make(chan *request)

type UTCFormatter struct {
	log.Formatter
}

func (u UTCFormatter) Format(e *log.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return u.Formatter.Format(e)
}

var logger = log.New()

func init() {
	fConfig := flag.String("config", "", "Configuration file to load")
	fHelp := flag.Bool("help", false, "Print this help")
	fNodeNamePrefix := flag.String("prefix", "", "This prefix will go at the beginning of each node name")
	fNumNodes := flag.String("nodes", "", "The number of nodes to simulate")
	fInterval := flag.String("interval", "", "Interval between a node's chef-client runs, in minutes")
	fSampleConfig := flag.Bool("sample-config", false, "Print out full sample configuration")
	fProfileLogs := flag.Bool("profile-logs", false, "Generates API request profile from specified chef-load log files")
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

	if !*fProfileLogs && *fConfig == "" {
		fmt.Println("Usage of chef-load:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *fProfileLogs {
		if len(flag.Args()) == 0 {
			log.Fatal("The -profile-logs option requires chef-load log file(s) to be specified")
		}
		logFiles = flag.Args()
		return
	}

	var err error
	config, err = loadConfig(*fConfig)
	if err != nil {
		log.WithField("error", err).Fatal("Could not load chef-load config file")
	}

	if *fNumNodes != "" {
		config.NumNodes, _ = strconv.Atoi(*fNumNodes)
	}

	if *fInterval != "" {
		config.Interval, _ = strconv.Atoi(*fInterval)
	}

	if *fNodeNamePrefix != "" {
		config.NodeNamePrefix = *fNodeNamePrefix
	}

	if config.ChefServerURL == "" && config.DataCollectorURL == "" {
		log.Fatal("You must set chef_server_url or data_collector_url or both")
	}

	if config.ChefServerURL != "" {
		config.RunChefClient = true
		if !strings.HasSuffix(config.ChefServerURL, "/") {
			config.ChefServerURL = config.ChefServerURL + "/"
		}
		if config.ClientName == "" || config.ClientKey == "" {
			log.Fatal("You must set client_name and client_key if chef_server_url is set")
		}
	}

	if config.DataCollectorURL != "" && config.ChefServerURL == "" {
		// make sure config.ChefServerURL is set to something because it is used
		// even when only in data-collector mode
		config.ChefServerURL = "https://chef.example.com/organizations/demo/"
	}

	logger.Formatter = UTCFormatter{&log.JSONFormatter{}}

	if err := os.MkdirAll(path.Dir(config.LogFile), 0755); err != nil {
		log.WithField("error", err).Fatal("Failed to create directory")
	}
	file, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		logger.Out = file
	} else {
		log.WithField("error", err).Fatal("Failed to log to file")
	}
}

func main() {
	amountOfRequests := make(amountOfRequests)

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

		for {
			select {
			case req := <-requests:
				amountOfRequests.addRequest(request{Method: req.Method, Url: req.Url, StatusCode: req.StatusCode})
			case sig := <-sigs:
				switch sig {
				case syscall.SIGINT, syscall.SIGTERM:
					if sig == syscall.SIGINT {
						fmt.Printf("\n%s Received Signal: INT\n", time.Now().UTC().Format(iso8601DateTime))
					} else {
						fmt.Printf("%s Received Signal: TERM\n", time.Now().UTC().Format(iso8601DateTime))
					}
					printAPIRequestProfile(amountOfRequests)
					fmt.Printf("%s Stopping chef-load\n", time.Now().UTC().Format(iso8601DateTime))
					os.Exit(0)
				case syscall.SIGUSR1:
					fmt.Printf("%s Received Signal: USR1\n", time.Now().UTC().Format(iso8601DateTime))
					printAPIRequestProfile(amountOfRequests)
				}
			}
		}
	}()

	if len(logFiles) != 0 {
		for _, logFile := range logFiles {
			fmt.Printf("%s Reading log file %s\n", time.Now().UTC().Format(iso8601DateTime), logFile)
			file, err := os.Open(logFile)
			defer file.Close()
			if err != nil {
				log.WithField("error", err).Fatalf("Could not read log file %s", logFile)
			}

			// create a new scanner and read the file line by line
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				req := request{}
				json.Unmarshal([]byte(scanner.Text()), &req)
				amountOfRequests.addRequest(request{Method: req.Method, Url: req.Url, StatusCode: req.StatusCode})
			}

			// check for errors
			if err = scanner.Err(); err != nil {
				log.WithField("error", err).Fatalf("Could not read log file %s", logFile)
			}
		}
		printAPIRequestProfile(amountOfRequests)
		os.Exit(0)
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

	fmt.Printf("%s Starting chef-load with %d nodes distributed evenly across a %d minute interval\n", time.Now().UTC().Format(iso8601DateTime), config.NumNodes, config.Interval)
	fmt.Printf("%s All API requests will be logged in %s\n", time.Now().UTC().Format(iso8601DateTime), config.LogFile)
	delayBetweenNodes := time.Duration(math.Ceil(float64(time.Duration(config.Interval)*(time.Minute/time.Nanosecond))/float64(config.NumNodes))) * time.Nanosecond
	firstRun := true
	for {
		for i := 1; i <= config.NumNodes; i++ {
			nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
			go chefClientRun(nodeClient, nodeName, firstRun, ohaiJSON, convergeJSON, complianceJSON)
			time.Sleep(delayBetweenNodes)
		}
		firstRun = false
	}
}
