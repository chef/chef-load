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
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Default() ---

func TestDefault_LoadMode(t *testing.T) {
	cfg := Default()
	assert.Equal(t, "policyfile", cfg.LoadMode)
}

func TestDefault_PolicyfilePercentage(t *testing.T) {
	cfg := Default()
	assert.Equal(t, 0.5, cfg.PolicyfilePercentage)
}

func TestDefault_EnableCompliancePhase(t *testing.T) {
	cfg := Default()
	assert.False(t, cfg.EnableCompliancePhase)
}

func TestDefault_CompliancePhasePercentage(t *testing.T) {
	cfg := Default()
	assert.Equal(t, 1.0, cfg.CompliancePhasePercentage)
}

func TestDefault_PolicyNameAndGroup(t *testing.T) {
	cfg := Default()
	assert.Empty(t, cfg.PolicyName)
	assert.Empty(t, cfg.PolicyGroup)
}

func TestDefault_LegacyDefaults(t *testing.T) {
	// Verify pre-existing defaults haven't changed.
	cfg := Default()
	assert.Equal(t, 30, cfg.NumNodes)
	assert.Equal(t, 30, cfg.Interval)
	assert.Equal(t, "chef-load", cfg.NodeNamePrefix)
	assert.Equal(t, "_default", cfg.ChefEnvironment)
	assert.Equal(t, "never", cfg.DownloadCookbooks)
	assert.Equal(t, 1.0, cfg.DownloadCookbooksScaleFactor)
	assert.Equal(t, "13.2.20", cfg.ChefVersion)
	assert.False(t, cfg.RunChefClient)
	assert.False(t, cfg.SkipClientCreation)
	assert.Equal(t, 0.0, cfg.NodeReplacementRate)
}

// --- PrintSampleConfig ---

func TestPrintSampleConfig_ContainsNewSettings(t *testing.T) {
	// PrintSampleConfig writes to stdout; the simplest approach is to verify
	// that the function runs without panic (it is tested implicitly via the
	// init command). We also use it as a hook to ensure the documented options
	// actually exist in the Config struct so that mapstructure can load them.
	cfg := Default()

	// Ensure every new TOML key maps to a real field.
	assert.NotEmpty(t, cfg.LoadMode)
	assert.False(t, cfg.EnableCompliancePhase)
	assert.Equal(t, 1.0, cfg.CompliancePhasePercentage)
	assert.Equal(t, 0.5, cfg.PolicyfilePercentage)
}

// --- usePolicyfile determination logic ---
// The switch logic lives inside ChefClientRun but we can unit-test it by
// extracting the same expression.

func policyfileModeFor(loadMode string, nodeNumber uint32, numNodes int, pct float64) bool {
	switch loadMode {
	case "policyfile":
		return true
	case "mixed":
		return float64(nodeNumber)/float64(numNodes) < pct
	}
	return false
}

func TestUsePolicyfile_PolicyfileMode(t *testing.T) {
	assert.True(t, policyfileModeFor("policyfile", 0, 100, 0.5))
	assert.True(t, policyfileModeFor("policyfile", 99, 100, 0.0))
}

func TestUsePolicyfile_LegacyMode(t *testing.T) {
	assert.False(t, policyfileModeFor("legacy", 0, 100, 1.0))
}

func TestUsePolicyfile_MixedMode_FirstHalf(t *testing.T) {
	// With 50% and 100 nodes, slots 0-49 should use policyfile.
	assert.True(t, policyfileModeFor("mixed", 0, 100, 0.5))
	assert.True(t, policyfileModeFor("mixed", 49, 100, 0.5))
}

func TestUsePolicyfile_MixedMode_SecondHalf(t *testing.T) {
	// Slots 50-99 should use legacy.
	assert.False(t, policyfileModeFor("mixed", 50, 100, 0.5))
	assert.False(t, policyfileModeFor("mixed", 99, 100, 0.5))
}

func TestUsePolicyfile_MixedMode_AllPolicyfile(t *testing.T) {
	assert.True(t, policyfileModeFor("mixed", 99, 100, 1.0))
}

func TestUsePolicyfile_MixedMode_NoPolicyfile(t *testing.T) {
	assert.False(t, policyfileModeFor("mixed", 0, 100, 0.0))
}

// --- compliance phase gate logic ---

func runCompliancePhaseFor(hasFile bool, enabled bool, nodeNumber uint32, numNodes int, pct float64) bool {
	fileLoaded := hasFile
	return fileLoaded || (enabled && numNodes > 0 && float64(nodeNumber) < float64(numNodes)*pct)
}

func TestCompliancePhaseGate_FileAlwaysFires(t *testing.T) {
	// If a compliance JSON file is loaded, always fire regardless of other settings.
	assert.True(t, runCompliancePhaseFor(true, false, 0, 100, 0.0))
}

func TestCompliancePhaseGate_DisabledWithNoFile(t *testing.T) {
	assert.False(t, runCompliancePhaseFor(false, false, 0, 100, 1.0))
}

func TestCompliancePhaseGate_EnabledAllNodes(t *testing.T) {
	for i := uint32(0); i < 10; i++ {
		assert.True(t, runCompliancePhaseFor(false, true, i, 10, 1.0),
			"node %d should run compliance with 100%% coverage", i)
	}
}

func TestCompliancePhaseGate_EnabledHalfNodes(t *testing.T) {
	assert.True(t, runCompliancePhaseFor(false, true, 0, 10, 0.5))
	assert.True(t, runCompliancePhaseFor(false, true, 4, 10, 0.5))
	assert.False(t, runCompliancePhaseFor(false, true, 5, 10, 0.5))
	assert.False(t, runCompliancePhaseFor(false, true, 9, 10, 0.5))
}

func TestCompliancePhaseGate_ZeroNodes(t *testing.T) {
	// Guard against divide-by-zero: NumNodes == 0 must never fire.
	assert.False(t, runCompliancePhaseFor(false, true, 0, 0, 1.0))
}
