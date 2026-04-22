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
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NodeRecord tracks a single pre-created node in the JSON log file.
type NodeRecord struct {
	NodeName  string `json:"node_name"`
	NodeUUID  string `json:"node_uuid"`   // deterministic: uuid.NewMD5(uuid.NameSpaceDNS, []byte(NodeName))
	CreatedAt string `json:"created_at"`  // RFC3339 timestamp
	Index     int    `json:"index"`        // positional index in the node pool
	RetiredAt string `json:"retired_at,omitempty"` // set when node is retired via elastic scale-down
}

// nodeLogMu serialises concurrent writes to the node log file.
var nodeLogMu sync.Mutex

// NewNodeRecord creates a NodeRecord for the given name and pool index using
// a deterministic UUID derived from the node name (consistent with the rest
// of the codebase).
func NewNodeRecord(nodeName string, index int) NodeRecord {
	nodeUUID := uuid.NewMD5(uuid.NameSpaceDNS, []byte(nodeName))
	return NodeRecord{
		NodeName:  nodeName,
		NodeUUID:  nodeUUID.String(),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Index:     index,
	}
}

// SaveNodeLog atomically writes records to path as a JSON array.
// Atomic via write-to-temp + rename (POSIX).
func SaveNodeLog(path string, records []NodeRecord) error {
	nodeLogMu.Lock()
	defer nodeLogMu.Unlock()
	return saveNodeLogLocked(path, records)
}

// saveNodeLogLocked is the unlocked inner write; callers must hold nodeLogMu.
func saveNodeLogLocked(path string, records []NodeRecord) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// LoadNodeLog reads and parses the JSON node log file at path.
func LoadNodeLog(path string) ([]NodeRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var records []NodeRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// AppendNodeLog appends newRecords to the existing log at path, creating the
// file if it does not exist. The write is atomic and thread-safe.
func AppendNodeLog(path string, newRecords []NodeRecord) error {
	nodeLogMu.Lock()
	defer nodeLogMu.Unlock()

	var existing []NodeRecord
	if data, err := os.ReadFile(path); err == nil {
		// Ignore unmarshal errors on a partially-written file; we'll overwrite.
		_ = json.Unmarshal(data, &existing)
	}
	combined := append(existing, newRecords...)
	return saveNodeLogLocked(path, combined)
}

// MarkRetired updates the RetiredAt field for each name in retired inside the
// log file at path.
func MarkRetired(path string, retired []string) error {
	nodeLogMu.Lock()
	defer nodeLogMu.Unlock()

	records, err := loadNodeLogLocked(path)
	if err != nil {
		return err
	}

	retiredSet := make(map[string]bool, len(retired))
	for _, n := range retired {
		retiredSet[n] = true
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for i := range records {
		if retiredSet[records[i].NodeName] && records[i].RetiredAt == "" {
			records[i].RetiredAt = now
		}
	}
	return saveNodeLogLocked(path, records)
}

// loadNodeLogLocked reads the log without acquiring the mutex (callers must hold it).
func loadNodeLogLocked(path string) ([]NodeRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var records []NodeRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}
