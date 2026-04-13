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

// --- amountOfRequests.addRequest URL normalisation ---

func TestAddRequest_NormalizeBookshelfURL(t *testing.T) {
	a := make(amountOfRequests)
	a.addRequest(request{Method: "GET", Url: "/bookshelf/org/data/some/path", StatusCode: 200})

	normalized := request{Method: "GET", Url: "/bookshelf/<...>", StatusCode: 200}
	assert.Equal(t, uint64(1), a[normalized])
}

func TestAddRequest_NormalizeNodeURL(t *testing.T) {
	a := make(amountOfRequests)
	a.addRequest(request{Method: "PUT", Url: "/nodes/chef-load-42", StatusCode: 200})

	normalized := request{Method: "PUT", Url: "/nodes/chef-load-<N>", StatusCode: 200}
	assert.Equal(t, uint64(1), a[normalized])
}

func TestAddRequest_NormalizeNodeURLWithSubpath(t *testing.T) {
	a := make(amountOfRequests)
	a.addRequest(request{Method: "GET", Url: "/nodes/chef-load-7/runs", StatusCode: 200})

	normalized := request{Method: "GET", Url: "/nodes/chef-load-<N>/runs", StatusCode: 200}
	assert.Equal(t, uint64(1), a[normalized])
}

func TestAddRequest_NormalizeRolesURL(t *testing.T) {
	a := make(amountOfRequests)
	a.addRequest(request{Method: "GET", Url: "/roles/my_role", StatusCode: 200})

	normalized := request{Method: "GET", Url: "/roles/<ROLENAME>", StatusCode: 200}
	assert.Equal(t, uint64(1), a[normalized])
}

func TestAddRequest_NonMatchingURLUnchanged(t *testing.T) {
	a := make(amountOfRequests)
	r := request{Method: "GET", Url: "/environments/prod", StatusCode: 200}
	a.addRequest(r)

	assert.Equal(t, uint64(1), a[r])
}

func TestAddRequest_NormalizedURLsGroupedTogether(t *testing.T) {
	// Different node numbers should map to the same normalized bucket.
	a := make(amountOfRequests)
	a.addRequest(request{Method: "GET", Url: "/nodes/chef-load-1", StatusCode: 200})
	a.addRequest(request{Method: "GET", Url: "/nodes/chef-load-99", StatusCode: 200})

	normalized := request{Method: "GET", Url: "/nodes/chef-load-<N>", StatusCode: 200}
	assert.Equal(t, uint64(2), a[normalized])
}

func TestAddRequest_DifferentStatusCodesAreDistinct(t *testing.T) {
	// Requests with the same URL but different status codes are separate buckets.
	a := make(amountOfRequests)
	a.addRequest(request{Method: "GET", Url: "/environments/prod", StatusCode: 200})
	a.addRequest(request{Method: "GET", Url: "/environments/prod", StatusCode: 404})

	assert.Equal(t, uint64(1), a[request{Method: "GET", Url: "/environments/prod", StatusCode: 200}])
	assert.Equal(t, uint64(1), a[request{Method: "GET", Url: "/environments/prod", StatusCode: 404}])
}
