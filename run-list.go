package main

import (
	"fmt"
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

func (rl runList) expand(nodeClient *chef.Client, chefEnvironment string) []string {
	recipes := []string{}
	appliedRoles := map[string]bool{}
	expandRunList(nodeClient, rl, &recipes, &appliedRoles, chefEnvironment)
	return recipes
}

func expandRunList(nodeClient *chef.Client, rl runList, recipesPtr *[]string, appliedRolesPtr *map[string]bool, chefEnvironment string) {
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
				expandRunList(nodeClient, roleRunListFor(nodeClient, entry.name, chefEnvironment), recipesPtr, appliedRolesPtr, chefEnvironment)
			}
		}
		expandRunList(nodeClient, rl, recipesPtr, appliedRolesPtr, chefEnvironment)
	}
}

func solveRunListDependencies(nodeClient *chef.Client, expandedRunList []string, chefEnvironment string) cookbooks {
	body := map[string][]string{"run_list": expandedRunList}

	var ckbks cookbooks
	_, err := apiRequest(*nodeClient, "POST", "environments/"+chefEnvironment+"/cookbook_versions", body, &ckbks, nil)
	if err != nil {
		fmt.Println(err)
	}
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
