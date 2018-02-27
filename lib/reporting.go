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
	"net/http"
	"strings"
	"time"

	"github.com/go-chef/chef"
	uuid "github.com/satori/go.uuid"
)

const rubyDateTime = "2006-01-02 15:04:05 -0700"

func reportingRunStart(nodeClient chef.Client, nodeName string, runUUID uuid.UUID, startTime time.Time) (*http.Response, error) {
	body := map[string]string{
		"action":     "start",
		"run_id":     runUUID.String(),
		"start_time": startTime.Format(rubyDateTime),
	}

	res, err := apiRequest(nodeClient, nodeName, "POST", "reports/nodes/"+nodeName+"/runs", body, nil, map[string]string{"X-Ops-Reporting-Protocol-Version": "0.1.0"})
	return res, err
}

func reportingRunStop(nodeClient chef.Client, nodeName string, runUUID uuid.UUID, startTime time.Time, endTime time.Time, rl runList) (*http.Response, error) {
	body := map[string]interface{}{
		"action":          "end",
		"data":            map[string]interface{}{},
		"end_time":        endTime.Format(rubyDateTime),
		"resources":       []interface{}{},
		"run_list":        `["` + strings.Join(rl.toStringSlice(), `","`) + `"]`,
		"start_time":      startTime.Format(rubyDateTime),
		"status":          "success",
		"total_res_count": "0",
	}

	res, err := apiRequest(nodeClient, nodeName, "POST", "reports/nodes/"+nodeName+"/runs/"+runUUID.String(), body, nil, map[string]string{"X-Ops-Reporting-Protocol-Version": "0.1.0"})
	return res, err
}
