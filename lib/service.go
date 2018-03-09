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

var logger = log.New()

const DateTimeFormat = "2006-01-02T15:04:05Z"

func Start(config *Config) {
	var (
		numRequests = make(amountOfRequests)
		requests    = make(chan *request)
		firstRun    = true
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
		"nodes":   config.NumNodes,
		"actions": config.NumActions,
		"minutes": config.Interval,
		"log":     config.LogFile,
	}).Info("Starting chef-load")

	delayBetweenNodes := time.Duration(math.Ceil(float64(time.Duration(config.Interval)*(time.Minute/time.Nanosecond))/float64(config.NumNodes))) * time.Nanosecond
	delayBetweenActions := time.Duration(math.Ceil(float64(time.Duration(config.Interval)*(time.Minute/time.Nanosecond))/float64(config.NumActions))) * time.Nanosecond

	// This goroutine is in charge to read requests and write them to disk
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

		for {
			select {
			case req := <-requests:
				numRequests.addRequest(request{Method: req.Method, Url: req.Url, StatusCode: req.StatusCode})
			case sig := <-sigs:
				log.WithFields(log.Fields{"syscall": sig}).Info("Signal received")
				printAPIRequestProfile(numRequests)
				log.Info("Stopping chef-load")
				os.Exit(0)
			}
		}
	}()

	// The Actions goroutine
	go func() {
		dataCollectorClient, _ := NewDataCollectorClient(&DataCollectorConfig{
			Token:   config.DataCollectorToken,
			URL:     config.DataCollectorURL,
			SkipSSL: true,
		}, requests)

		for i := 1; i <= config.NumActions; i++ {
			go chefAction(config, randomActionType(), dataCollectorClient)
			time.Sleep(delayBetweenActions)
		}
	}()

	// The Nodes (CCRs) goroutine
	for {
		for i := 1; i <= config.NumNodes; i++ {
			nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
			go ChefClientRun(config, nodeName, firstRun, requests)
			time.Sleep(delayBetweenNodes)
		}
		firstRun = false
	}
}
