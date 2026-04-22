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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNodeRecord(t *testing.T) {
	rec := NewNodeRecord("chef-load-42", 42)

	assert.Equal(t, "chef-load-42", rec.NodeName)
	assert.Equal(t, 42, rec.Index)
	assert.NotEmpty(t, rec.NodeUUID)
	assert.Empty(t, rec.RetiredAt)

	// UUID must be deterministic.
	rec2 := NewNodeRecord("chef-load-42", 42)
	assert.Equal(t, rec.NodeUUID, rec2.NodeUUID)

	// Different names must produce different UUIDs.
	rec3 := NewNodeRecord("chef-load-99", 99)
	assert.NotEqual(t, rec.NodeUUID, rec3.NodeUUID)

	// Timestamp must be parseable as RFC3339.
	_, err := time.Parse(time.RFC3339, rec.CreatedAt)
	assert.NoError(t, err)
}

func TestSaveAndLoadNodeLog_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nodes.json")

	records := []NodeRecord{
		NewNodeRecord("chef-load-0", 0),
		NewNodeRecord("chef-load-1", 1),
		NewNodeRecord("chef-load-2", 2),
	}

	require.NoError(t, SaveNodeLog(path, records))

	loaded, err := LoadNodeLog(path)
	require.NoError(t, err)
	require.Len(t, loaded, len(records))

	for i, r := range loaded {
		assert.Equal(t, records[i].NodeName, r.NodeName)
		assert.Equal(t, records[i].NodeUUID, r.NodeUUID)
		assert.Equal(t, records[i].Index, r.Index)
	}
}

func TestSaveNodeLog_Atomic(t *testing.T) {
	// SaveNodeLog must not leave a partial .tmp file on success.
	dir := t.TempDir()
	path := filepath.Join(dir, "nodes.json")

	require.NoError(t, SaveNodeLog(path, []NodeRecord{NewNodeRecord("n0", 0)}))

	tmpPath := path + ".tmp"
	_, err := os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "tmp file should be removed after atomic rename")
}

func TestSaveNodeLog_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nodes.json")

	records := []NodeRecord{NewNodeRecord("chef-load-0", 0)}
	require.NoError(t, SaveNodeLog(path, records))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var raw []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Len(t, raw, 1)
}

func TestLoadNodeLog_FileNotFound(t *testing.T) {
	_, err := LoadNodeLog("/nonexistent/path/nodes.json")
	assert.Error(t, err)
}

func TestLoadNodeLog_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nodes.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0644))

	_, err := LoadNodeLog(path)
	assert.Error(t, err)
}

func TestAppendNodeLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nodes.json")

	first := []NodeRecord{NewNodeRecord("chef-load-0", 0)}
	require.NoError(t, SaveNodeLog(path, first))

	second := []NodeRecord{NewNodeRecord("chef-load-1", 1)}
	require.NoError(t, AppendNodeLog(path, second))

	loaded, err := LoadNodeLog(path)
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "chef-load-0", loaded[0].NodeName)
	assert.Equal(t, "chef-load-1", loaded[1].NodeName)
}

func TestAppendNodeLog_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new-nodes.json")

	// File does not exist yet — AppendNodeLog should create it.
	rec := NewNodeRecord("chef-load-0", 0)
	require.NoError(t, AppendNodeLog(path, []NodeRecord{rec}))

	loaded, err := LoadNodeLog(path)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, rec.NodeName, loaded[0].NodeName)
}

func TestMarkRetired(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nodes.json")

	records := []NodeRecord{
		NewNodeRecord("chef-load-0", 0),
		NewNodeRecord("chef-load-1", 1),
		NewNodeRecord("chef-load-2", 2),
	}
	require.NoError(t, SaveNodeLog(path, records))

	require.NoError(t, MarkRetired(path, []string{"chef-load-1"}))

	loaded, err := LoadNodeLog(path)
	require.NoError(t, err)
	require.Len(t, loaded, 3)

	assert.Empty(t, loaded[0].RetiredAt, "chef-load-0 should not be retired")
	assert.NotEmpty(t, loaded[1].RetiredAt, "chef-load-1 should be retired")
	assert.Empty(t, loaded[2].RetiredAt, "chef-load-2 should not be retired")

	// Calling MarkRetired again should not change the timestamp.
	originalRetiredAt := loaded[1].RetiredAt
	time.Sleep(time.Second) // ensure a different timestamp would be generated
	require.NoError(t, MarkRetired(path, []string{"chef-load-1"}))

	loaded2, err := LoadNodeLog(path)
	require.NoError(t, err)
	assert.Equal(t, originalRetiredAt, loaded2[1].RetiredAt, "retired_at should not be updated on second call")
}
