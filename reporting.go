package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chef/chef"
	uuid "github.com/satori/go.uuid"
)

const rubyDateTime = "2006-01-02 15:04:05 -0700"

func reportingRunStart(nodeClient chef.Client, nodeName string, runUUID uuid.UUID, startTime time.Time) (*http.Response, error) {
	startRunBody := map[string]string{
		"action":     "start",
		"run_id":     runUUID.String(),
		"start_time": startTime.Format(rubyDateTime),
	}
	data, err := chef.JSONReader(startRunBody)
	if err != nil {
		fmt.Println(err)
	}

	res, err := apiRequest(nodeClient, "POST", "reports/nodes/"+nodeName+"/runs", data, nil, map[string]string{"X-Ops-Reporting-Protocol-Version": "0.1.0"})
	return res, err
}

func reportingRunStop(nodeClient chef.Client, nodeName string, runUUID uuid.UUID, startTime time.Time, endTime time.Time, rl runList) (*http.Response, error) {
	endRunBody := map[string]interface{}{
		"action":          "end",
		"data":            map[string]interface{}{},
		"end_time":        endTime.Format(rubyDateTime),
		"resources":       []interface{}{},
		"run_list":        `["` + strings.Join(rl.toStringSlice(), `","`) + `"]`,
		"start_time":      startTime.Format(rubyDateTime),
		"status":          "success",
		"total_res_count": "0",
	}
	data, err := chef.JSONReader(endRunBody)
	if err != nil {
		fmt.Println(err)
	}

	res, err := apiRequest(nodeClient, "POST", "reports/nodes/"+nodeName+"/runs/"+runUUID.String(), data, nil, map[string]string{"X-Ops-Reporting-Protocol-Version": "0.1.0"})
	return res, err
}
