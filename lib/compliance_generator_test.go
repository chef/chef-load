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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- generateNodeName ---

func TestGenerateNodeName(t *testing.T) {
	nodeName := generateNodeName("prefix")

	nodeNameTokenized := strings.Split(nodeName, "-")
	assert.Len(t, nodeNameTokenized, 4, "")
}

// --- int2ip ---

func TestInt2ip_Zero(t *testing.T) {
	assert.Equal(t, "0.0.0.0", int2ip(0).String())
}

func TestInt2ip_Loopback(t *testing.T) {
	assert.Equal(t, "127.0.0.1", int2ip(0x7F000001).String())
}

func TestInt2ip_PrivateRange(t *testing.T) {
	assert.Equal(t, "192.168.0.1", int2ip(0xC0A80001).String())
}

func TestInt2ip_Broadcast(t *testing.T) {
	assert.Equal(t, "255.255.255.255", int2ip(0xFFFFFFFF).String())
}

// --- intervalMinutes ---
// Boundaries (with nodesCount=100, maxScansPerDay=4):
//   divisor ≤ 0.10 → 1440/maxScansPerDay
//   divisor ≤ 0.40 → 1440
//   divisor ≤ 0.80 → 10080
//   divisor >  0.80 → 43200

func TestIntervalMinutes_TenPercentBoundary(t *testing.T) {
	// index 0 (0%) and index 10 (10%) both fall in the first bucket.
	assert.Equal(t, 360, intervalMinutes(100, 0, 4))
	assert.Equal(t, 360, intervalMinutes(100, 10, 4))
}

func TestIntervalMinutes_ThirtyPercentBucket(t *testing.T) {
	// index 11 (11%) just crosses into the second bucket.
	assert.Equal(t, 1440, intervalMinutes(100, 11, 4))
	assert.Equal(t, 1440, intervalMinutes(100, 40, 4))
}

func TestIntervalMinutes_FortyPercentBucket(t *testing.T) {
	// index 41 (41%) crosses into the third bucket.
	assert.Equal(t, 10080, intervalMinutes(100, 41, 4))
	assert.Equal(t, 10080, intervalMinutes(100, 80, 4))
}

func TestIntervalMinutes_FinalBucket(t *testing.T) {
	// index 81 (81%) falls in the final bucket.
	assert.Equal(t, 43200, intervalMinutes(100, 81, 4))
	assert.Equal(t, 43200, intervalMinutes(100, 99, 4))
}

// --- intervalToString ---

func TestIntervalToString_Hours(t *testing.T) {
	assert.Equal(t, "1 hour(s)", intervalToString(60))
	assert.Equal(t, "12 hour(s)", intervalToString(720))
	assert.Equal(t, "23 hour(s)", intervalToString(1380))
}

func TestIntervalToString_OneDayBoundary(t *testing.T) {
	// 1440 minutes = 24 hours, which is exactly the day boundary.
	assert.Equal(t, "1 day(s)", intervalToString(1440))
}

func TestIntervalToString_Days(t *testing.T) {
	assert.Equal(t, "7 day(s)", intervalToString(10080))
	assert.Equal(t, "30 day(s)", intervalToString(43200))
}

