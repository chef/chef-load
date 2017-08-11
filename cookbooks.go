package main

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

func (ckbkFile cookbookFile) download(nodeClient *chef.Client) error {
	_, err := apiRequest(*nodeClient, "GET", ckbkFile.URL, nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (ckbk cookbook) download(nodeClient *chef.Client) error {
	for _, ckbkFile := range ckbk.Attributes {
		err := ckbkFile.download(nodeClient)
		if err != nil {
			return err
		}
	}
	for _, ckbkFile := range ckbk.Definitions {
		err := ckbkFile.download(nodeClient)
		if err != nil {
			return err
		}
	}
	for _, ckbkFile := range ckbk.Files {
		err := ckbkFile.download(nodeClient)
		if err != nil {
			return err
		}
	}
	for _, ckbkFile := range ckbk.Libraries {
		err := ckbkFile.download(nodeClient)
		if err != nil {
			return err
		}
	}
	for _, ckbkFile := range ckbk.Providers {
		err := ckbkFile.download(nodeClient)
		if err != nil {
			return err
		}
	}
	for _, ckbkFile := range ckbk.Recipes {
		err := ckbkFile.download(nodeClient)
		if err != nil {
			return err
		}
	}
	for _, ckbkFile := range ckbk.Resources {
		err := ckbkFile.download(nodeClient)
		if err != nil {
			return err
		}
	}
	for _, ckbkFile := range ckbk.RootFiles {
		err := ckbkFile.download(nodeClient)
		if err != nil {
			return err
		}
	}
	for _, ckbkFile := range ckbk.Templates {
		err := ckbkFile.download(nodeClient)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ckbks cookbooks) download(nodeClient *chef.Client) error {
	for _, ckbk := range ckbks {
		err := ckbk.download(nodeClient)
		if err != nil {
			return err
		}
	}
	return nil
}
