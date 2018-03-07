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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/go-chef/chef"
	log "github.com/sirupsen/logrus"
)

func apiRequest(nodeClient chef.Client, nodeName, chefVersion, method, url string,
	body, v interface{}, headers map[string]string, requests chan *request) (*http.Response, error) {

	var bodyJSON io.Reader = nil
	if body != nil {
		var err error
		bodyJSON, err = chef.JSONReader(body)
		if err != nil {
			log.WithField("error", err).Error("Could not convert data to JSON")
		}
	}

	req, _ := nodeClient.NewRequest(method, url, bodyJSON)
	req.Header.Set("X-Ops-Server-Api-Version", "1")
	req.Header.Set("X-Chef-Version", chefVersion)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	t0 := time.Now()
	res, err := nodeClient.Do(req, v)
	request_time := time.Now().Sub(t0)
	statusCode := 999
	if res != nil {
		defer res.Body.Close()
		statusCode = res.StatusCode
	}

	requests <- &request{
		Method:     req.Method,
		Url:        req.URL.String(),
		StatusCode: statusCode,
	}

	logger.WithFields(log.Fields{
		"node_name":            nodeName,
		"method":               req.Method,
		"url":                  req.URL.String(),
		"status_code":          statusCode,
		"request_time_seconds": float64(request_time.Nanoseconds()/1e6) / 1000,
	}).Info("API Request")

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
		log.WithField("error", err).Error("Could not create API client")
	}
	return *client
}

func getPrivateKey(privateKeyPath string) string {
	fileContent, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		log.WithField("error", err).Error("Could not read private key %s", privateKeyPath)
	}
	privateKey := string(fileContent)
	return privateKey
}

func parseJSONFile(jsonFile string) map[string]interface{} {
	jsonContent := map[string]interface{}{}

	file, err := os.Open(jsonFile)
	if err != nil {
		log.WithField("error", err).Error("Could not open JSON file %s", jsonFile)
		return jsonContent
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&jsonContent)
	if err != nil {
		log.WithField("error", err).Error("Could not decode JSON file %s", jsonFile)
		return jsonContent
	}
	return jsonContent
}

type amountOfRequests map[request]uint64

func (a amountOfRequests) addRequest(req request) {
	re := regexp.MustCompile("/bookshelf/.*")
	req.Url = re.ReplaceAllString(req.Url, "/bookshelf/<...>")

	re = regexp.MustCompile("(/nodes/.*-)\\d+(/.*)?")
	req.Url = re.ReplaceAllString(req.Url, "$1<N>$2")
	a[req]++
}

func printAPIRequestProfile(numRequests map[request]uint64) {
	log.Info("Printing profile of API requests")

	var (
		requests    []request
		maxAmount   uint64
		totalAmount uint64
	)

	for request, amount := range numRequests {
		requests = append(requests, request)
		if amount > maxAmount {
			maxAmount = amount
		}
		totalAmount += amount
	}

	sort.Slice(requests, func(i, j int) bool {
		switch {
		case requests[i].Url < requests[j].Url:
			return true
		case requests[i].Url == requests[j].Url:
			switch {
			case requests[i].Method < requests[j].Method:
				return true
			case requests[i].Method == requests[j].Method:
				if requests[i].StatusCode < requests[j].StatusCode {
					return true
				}
			}
		}
		return false
	})

	log.Info("Total API Requests: ", totalAmount)
	amountHeader := "Subtotal"
	amountFieldWidth := len(amountHeader)
	if maxAmountWidth := len(strconv.FormatUint(maxAmount, 10)); maxAmountWidth > amountFieldWidth {
		amountFieldWidth = maxAmountWidth
	}
	log.Info(fmt.Sprintf("%% of Total | %-*s | Status | Method | URL", amountFieldWidth, amountHeader))
	for _, request := range requests {
		count := numRequests[request]
		percentOfTotal := float64(count) / float64(totalAmount) * 100.0
		log.Info(fmt.Sprintf("%-10.2f   %-*d   %-6d   %-6s   %s",
			percentOfTotal, amountFieldWidth, count, request.StatusCode, request.Method, request.Url))
	}
}
