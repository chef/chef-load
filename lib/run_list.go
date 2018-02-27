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
	"regexp"

	"github.com/go-chef/chef"
)

type runListItem struct {
	name     string
	itemType string
	version  string
}

type runList []runListItem

func (rl runList) append(rli runListItem) runList {
	return append(rl, rli)
}

func (rl runList) length() int {
	return len(rl)
}

func (rl runList) shift() (runListItem, runList) {
	rli := rl[0]
	rl = rl[1:]
	return rli, rl
}

func (rl runList) toStringSlice() []string {
	var stringSlice []string
	for _, rli := range rl {
		if rli.version == "" {
			stringSlice = append(stringSlice, rli.itemType+"["+rli.name+"]")
		} else {
			stringSlice = append(stringSlice, rli.itemType+"["+rli.name+"@"+rli.version+"]")
		}
	}
	return stringSlice
}

func (rl runList) expand(nodeClient *chef.Client, nodeName string, chefEnvironment string) []string {
	recipes := []string{}
	appliedRoles := map[string]bool{}
	expandRunList(nodeClient, nodeName, rl, &recipes, &appliedRoles, chefEnvironment)
	return recipes
}

func expandRunList(nodeClient *chef.Client, nodeName string, rl runList, recipesPtr *[]string, appliedRolesPtr *map[string]bool, chefEnvironment string) {
	var entry runListItem
	if rl.length() > 0 {
		entry, rl = rl.shift()
		switch entry.itemType {
		case "recipe":
			var recipe string
			if entry.version == "" {
				recipe = entry.name
			} else {
				recipe = entry.name + "@" + entry.version
			}
			*recipesPtr = append(*recipesPtr, recipe)
		case "role":
			if !(*appliedRolesPtr)[entry.name] {
				(*appliedRolesPtr)[entry.name] = true
				roleRunList := roleRunListFor(nodeClient, nodeName, entry.name, chefEnvironment)
				expandRunList(nodeClient, nodeName, roleRunList, recipesPtr, appliedRolesPtr, chefEnvironment)
			}
		}
		expandRunList(nodeClient, nodeName, rl, recipesPtr, appliedRolesPtr, chefEnvironment)
	}
}

func solveRunListDependencies(nodeClient *chef.Client, nodeName string, expandedRunList []string, chefEnvironment string) cookbooks {
	body := map[string][]string{"run_list": expandedRunList}

	var ckbks cookbooks
	apiRequest(*nodeClient, nodeName, "POST", "environments/"+chefEnvironment+"/cookbook_versions", body, &ckbks, nil)
	return ckbks
}

func parseRunList(unparsedRunList []string) runList {
	var qualifiedRecipeRegExp = regexp.MustCompile(`^recipe\[([^\]@]+)(@([0-9]+(\.[0-9]+){1,2}))?\]$`)
	var qualifiedRoleRegExp = regexp.MustCompile(`^role\[([^\]]+)\]$`)
	var unqualifiedRecipeRegExp = regexp.MustCompile(`^([^@]+)(@([0-9]+(\.[0-9]+){1,2}))?$`)

	rl := runList{}
	for _, item := range unparsedRunList {
		match := qualifiedRecipeRegExp.FindStringSubmatch(item)
		if len(match) > 0 {
			// recipe[recipe_name]
			// recipe[recipe_name@1.0.0]
			rli := runListItem{name: match[1], itemType: "recipe", version: match[3]}
			rl = rl.append(rli)
			continue
		}
		match = qualifiedRoleRegExp.FindStringSubmatch(item)
		if len(match) > 0 {
			// role[role_name]
			rli := runListItem{name: match[1], itemType: "role"}
			rl = rl.append(rli)
			continue
		}
		match = unqualifiedRecipeRegExp.FindStringSubmatch(item)
		if len(match) > 0 {
			// recipe_name
			// recipe_name@1.0.0
			rli := runListItem{name: match[1], itemType: "recipe", version: match[3]}
			rl = rl.append(rli)
			continue
		}
	}
	return rl
}
