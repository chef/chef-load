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

	"github.com/go-chef/chef"
)

// PolicyCookbookLock holds the version lock data for a single cookbook entry
// inside a policyfile's cookbook_locks section.
type PolicyCookbookLock struct {
	// Identifier is the SHA1 that uniquely identifies the cookbook artifact.
	// Used by the native policyfile API (policy_document_native_api = true).
	Identifier string `json:"identifier"`

	// DottedDecimalIdentifier is an x.y.z representation of the identifier
	// used by the compatibility-mode policyfile API.
	DottedDecimalIdentifier string `json:"dotted_decimal_identifier"`

	// Version is the semver version string of the locked cookbook.
	Version string `json:"version"`
}

// PolicyDocument represents the policyfile document returned by the Chef Server
// via GET /organizations/<org>/policy_groups/<group>/policies/<name>.
type PolicyDocument struct {
	RevisionID         string                        `json:"revision_id"`
	Name               string                        `json:"name"`
	RunList            []string                      `json:"run_list"`
	NamedRunLists      map[string][]string           `json:"named_run_lists,omitempty"`
	CookbookLocks      map[string]PolicyCookbookLock `json:"cookbook_locks"`
	DefaultAttributes  map[string]interface{}        `json:"default_attributes,omitempty"`
	OverrideAttributes map[string]interface{}        `json:"override_attributes,omitempty"`
}

// fetchPolicy retrieves the policyfile document from the Chef Server using the
// native policyfile API endpoint:
//
//	GET /organizations/<org>/policy_groups/<group>/policies/<name>
func fetchPolicy(nodeClient chef.Client, nodeName, chefVersion, policyGroup, policyName string, requests chan *request) (PolicyDocument, error) {
	var policy PolicyDocument
	url := "policy_groups/" + policyGroup + "/policies/" + policyName
	_, err := apiRequest(nodeClient, nodeName, chefVersion, "GET", url, nil, &policy, nil, requests) //nolint:bodyclose
	return policy, err
}

// parsePolicyRecipeSpec strips the "recipe[" prefix and trailing "]" from a
// policy run_list entry, returning the bare "cookbook::recipe" name.
// For example: "recipe[apache::default]" -> "apache::default".
// If the spec does not match the expected format it is returned as-is.
func parsePolicyRecipeSpec(spec string) string {
	if strings.HasPrefix(spec, "recipe[") && strings.HasSuffix(spec, "]") {
		return spec[7 : len(spec)-1]
	}
	return spec
}

// policyRunListRecipes extracts the bare recipe names (cookbook::recipe) from a
// policyfile run_list for use in node automatic attributes.
func policyRunListRecipes(policyRunList []string) []string {
	recipes := make([]string, 0, len(policyRunList))
	for _, spec := range policyRunList {
		recipes = append(recipes, parsePolicyRecipeSpec(spec))
	}
	return recipes
}

// resolveAndDownloadPolicyfileCookbooks fetches the cookbook artifact manifest for
// each entry in the policy's cookbook_locks from the Chef Server's native
// cookbook_artifacts endpoint:
//
//	GET /organizations/<org>/cookbook_artifacts/<name>/<identifier>
//
// When doDownload is true, individual cookbook files are downloaded with the
// given probability (0.0–1.0).
func resolveAndDownloadPolicyfileCookbooks(
	nodeClient *chef.Client,
	nodeName, chefVersion string,
	policy PolicyDocument,
	doDownload bool,
	downloadProbability float64,
	requests chan *request,
) {
	for name, lock := range policy.CookbookLocks {
		var ckbk cookbook
		url := "cookbook_artifacts/" + name + "/" + lock.Identifier
		_, _ = apiRequest(*nodeClient, nodeName, chefVersion, "GET", url, nil, &ckbk, nil, requests) //nolint:bodyclose
		if doDownload {
			ckbk.download(nodeClient, nodeName, chefVersion, downloadProbability, requests)
		}
	}
}
