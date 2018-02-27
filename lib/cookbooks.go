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

import "github.com/go-chef/chef"

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

type cookbooks map[string]cookbook

func (ckbkFile cookbookFile) download(nodeClient *chef.Client, nodeName string) {
	apiRequest(*nodeClient, nodeName, "GET", ckbkFile.URL, nil, nil, nil)
}

func (ckbk cookbook) download(nodeClient *chef.Client, nodeName string) {
	for _, ckbkFile := range ckbk.Attributes {
		ckbkFile.download(nodeClient, nodeName)
	}
	for _, ckbkFile := range ckbk.Definitions {
		ckbkFile.download(nodeClient, nodeName)
	}
	for _, ckbkFile := range ckbk.Files {
		ckbkFile.download(nodeClient, nodeName)
	}
	for _, ckbkFile := range ckbk.Libraries {
		ckbkFile.download(nodeClient, nodeName)
	}
	for _, ckbkFile := range ckbk.Providers {
		ckbkFile.download(nodeClient, nodeName)
	}
	for _, ckbkFile := range ckbk.Recipes {
		ckbkFile.download(nodeClient, nodeName)
	}
	for _, ckbkFile := range ckbk.Resources {
		ckbkFile.download(nodeClient, nodeName)
	}
	for _, ckbkFile := range ckbk.RootFiles {
		ckbkFile.download(nodeClient, nodeName)
	}
	for _, ckbkFile := range ckbk.Templates {
		ckbkFile.download(nodeClient, nodeName)
	}
}

func (ckbks cookbooks) download(nodeClient *chef.Client, nodeName string) {
	for _, ckbk := range ckbks {
		ckbk.download(nodeClient, nodeName)
	}
}
