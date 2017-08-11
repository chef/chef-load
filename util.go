package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-chef/chef"
)

func apiRequest(nodeClient chef.Client, method, url string, body interface{}, v interface{}, headers map[string]string) (*http.Response, error) {
	var bodyJSON io.Reader = nil
	if body != nil {
		var err error
		bodyJSON, err = chef.JSONReader(body)
		if err != nil {
			fmt.Println(err)
		}
	}

	req, _ := nodeClient.NewRequest(method, url, bodyJSON)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	res, err := nodeClient.Do(req, v)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return res, err
	}
	ioutil.ReadAll(res.Body)
	return res, err
}

func getAPIClient(clientName, privateKeyPath, chefServerURL string) chef.Client {
	privateKey := getPrivateKey(privateKeyPath)

	client, err := chef.NewClient(&chef.Config{
		Name:    clientName,
		Key:     privateKey,
		BaseURL: chefServerURL,
		SkipSSL: true,
	})
	if err != nil {
		fmt.Println("Issue setting up client:", err)
	}
	return *client
}

func getPrivateKey(privateKeyPath string) string {
	fileContent, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		fmt.Println("Couldn't read private key:", err)
	}
	privateKey := string(fileContent)
	return privateKey
}

func parseJSONFile(jsonFile string) map[string]interface{} {
	jsonContent := map[string]interface{}{}

	file, err := os.Open(jsonFile)
	if err != nil {
		fmt.Println("Couldn't open ohai JSON file ", jsonFile, ": ", err)
		return jsonContent
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&jsonContent)
	if err != nil {
		fmt.Println("Couldn't decode ohai JSON file ", jsonFile, ": ", err)
		return jsonContent
	}
	return jsonContent
}
