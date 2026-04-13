//
// Copyright:: Copyright 2026 Progress Software Corporation, Inc.
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
	"testing"
	"time"

	"github.com/go-chef/chef"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// helpers

func testNodeDetails() NodeDetails {
	return NodeDetails{
		name:        "test-node",
		environment: "production",
		ipAddr:      "10.0.0.1",
		fqdn:        "test-node.example.com",
		sourceFqdn:  "chef.example.com",
		orgName:     "myorg",
		policyName:  "base",
		policyGroup: "prod",
		chefTags:    []string{"tag1"},
		roles:       []string{"web"},
		recipes:     []string{"base::default"},
		nodeUUID:    uuid.NewMD5(uuid.NameSpaceDNS, []byte("test-node")),
	}
}

func testConfig() *Config {
	cfg := Default()
	cfg.ChefServerURL = "https://chef.example.com/organizations/myorg"
	cfg.DataCollectorToken = "test-token"
	cfg.ChefVersion = "18.0.0"
	return &cfg
}

// --- syntheticComplianceReport ---

func TestSyntheticComplianceReport_Shape(t *testing.T) {
	r := syntheticComplianceReport()
	assert.Equal(t, "6.0.0", r["version"], "version field")
	stats, ok := r["statistics"].(map[string]interface{})
	if !assert.True(t, ok, "statistics should be a map") {
		return
	}
	assert.NotZero(t, stats["duration"], "duration field")
	profiles, ok := r["profiles"].([]interface{})
	if !assert.True(t, ok, "profiles should be a slice") {
		return
	}
	assert.Empty(t, profiles, "profiles should be empty for synthetic report")
}

func TestSyntheticComplianceReport_ReturnsNewMapEachCall(t *testing.T) {
	r1 := syntheticComplianceReport()
	r2 := syntheticComplianceReport()
	r1["version"] = "mutated"
	assert.Equal(t, "6.0.0", r2["version"], "mutations must not bleed between calls")
}

// --- dataCollectorComplianceReport ---

func TestDataCollectorComplianceReport_RequiredFields(t *testing.T) {
	nd := testNodeDetails()
	reportUUID := uuid.New()
	runUUID := uuid.New()
	endTime := time.Now().UTC()
	body := syntheticComplianceReport()

	result := dataCollectorComplianceReport(nd, reportUUID, runUUID, endTime, body).(map[string]interface{})

	assert.Equal(t, "inspec_report", result["type"])
	assert.Equal(t, nd.name, result["node_name"])
	assert.Equal(t, nd.environment, result["environment"])
	assert.Equal(t, nd.fqdn, result["fqdn"])
	assert.Equal(t, nd.sourceFqdn, result["source_fqdn"])
	assert.Equal(t, nd.orgName, result["organization_name"])
	assert.Equal(t, nd.policyName, result["policy_name"])
	assert.Equal(t, nd.policyGroup, result["policy_group"])
	assert.Equal(t, nd.chefTags, result["chef_tags"])
	assert.Equal(t, nd.ipAddr, result["ipaddress"])
	assert.Equal(t, nd.nodeUUID, result["node_uuid"])
	assert.Equal(t, reportUUID, result["report_uuid"])
	assert.Equal(t, runUUID, result["run_uuid"])
	assert.Equal(t, endTime.Format(DateTimeFormat), result["end_time"])
}

func TestDataCollectorComplianceReport_RunUUIDIsNilForStandaloneScans(t *testing.T) {
	// Standalone compliance scans (from the generate path) should carry a nil
	// run_uuid so the Automate ingest pipeline doesn't attempt to correlate them
	// with a CCR.
	nd := testNodeDetails()
	result := dataCollectorComplianceReport(nd, uuid.New(), uuid.Nil, time.Now().UTC(),
		syntheticComplianceReport()).(map[string]interface{})
	assert.Equal(t, uuid.Nil, result["run_uuid"])
}

func TestDataCollectorComplianceReport_ControlsFieldStripped(t *testing.T) {
	nd := testNodeDetails()
	body := syntheticComplianceReport()
	body["controls"] = []interface{}{"should-be-deleted"}

	result := dataCollectorComplianceReport(nd, uuid.New(), uuid.New(), time.Now().UTC(), body).(map[string]interface{})
	assert.Nil(t, result["controls"], "controls field must be stripped")
}

// --- dataCollectorRunStart ---

