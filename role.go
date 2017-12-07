//
// Copyright:: Copyright 2017 Chef Software, Inc.
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

package main

import (
	"encoding/json"

	"github.com/go-chef/chef"
)

type role struct {
	ChefType           string                     `json:"chef_type"`
	DefaultAttributes  map[string]json.RawMessage `json:"default_attributes"`
	Description        string                     `json:"description"`
	EnvRunLists        map[string][]string        `json:"env_run_lists"`
	JSONClass          string                     `json:"json_class"`
	Name               string                     `json:"name"`
	OverrideAttributes map[string]json.RawMessage `json:"override_attributes"`
	RunList            []string                   `json:"run_list"`
}

func roleRunListFor(nodeClient *chef.Client, nodeName string, roleName, chefEnvironment string) runList {
	var r role
	apiRequest(*nodeClient, nodeName, "GET", "roles/"+roleName, nil, &r, nil)

	var roleRunList runList
	envRunList, envRunListExists := r.EnvRunLists[chefEnvironment]
	if chefEnvironment != "_default" && envRunListExists {
		roleRunList = parseRunList(envRunList)
	} else {
		roleRunList = parseRunList(r.RunList)
	}
	return roleRunList
}
