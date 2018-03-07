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
	"errors"
	"fmt"
	"math/rand"
	"time"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// ActionType will be our enum to identity a list of actions types
type ActionType int

// Supported Action Types (EntityType)
const (
	nodeAction ActionType = iota
	cookbookAction
	dataBagAction
	environmentAction
	roleAction
	policyfileAction
	// TODO: (@afiune) Add latter when compliance joins the pool party
	//profileAction
	//scanJobsAction
)

// Strings of the supported Action Type list above
var actionTypeString = map[ActionType]string{
	nodeAction:        "node",
	cookbookAction:    "cookbook",
	dataBagAction:     "bag",
	environmentAction: "environment",
	roleAction:        "role",
	policyfileAction:  "policyfile",

	// In the RFC we have more action/entity types
	// https://github.com/chef/chef-rfc/blob/master/rfc077-mode-agnostic-data-collection.md#action-schema
	// TODO: (@afiune) How do we handle those?
	//"user",
	//"permission",
	//"item",
	//"group",
	//"client",

	// TODO: (@afiune) Add latter when compliance joins the pool party
	//profileAction:     "profile",
	//scanJobsAction:    "scanjobs",
}

// Task will be our enum to identity a list of tasks
type Task int

// Supported Tasks
const (
	createTask Task = iota
	updateTask
	deleteTask
)

// Strings of the supported Tasks list above
var tasksString = map[Task]string{
	createTask: "create",
	updateTask: "update",
	deleteTask: "delete",
}

type actionRequest struct {
	ID               uuid.UUID   `json:"id"`
	MessageType      string      `json:"message_type"`
	MessageVersion   string      `json:"message_version"`
	EntityType       string      `json:"entity_type"`
	actionType       ActionType  `json:"-"`
	EntityName       string      `json:"entity_name"`
	Task             string      `json:"task"`
	OrganizationName string      `json:"organization_name"`
	ServiceHostname  string      `json:"service_hostname"`
	RecordedAt       time.Time   `json:"recorded_at"`
	RemoteHostname   string      `json:"remote_hostname"`
	RequestID        string      `json:"request_id"`
	RequestorName    string      `json:"requestor_name"`
	RequestorType    string      `json:"requestor_type"`
	UserAgent        string      `json:"user_agent"`
	RemoteRequestID  string      `json:"remote_request_id,omitempty"`
	Data             interface{} `json:"data"`
}

func defaultActionRequest() *actionRequest {
	ID, _ := uuid.NewV4()
	return &actionRequest{
		ID:               ID,
		MessageType:      "action",
		MessageVersion:   "0.1.0",
		actionType:       nodeAction,
		EntityType:       actionTypeString[nodeAction],
		EntityName:       "",
		Task:             "",
		OrganizationName: "_default",
		ServiceHostname:  "",
		RecordedAt:       time.Now(),
		RemoteHostname:   "",
		RequestID:        "",
		RequestorName:    "",
		RequestorType:    "chef-load",
		UserAgent:        "chef-load-4.0.0", // Create a version?
		RemoteRequestID:  "",
		Data:             map[string]string{},
	}
}

func newActionRequest(aType ActionType) *actionRequest {
	a := defaultActionRequest()
	a.SetEntityType(aType)
	return a
}

func newRandomActionRequest(aType ActionType) *actionRequest {
	a := newActionRequest(aType)
	a.randomize()
	return a
}

func randomActionType() ActionType {
	return ActionType(rand.Intn(len(actionTypeString)))
}

func randomTask() Task {
	return Task(rand.Intn(len(tasksString)))
}

func (ar *actionRequest) SetTask(t Task) {
	ar.Task = tasksString[t]
}

func (ar *actionRequest) SetEntityType(t ActionType) {
	ar.actionType = t
	ar.EntityType = actionTypeString[t]
}

func randomEntityName() string {
	return entityNameList[rand.Intn(len(entityNameList))]
}

func randomRequestorName() string {
	return requestorNameList[rand.Intn(len(requestorNameList))]
}

func randomTime() time.Time {
	return time.Now().AddDate(0, 0, rand.Intn(7)*-1)
}

// This function will randomize the Chef Action instance depending on the action type
func (ar *actionRequest) randomize() {
	ar.SetTask(randomTask())
	ar.EntityName = randomEntityName()
	ar.RequestorName = randomRequestorName()
	ar.ServiceHostname = getRandom("source_fqdn")
	ar.OrganizationName = getRandom("organization")
	ar.RecordedAt = randomTime()

	// Custom settings for specific actions
	//
	// We might use this if we have to customize specific fields for each action type
	switch ar.actionType {
	case nodeAction:
	case cookbookAction:
	case dataBagAction:
	case environmentAction:
	case roleAction:
	case policyfileAction:
	// TODO: (@afiune) Add latter when compliance joins the pool party
	//case profileAction:
	//case scanJobsAction:
	default:
	}
}

func (ar *actionRequest) String() string {
	return fmt.Sprintf("%s::%s", ar.EntityType, ar.Task)
}

func GenerateChefActions(config *Config, requests chan *request) error {
	log.WithFields(log.Fields{
		"actions":     config.NumActions,
		"random_data": config.RandomData,
	}).Info("Generating chef actions")

	rand.Seed(time.Now().UTC().UnixNano())

	dataCollectorClient, err := NewDataCollectorClient(&DataCollectorConfig{
		Token:   config.DataCollectorToken,
		URL:     config.DataCollectorURL,
		SkipSSL: true,
	}, requests)
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating DataCollectorClient: %+v \n", err))
	}

	for i := 1; i <= config.NumActions; i++ {
		// TODO: Check the errors
		chefAction(config, randomActionType(), dataCollectorClient)
	}
	return nil
}

func chefAction(config *Config, aType ActionType, dataCollectorClient *DataCollectorClient) error {
	action := newRandomActionRequest(aType)
	return chefAutomateSendMessage(dataCollectorClient, action.String(), action)
}
