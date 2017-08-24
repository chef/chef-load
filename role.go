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
	_, err := apiRequest(*nodeClient, nodeName, "GET", "roles/"+roleName, nil, &r, nil)
	if err != nil {
		printError(nodeName, err)
	}

	var roleRunList runList
	envRunList, envRunListExists := r.EnvRunLists[chefEnvironment]
	if chefEnvironment != "_default" && envRunListExists {
		roleRunList = parseRunList(envRunList)
	} else {
		roleRunList = parseRunList(r.RunList)
	}
	return roleRunList
}
