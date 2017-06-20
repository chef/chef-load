package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chef/chef"
)

func apiRequest(nodeClient chef.Client, method, url string, data io.Reader) (*http.Response, error) {
	req, _ := nodeClient.NewRequest("GET", url, data)
	res, err := nodeClient.Do(req, nil)
	if err != nil {
		// can't print res here if it is nil
		// fmt.Println(res.StatusCode)
		// TODO: should this be handled better than just skipping over it?
		fmt.Println(err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		ioutil.ReadAll(res.Body)
	}
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

func getStatusCode(err error) int {
	errFields := strings.Fields(err.Error())
	statusCode, _ := strconv.Atoi(errFields[len(errFields)-1])
	return statusCode
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