func TestDataCollectorRunStart_MessageType(t *testing.T) {
	cfg := testConfig()
	runUUID := uuid.New()
	nodeUUID := uuid.New()
	startTime := time.Now().UTC()

	body := dataCollectorRunStart(cfg, "test-node", "chef.example.com", "myorg", runUUID, nodeUUID, startTime).(map[string]interface{})

	assert.Equal(t, "run_start", body["message_type"])
	assert.Equal(t, "1.1.0", body["message_version"])
	assert.Equal(t, "chef_client", body["source"])
	assert.Equal(t, "test-node", body["node_name"])
	assert.Equal(t, "myorg", body["organization_name"])
	assert.Equal(t, runUUID.String(), body["run_id"])
	assert.Equal(t, runUUID.String(), body["id"])
	assert.Equal(t, nodeUUID.String(), body["entity_uuid"])
	assert.Equal(t, startTime.Format(DateTimeFormat), body["start_time"])
}

func TestDataCollectorRunStart_InfersChefServerFQDN(t *testing.T) {
	// When chefServerFQDN is empty, it should be derived from ChefServerURL.
	cfg := testConfig()
	body := dataCollectorRunStart(cfg, "node", "", "myorg", uuid.New(), uuid.New(), time.Now()).(map[string]interface{})
	assert.Equal(t, "chef.example.com", body["chef_server_fqdn"])
}

// --- dataCollectorRunStop ---

func testRunStopBody(t *testing.T, expandedRunListID string) map[string]interface{} {
	t.Helper()
	cfg := testConfig()
	node := chef.Node{Name: "test-node"}
	runUUID := uuid.New()
	nodeUUID := uuid.New()
	start := time.Now().UTC()
	end := start.Add(30 * time.Second)
	rl := parseRunList([]string{"recipe[base::default]"})

	result := dataCollectorRunStop(cfg, node, "test-node", "chef.example.com", "myorg", "success",
		rl, rl, runUUID, nodeUUID, start, end, map[string]interface{}{}, expandedRunListID)
	return result.(map[string]interface{})
}

func TestDataCollectorRunStop_MessageType(t *testing.T) {
	body := testRunStopBody(t, "_default")
	assert.Equal(t, "run_converge", body["message_type"])
	assert.Equal(t, "1.1.0", body["message_version"])
	assert.Equal(t, "chef_client", body["source"])
	assert.Equal(t, "success", body["status"])
}

func TestDataCollectorRunStop_LegacyExpandedRunListID(t *testing.T) {
	body := testRunStopBody(t, "_default")
	erl := body["expanded_run_list"].(map[string]interface{})
	assert.Equal(t, "_default", erl["id"], "traditional runs use chef environment as expanded_run_list id")
}

func TestDataCollectorRunStop_PolicyfileExpandedRunListID(t *testing.T) {
	body := testRunStopBody(t, "_policy_node")
	erl := body["expanded_run_list"].(map[string]interface{})
	assert.Equal(t, "_policy_node", erl["id"], "policyfile runs must use _policy_node as expanded_run_list id")
}

func TestDataCollectorRunStop_RunListContents(t *testing.T) {
	body := testRunStopBody(t, "_default")
	rl := body["run_list"].([]interface{})
	if !assert.Len(t, rl, 1) {
		return
	}
	assert.Equal(t, "recipe[base::default]", rl[0])
}

func TestDataCollectorRunStop_ExpandedRunListItems(t *testing.T) {
	body := testRunStopBody(t, "_default")
	erl := body["expanded_run_list"].(map[string]interface{})
	items := erl["run_list"].([]interface{})
	if !assert.Len(t, items, 1) {
		return
	}
	item := items[0].(expandedRunListItem)
	assert.Equal(t, "base::default", item.Name)
	assert.Equal(t, "recipe", item.ItemType)
	assert.False(t, item.Skipped)
}

func TestDataCollectorRunStop_ConvergeJSONOverridesRunList(t *testing.T) {
	// When converge_status_json_file supplies run_list and expanded_run_list,
	// those values must take precedence over anything computed from config.
	cfg := testConfig()
	node := chef.Node{Name: "test-node"}
	runUUID := uuid.New()
	nodeUUID := uuid.New()
	start := time.Now().UTC()
	end := start.Add(5 * time.Second)
	overrideRL := []interface{}{"recipe[override::default]"}
	overrideERL := map[string]interface{}{
		"id":       "from-file",
		"run_list": []interface{}{},
	}
	convergeJSON := map[string]interface{}{
		"run_list":          overrideRL,
		"expanded_run_list": overrideERL,
	}

	body := dataCollectorRunStop(cfg, node, "test-node", "chef.example.com", "myorg", "success",
		parseRunList([]string{}), parseRunList([]string{}), runUUID, nodeUUID, start, end,
		convergeJSON, "_default").(map[string]interface{})

	assert.Equal(t, overrideRL, body["run_list"])
	assert.Equal(t, overrideERL, body["expanded_run_list"])
}
