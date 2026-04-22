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
	"sync"
	"sync/atomic"
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
	// activeCount tracks the number of active CCR runner slots. It is read by
	// the SIGUSR1 handler and written by the burst/scale-down goroutines.
	var activeCount int32

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
				if sig == syscall.SIGUSR1 {
					log.Printf("Active node pool size: %d", atomic.LoadInt32(&activeCount))
				}
				printAPIRequestProfile(startTime, requestAggregator)
				if sig != syscall.SIGUSR1 {
					log.Info("Stopping chef-load")
					os.Exit(0)
				}
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

	// -----------------------------------------------------------------------
	// Node pool initialisation (with optional pre-creation)
	// -----------------------------------------------------------------------

	var (
		nodesMu       sync.Mutex
		ccrCompletion = make(chan int, config.NumNodes)
		nodeNameIdx   = config.NumNodes // new names start beyond the pre-created range
		nodes         = make([]runner, config.NumNodes)
		fleetRecords  []NodeRecord // saved for elastic scale-down replacement
	)

	if config.PrecreateNodes && config.RunChefClient {
		records, err := LoadOrPrecreate(config, requests)
		if err != nil {
			log.WithField("error", err).Warn("Pre-creation completed with errors; continuing with available records")
		}
		fleetRecords = records

		for i := 0; i < config.NumNodes; i++ {
			if i < len(records) {
				nodes[i] = runner{NodeName: records[i].NodeName, FirstRun: false}
			} else {
				// Partial pre-creation: fall back to generated names.
				nodes[i] = runner{NodeName: config.NodeNamePrefix + "-" + strconv.Itoa(i), FirstRun: false}
			}
		}
		if len(records) > config.NumNodes {
			nodeNameIdx = len(records)
		}
	} else {
		// Original behaviour: generate names sequentially, all FirstRun=true.
		nodeNameIdx = 0
		for i := 0; i < config.NumNodes; i++ {
			nodes[i] = runner{NodeName: config.NodeNamePrefix + "-" + strconv.Itoa(nodeNameIdx), FirstRun: true}
			nodeNameIdx++
		}
	}

	atomic.StoreInt32(&activeCount, int32(config.NumNodes))

	// Trigger the first CCR for every node slot.
	for i := 0; i < config.NumNodes; i++ {
		ccrCompletion <- i
	}

	// -----------------------------------------------------------------------
	// Realistic trickle goroutine
	//
	// When precreate_nodes is true and new_node_creates_per_minute > 0 (and
	// elastic_burst_size == 0), periodically swap a random CCR slot for a
	// brand-new node, simulating steady fleet provisioning at ~N nodes/min.
	// -----------------------------------------------------------------------
	if config.PrecreateNodes && config.NewNodeCreatesPerMinute > 0 && config.ElasticBurstSize == 0 {
		tickInterval := time.Duration(float64(time.Minute) / float64(config.NewNodeCreatesPerMinute))
		go func() {
			ticker := time.NewTicker(tickInterval)
			defer ticker.Stop()
			for range ticker.C {
				nodesMu.Lock()
				n := rand.Intn(config.NumNodes)
				newName := config.NodeNamePrefix + "-" + strconv.Itoa(nodeNameIdx)
				rec := NewNodeRecord(newName, nodeNameIdx)
				nodeNameIdx++
				nodes[n] = runner{NodeName: newName, FirstRun: true}
				nodesMu.Unlock()

				go func(r NodeRecord) {
					if appendErr := AppendNodeLog(config.NodeLogFile, []NodeRecord{r}); appendErr != nil {
						log.WithField("error", appendErr).Warn("Could not append trickle node to log")
					}
				}(rec)
			}
		}()
	}

	// -----------------------------------------------------------------------
	// Elastic burst goroutine
	//
	// When elastic_burst_size > 0, spin up a batch of new nodes on each tick.
	// Respects elastic_max_nodes cap and optionally scales down after 2×
	// burst interval when elastic_scale_down is true.
	// -----------------------------------------------------------------------
	if config.ElasticBurstSize > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(config.ElasticBurstInterval) * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				burstSize := config.ElasticBurstSize

				if config.ElasticMaxNodes > 0 {
					current := int(atomic.LoadInt32(&activeCount))
					remaining := config.ElasticMaxNodes - current
					if remaining <= 0 {
						log.Printf("Elastic burst: pool at cap (%d/%d), skipping burst",
							current, config.ElasticMaxNodes)
						continue
					}
					if burstSize > remaining {
						burstSize = remaining
					}
				}

				nodesMu.Lock()
				burstSlots := make([]int, burstSize)
				newRecords := make([]NodeRecord, burstSize)
				for j := 0; j < burstSize; j++ {
					newName := config.NodeNamePrefix + "-" + strconv.Itoa(nodeNameIdx)
					newRecords[j] = NewNodeRecord(newName, nodeNameIdx)
					nodeNameIdx++
					n := rand.Intn(len(nodes))
					burstSlots[j] = n
					nodes[n] = runner{NodeName: newName, FirstRun: true}
				}
				newActive := atomic.AddInt32(&activeCount, int32(burstSize))
				nodesMu.Unlock()

				log.Printf("Elastic burst: +%d nodes, active pool: %d", burstSize, newActive)

				go func(recs []NodeRecord) {
					if appendErr := AppendNodeLog(config.NodeLogFile, recs); appendErr != nil {
						log.WithField("error", appendErr).Warn("Could not append burst nodes to log")
					}
				}(newRecords)

				if config.ElasticScaleDown {
					retireDelay := time.Duration(config.ElasticBurstInterval*2) * time.Minute
					retiredNames := make([]string, len(newRecords))
					for k, r := range newRecords {
						retiredNames[k] = r.NodeName
					}
					go func(slots []int, names []string) {
						time.Sleep(retireDelay)
						nodesMu.Lock()
						fleetLen := len(fleetRecords)
						for _, slot := range slots {
							if fleetLen > 0 {
								// Replace burst slot with a fleet node so the slot
								// continues to run CCRs, simulating scale-down where
								// replacement capacity already exists in the fleet.
								nodes[slot] = runner{
									NodeName: fleetRecords[rand.Intn(fleetLen)].NodeName,
									FirstRun: false,
								}
							}
						}
						newActive := atomic.AddInt32(&activeCount, -int32(len(slots)))
						nodesMu.Unlock()
						log.Printf("Elastic scale-down: -%d nodes, active pool: %d",
							len(slots), newActive)
						go func() {
							if markErr := MarkRetired(config.NodeLogFile, names); markErr != nil {
								log.WithField("error", markErr).Warn("Could not mark nodes retired in log")
							}
						}()
					}(burstSlots, retiredNames)
				}
			}
		}()
	}

	// -----------------------------------------------------------------------
	// Main CCR loop
	// -----------------------------------------------------------------------
	var timeout = false
	for {
		if !timeout {
			time.Sleep(delayBetweenConverges)
		}
		select {
		case n := <-ccrCompletion:
			timeout = false
			nodesMu.Lock()
			if !config.SkipClientCreation && rand.Float64() < config.NodeReplacementRate {
				nodes[n] = runner{NodeName: config.NodeNamePrefix + "-" + strconv.Itoa(nodeNameIdx), FirstRun: true}
				nodeNameIdx++
			}
			nodeName := nodes[n].NodeName
			firstRun := nodes[n].FirstRun
			nodes[n].FirstRun = false
			nodesMu.Unlock()
			go ChefClientRun(config, nodeName, firstRun, requests, ccrCompletion, uint32(n))
		case <-time.After(time.Millisecond * 100):
			fmt.Println("All clients busy, waiting for one to complete before next run. Server may be responding slowly")
			timeout = true
		}
	}
}
