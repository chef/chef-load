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

	"github.com/stretchr/testify/assert"
)

// --- parsePolicyRecipeSpec ---

func TestParsePolicyRecipeSpec_QualifiedRecipe(t *testing.T) {
	assert.Equal(t, "apache::default", parsePolicyRecipeSpec("recipe[apache::default]"))
}

func TestParsePolicyRecipeSpec_DefaultRecipe(t *testing.T) {
	// Recipes without an explicit recipe name are expressed as cookbook::default.
	assert.Equal(t, "base::default", parsePolicyRecipeSpec("recipe[base::default]"))
}

func TestParsePolicyRecipeSpec_NoWrapper(t *testing.T) {
	// If the spec does not match the expected format it is returned as-is.
	assert.Equal(t, "apache::default", parsePolicyRecipeSpec("apache::default"))
}

func TestParsePolicyRecipeSpec_EmptyString(t *testing.T) {
	assert.Equal(t, "", parsePolicyRecipeSpec(""))
}

func TestParsePolicyRecipeSpec_OnlyPrefix(t *testing.T) {
	// "recipe[" without the closing "]" is not a valid spec and must be returned unchanged.
	assert.Equal(t, "recipe[apache::default", parsePolicyRecipeSpec("recipe[apache::default"))
}

// --- policyRunListRecipes ---

func TestPolicyRunListRecipes_Empty(t *testing.T) {
	assert.Equal(t, []string{}, policyRunListRecipes([]string{}))
}

func TestPolicyRunListRecipes_SingleEntry(t *testing.T) {
	got := policyRunListRecipes([]string{"recipe[base::default]"})
	assert.Equal(t, []string{"base::default"}, got)
}

func TestPolicyRunListRecipes_MultipleEntries(t *testing.T) {
	input := []string{
		"recipe[base::default]",
		"recipe[apache::default]",
		"recipe[hardening::ssh]",
	}
	want := []string{"base::default", "apache::default", "hardening::ssh"}
	assert.Equal(t, want, policyRunListRecipes(input))
}

func TestPolicyRunListRecipes_PreservesOrder(t *testing.T) {
	input := []string{
		"recipe[z::default]",
		"recipe[a::default]",
		"recipe[m::default]",
	}
	want := []string{"z::default", "a::default", "m::default"}
	assert.Equal(t, want, policyRunListRecipes(input))
}
