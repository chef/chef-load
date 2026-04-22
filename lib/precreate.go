//
// Copyright:: Copyright 2024 Chef Software, Inc.
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
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chef/chef"
	log "github.com/sirupsen/logrus"
)

const searchPageSize = 1000

// PrecreateNodes creates all NumNodes client + node objects on the Chef Server
// using a worker pool of PrecreateWorkers goroutines. Each node requires 2 API
// requests (POST clients + POST nodes); 409 Conflict is treated as success so
// the function is idempotent.
//
// With PrecreateWorkers=50 and ~100 ms per request, ~160 k nodes complete in
// approximately 10 minutes.
func PrecreateNodes(config *Config, requests chan *request) ([]NodeRecord, error) {
	if !config.RunChefClient {
		log.Info("Precreate: chef_server_url not set, generating node records without API calls")
		records := make([]NodeRecord, config.NumNodes)
		for i := 0; i < config.NumNodes; i++ {
			nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
			records[i] = NewNodeRecord(nodeName, i)
		}
		return records, nil
	}

	chefClient, err := newChefAPIClient(config)
	if err != nil {
		return nil, fmt.Errorf("precreate: failed to create API client: %w", err)
	}

	type result struct {
		record NodeRecord
		err    error
	}

	workCh := make(chan int, config.PrecreateWorkers)
	resultCh := make(chan result, config.NumNodes)

	var wg sync.WaitGroup
	for w := 0; w < config.PrecreateWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range workCh {
				nodeName := config.NodeNamePrefix + "-" + strconv.Itoa(i)
				rec, precreateErr := precreateNode(config, chefClient, nodeName, i, requests)
				resultCh <- result{record: rec, err: precreateErr}
			}
		}()
	}

	// Feed work indices then close the channel after all workers finish.
	go func() {
		for i := 0; i < config.NumNodes; i++ {
			workCh <- i
		}
		close(workCh)
		wg.Wait()
		close(resultCh)
	}()

	records := make([]NodeRecord, 0, config.NumNodes)
	var errs []error
	done := 0
	for r := range resultCh {
		if r.err != nil {
			errs = append(errs, r.err)
		} else {
			records = append(records, r.record)
		}
		done++
		if done%1000 == 0 {
			log.Printf("Pre-created %d/%d nodes...", done, config.NumNodes)
		}
	}

	log.Printf("Pre-creation complete: %d nodes created (%d errors)", len(records), len(errs))
	if len(errs) > 0 {
		return records, fmt.Errorf("precreate: %d node(s) failed (first: %w)", len(errs), errs[0])
	}
	return records, nil
}

// precreateNode creates a single client + node object. It issues 2 requests:
// POST clients then POST nodes. 409 Conflict is treated as success.
func precreateNode(config *Config, chefClient *chef.Client, nodeName string, index int, requests chan *request) (NodeRecord, error) {
	// Take a value copy so each goroutine has its own stack frame for the
	// chef.Client value while sharing the underlying http.Client pointer.
	nodeClient := *chefClient

	// --- POST clients ---
	clientBody := map[string]interface{}{
		"admin":     false,
		"name":      nodeName,
		"validator": false,
	}
	if config.ChefServerCreatesClientKey {
		clientBody["create_key"] = true
	}
	res, err := apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "clients", clientBody, nil, nil, requests)
	if err != nil && !isConflict(res) {
		return NodeRecord{}, fmt.Errorf("POST clients/%s: %w", nodeName, err)
	}

	// --- POST nodes ---
	var node chef.Node
	switch config.LoadMode {
	case "policyfile", "mixed":
		node = chef.Node{Name: nodeName, PolicyName: config.PolicyName, PolicyGroup: config.PolicyGroup}
	default:
		node = chef.Node{Name: nodeName, Environment: config.ChefEnvironment}
	}
	res, err = apiRequest(nodeClient, nodeName, config.ChefVersion, "POST", "nodes", node, nil, nil, requests)
	if err != nil && !isConflict(res) {
		return NodeRecord{}, fmt.Errorf("POST nodes/%s: %w", nodeName, err)
	}

	return NewNodeRecord(nodeName, index), nil
}

// isConflict returns true when the HTTP response has status 409 Conflict,
// indicating the resource already exists (idempotent pre-creation).
func isConflict(res *http.Response) bool {
	return res != nil && res.StatusCode == http.StatusConflict
}

