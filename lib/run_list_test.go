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

// --- runList.length ---

func TestRunListLength_Empty(t *testing.T) {
	assert.Equal(t, 0, runList{}.length())
}

func TestRunListLength_NonEmpty(t *testing.T) {
	rl := runList{{name: "base::default", itemType: "recipe"}}
	assert.Equal(t, 1, rl.length())
}

// --- runList.shift ---

func TestRunListShift_ReturnsFirstAndRemainder(t *testing.T) {
	rl := runList{
		{name: "base::default", itemType: "recipe"},
		{name: "web::setup", itemType: "recipe"},
	}
	first, rest := rl.shift()
	assert.Equal(t, "base::default", first.name)
	assert.Equal(t, "recipe", first.itemType)
	if !assert.Equal(t, 1, rest.length()) {
		return
	}
	assert.Equal(t, "web::setup", rest[0].name)
}

func TestRunListShift_SingleItem(t *testing.T) {
	rl := runList{{name: "base::default", itemType: "recipe"}}
	first, rest := rl.shift()
	assert.Equal(t, "base::default", first.name)
	assert.Equal(t, 0, rest.length())
}

// --- runList.toStringSlice ---

func TestRunListToStringSlice_Empty(t *testing.T) {
	assert.Empty(t, runList{}.toStringSlice())
}

func TestRunListToStringSlice_RecipeWithoutVersion(t *testing.T) {
	rl := runList{{name: "base::default", itemType: "recipe"}}
	assert.Equal(t, []string{"recipe[base::default]"}, rl.toStringSlice())
}

func TestRunListToStringSlice_RecipeWithVersion(t *testing.T) {
	rl := runList{{name: "base::default", itemType: "recipe", version: "1.2.3"}}
	assert.Equal(t, []string{"recipe[base::default@1.2.3]"}, rl.toStringSlice())
}

func TestRunListToStringSlice_Role(t *testing.T) {
	rl := runList{{name: "webserver", itemType: "role"}}
	assert.Equal(t, []string{"role[webserver]"}, rl.toStringSlice())
}

func TestRunListToStringSlice_Mixed(t *testing.T) {
	rl := runList{
		{name: "webserver", itemType: "role"},
		{name: "apache::default", itemType: "recipe"},
		{name: "hardening::ssh", itemType: "recipe", version: "2.0.0"},
	}
	want := []string{"role[webserver]", "recipe[apache::default]", "recipe[hardening::ssh@2.0.0]"}
	assert.Equal(t, want, rl.toStringSlice())
}

// --- parseRunList ---

func TestParseRunList_Empty(t *testing.T) {
	rl := parseRunList([]string{})
	assert.Equal(t, 0, rl.length())
}

func TestParseRunList_QualifiedRecipe(t *testing.T) {
	rl := parseRunList([]string{"recipe[apache::default]"})
	if !assert.Equal(t, 1, rl.length()) {
		return
	}
	assert.Equal(t, "apache::default", rl[0].name)
	assert.Equal(t, "recipe", rl[0].itemType)
	assert.Equal(t, "", rl[0].version)
}

func TestParseRunList_QualifiedRecipeWithVersion(t *testing.T) {
	rl := parseRunList([]string{"recipe[apache::default@2.1.0]"})
	if !assert.Equal(t, 1, rl.length()) {
		return
	}
	assert.Equal(t, "apache::default", rl[0].name)
	assert.Equal(t, "recipe", rl[0].itemType)
	assert.Equal(t, "2.1.0", rl[0].version)
}

func TestParseRunList_QualifiedRole(t *testing.T) {
	rl := parseRunList([]string{"role[webserver]"})
	if !assert.Equal(t, 1, rl.length()) {
		return
	}
	assert.Equal(t, "webserver", rl[0].name)
	assert.Equal(t, "role", rl[0].itemType)
}

func TestParseRunList_UnqualifiedRecipe(t *testing.T) {
	rl := parseRunList([]string{"base"})
	if !assert.Equal(t, 1, rl.length()) {
		return
	}
	assert.Equal(t, "base", rl[0].name)
	assert.Equal(t, "recipe", rl[0].itemType)
	assert.Equal(t, "", rl[0].version)
}

func TestParseRunList_UnqualifiedRecipeWithVersion(t *testing.T) {
	rl := parseRunList([]string{"base@1.0.0"})
	if !assert.Equal(t, 1, rl.length()) {
		return
	}
	assert.Equal(t, "base", rl[0].name)
	assert.Equal(t, "recipe", rl[0].itemType)
	assert.Equal(t, "1.0.0", rl[0].version)
}

func TestParseRunList_MultipleItems(t *testing.T) {
	rl := parseRunList([]string{
		"recipe[base::default]",
		"role[webserver]",
		"recipe[hardening::ssh@2.0.0]",
	})
	if !assert.Equal(t, 3, rl.length()) {
		return
	}
	assert.Equal(t, "recipe", rl[0].itemType)
	assert.Equal(t, "role", rl[1].itemType)
	assert.Equal(t, "2.0.0", rl[2].version)
}

func TestParseRunList_OrderPreserved(t *testing.T) {
	input := []string{"recipe[z::default]", "recipe[a::default]", "recipe[m::default]"}
	rl := parseRunList(input)
	if !assert.Equal(t, 3, rl.length()) {
		return
	}
	assert.Equal(t, "z::default", rl[0].name)
	assert.Equal(t, "a::default", rl[1].name)
	assert.Equal(t, "m::default", rl[2].name)
}

// --- parseRunLists ---

func TestParseRunLists_Empty(t *testing.T) {
	rls := parseRunLists([][]string{})
	assert.Empty(t, rls)
}

func TestParseRunLists_SingleList(t *testing.T) {
	rls := parseRunLists([][]string{{"recipe[base::default]"}})
	if !assert.Len(t, rls, 1) {
		return
	}
	assert.Equal(t, 1, rls[0].length())
}

func TestParseRunLists_MultipleLists(t *testing.T) {
	rls := parseRunLists([][]string{
		{"recipe[base::default]"},
		{"recipe[web::setup]", "role[webserver]"},
	})
	if !assert.Len(t, rls, 2) {
		return
	}
	assert.Equal(t, 1, rls[0].length())
	assert.Equal(t, 2, rls[1].length())
}
