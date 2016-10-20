package main

import (
	"encoding/json"
	"fmt"

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

func roleRunListFor(nodeClient *chef.Client, roleName, chefEnvironment string) runList {
	req, _ := nodeClient.NewRequest("GET", "roles/"+roleName, nil)
	res, err := nodeClient.Do(req, nil)
	if err != nil {
		// can't print res here if it is nil
		// fmt.Println(res.StatusCode)
		// TODO: should this be handled better than just skipping over it?
		fmt.Println(err)
	}
	defer res.Body.Close()

	var r role
	json.NewDecoder(res.Body).Decode(&r)

	var roleRunList runList
	if chefEnvironment == "_default" {
		roleRunList = parseRunList(r.RunList)
	} else {
		roleRunList = parseRunList(r.EnvRunLists[chefEnvironment])
	}
	return roleRunList
}