// BulkVerifyNodes uses the Chef Server Search API to check which nodes from
// records actually exist. It paginates results 1 000 rows at a time and uses
// a partial-search POST body to retrieve only node names, minimising response
// payload for large fleets.
//
// Returns (allExist, missing names, error).
func BulkVerifyNodes(config *Config, records []NodeRecord) (bool, []string, error) {
	if !config.RunChefClient {
		return true, nil, nil
	}

	chefClient, err := newChefAPIClient(config)
	if err != nil {
		return false, nil, fmt.Errorf("bulk verify: failed to create API client: %w", err)
	}

	start := time.Now()
	searchQuery := chef.SearchQuery{
		Index:  "node",
		Query:  fmt.Sprintf("name:%s-*", config.NodeNamePrefix),
		SortBy: "X_CHEF_id_CHEF_X asc",
		Start:  0,
		Rows:   searchPageSize,
	}
	// Request only the name attribute to minimise response size.
	partialParams := map[string]interface{}{
		"name": []string{"name"},
	}

	serverNodes := make(map[string]bool, len(records))
	for {
		res, searchErr := searchQuery.DoPartialJSON(chefClient, partialParams)
		if searchErr != nil {
			return false, nil, fmt.Errorf("bulk verify: search failed: %w", searchErr)
		}

		for _, row := range res.Rows {
			var partial struct {
				Name string `json:"name"`
			}
			if jsonErr := json.Unmarshal(row.Data, &partial); jsonErr == nil && partial.Name != "" {
				serverNodes[partial.Name] = true
			}
		}

		if searchQuery.Start+searchPageSize >= res.Total {
			break
		}
		searchQuery.Start += searchPageSize
	}

	var missing []string
	for _, rec := range records {
		if !serverNodes[rec.NodeName] {
			missing = append(missing, rec.NodeName)
		}
	}

	elapsed := time.Since(start).Round(time.Millisecond)
	log.Printf("Verified %d nodes in %s via search (%d on server, %d missing)",
		len(records), elapsed, len(serverNodes), len(missing))

	return len(missing) == 0, missing, nil
}

// VerifyNodesExist checks that every record in records exists on the Chef
// Server. It delegates to BulkVerifyNodes (Search API) and falls back to
// individual GET requests if the search returns an error.
func VerifyNodesExist(config *Config, records []NodeRecord, requests chan *request) (bool, []string) {
	allExist, missing, err := BulkVerifyNodes(config, records)
	if err != nil {
		log.WithField("error", err).Warn("Bulk search verification failed, falling back to individual GETs")
		missing = verifyNodesIndividually(config, nodeNames(records), requests)
		return len(missing) == 0, missing
	}
	return allExist, missing
}

// verifyNodesIndividually does a GET per node using PrecreateWorkers goroutines.
// Used as a fallback when the Search index is unavailable.
func verifyNodesIndividually(config *Config, names []string, requests chan *request) []string {
	if !config.RunChefClient || len(names) == 0 {
		return nil
	}

	chefClient, err := newChefAPIClient(config)
	if err != nil {
		log.WithField("error", err).Warn("Could not create API client for individual verification; assuming all nodes missing")
		return names
	}

	workCh := make(chan string, config.PrecreateWorkers)
	var mu sync.Mutex
	var missing []string
	var wg sync.WaitGroup

	workers := config.PrecreateWorkers
	if workers < 1 {
		workers = 1
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range workCh {
				nc := *chefClient
				res, _ := apiRequest(nc, name, config.ChefVersion, "GET", "nodes/"+name, nil, nil, nil, requests)
				if res == nil || res.StatusCode == http.StatusNotFound {
					mu.Lock()
					missing = append(missing, name)
					mu.Unlock()
				}
			}
		}()
	}

	for _, name := range names {
		workCh <- name
	}
	close(workCh)
	wg.Wait()
	return missing
}

