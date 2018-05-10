//
// Copyright:: Copyright 2018 Chef Software, Inc.
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
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type LivenessRequest struct {
	ChefServerFQDN   string    `json:"chef_server_fqdn"`
	Source           string    `json:"source"`
	MessageVersion   string    `json:"message_version"`
	EventType        string    `json:"event_type"`
	OrganizationName string    `json:"organization_name"`
	NodeName         string    `json:"node_name"`
	EntityUUID       uuid.UUID `json:"entity_uuid"`
	Timestamp        time.Time `json:"@timestamp"`
}

func newLivenessPingRequest(nodeName, chefServerFQDN, chefServerOrg string) *LivenessRequest {
	return &LivenessRequest{
		Timestamp:        time.Now(),
		Source:           "liveness_agent",
		EventType:        "node_ping",
		MessageVersion:   "0.0.1",
		ChefServerFQDN:   chefServerFQDN,
		OrganizationName: chefServerOrg,
		NodeName:         nodeName,
		EntityUUID:       uuid.NewV3(uuid.NamespaceDNS, nodeName),
	}
}

func (lr *LivenessRequest) String() string {
	return fmt.Sprintf("%s::%s", lr.EventType, lr.NodeName)
}

func GenerateLivenessData(config *Config, requests chan *request) error {
	log.WithFields(log.Fields{
		"nodes": config.NumNodes,
	}).Info("Generating liveness agent data")

	rand.Seed(time.Now().UTC().UnixNano())

	dataCollectorClient, err := NewDataCollectorClient(&DataCollectorConfig{
		Token:   config.DataCollectorToken,
		URL:     config.DataCollectorURL,
		SkipSSL: true,
	}, requests)
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating DataCollectorClient: %+v \n", err))
	}

	chefServerURL, err := url.ParseRequestURI(config.ChefServerURL)
	if err != nil {
		return errors.New(fmt.Sprintf("Error parsing ChefServer URL: %+v \n", err))
	}

	for i := 1; i <= config.NumNodes; i++ {
		nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
		livenessPing(nodeName, chefServerURL, dataCollectorClient)
	}
	return nil
}

func livenessPing(nodeName string, chefServerURL *url.URL, dataCollectorClient *DataCollectorClient) (int, error) {
	var (
		chefServerFQDN = chefServerURL.Host
		chefServerOrg  = strings.Split(chefServerURL.Path, "/")[2]
	)
	lvPing := newLivenessPingRequest(nodeName, chefServerFQDN, chefServerOrg)
	return chefAutomateSendMessage(dataCollectorClient, lvPing.String(), lvPing)
}
