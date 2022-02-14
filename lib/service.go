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
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

type UTCFormatter struct {
	log.Formatter
}

func (u UTCFormatter) Format(e *log.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return u.Formatter.Format(e)
}

type runner struct {
	NodeName string `json:"string"`
	FirstRun bool
}

type request struct {
	Method     string `json:"method"`
	Url        string `json:"url"`
	StatusCode int    `json:"status_code"`
}

var logger = log.New()

const DateTimeFormat = "2006-01-02T15:04:05Z"

func Start(config *Config) {
	var (
		requestAggregator = make(amountOfRequests)
		requests          = make(chan *request)
	)

	logger.Formatter = UTCFormatter{&log.JSONFormatter{}}
	logger.SetNoLock()

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
		"nodes":               config.NumNodes,
		"actions":             config.NumActions,
		"interval":            config.Interval,
		"prefix":              config.NodeNamePrefix,
		"skip-create-clients": config.SkipClientCreation,
	}).Info("Starting chef-load")

	var (
		delayBetweenConverges = time.Duration(math.Ceil(float64(time.Duration(config.Interval)*(time.Minute/time.Nanosecond))/float64(config.NumNodes))) * time.Nanosecond

		// hardcode each node's liveness ping interval to 30 minutes
		delayBetweenLivenessAgentPing = time.Duration(math.Ceil(float64(time.Duration(30)*(time.Minute/time.Nanosecond))/float64(config.NumNodes))) * time.Nanosecond
	)

	log.Printf("Delay between converges = %s\n", delayBetweenConverges)
	var delayBetweenActions time.Duration
	if config.NumActions > 0 {
		delayBetweenActions = time.Duration(math.Ceil(float64(time.Duration(config.Interval)*(time.Minute/time.Nanosecond))/float64(config.NumActions))) * time.Nanosecond
	}

	var startTime = time.Now()
	// This goroutine aggregates API requests and handles and handles interrupt
	// to display a final report.
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

		for {
			select {
			case req := <-requests:
				requestAggregator.addRequest(request{Method: req.Method, Url: req.Url, StatusCode: req.StatusCode})
			case sig := <-sigs:
				log.WithFields(log.Fields{"syscall": sig}).Info("Signal received")
				printAPIRequestProfile(startTime, requestAggregator)
				log.Info("Stopping chef-load")
				os.Exit(0)
			}
		}
	}()

	if config.LivenessAgent {
		// The liveness agent goroutine
		go func() {
			// TODO Check errors!
			var (
				dataCollectorClient, _ = NewDataCollectorClient(&DataCollectorConfig{
					Token:   config.DataCollectorToken,
					URL:     config.DataCollectorURL,
					SkipSSL: true,
				}, requests)

				chefServerURL, _ = url.ParseRequestURI(config.ChefServerURL)
			)

			// Never stop sending liveness ping
			for {
				for i := 1; i <= config.NumNodes; i++ {
					nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
					go livenessPing(nodeName, chefServerURL, dataCollectorClient)
					time.Sleep(delayBetweenLivenessAgentPing)
				}
			}
		}()
	}

	// The Actions goroutine
	if config.DataCollectorURL != "" && config.NumActions > 0 {
		go func() {
			dataCollectorClient, _ := NewDataCollectorClient(&DataCollectorConfig{
				Token:   config.DataCollectorToken,
				URL:     config.DataCollectorURL,
				SkipSSL: true,
			}, requests)

			// Never stop sending actions
			for {
				for i := 1; i <= config.NumActions; i++ {
					go chefAction(config, randomActionType(), dataCollectorClient)
					time.Sleep(delayBetweenActions)
				}
			}
		}()
	}
	// Cleanup: split these sections into their own functions

	// The Nodes (CCRs) goroutine
	// var nodeNames = make([]string, config.NumNodes)
	var ccrCompletion = make(chan int, config.NumNodes)
	// var replaceNode = false
	var nodeNameIdx = 0

	// Create initial group of runs at the scheduled interval
	var nodes = make([]runner, config.NumNodes)
	for i := 0; i < config.NumNodes; i++ {
		nodes[i] = runner{NodeName: config.NodeNamePrefix + "-" + strconv.Itoa(nodeNameIdx), FirstRun: true}
		nodeNameIdx++
		ccrCompletion <- i // trigger the first run for node 'i'
	}

	var timeout = false
	//var lastRunStart = time.Now()
	for {
		if !timeout {
			time.Sleep(delayBetweenConverges)
		}
		select {
		case n := <-ccrCompletion:
			timeout = false
			if rand.Float64() < config.NodeReplacementRate {
				nodes[n] = runner{NodeName: config.NodeNamePrefix + "-" + strconv.Itoa(nodeNameIdx), FirstRun: true}
				nodeNameIdx++
			}
			// confirming that throttle effect ensures we have a maximum of NumNodes concurrent CCRs happening
			// log.Printf("[node %s] starting CCR. Elapsed since most recent CCR on any node: %s", nodes[n].NodeName, time.Since(lastRunStart))
			// lastRunStart = time.Now()
			go ChefClientRun(config, nodes[n].NodeName, nodes[n].FirstRun, requests, ccrCompletion, uint32(n))
			nodes[n].FirstRun = false
		case <-time.After(time.Millisecond * 100):
			fmt.Println("All clients busy, waiting for one to complete before next run. Server may be responding slowly")
			timeout = true
		}
	}
}