// LoadOrPrecreate implements the resume-or-create logic for the pre-creation
// phase:
//
//  1. If the node log file exists and is non-empty, verify all nodes are
//     present on the server via BulkVerifyNodes. If all exist, return the
//     records immediately (skip pre-creation).
//
//  2. If some nodes are missing, re-create only those nodes and update the
//     log file.
//
//  3. If the log file does not exist, run a full PrecreateNodes and save the
//     log file.
func LoadOrPrecreate(config *Config, requests chan *request) ([]NodeRecord, error) {
	existing, err := LoadNodeLog(config.NodeLogFile)
	if err == nil && len(existing) > 0 {
		log.Printf("Node log found (%d records). Verifying nodes exist on server...", len(existing))

		allExist, missing := VerifyNodesExist(config, existing, requests)
		if allExist {
			log.Printf("All %d nodes verified. Skipping pre-creation.", len(existing))
			return existing, nil
		}

		log.Printf("%d node(s) missing from server. Re-creating...", len(missing))
		missingRecords := filterByName(existing, missing)
		newRecords, recreateErr := recreateMissingNodes(config, missingRecords, requests)
		if recreateErr != nil {
			return existing, recreateErr
		}

		// Merge: update existing records with freshly re-created ones.
		nameMap := make(map[string]NodeRecord, len(newRecords))
		for _, r := range newRecords {
			nameMap[r.NodeName] = r
		}
		merged := make([]NodeRecord, len(existing))
		copy(merged, existing)
		for i, r := range merged {
			if nr, ok := nameMap[r.NodeName]; ok {
				merged[i] = nr
			}
		}
		if saveErr := SaveNodeLog(config.NodeLogFile, merged); saveErr != nil {
			log.WithField("error", saveErr).Warn("Could not update node log after re-creation")
		}
		return merged, nil
	}

	// No existing log — full pre-creation.
	log.Printf("No node log found at %s. Starting full pre-creation of %d nodes...",
		config.NodeLogFile, config.NumNodes)
	records, precreateErr := PrecreateNodes(config, requests)
	if precreateErr != nil {
		// Save whatever was created so we can resume next time.
		_ = SaveNodeLog(config.NodeLogFile, records)
		return records, precreateErr
	}
	if saveErr := SaveNodeLog(config.NodeLogFile, records); saveErr != nil {
		log.WithField("error", saveErr).Warn("Could not save node log after pre-creation")
	}
	return records, nil
}

// recreateMissingNodes creates client + node objects for a subset of records
// (those whose names are missing from the server).
func recreateMissingNodes(config *Config, records []NodeRecord, requests chan *request) ([]NodeRecord, error) {
	if !config.RunChefClient || len(records) == 0 {
		return records, nil
	}

	chefClient, err := newChefAPIClient(config)
	if err != nil {
		return nil, fmt.Errorf("recreate: failed to create API client: %w", err)
	}

	type result struct {
		record NodeRecord
		err    error
	}

	workCh := make(chan NodeRecord, config.PrecreateWorkers)
	resultCh := make(chan result, len(records))
	var wg sync.WaitGroup

	workers := config.PrecreateWorkers
	if workers < 1 {
		workers = 1
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rec := range workCh {
				newRec, recErr := precreateNode(config, chefClient, rec.NodeName, rec.Index, requests)
				resultCh <- result{record: newRec, err: recErr}
			}
		}()
	}

	go func() {
		for _, r := range records {
			workCh <- r
		}
		close(workCh)
		wg.Wait()
		close(resultCh)
	}()

	var out []NodeRecord
	var errs []error
	for r := range resultCh {
		if r.err != nil {
			errs = append(errs, r.err)
		} else {
			out = append(out, r.record)
		}
	}
	if len(errs) > 0 {
		return out, fmt.Errorf("recreate: %d node(s) failed (first: %w)", len(errs), errs[0])
	}
	return out, nil
}

// --- helpers ---

func nodeNames(records []NodeRecord) []string {
	names := make([]string, len(records))
	for i, r := range records {
		names[i] = r.NodeName
	}
	return names
}

func filterByName(records []NodeRecord, names []string) []NodeRecord {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	var out []NodeRecord
	for _, r := range records {
		if set[r.NodeName] {
			out = append(out, r)
		}
	}
	return out
}

// RunPrecreate is the top-level entry point for the standalone `precreate`
// command. It sets up a request channel + consumer, calls PrecreateNodes,
// saves the node log, and prints a summary. Mirrors the pattern used by
// Start() so the unexported *request type stays internal to the lib package.
func RunPrecreate(config *Config) error {
	if !config.RunChefClient {
		return fmt.Errorf("precreate requires chef_server_url, client_name, and client_key to be set")
	}

	requests := make(chan *request)
	var totalRequests int
	reqDone := make(chan struct{})
	go func() {
		defer close(reqDone)
		for range requests {
			totalRequests++
		}
	}()

	records, precreateErr := PrecreateNodes(config, requests)
	close(requests)
	<-reqDone

	if saveErr := SaveNodeLog(config.NodeLogFile, records); saveErr != nil {
		log.WithField("error", saveErr).Warn("Could not save node log")
	}

	log.Printf("Precreate summary: %d nodes created, %d API requests, log: %s",
		len(records), totalRequests, config.NodeLogFile)
	return precreateErr
}
