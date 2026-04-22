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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- helpers ----------------------------------------------------------------

func makeRecords(prefix string, n int) []NodeRecord {
	records := make([]NodeRecord, n)
	for i := 0; i < n; i++ {
		name := prefix + "-" + string(rune('0'+i))
		if i >= 10 {
			name = prefix + "-x" // ensure uniqueness for >9
		}
		records[i] = NewNodeRecord(name, i)
	}
	return records
}

// ---- nodeNames --------------------------------------------------------------

func TestNodeNames(t *testing.T) {
	records := []NodeRecord{
		{NodeName: "a"},
		{NodeName: "b"},
		{NodeName: "c"},
	}
	assert.Equal(t, []string{"a", "b", "c"}, nodeNames(records))
}

func TestNodeNames_Empty(t *testing.T) {
	assert.Empty(t, nodeNames(nil))
}

// ---- filterByName -----------------------------------------------------------

func TestFilterByName(t *testing.T) {
	records := []NodeRecord{
		{NodeName: "node-0"},
		{NodeName: "node-1"},
		{NodeName: "node-2"},
	}
	filtered := filterByName(records, []string{"node-0", "node-2"})
	require.Len(t, filtered, 2)
	assert.Equal(t, "node-0", filtered[0].NodeName)
	assert.Equal(t, "node-2", filtered[1].NodeName)
}

func TestFilterByName_NoneMatch(t *testing.T) {
	records := []NodeRecord{{NodeName: "node-0"}}
	assert.Empty(t, filterByName(records, []string{"node-99"}))
}

// ---- PrecreateNodes (offline / RunChefClient=false) ------------------------

func TestPrecreateNodes_NoChefServer(t *testing.T) {
	cfg := Default()
	cfg.RunChefClient = false
	cfg.NumNodes = 5
	cfg.NodeNamePrefix = "tst"

	requests := make(chan *request, 100)
	defer func() {
		// drain
		for len(requests) > 0 {
			<-requests
		}
	}()

	records, err := PrecreateNodes(&cfg, requests)
	require.NoError(t, err)
	require.Len(t, records, 5)

	for i, r := range records {
		assert.Equal(t, "tst-"+string(rune('0'+i)), r.NodeName)
		assert.Equal(t, i, r.Index)
		assert.NotEmpty(t, r.NodeUUID)
	}
}

func TestPrecreateNodes_UUIDDeterministic(t *testing.T) {
	cfg := Default()
	cfg.RunChefClient = false
	cfg.NumNodes = 3
	cfg.NodeNamePrefix = "det"

	requests := make(chan *request, 50)
	defer func() {
		for len(requests) > 0 {
			<-requests
		}
	}()

	records1, err1 := PrecreateNodes(&cfg, requests)
	records2, err2 := PrecreateNodes(&cfg, requests)
	require.NoError(t, err1)
	require.NoError(t, err2)

	for i := range records1 {
		assert.Equal(t, records1[i].NodeUUID, records2[i].NodeUUID)
	}
}

// ---- LoadOrPrecreate (offline) ----------------------------------------------

func TestLoadOrPrecreate_NoLogFile(t *testing.T) {
	dir := t.TempDir()
	cfg := Default()
	cfg.RunChefClient = false
	cfg.NumNodes = 4
	cfg.NodeNamePrefix = "lop"
	cfg.NodeLogFile = filepath.Join(dir, "nodes.json")

	requests := make(chan *request, 100)
	defer func() {
		for len(requests) > 0 {
			<-requests
		}
	}()

	records, err := LoadOrPrecreate(&cfg, requests)
	require.NoError(t, err)
	require.Len(t, records, 4)

	// Log file should have been created.
	_, statErr := os.Stat(cfg.NodeLogFile)
	require.NoError(t, statErr)
}

func TestLoadOrPrecreate_ExistingLog_AllPresent(t *testing.T) {
	dir := t.TempDir()
	cfg := Default()
	cfg.RunChefClient = false // skip real server calls; BulkVerifyNodes returns true
	cfg.NumNodes = 3
	cfg.NodeNamePrefix = "resume"
	cfg.NodeLogFile = filepath.Join(dir, "nodes.json")

	// Seed the log file with pre-existing records.
	existing := []NodeRecord{
		NewNodeRecord("resume-0", 0),
		NewNodeRecord("resume-1", 1),
		NewNodeRecord("resume-2", 2),
	}
	require.NoError(t, SaveNodeLog(cfg.NodeLogFile, existing))

	requests := make(chan *request, 100)
	defer func() {
		for len(requests) > 0 {
			<-requests
		}
	}()

	records, err := LoadOrPrecreate(&cfg, requests)
	require.NoError(t, err)
	require.Len(t, records, 3)

	// The records loaded from the log file should have the original UUIDs.
	for i, r := range records {
		assert.Equal(t, existing[i].NodeName, r.NodeName)
		assert.Equal(t, existing[i].NodeUUID, r.NodeUUID)
	}
}

// ---- BulkVerifyNodes (offline) ---------------------------------------------

func TestBulkVerifyNodes_RunChefClientFalse(t *testing.T) {
	cfg := Default()
	cfg.RunChefClient = false
	records := []NodeRecord{
		NewNodeRecord("n-0", 0),
		NewNodeRecord("n-1", 1),
	}
	allExist, missing, err := BulkVerifyNodes(&cfg, records)
	require.NoError(t, err)
	assert.True(t, allExist)
	assert.Empty(t, missing)
}
