//
// Copyright:: Copyright 2017 Chef Software, Inc.
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

package main

// This file will have the functions that will generate random data,
// it involve creating fake Chef Nodes and Chef Runs that can be sent
// to the data-collector endpoint

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/go-chef/chef"
)

func generateRandomData(nodeClient chef.Client, ohaiJSON, convergeJSON, complianceJSON map[string]interface{}, requests amountOfRequests) (err error) {
	channels := make([]<-chan error, config.NumNodes)

	for i := 0; i < config.NumNodes; i++ {
		nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i+1)
		fmt.Printf(".")
		channels[i] = ccr(nodeClient, nodeName, true, ohaiJSON, convergeJSON, complianceJSON)
	}

	fmt.Println("\n")

	for n := range merge(channels...) {
		if n != nil {
			fmt.Println("Error: ", n)
			err = n
		}
	}

	printAPIRequestProfile(requests)

	return err
}

func ccr(nodeClient chef.Client, nodeName string, firstRun bool,
	ohaiJSON, convergeJSON, complianceJSON map[string]interface{}) <-chan error {
	out := make(chan error)
	go func() {
		chefClientRun(nodeClient, nodeName, true, ohaiJSON, convergeJSON, complianceJSON)
		close(out)
	}()
	return out
}

func merge(cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan error) {
		for err := range c {
			out <- err
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
