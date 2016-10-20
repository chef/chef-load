package main

import (
	"encoding/json"
	"fmt"

	"github.com/go-chef/chef"
)

type cookbookFile struct {
	Checksum    string `json:"checksum"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Specificity string `json:"specificity"`
	URL         string `json:"url"`
}

type cookbook struct {
	CookbookName string         `json:"cookbook_name"`
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	Attributes   []cookbookFile `json:"attributes"`
	Definitions  []cookbookFile `json:"definitions"`
	Files        []cookbookFile `json:"files"`
	Libraries    []cookbookFile `json:"libraries"`
	Providers    []cookbookFile `json:"providers"`
	Recipes      []cookbookFile `json:"recipes"`
	Resources    []cookbookFile `json:"resources"`
	RootFiles    []cookbookFile `json:"root_files"`
	Templates    []cookbookFile `json:"templates"`
}

func getCookbookFile(nodeClient *chef.Client, cookbookFile string) {
	req, err := nodeClient.NewRequest("GET", cookbookFile, nil)
	res, err := nodeClient.Do(req, nil)
	if err != nil {
		fmt.Println(err)
		// TODO: need to handle errors better
		return
	}
	defer res.Body.Close()
}

func downloadCookbooks(nodeClient *chef.Client, cookbooks map[string]json.RawMessage) {
	for _, v := range cookbooks {
		var cookbook cookbook
		json.Unmarshal(v, &cookbook)

		for _, cookbookFile := range cookbook.Attributes {
			getCookbookFile(nodeClient, cookbookFile.URL)
		}
		for _, cookbookFile := range cookbook.Definitions {
			getCookbookFile(nodeClient, cookbookFile.URL)
		}
		for _, cookbookFile := range cookbook.Files {
			getCookbookFile(nodeClient, cookbookFile.URL)
		}
		for _, cookbookFile := range cookbook.Libraries {
			getCookbookFile(nodeClient, cookbookFile.URL)
		}
		for _, cookbookFile := range cookbook.Providers {
			getCookbookFile(nodeClient, cookbookFile.URL)
		}
		for _, cookbookFile := range cookbook.Recipes {
			getCookbookFile(nodeClient, cookbookFile.URL)
		}
		for _, cookbookFile := range cookbook.Resources {
			getCookbookFile(nodeClient, cookbookFile.URL)
		}
		for _, cookbookFile := range cookbook.RootFiles {
			getCookbookFile(nodeClient, cookbookFile.URL)
		}
		for _, cookbookFile := range cookbook.Templates {
			getCookbookFile(nodeClient, cookbookFile.URL)
		}
	}
}
