//
// Copyright:: Copyright 2017-2018 Chef Software, Inc.
// License:: Apache License, Version 2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package chef_load

import (
	"math"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chef/chef"
	log "github.com/sirupsen/logrus"
)

type UTCFormatter struct {
	log.Formatter
}

func (u UTCFormatter) Format(e *log.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return u.Formatter.Format(e)
}

type request struct {
	Method     string `json:"method"`
	Url        string `json:"url"`
	StatusCode int    `json:"status_code"`
}

var (
	logger   = log.New()
	requests = make(chan *request)
)

const DateTimeFormat = "2006-01-02T15:04:05Z"

func Start(config *Config) {
	var (
		nodeClient       chef.Client
		ohaiJSON         = map[string]interface{}{}
		convergeJSON     = map[string]interface{}{}
		complianceJSON   = map[string]interface{}{}
		amountOfRequests = make(amountOfRequests)
		firstRun         = true
	)

	if config.RunChefClient {
		nodeClient = getAPIClient(config.ClientName, config.ClientKey, config.ChefServerURL)
	}

	if config.OhaiJSONFile != "" {
		ohaiJSON = parseJSONFile(config.OhaiJSONFile)
	}
	if config.ConvergeStatusJSONFile != "" {
		convergeJSON = parseJSONFile(config.ConvergeStatusJSONFile)
	}

	if config.ComplianceStatusJSONFile != "" {
		complianceJSON = parseJSONFile(config.ComplianceStatusJSONFile)
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

	log.WithFields(log.Fields{
		"nodes":   config.NumNodes,
		"minutes": config.Interval,
		"log":     config.LogFile,
	}).Info("Starting chef-load")

	delayBetweenNodes := time.Duration(math.Ceil(float64(time.Duration(config.Interval)*(time.Minute/time.Nanosecond))/float64(config.NumNodes))) * time.Nanosecond

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

		for {
			select {
			case req := <-requests:
				amountOfRequests.addRequest(request{Method: req.Method, Url: req.Url, StatusCode: req.StatusCode})
			case sig := <-sigs:
				log.WithFields(log.Fields{"syscall": sig}).Info("Signal received")
				printAPIRequestProfile(amountOfRequests)
				log.Info("Stopping chef-load")
				os.Exit(0)
			}
		}
	}()

	for {
		for i := 1; i <= config.NumNodes; i++ {
			nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
			go ChefClientRun(config, nodeClient, nodeName, firstRun, ohaiJSON, convergeJSON, complianceJSON)
			time.Sleep(delayBetweenNodes)
		}
		firstRun = false
	}
}
